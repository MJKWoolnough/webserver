package proxy // import "vimagination.zapto.org/webserver/proxy"

import (
	"bytes"
	"io"
	"net"
	"strings"
	"sync"

	"vimagination.zapto.org/byteio"
	"vimagination.zapto.org/memio"
)

// MaxHeaderSize represents the maximum size that the proxy will look throught to find a Host header
const MaxHeaderSize = 8 << 10 // 8KB

// Two preconstructed responses to certain errors
var (
	HeadersTooLarge = []byte("HTTP/1.0 413\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	BadRequest      = []byte("HTTP/1.0 400\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
)

var pool = sync.Pool{
	New: func() interface{} {
		return make(memio.Buffer, MaxHeaderSize+1)
	},
}

func (p *Proxy) run(l net.Listener, encrypted bool) error {
	for {
		c, err := l.Accept()
		if err != nil {
			if oe, ok := err.(*net.OpError); ok {
				if oe.Temporary() {
					continue
				}
			}
			return err
		}
		go p.handleConn(c, encrypted)
	}
}

func (p *Proxy) handleConn(c net.Conn, encrypted bool) {
	buf := pool.Get().(memio.Buffer)
	defer pool.Put(buf)
	defer c.Close()
	var (
		hostname   string
		readLength int
	)
	if encrypted {
		hostname, readLength = readEncrypted(c, buf)
	} else {
		hostname, readLength = readHTTP(c, buf)
	}
	if readLength == MaxHeaderSize {
		c.Write(HeadersTooLarge)
		return
	}

	if readLength < 0 {
		c.Write(BadRequest)
		return
	}

	pos := strings.IndexByte(hostname, ':')
	if pos >= 0 {
		hostname = hostname[:pos]
	}

	p.mu.RLock()
	h, ok := p.hostnames[hostname]
	if !ok {
		h = p.defaultHost
	}
	p.mu.RUnlock()
	t := h.getTransfer(encrypted)
	t.Transfer(c, buf[:readLength])
}

func readEncrypted(c net.Conn, buf memio.Buffer) (string, int) {
	_, err := io.ReadFull(c, buf[:5])
	if err != nil {
		return "", -1
	}
	r := byteio.StickyBigEndianReader{
		Reader: &buf,
	}
	if r.ReadUint8() != 22 {
		//not a handshake, error out
		return "", -1
	}

	buf = buf[1:] // skip major version
	buf = buf[1:] // skip minor version

	length := r.ReadUint16()

	if len(buf) < int(length) {
		return "", -1
	} else {
		buf = buf[:length]
	}
	_, err = io.ReadFull(c, buf)
	if err != nil {
		return "", -1
	}

	if r.ReadUint8() != 1 {
		// not a client_hello, error out
		return "", -1
	}

	l := int(r.ReadUint24())
	if l != len(buf) {
		// incorrect length
		return "", -1
	}

	buf = buf[1:] // skip major version
	buf = buf[1:] // skip minor version

	buf = buf[4:]  // skip gmt_unix_time
	buf = buf[28:] // skip random_bytes

	sessionLength := r.ReadUint8()
	if sessionLength > 32 || len(buf) < int(sessionLength) {
		// invalid length
		return "", -1
	}
	buf = buf[sessionLength:] // skip session id

	cipherSuiteLength := r.ReadUint16()
	if cipherSuiteLength == 0 || len(buf) < int(cipherSuiteLength) {
		// invalid length
		return "", -1
	}
	buf = buf[cipherSuiteLength:] // skip cipher suites

	compressionMethodLength := r.ReadUint8()
	if compressionMethodLength < 1 {
		// invalid length
		return "", -1
	}
	buf = buf[compressionMethodLength:] // skip compression methods

	extsLength := r.ReadUint16()
	if len(buf) < int(extsLength) {
		// invalid length
		return "", -1
	}
	buf = buf[:extsLength]

	for len(buf) > 0 {
		extType := r.ReadUint16()
		extLength := r.ReadUint16()
		if len(buf) < int(extLength) {
			// invalid length
			return "", -1
		}
		if extType == 0 { // server_name
			l := r.ReadUint16()
			if l != extLength-2 {
				// invalid length
				return "", -1
			}

			buf = buf[1:] // skip name_type

			nameLength := r.ReadUint16()
			if len(buf) < int(nameLength) {
				// invalid length
				return "", -1
			}
			return string(buf[:nameLength]), 5 + int(length)
		} else {
			buf = buf[extLength:]
		}
	}
	return "", 5 + int(length)
}

func readHTTP(c net.Conn, buf []byte) (string, int) {
	var (
		last       int
		char       = make([]byte, 1, 1)
		readLength int
	)
	buf = buf[:0]
	for readLength < MaxHeaderSize {
		n, err := c.Read(char)
		if err != nil {
			readLength = -1
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
		return string(bytes.TrimSpace(line[p+1:])), readLength
	}
	return
}
