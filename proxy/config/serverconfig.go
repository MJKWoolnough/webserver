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

// Webserver Reverse Proxy configuation editor.
package main

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"github.com/MJKWoolnough/crypto/asymmetric"
	"github.com/MJKWoolnough/webserver/client/types"
	"net/rpc"
	"os"
	"strings"
	"time"
)

var (
	mode          *string = flag.String("m", "", "Mode [n,u,r].")
	serverName    *string = flag.String("s", "", "Server name.")
	aliases       *string = flag.String("a", "", "Server Aliases.")
	defaultServer *bool   = flag.Bool("d", false, "Make Default.")
	publicKey     *string = flag.String("k", "", "Server Public Key.")
	privateKey    *string = flag.String("p", "", "Proxy Server's Private Key.")
	network       *string = flag.String("n", "", "Proxy Server's network.")
	address       *string = flag.String("l", "", "Proxy Server's Address.")
	//ssl
)

type signer interface {
	Sign(*rsa.PrivateKey) error
}

func main() {
	flag.Parse()
	var (
		method string
		args   signer
		key    *rsa.PrivateKey
	)
	if *privateKey != "" {
		p, err := os.Open(*privateKey)
		if err == nil {
			key, err = asymmetric.PrivateKey(p)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
	if *serverName == "" {
		fmt.Fprintln(os.Stderr, "No server name specified.")
		os.Exit(1)
	}
	if *network != "tcp" && *network != "unix" {
		fmt.Fprintln(os.Stderr, "Invalid proxy network specified")
		os.Exit(1)
	}
	if *address == "" {
		fmt.Fprintln(os.Stderr, "No proxy address specified")
		os.Exit(1)
	}
	if *mode == "n" || *mode == "u" {
		if *mode == "n" {
			method = "Servers.Register"
		} else {
			method = "Servers.Update"
		}
		s := new(types.NewServer)
		s.ServerName = *serverName

		aliasList := make([]string, 0)
		if *defaultServer {
			aliasList = append(aliasList, "")
		}
		if *aliases != "" {
			aliasList = append(aliasList, strings.Split(*aliases, ",")...)
		}
		if len(aliasList) > 0 {
			s.Aliases = aliasList
		}
		if *publicKey == "" {
			fmt.Fprintln(os.Stderr, "No server public key specified.")
			os.Exit(1)
		} else {
			p, err := os.Open(*publicKey)
			if err == nil {
				s.PublicKey, err = asymmetric.PublicKey(p)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		}
		args = s
	} else if *mode == "r" {
		method = "Servers.Remove"
		s := new(types.RemoveServer)
		s.ServerName = *serverName
		args = s
	} else {
		fmt.Fprintln(os.Stderr, "Invalid mode specified")
		os.Exit(1)
	}
	if err := args.Sign(key); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	client, err := rpc.Dial(*network, *address)
	if err == nil {
		var reply bool
		err = client.Call(method, args, &reply)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	} else {
		fmt.Println("Success!")
	}
}
