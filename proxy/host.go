package proxy

import (
	"os/exec"
	"sync"
)

type Host struct {
	proxy *Proxy

	mu                          sync.RWMutex
	cmd                         *exec.Cmd
	aliases                     []string
	httpTransfer, httpsTransfer *transfer
}

func (p *Proxy) NewHost(c *exec.Cmd) *Host {
	return &Host{
		cmd:   c,
		proxy: p,
	}
}

func (h *Host) AddAliases(names ...string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
NameLoop:
	for _, name := range names {
		for _, alias := range h.aliases {
			if alias == name {
				continue NameLoop
			}
		}
		if h.proxy.addAlias(h, name) {
			h.aliases = append(h.aliases, name)
		} else {
			return ErrAliasInUse{name}
		}
	}
	return nil
}

func (h *Host) RemoveAlias(names ...string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
NameLoop:
	for _, name := range names {
		for n, alias := range h.aliases {
			if alias == name {
				h.proxy.removeAlias(name)
				copy(h.aliases[n:], h.aliases[n+1:])
				continue NameLoop
			}
		}
		return ErrUnknownAlias{name}
	}
	return nil
}

func (h *Host) Aliases() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s := make([]string, len(h.aliases))
	copy(s, h.aliases)
	return s
}

func (h *Host) Restart() error {
	return nil
}

func (h *Host) Stop() error {
	return nil
}

func (h *Host) Replace(c *exec.Cmd) error {
	return nil
}

func (h *Host) getTransfer(encrypted bool) *transfer {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if encrypted {
		return h.httpsTransfer
	}
	return h.httpTransfer
}

type ErrAliasInUse struct {
	Name string
}

func (e ErrAliasInUse) Error() string {
	return "server alias already in use: " + e.Name
}

type ErrUnknownAlias struct {
	Name string
}

func (e ErrUnknownAlias) Error() string {
	return "server alias not assigned to this host: " + e.Name
}
