package proxy

import (
	"errors"
	"os/exec"
	"strconv"
	"sync"
)

type Host struct {
	proxy *Proxy

	mu                          sync.RWMutex
	cmd                         *exec.Cmd
	aliases                     []string
	httpTransfer, httpsTransfer *transfer
}

func (p *Proxy) NewHost(c *exec.Cmd) (*Host, error) {
	h := &Host{
		cmd:   c,
		proxy: p,
	}
	var err error
	h.httpTransfer, h.httpsTransfer, err = h.setupCmd(c)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Host) setupCmd(c *exec.Cmd) (*transfer, *transfer, error) {
	select {
	case <-h.proxy.closed:
		return nil, nil, ErrProxyClosed
	default:
	}
	var (
		http, https *transfer
		done        bool
	)
	defer func() {
		if !done {
			if http != nil {
				http.Close()
			}
			if https != nil {
				https.Close()
			}
		}
	}()
	if h.proxy.http != nil {
		c.Env = append(c.Env, "proxyHTTPSocket="+strconv.FormatUint(uint64(len(c.ExtraFiles))+3, 10))
		var err error
		http, err = newTransfer()
		if err != nil {
			return nil, nil, err
		}
		c.ExtraFiles = append(c.ExtraFiles, http.f)
	}
	if h.proxy.https != nil {
		c.Env = append(c.Env, "proxyHTTPSSocket="+strconv.FormatUint(uint64(len(c.ExtraFiles))+3, 10))
		var err error
		https, err = newTransfer()
		if err != nil {
			return nil, nil, err
		}
		c.ExtraFiles = append(c.ExtraFiles, https.f)
	}
	if err := c.Start(); err != nil {
		return nil, nil, err
	}
	done = true
	return http, https, nil
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
	h.mu.Lock()
	defer h.mu.Unlock()
	cmd := exec.Command(h.cmd.Path, h.cmd.Args...)
	http, https, err := h.setupCmd(cmd)
	if err != nil {
		return err
	}
	h.httpTransfer.Close()
	h.httpsTransfer.Close()
	h.cmd = cmd
	h.httpTransfer = http
	h.httpsTransfer = https
	return nil
}

func (h *Host) Stop() error {
	if h.proxy.IsDefault(h) {
		select {
		case <-h.proxy.closed:
		default:
			return ErrIsDefault
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	var err error
	if h.httpTransfer != nil {
		err = h.httpTransfer.Close()
		h.httpTransfer = nil
	}
	if h.httpsTransfer != nil {
		if e := h.httpsTransfer.Close(); e != nil {
			err = e
		}
		h.httpsTransfer = nil
	}
	for _, alias := range h.aliases {
		h.proxy.removeAlias(alias)
	}
	h.aliases = h.aliases[:0]
	return err
}

func (h *Host) Replace(c *exec.Cmd) error {
	http, https, err := h.setupCmd(c)
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.httpTransfer.Close()
	h.httpsTransfer.Close()
	h.cmd = c
	h.httpTransfer = http
	h.httpsTransfer = https
	h.mu.Unlock()
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

var (
	ErrIsDefault   = errors.New("host is default")
	ErrProxyClosed = errors.New("proxy closed")
)
