package client

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var (
	proxyHTTPSocket, proxyHTTPSSocket net.Listener
	server                            = new(http.Server)

	mu               sync.RWMutex
	started, stopped bool
	serverError      error

	wait = make(chan struct{})
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
	// setup signals
}

func Setup(s *http.Server) error {
	if started {
		return ErrRunning
	}
	if proxyHTTPSocket == nil && proxyHTTPSSocket == nil {
		return ErrNoSocket
	}
	if s == nil {
		s = new(http.Server)
	}
	server = s
	return nil
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

func run() {
	var wg sync.WaitGroup
	mu.Lock()
	started = true
	mu.Unlock()
	if proxyHTTPSocket != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			server.Serve(proxyHTTPSocket)
		}()
	}
	if proxyHTTPSSocket != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if server.TLSConfig == nil {
				server.Serve(proxyHTTPSSocket)
			} else {
				server.Serve(tls.NewListener(proxyHTTPSSocket, server.TLSConfig))
			}

		}()
	}

	wg.Wait()
	close(wait)
}

func Run() error {
	mu.RLock()
	s1 := started
	s2 := stopped
	mu.RUnlock()
	if s1 {
		return ErrRunning
	}
	if s2 {
		return ErrStopped
	}
	go run()
	<-wait
	return nil
}

func Start() error {
	mu.RLock()
	s1 := started
	s2 := stopped
	mu.RUnlock()
	if s1 {
		return ErrRunning
	}
	if s2 {
		return ErrStopped
	}
	go run()
	return nil
}

func Wait() error {
	mu.RLock()
	s := started
	mu.RUnlock()
	if !s {
		return ErrNotRunning
	}
	<-wait
	openConnections.Wait()
	return nil
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if stopped {
		return ErrStopped
	}
	stopped = true
	if started {
		err := proxyHTTPSocket.Close()
		if e := proxyHTTPSSocket.Close(); e != nil {
			return e
		}
		return err
	}
	return nil
}

// Errors
var (
	ErrNoSocket   = errors.New("no sockets setup")
	ErrRunning    = errors.New("already running")
	ErrNotRunning = errors.New("not running")
	ErrStopped    = errors.New("stopped")
)
