package proxy

import (
	"errors"
	"net"
	"sync"
)

type Proxy struct {
	http, https net.Listener

	started bool
	closed  chan struct{}
	err     error

	mu          sync.RWMutex
	hostnames   map[string]*Host
	defaultHost *Host
}

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

func (p *Proxy) Default(h *Host) error {
	if h.proxy != p {
		return ErrInvalidHost
	}
	p.mu.Lock()
	p.defaultHost = h
	p.mu.Unlock()
	return nil
}

func (p *Proxy) IsDefault(h *Host) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.defaultHost == h
}

func (p *Proxy) runConns() error {
	p.started = true
	defer close(p.closed)
	ec := make(chan error, 1)
	if p.http != nil {
		defer p.http.Close()
		go func() {
			ec <- p.run(false)
		}()
	}
	if p.https != nil {
		defer p.https.Close()
		go func() {
			ec <- p.run(true)
		}()
	}
	p.err <- ec
	return p.err
}

func (p *Proxy) Run() error {
	if p.started {
		return ErrRunning
	}
	return p.runConns()
}

func (p *Proxy) Start() error {
	if p.started {
		return ErrRunning
	}
	go p.runConns()
	return nil
}

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
	i, ok := p.hostnames[name]
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
)
