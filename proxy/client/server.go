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

	wait = make(chan error)
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

// Setup will setup the client with the given http Server.
//
// This is optional as it will use the http DefaultServer by default
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

// SetupTLS is used to setup the http Server and tls information
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

// SetupTest will bypass the proxy and create a listener itself from the given
// details.
//
// Useful when testing a server and don't want to run the full proxy to do so
func SetupTest(lnet, laddr string) error {
	l, err := net.Listen(lnet, laddr)
	if err != nil {
		return err
	}
	proxyHTTPSocket = l
	return nil
}

func run() {
	var wg sync.WaitGroup
	mu.Lock()
	started = true
	mu.Unlock()
	ec := make(chan error, 2)
	if proxyHTTPSocket != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ec <- server.Serve(proxyHTTPSocket)
		}()
	}
	if proxyHTTPSSocket != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if server.TLSConfig == nil {
				ec <- server.Serve(proxyHTTPSSocket)
			} else {
				ec <- server.Serve(tls.NewListener(proxyHTTPSSocket, server.TLSConfig))
			}

		}()
	}
	wg.Wait()
	close(ec)
	err1 := <-ec
	err2 := <-ec
	if err1 != nil {
		wait <- err1
	} else if err2 != nil {
		wait <- err2
	}
	close(wait)
}

// Run starts the client and blocks until the client stops, returning the error
// if any
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
	return <-wait
}

// Start starts the client and returns immediately
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

// Wait waits for the client to stop and returns any error
func Wait() error {
	mu.RLock()
	s := started
	mu.RUnlock()
	if !s {
		return ErrNotRunning
	}
	err := <-wait
	openConnections.Wait()
	return err
}

// Close closes all open proxied listeners and stop the client
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if stopped {
		return ErrStopped
	}
	stopped = true
	if started {
		err := server.Close()
		if proxyHTTPSocket != nil {
			e := proxyHTTPSocket.Close()
			if err == nil {
				err = e
			}
		}
		if proxyHTTPSSocket != nil {
			e := proxyHTTPSSocket.Close()
			if err == nil {
				err = e
			}
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
