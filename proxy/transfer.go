package proxy

import (
	"encoding/binary"
	"errors"
	"net"
	"os"
	"sync"
	"syscall"
)

type transfer struct {
	mu sync.Mutex
	f  *os.File
	c  *net.UnixConn
}

func newTransfer() (*transfer, error) {
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
	return &transfer{
		f: f,
		c: uc,
	}, nil

}

func (t *transfer) Transfer(c net.Conn, buf []byte) error {
	f, err := c.(file).File()
	if err != nil {
		return err
	}
	length := make([]byte, 4)
	binary.LittleEndian.PutUint32(length, uint32(len(buf)))
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, _, err = t.c.WriteMsgUnix(length, syscall.UnixRights(int(f.Fd())), nil); err != nil {
		return err
	}
	_, err = t.c.Write(buf)
	return err
}

func (t *transfer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	err := t.c.Close()
	t.f.Close()
	return err
}

// Errors
var ErrBadSocket = errors.New("bad socket type")
