package client

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
)

var (
	proxyHTTPSocket, proxyHTTPSSocket net.Listener
	addr                              string
	server                            = new(http.Server)
)

func init() {
	if hfd, ok := os.LookupEnv("proxyHTTPSocket"); ok {
		fd, _ := strconv.ParseUint(hfd, 10, 0)
		l, err := newListener(uintptr(fd))
		if err == nil { // panic???
			proxyHTTPSocket = l
		}
		os.Unsetenv("proxyHTTPSocket")
	}
	if sfd, ok := os.LookupEnv("proxyHTTPSSocket"); ok {
		fd, _ := strconv.ParseUint(sfd, 10, 0)
		l, err := newListener(uintptr(fd))
		if err == nil { // panic???
			proxyHTTPSSocket = l
		}
		os.Unsetenv("proxyHTTPSSocket")
	}
	if paddr, ok := os.LookupEnv("proxyAddr"); ok {
		addr = paddr
		os.Unsetenv("proxyAddr")
	}
	// setup signals
}

func Setup(s *http.Server) error {
	if started {
		return ErrRunning
	}
	if proxyHTTPSocketFD == 0 && proxySSLSocketFD == 0 {
		return ErrNoSocket
	}
	if s == nil {
		sever = new(server)
	}
}

func SetupTLS(s *http.Server, certFile, keyFile string) error {
	if started {
		return ErrRunning
	}
	if len(s.TLSConfig.Certificates) == 0 || certFile != "" || keyFile != "" {
		var err error
		s.TLSConfig.Certificates = make([]tls.Certificate, 1)
		s.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
	}
	return Setup(s)
}

func Run() error {
	if started {
		return ErrRunning
	}
	return nil
}

func Start() error {
	if started {
		return ErrRunning
	}
	return nil
}

func Wait() error {
	if !started {
		return ErrNotRunning
	}
	return nil
}

func Close() error {
	if started {

	}
	return nil
}

// Errors
var (
	ErrNoSocket   = errors.New("no sockets setup")
	ErrRunning    = errors.New("already running")
	ErrNotRunning = errors.New("not running")
)
