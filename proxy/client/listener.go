package client

import (
	"encoding/binary"
	"errors"
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var openConnections sync.WaitGroup

type listener struct {
	unix *net.UnixConn
}

func newListener(socketFD uintptr) (net.Listener, error) {
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
	if ka, ok := c.(keepAlive); ok {
		ka.SetKeepAlive(true)
		ka.SetKeepAlivePeriod(3 * time.Minute)
	}
	conn := &conn{
		buf:  buf,
		Conn: c,
	}
	openConnections.Add(1)
	runtime.SetFinalizer(conn, connClose)
	return conn, nil
}

type keepAlive interface {
	SetKeepAlive(bool) error
	SetKeepAlivePeriod(time.Duration) error
}

func connClose(*conn) {
	openConnections.Done()
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
	closed bool
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

func (c *conn) Close() error {
	if c.closed {
		return nil
	}
	runtime.SetFinalizer(c, nil)
	connClose(nil)
	c.closed = true
	return c.Conn.Close()
}

// Errors
var (
	ErrInvalidSocket = errors.New("invalid socket type")
	ErrInvalidSCM    = errors.New("invalid number of socket control messages")
	ErrInvalidFDs    = errors.New("invalid number of file descriptors")
)
