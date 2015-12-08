package client

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
)

type addr struct{}

func (addr) Network() string {
	return "proxy"
}

func (addr) String() string {
	return "proxy"
}

type conn struct {
	net.Conn
	buf []byte
}

func (c *conn) Read(b []byte) (int, error) {
	if len(c.buf) == 0 {
		return c.Conn.Read(b)
	}
	n := copy(b, c.buf)
	c.buf = c.buf[n:]
	if len(c.buf) == 0 {
		c.buf = nil
	}
	if n < len(b) {
		m, err := c.Conn.Read(b[n:])
		return n + m, err
	}
	return n, nil
}

type listener struct {
	data <-chan *conn
}

func (l *listener) Accept() (net.Conn, error) {
	c, ok := <-l.data
	if !ok {
		return nil, &net.OpError{
			Op:     "accept",
			Net:    "proxy",
			Source: addr{},
			Addr:   addr{},
			Err:    ErrClosing,
		}
	}
	return c, nil
}

func (l *listener) Close() error {
	return nil
}

func (l *listener) Addr() net.Addr {
	return addr{}
}

func New(r io.Reader) (net.Listener, net.Listener) {
	nc := make(chan *conn)
	ec := make(chan *conn)

	go run(r, nc, ec)

	return &listener{nc}, &listener{ec}
}

func run(r io.Reader, nc, ec chan *conn) {
	var (
		connType [1]byte
		fd       [8]byte
		length   [4]byte
		err      error
	)
	for {
		_, err = r.Read(connType[:])
		_, err = r.Read(fd[:])
		_, err = r.Read(length[:])
		buf := make([]byte, int(binary.LittleEndian.Uint32(length[:])))
		_, err = r.Read(buf)
		if err != nil {
			continue
		}

		var fc net.Conn

		fc, err = net.FileConn(os.NewFile(uintptr(binary.LittleEndian.Uint64(fd[:])), ""))

		c := &conn{
			Conn: fc,
			buf:  buf,
		}

		if connType[0] == 0 {
			nc <- c
		} else {
			ec <- c
		}
	}

}

// Errors
var ErrClosing = errors.New("use of closed network connection")
