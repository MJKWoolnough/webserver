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

// Package client contains the registration methods to connect a client to a webserver/proxy.
package client

import (
	"github.com/MJKWoolnough/crypto-asymmetric"
	"encoding/binary"
	"flag"
	"github.com/MJKWoolnough/webserver/client/types"
	"io"
	"net"
	"net/rpc"
)

var (
	network *string = flag.String("n", "", "Proxy Server's network.")
	address *string = flag.String("l", "", "Proxy Server's Address.")
)

type errorString struct {
	err string
}

func (e errorString) Error() string {
	return e.err
}

type wConn struct {
	net.Conn
	net, addr string
}

func (c *wConn) RemoteAddr() net.Addr {
	return c
}

func (c *wConn) Network() string {
	return c.net
}

func (c *wConn) String() string {
	return c.addr
}

type wListen struct {
	net.Listener
}

func (l *wListen) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	var netLen, addrLen uint32
	if err = binary.Read(c, binary.BigEndian, &netLen); err != nil {
		return nil, err
	}
	net := make([]byte, netLen, netLen)
	if err = binary.Read(c, binary.BigEndian, &net); err != nil {
		return nil, err
	}
	if err = binary.Read(c, binary.BigEndian, &addrLen); err != nil {
		return nil, err
	}
	addr := make([]byte, addrLen, addrLen)
	if err = binary.Read(c, binary.BigEndian, &addr); err != nil {
		return nil, err
	}
	return &wConn{c, string(net), string(addr)}, nil
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

// Register attempts to set-up the connection to the proxy server and returns a Listener to pass to the http package.
func Register(serverName string, addr net.Addr, privateKey io.Reader) (net.Listener, error) {
	flag.Parse()
	var (
		connect types.ConnectServer
		conn    net.Listener
	)
	if serverName == "" {
		return nil, errorString{"No servername set"}
	}
	connect.ServerName = serverName
	if a, err := net.Listen(addr.Network(), addr.String()); err != nil {
		return nil, errorString{"Error creating connection for server - " + addr.Network() + " - " + addr.String() + " : " + err.Error()}
	} else {
		conn = a
	}
	connect.Net = conn.Addr().Network()
	connect.Addr = conn.Addr().String()
	if pKey, err := asymmetric.PrivateKey(privateKey); err != nil {
		conn.Close()
		return nil, err
	} else {
		connect.Sign(pKey)
	}
	client, err := rpc.Dial(*network, *address)
	if err == nil {
		var reply bool
		err = client.Call("Servers.Connect", connect, &reply)
		client.Close()
	}
	if err != nil {
		return nil, err
	}
	return &wListen{conn}, nil
}
