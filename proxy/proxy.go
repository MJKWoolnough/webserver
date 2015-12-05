package proxy

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
)

func New() {

}

func startServer() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}
	for {
		c, err := l.Accept()
		if err != nil {
			continue
		}
		go handleConn(c, false)
	}
}

const MaxXHeaderSize = http.DefaultMaxHeaderBytes

var (
	HeadersTooLarge = []byte("HTTP/1.0 413\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	BadRequest      = []byte("HTTP/1.0 400\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
)

var pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1+8+4+MaxHeaderSize)
	},
}

func handleConn(c net.Conn, encrypted bool) {
	buf := pool.Get().([]byte)
	defer pool.Put(buf)
	defer c.Close()
	var hostname string
	if encrypted {
		buf[0] = 1
		recordHeader := buf[1+8+4 : 1+8+4+5]
		_, err := io.ReadFull(c, dataHeader)
		if err != nil || recordHeader[0] == 0x80 {
			return
		}
		dataLength := int(recordHeader[3])<<8 | int(recordHeader[4])
		if dataLength < 42 || dataLength > MaxHeaderSize {
			return
		}
		data := buf[1+8+4+5 : 1+8+4+5+dataLength]
		_, err = io.ReadFull(c, data)
		if err != nil {
			return
		}
		buf = buf[:1+8+4+5+dataLength]

		sessionIDLen := int(data[38])
		if sessionIDLen > 32 || len(data) < 39+int(sessionIDLen) {
			goto Skip
		}
		data = data[39+sessionIDLen:]
		if len(data) < 2 {
			goto Skip
			return
		}

		cipherSuiteLen := int(data[0])<<8 | int(data[1])
		if cipherSuiteLen%2 == 1 || len(data) < 2+cipherSuiteLen {
			goto Skip
		}
		data = data[2+cipherSuiteLen:]

		if len(data) < 1 {
			goto Skip
		}
		compressionMethodsLen := int(data[0])
		if len(data) < 1+compressionMethodsLen {
			goto Skip
		}
		data = data[1+compressionMethodsLen:]

		if len(data) > 0 {
			if len(data) < 2 {
				goto Skip
			}
			extensionsLength := int(data[0])<<8 | int(data[1])
			if extensionsLength != len(data) {
				goto Skip
			}
		ExtLoop:
			for len(data) != 0 {
				if len(data) < 4 {
					goto Skip
				}
				extension := uint16(data[0])<<8 | uint16(data[1])
				length := int(data[2])<<8 | int(data[3])
				data = data[4:]
				if len(data) < length {
					goto Skip
				}
				if extension == 0 { //serverName
					d := data[:length]
					if len(d) < 2 {
						goto Skip
					}
					namesLen := int(d[0])<<8 | int(d[1])
					d = d[2:]
					if len(d) != namesLen {
						goto Skip
					}
					for len(d) > 0 {
						if len(d) < 3 {
						}
						nameType := d[0]
						nameLen := int(d[1])<<8 | int(d[2])
						d = d[3:]
						if len(d) < nameLen {
							goto Skip
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
	} else {
		buf[0] = 0
		buf = buf[:1+8+4]
		var (
			last = len(buf)
			char = make([]byte, 1, 1)
			size int
		)
		for size < MaxHeaderSize {
			n, err := c.Read(char)
			if err != nil {
				c.Write(BadRequest)
				return
			}
			if n != 1 {
				continue
			}
			size++
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
			if string(bytes.TrimSpace(line[:p])) != "Host" {
				continue
			}
			hostname = string(bytes.TrimSpace(line[p+1:]))
			break
		}
		if size == MaxHeaderSize {
			c.Write(HeadersTooLarge)
			c.Close()
			return
		}
	}
Skip:

	nf := c.(interface {
		File() (*os.File, error)
	})
	f, _ := nf.File()
	binary.LittleEndian.PutUint64(buf[1:1+8], uint64(f.Fd()))
	binary.LittleEndian.PutUint32(buf[1+8:1+8+4], uint32(len(buf)-(1+8+4)))
	toHost(hostname, buf)
}

func toHost(hostname string, buf []byte) {
	//get host
	//send buf to host
}
