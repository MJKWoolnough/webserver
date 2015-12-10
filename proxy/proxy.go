package proxy

import (
	"errors"
	"net"
	"sync"
)

type Proxy struct {
	http, ssl net.Listener

	mu          sync.RWMutex
	hostnames   map[string]*Host
	defaultHost *Host
}

func New(http, ssl net.Listener) *Proxy {
	return &Proxy{
		http: http,
		ssl:  ssl,
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

func (p *Proxy) Run() error {
	return nil
}

func (p *Proxy) Start() error {
	return nil
}

func (p *Proxy) Wait() error {
	return nil
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
)
