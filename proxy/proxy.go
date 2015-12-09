package proxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
)

type file interface {
	File() (*os.File, error)
}

type host struct {
	sync.Mutex
	f *os.File
	*net.UnixConn
}

func newHost() (*host, error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	u := os.NewFile(uintptr(fds[0]), "")
	f := os.NewFile(uintptr(fds[1]), "")
	fc, err := net.FileConn(u)
	if err != nil { // really shouldn't happen!
		u.Close()
		f.Close()
		return nil, err
	}
	uc, ok := fc.(*net.UnixConn)
	if !ok { // ... again, really shouldn't happen
		return nil, ErrBadSocket
	}
	return &host{
		f:        f,
		UnixConn: uc,
	}, nil
}

func (h *host) transfer(c net.Conn, buf []byte) error {
	f, err := c.(file).File()
	if err != nil {
		return err
	}
	length := make([]byte, 4)
	binary.LittleEndian.PutUint32(length, uint32(len(buf)))
	h.Lock()
	defer h.Unlock()
	if _, _, err = h.WriteMsgUnix(length, syscall.UnixRights(int(f.Fd())), nil); err != nil {
		return err
	}
	_, err = h.Write(buf)
	return err
}

type Proxy struct {
	l         net.Listener
	encrypted bool

	mu    sync.RWMutex
	hosts map[string]*host
}

// New creates a new proxy for the given Listener
func New(l net.Listener, encrypted bool) *Proxy {
	return &Proxy{
		l:         l,
		encrypted: encrypted,
		hosts:     make(map[string]*host),
	}
}

// Get retrieves the connection for a given host
func (p *Proxy) Get(name string) *os.File {
	p.mu.RLock()
	defer p.mu.RUnlock()
	h, ok := p.hosts[name]
	if !ok {
		return nil
	}
	return h.f
}

// AddHost adds a new host, with its aliases, and returns a File to be passed
// to a child process
func (p *Proxy) AddHost(names ...string) (*os.File, error) {
	h, err := newHost()
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	for _, name := range names {
		p.hosts[name] = h
	}
	p.mu.Unlock()
	return h.f, nil
}

// RemoveHost removes a single alias for a server.
//
// This does not close any connections. If this is the last alias for the
// server, you should use CloseHost instead.
func (p *Proxy) RemoveHost(name string) error {
	if name == "" {
		return ErrNoRemoveDefault
	}
	p.mu.Lock()
	delete(p.hosts, name)
	p.mu.Unlock()
	return nil
}

// CloseHost removes all of the aliases for the server and closes the
// connection
func (p *Proxy) CloseHost(name string) error {
	p.mu.Lock()
	defer p.mu.Lock()
	h, ok := p.hosts[name]
	if !ok {
		return nil
	}
	for n, nh := range p.hosts {
		if h == nh {
			delete(p.hosts, n)
		}
	}
	return h.Close()
}

func (p *Proxy) Start() error {
	p.mu.RLock()
	_, ok := p.hosts[""]
	p.mu.RUnlock()
	if !ok {
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
		go p.handleConn(c)
	}
}

func (p *Proxy) Run() error {
	p.mu.RLock()
	_, ok := p.hosts[""]
	p.mu.RUnlock()
	if !ok {
		return ErrNoDefault
	}
	return p.run()
}

func (p *Proxy) Close() error {
	return p.l.Close()
}

const MaxHeaderSize = 1 << 13 // 8KB

var (
	HeadersTooLarge = []byte("HTTP/1.0 413\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	BadRequest      = []byte("HTTP/1.0 400\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
)

var pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 8+4+MaxHeaderSize+1)
	},
}

func (p *Proxy) handleConn(c net.Conn) {
	buf := pool.Get().([]byte)
	defer pool.Put(buf)
	defer c.Close()
	var (
		hostname   string
		readLength int
	)
	if p.encrypted {
		hostname, readLength = readEncrypted(c, buf)
	} else {
		hostname, readLength = readHTTP(c, buf)
	}
	if readLength == MaxHeaderSize {
		c.Write(HeadersTooLarge)
		return
	}

	pos := strings.IndexByte(hostname, ':')
	if pos >= 0 {
		hostname = hostname[:pos]
	}

	p.mu.RLock()
	h, ok := p.hosts[hostname]
	if !ok {
		h = p.hosts[""]
	}
	p.mu.RUnlock()
	h.transfer(c, buf[:readLength])
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
	buf = buf[:0]
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
	ErrNoDefault       = errors.New("no default set")
	ErrNoRemoveDefault = errors.New("cannot remove default host, only replace/update")
	ErrBadSocket       = errors.New("bad socket type")
)
