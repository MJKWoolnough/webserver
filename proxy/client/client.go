package client

import (
	"encoding/binary"
	"errors"
	"net"
	"os"
	"syscall"
)

type listener struct {
	unix *net.UnixConn
}

func Listen(socketFD uintptr) (net.Listener, error) {
	c, err := net.FileConn(os.NewFile(socketFD, ""))
	if err != nil {
		return nil, err
	}
	u, ok := c.(*net.UnixConn)
	if !ok {
		return nil, ErrInvalidSocket
	}
	return &listener{unix: u}, nil
}

func (l *listener) Accept() (net.Conn, error) {
	length := make([]byte, 4)

	oob := make([]byte, syscall.CmsgSpace(4))

	_, _, _, _, err := l.unix.ReadMsgUnix(length, oob)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, binary.LittleEndian.Uint32(length))
	_, err = l.unix.Read(buf)
	if err != nil {
		return nil, err
	}
	msg, err := syscall.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, err
	}
	if len(msg) != 1 {
		return nil, ErrInvalidSCM
	}
	fd, err := syscall.ParseUnixRights(&msg[0])
	if err != nil {
		return nil, err
	}
	if len(fd) != 1 {
		return nil, ErrInvalidFDs
	}
	c, err := net.FileConn(os.NewFile(uintptr(fd[0]), ""))
	if err != nil {
		return nil, err
	}
	return &conn{
		buf:  buf,
		Conn: c,
	}, nil
}

func (l *listener) Close() error {
	return l.unix.Close()
}

func (l *listener) Addr() net.Addr {
	return l.unix.LocalAddr()
}

type conn struct {
	buf []byte
	net.Conn
}

func (c *conn) Read(b []byte) (int, error) {
	if c.buf == nil {
		return c.Conn.Read(b)
	}
	n := copy(b, c.buf)
	c.buf = c.buf[n:]
	if len(c.buf) == 0 {
		c.buf = nil
		m, err := c.Conn.Read(b[n:])
		return n + m, err
	}
	return n, nil
}

// Errors
var (
	ErrInvalidSocket = errors.New("invalid socket type")
	ErrInvalidSCM    = errors.New("invalid number of socket control messages")
	ErrInvalidFDs    = errors.New("invalid number of file descriptors")
)
