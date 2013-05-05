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

// Package types contains types and methods shared by the webserver/client and webserver/proxy packages.
package types

import (
	"crypto/rsa"
	"github.com/MJKWoolnough/crypto/asymmetric"
	"time"
)

// NewServer contains the information needed to register a new client with the proxy.
type NewServer struct {
	ServerName string
	Aliases    []string
	PublicKey  *rsa.PublicKey
	//sslCert
	Time      time.Time
	Signature []byte
}

func (n *NewServer) GetData() (serverName string, signature []byte, t time.Time) {
	serverName, signature, t = n.ServerName, n.Signature, n.Time
	n.Signature = nil
	return
}

func (n *NewServer) Sign(key *rsa.PrivateKey) (err error) {
	n.Time = time.Now()
	n.Signature, err = asymmetric.Sign(n, key)
	return
}

// ConnectServer contains the information needed to update the proxys information about the client.
type ConnectServer struct {
	ServerName string
	Net, Addr  string
	Time       time.Time
	Signature  []byte
}

func (c *ConnectServer) GetData() (serverName string, signature []byte, t time.Time) {
	serverName, signature, t = c.ServerName, c.Signature, c.Time
	c.Signature = nil
	return
}

func (c *ConnectServer) Sign(key *rsa.PrivateKey) (err error) {
	c.Time = time.Now()
	c.Signature, err = asymmetric.Sign(c, key)
	return
}

// ConnectServer contains the information needed to remove the client.
type RemoveServer struct {
	ServerName string
	Time       time.Time
	Signature  []byte
}

func (r *RemoveServer) GetData() (serverName string, signature []byte, t time.Time) {
	serverName, signature, t = r.ServerName, r.Signature, r.Time
	r.Signature = nil
	return
}

func (r *RemoveServer) Sign(key *rsa.PrivateKey) (err error) {
	r.Time = time.Now()
	r.Signature, err = asymmetric.Sign(r, key)
	return
}
