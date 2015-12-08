package proxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
)

type Proxy struct {
	l   net.Listener
	ssl bool

	mu          sync.RWMutex
	hosts       map[string]io.Writer
	defaultHost string
}

func New(l net.Listener) *Proxy {
	return &Proxy{
		l:     l,
		ssl:   false,
		hosts: make(map[string]io.Writer),
	}
}

func (p *Proxy) Update(name string, w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hosts[name] = w
	if p.defaultHost == "" {
		p.defaultHost = name
	}
}

func (p *Proxy) Default(name string) error {
	if name == "" {
		return ErrNoDefault
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.hosts[name]; !ok {
		return ErrNoDefault
	}
	p.defaultHost = name
	return nil
}

func (p *Proxy) Remove(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.defaultHost == name {
		return ErrNoRemoveDefault
	}
	delete(p.hosts, name)
	return nil
}

func (p *Proxy) Start() error {
	if p.defaultHost == "" {
		return ErrNoDefault
	}
	go p.run()
	return nil
}

func (p *Proxy) run() error {
	for {
		c, err := p.l.Accept()
		if err != nil {
			if oe, ok := err.(*net.OpError); ok {
				if oe.Temporary() {
					continue
				}
			}
			return err
		}
		go p.handleConn(c, p.ssl)
	}
}

func (p *Proxy) Run() error {
	if p.defaultHost == "" {
		return ErrNoDefault
	}
	return p.run()
}

func (p *Proxy) Close() error {
	return p.l.Close()
}

const MaxHeaderSize = http.DefaultMaxHeaderBytes

var (
	HeadersTooLarge = []byte("HTTP/1.0 413\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	BadRequest      = []byte("HTTP/1.0 400\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
)

var pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1+8+4+MaxHeaderSize+1)
	},
}

func (p *Proxy) handleConn(c net.Conn, encrypted bool) {
	buf := pool.Get().([]byte)
	defer pool.Put(buf)
	defer c.Close()
	var (
		hostname   string
		readLength int
	)
	if encrypted {
		buf[0] = 1
		hostname, readLength = readEncrypted(c, buf[1+8+4:])
	} else {
		buf[0] = 0
		hostname, readLength = readHTTP(c, buf[1+8+4:])
	}
	if readLength == MaxHeaderSize {
		c.Write(HeadersTooLarge)
		return
	}

	nf := c.(interface {
		File() (*os.File, error)
	})
	f, _ := nf.File()
	binary.LittleEndian.PutUint64(buf[1:1+8], uint64(f.Fd()))
	binary.LittleEndian.PutUint32(buf[1+8:1+8+4], uint32(readLength))

	p.mu.RLock()
	h, ok := p.hosts[hostname]
	if !ok {
		h = p.hosts[p.defaultHost]
	}
	p.mu.RUnlock()
	if _, err := h.Write(buf[:1+8+4+readLength]); err != nil {
		f.Close()
	}
}

func readEncrypted(c net.Conn, buf []byte) (hostname string, readLength int) {
	recordHeader := buf[:5]
	_, err := io.ReadFull(c, recordHeader)
	if err != nil || recordHeader[0] == 0x80 {
		return
	}
	readLength = 5
	dataLength := int(recordHeader[3])<<8 | int(recordHeader[4])
	if dataLength < 42 || dataLength > MaxHeaderSize {
		return
	}
	readLength += dataLength
	data := buf[5:readLength]
	_, err = io.ReadFull(c, data)
	if err != nil {
		return
	}
	buf = buf[:1+8+4+5+dataLength]

	sessionIDLen := int(data[38])
	if sessionIDLen > 32 || len(data) < 39+int(sessionIDLen) {
		return
	}
	data = data[39+sessionIDLen:]
	if len(data) < 2 {
		return
	}

	cipherSuiteLen := int(data[0])<<8 | int(data[1])
	if cipherSuiteLen%2 == 1 || len(data) < 2+cipherSuiteLen {
		return
	}
	data = data[2+cipherSuiteLen:]

	if len(data) < 1 {
		return
	}
	compressionMethodsLen := int(data[0])
	if len(data) < 1+compressionMethodsLen {
		return
	}
	data = data[1+compressionMethodsLen:]

	if len(data) > 0 {
		if len(data) < 2 {
			return
		}
		extensionsLength := int(data[0])<<8 | int(data[1])
		if extensionsLength != len(data) {
			return
		}
	ExtLoop:
		for len(data) != 0 {
			if len(data) < 4 {
				return
			}
			extension := uint16(data[0])<<8 | uint16(data[1])
			length := int(data[2])<<8 | int(data[3])
			data = data[4:]
			if len(data) < length {
				return
			}
			if extension == 0 { //serverName
				d := data[:length]
				if len(d) < 2 {
					return
				}
				namesLen := int(d[0])<<8 | int(d[1])
				d = d[2:]
				if len(d) != namesLen {
					return
				}
				for len(d) > 0 {
					if len(d) < 3 {
					}
					nameType := d[0]
					nameLen := int(d[1])<<8 | int(d[2])
					d = d[3:]
					if len(d) < nameLen {
						return
					}
					if nameType == 0 {
						hostname = string(d[:nameLen])
						break ExtLoop
					}
					d = d[nameLen:]
				}
			}
		}
	}
	return
}

func readHTTP(c net.Conn, buf []byte) (hostname string, readLength int) {
	var (
		last int
		char = make([]byte, 1, 1)
	)
	for readLength < MaxHeaderSize {
		n, err := c.Read(char)
		if err != nil {
			c.Write(BadRequest)
			return
		}
		if n != 1 {
			continue
		}
		readLength++
		buf = append(buf, char[0])
		if char[0] != '\n' {
			continue
		}
		line := buf[last:]
		if len(line) == 2 && line[0] == '\r' && line[1] == '\n' {
			break
		}
		last = len(buf)
		p := bytes.IndexByte(line, ':')
		if p < 0 {
			continue
		}
		if headerName := bytes.TrimSpace(line[:p]); len(headerName) != 4 || headerName[0] != 'H' || headerName[1] != 'o' || headerName[2] != 's' || headerName[3] != 't' {
			continue
		}
		hostname = string(bytes.TrimSpace(line[p+1:]))
		break
	}
	return
}

// Errors
var (
	ErrNoDefault       = errors.New("default host empty")
	ErrNoRemoveDefault = errors.New("cannot remove default host")
)
