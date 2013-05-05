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

// Package client is a drop-in replacement webserver/client to create a standalone
// webserver without the need of a proxy - useful for testing.
package client

import (
	"io"
	"net"
)

type errorString struct {
	err string
}

func (e errorString) Error() string {
	return e.err
}

type unixSocket struct {
	addr string
}

func NewUnixSocket(path string) net.Addr {
	return unixSocket{path}
}

func (u unixSocket) Network() string {
	return "unix"
}

func (u unixSocket) String() string {
	return u.addr
}

type tcpSocket struct {
	addr string
}

func NewTCP4Socket(addr, port string) net.Addr {
	if port == "" {
		port = "0"
	}
	return tcpSocket{addr + ":" + port}
}

func (t tcpSocket) Network() string {
	return "tcp"
}

func (t tcpSocket) String() string {
	return t.addr
}

// Register creates a tcp connection on port 12346 and returns the Listener.
func Register(serverName string, addr net.Addr, privateKey io.Reader) (conn net.Listener, err error) {
	if conn, err = net.Listen("tcp", "127.0.0.1:12346"); err != nil {
		err = errorString{"Error creating connection for server - " + addr.Network() + " - " + addr.String() + " : " + err.Error()}
	}
	return
}
