package proxy

import (
	"errors"
	"net"
	"sync"
)

// Proxy repsents a listener that will proxy connections to hosts
type Proxy struct {
	http, https net.Listener

	started bool
	closed  chan struct{}
	err     error

	mu          sync.RWMutex
	hostnames   map[string]*Host
	defaultHost *Host
}

// New creates a new Proxy will optional http and https listeners
func New(http, https net.Listener) *Proxy {
	if http == nil && https == nil {
		return nil
	}
	return &Proxy{
		http:      http,
		https:     https,
		closed:    make(chan struct{}),
		hostnames: make(map[string]*Host),
	}
}

// Default sets the default host, one which will be used if a host cannot be
// determined from the headers
func (p *Proxy) Default(h *Host) error {
	if h.proxy != p {
		return ErrInvalidHost
	}
	p.mu.Lock()
	p.defaultHost = h
	p.mu.Unlock()
	return nil
}

// IsDefault returns whether the given host is currently the default
func (p *Proxy) IsDefault(h *Host) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.defaultHost == h
}

func (p *Proxy) runConns() error {
	p.started = true
	ec := make(chan error, 1)
	http := p.http
	if http != nil {
		defer http.Close()
		go func() {
			ec <- p.run(http, false)
		}()
	}
	https := p.https
	if https != nil {
		defer https.Close()
		go func() {
			ec <- p.run(https, true)
		}()
	}
	p.err = <-ec
	close(p.closed)
	go p.closeHosts()
	return p.err
}

func (p *Proxy) closeHosts() {
	for _, host := range p.hostnames {
		host.Stop()
	}
}

// Run starts the proxy and waits until it is closed to return any errors
func (p *Proxy) Run() error {
	if p.started {
		return ErrRunning
	}
	if p.defaultHost == nil {
		return ErrNoDefault
	}
	return p.runConns()
}

// Start starts the proxy and returns immediately
func (p *Proxy) Start() error {
	if p.started {
		return ErrRunning
	}
	if p.defaultHost == nil {
		return ErrNoDefault
	}
	go p.runConns()
	return nil
}

// Wait will block until the proxy is stopped
func (p *Proxy) Wait() error {
	if !p.started {
		return ErrNotRunning
	}
	<-p.closed
	return p.err
}

func (p *Proxy) addAlias(h *Host, name string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.hostnames[name]
	if ok {
		return false
	}
	p.hostnames[name] = h
	return true
}

func (p *Proxy) removeAlias(name string) {
	p.mu.Lock()
	delete(p.hostnames, name)
	p.mu.Unlock()
}

// Errors
var (
	ErrInvalidHost = errors.New("invalid host")
	ErrRunning     = errors.New("already running")
	ErrNotRunning  = errors.New("not running")
	ErrNoDefault   = errors.New("no default host set")
)
