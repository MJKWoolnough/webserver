// Copyright (c) 2013 - Michael Woolnough <michael.woolnough@gmail.com>
// 
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met: 
// 
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer. 
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution. 
// 
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Package cache allows clients to store resources that expire after a given duration.
package cache

import (
	"sync"
	"time"
)

// Resource represents the clients data.
type Resource interface {
	// Expire will be executed after the given duration has passed allowing the data
	// processed or save in another format.
	Expire(time.Time)
}

type res struct {
	data Resource
	stop chan struct{}
}

func (r *res) expire(n string, e time.Duration) {
	expiry := time.NewTimer(e)
	defer expiry.Stop()
	select {
	case t := <-expiry.C:
		lock.Lock()
		defer lock.Unlock()
		delete(resources, n)
		go r.data.Expire(t)
	case <-r.stop:
	}
}

// ErrNotFound is returned when trying to Read or Remove a name that does not exist.
type ErrNotFound struct {
	name string
}

func (e ErrNotFound) Error() string {
	return "Resource " + e.name + " not found"
}

var (
	resources map[string]*res
	lock      sync.RWMutex
)

// Insert adds a new named resource into the cache which will expire after the given duration.
// If a resource with the same name already exists it will Remove that resource, without executing
// the Expire function, and replace it with the new resource.
func Insert(name string, resource Resource, duration time.Duration) {
	r := &res{resource, make(chan struct{})}
	go r.expire(name, duration)
	lock.Lock()
	defer lock.Unlock()
	if r, ok := resources[name]; ok {
		delete(resources, name)
		close(r.stop)
	}
	resources[name] = r
}

// Read allows a client to retrive a named resource from the cache.
func Read(name string) (Resource, error) {
	lock.RLock()
	defer lock.RUnlock()
	r, ok := resources[name]
	if !ok {
		return nil, &ErrNotFound{name}
	}
	return r.data, nil
}

// Remove will delete the named resource from the cache.
// The Expiry method will not be executed.
func Remove(name string) error {
	lock.Lock()
	defer lock.Unlock()
	if r, ok := resources[name]; ok {
		delete(resources, name)
		close(r.stop)
	} else {
		return &ErrNotFound{name}
	}
	return nil
}
