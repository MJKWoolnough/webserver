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

// Webserver Reverse Proxy
package main

import (
	"crypto/rsa"
	"encoding/binary"
	"encoding/gob"
	"flag"
	"github.com/MJKWoolnough/crypto/asymmetric"
	"github.com/MJKWoolnough/webserver/client/types"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

var (
	configFilename *string = flag.String("s", "", "Server Configuration file.")
	keyFilename    *string = flag.String("p", "", "Public Key file.")
	logFilename    *string = flag.String("l", "", "Log file.")
	unixSocket     *string = flag.String("u", "", "Unix socket for rpc server.")
	ipAddress      *string = flag.String("i", "", "IP Address/port for rpc server ip:port.")
	httpAddress    *string = flag.String("h", "", "IP Address/port for http server ip:port.")
	publicKey      *rsa.PublicKey
	logFile        *log.Logger
)

const (
	HTTP_400 = "400 Bad Request"
	HTTP_413 = "413 Request Entity Too Large"
	HTTP_502 = "502 Bad Gateway"
	HTTP_503 = "503 Service Unavailable"
)

type signed interface {
	GetData() (string, []byte, time.Time)
}

type server struct {
	ServerName string
	Net, Addr  string
	Aliases    []string
	PublicKey  *rsa.PublicKey
}

type errorString string

func (e errorString) Error() string {
	return string(e)
}

type Servers struct {
	lock       sync.RWMutex
	serverList []*server
	aliases    map[string]*server
}

func (s *Servers) Register(args types.NewServer, reply *bool) error {
	*reply = false
	if err := checkSignature(&args, publicKey, "Register"); err != nil {
		return errorString("Register: " + err.Error())
	}
	s.lock.RLock()
	if _, ok := s.aliases[args.ServerName]; ok {
		s.lock.RUnlock()
		logFile.Printf("Register: Server name already taken - %q\n", args.ServerName)
		return errorString("Servername is already taken")
	}
	for _, name := range args.Aliases {
		if _, ok := s.aliases[name]; ok {
			s.lock.RUnlock()
			logFile.Printf("Register: For server %q, alias already taken - %q\n", args.ServerName, name)
			return errorString("Alias is already taken: " + name)
		}
	}
	s.lock.RUnlock()
	serv := new(server)
	serv.ServerName = args.ServerName
	serv.Aliases = args.Aliases
	serv.PublicKey = args.PublicKey
	//ssl cert
	s.lock.Lock()
	s.serverList = append(s.serverList, serv)
	s.aliases[args.ServerName] = serv
	for _, name := range args.Aliases {
		s.aliases[name] = serv
	}
	s.lock.Unlock()
	go s.save()
	*reply = true
	logFile.Printf("Register: Registered new server - %q - Aliases %v\n", args.ServerName, args.Aliases)
	return nil
}

func (s *Servers) Update(args types.NewServer, reply *bool) error {
	*reply = false
	if err := checkSignature(&args, publicKey, "Update"); err != nil {
		return errorString("Update: " + err.Error())
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if serv, ok := s.aliases[args.ServerName]; !ok {
		s.lock.Unlock()
		logFile.Printf("Update: Could not find server %q to update.\n", args.ServerName)
		return errorString(args.ServerName + " server not found in server list")
	} else {
		for _, name := range args.Aliases {
			if serv, ok := s.aliases[name]; ok && serv.ServerName != args.ServerName {
				logFile.Printf("Update: For server %q, alias already taken - %q\n", args.ServerName, name)
				return errorString("Alias is already taken: " + name)
			}
		}
		serv.PublicKey = args.PublicKey
		//sslCert
		for _, oldAlias := range serv.Aliases {
			found := false
			for _, newAlias := range args.Aliases {
				if oldAlias == newAlias {
					found = true
					break
				}
			}
			if !found {
				delete(s.aliases, oldAlias)
			}
		}
		for _, name := range args.Aliases {
			s.aliases[name] = serv
		}
		serv.Aliases = args.Aliases
	}
	go s.save()
	*reply = true
	logFile.Printf("Update: Server %q updated.\n", args.ServerName)
	return nil
}

func (s *Servers) Remove(args types.RemoveServer, reply *bool) error {
	*reply = false
	if err := checkSignature(&args, publicKey, "Remove"); err != nil {
		return errorString("Remove: " + err.Error())
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if serv, ok := s.aliases[args.ServerName]; !ok {
		logFile.Printf("Remove: Could not find server %q to remove.\n", args.ServerName)
		return errorString(args.ServerName + " server not found in server list")
	} else {
		delete(s.aliases, args.ServerName)
		for _, alias := range serv.Aliases {
			delete(s.aliases, alias)
		}
	}
	for i, serv := range s.serverList {
		if serv.ServerName == args.ServerName {
			ls := len(s.serverList)
			s.serverList[ls-1], s.serverList[i], s.serverList = nil, s.serverList[ls-1], s.serverList[:ls-1]
			break
		}
	}
	go s.save()
	*reply = true
	logFile.Printf("Remove: Server %q removed.\n", args.ServerName)
	return nil
}

func (s *Servers) Connect(args types.ConnectServer, reply *bool) error {
	*reply = false
	if time.Since(args.Time).Seconds()+300 > 600 {
		logFile.Printf("Connect: %q failed Time check with %q\n", args.ServerName, args.Time.Format("2006-01-02 15:04:05"))
		return errorString("Failed Time check - check the Time!")
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if serv, ok := s.aliases[args.ServerName]; !ok {
		return errorString(args.ServerName + " server not found in server list")
	} else {
		sig := args.Signature
		args.Signature = nil
		if err := asymmetric.SignCheck(&args, sig, serv.PublicKey); err != nil {
			logFile.Printf("Connect: %q failed Signature check\n", args.ServerName)
			return errorString("Failed Signature check: " + err.Error())
		}
		serv.Net = args.Net
		serv.Addr = args.Addr
	}
	go s.save()
	*reply = true
	logFile.Printf("Connect: %q updated connection status %q, %q\n", args.ServerName, args.Net, args.Addr)
	return nil
}

func (s *Servers) serve(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 0)
	char := make([]byte, 1, 1)
	line := make([]byte, 8192, 8192)
	var (
		chosenStr string
		chosenSer *server
	)
breakpoint:
	for i := 0; i <= 100; i++ {
		if i == 100 {
			httpError(HTTP_413, conn)
			return
		}
		for j := 0; j <= 8192; j++ {
			if j == 8192 {
				httpError(HTTP_413, conn)
				return
			}
			if c, err := conn.Read(char); err != nil || c != 1 {
				httpError(HTTP_400, conn)
			}
			line[j] = char[0]
			if char[0] == '\n' {
				buf = append(buf, line[:j+1]...)
				if string(line[:2]) == "\r\n" {
					break breakpoint
				} else if string(line[:6]) == "Host: " {
					minus := 0
					if line[j-1] == '\r' {
						minus = 1
					}
					tchosenStr := line[6 : j-minus]
					bchosenStr := make([]byte, 0, len(tchosenStr))
					for _, c := range tchosenStr {
						if c == ':' {
							break
						}
						bchosenStr = append(bchosenStr, c)
					}
					chosenStr = string(bchosenStr)
				}
				break
			}
		}
	}
	if serv, ok := s.aliases[chosenStr]; !ok {
		if serv, ok = s.aliases[""]; !ok {
			httpError(HTTP_503, conn)
			return
		}
		chosenSer = serv
	} else {
		chosenSer = serv
	}
	if serveConn, err := net.Dial(chosenSer.Net, chosenSer.Addr); err != nil {
		httpError(HTTP_502, conn)
		return
	} else {
		ra := conn.RemoteAddr()
		net := []byte(ra.Network())
		log.Printf("Connection from %q %q to %q %q\n", ra.Network(), ra.String(), chosenSer.Net, chosenSer.Addr)
		address := []byte(ra.String())
		err := binary.Write(serveConn, binary.BigEndian, uint32(len(net)))
		if err == nil {
			err = binary.Write(serveConn, binary.BigEndian, net)
			if err == nil {
				err = binary.Write(serveConn, binary.BigEndian, uint32(len(address)))
				if err == nil {
					err = binary.Write(serveConn, binary.BigEndian, address)
					if err == nil {
						_, err = serveConn.Write(buf)
					}
				}
			}
		}
		if err != nil {
			log.Print(err)
		} else {
			errc := make(chan error, 1)
			go cp(conn, serveConn, errc)
			go cp(serveConn, conn, errc)
			if err := <-errc; err != nil {
				log.Printf("Serve: Connection closed with error: %q\n", err.Error())
			}
			conn.Close()
			serveConn.Close()
		}
	}
}

func cp(a, b net.Conn, errc chan<- error) {
	_, err := io.Copy(a, b)
	errc <- err
}

func (s *Servers) save() {
	if *configFilename != "" {
		f, err := os.Create(*configFilename)
		if err == nil {
			s.lock.RLock()
			err = gob.NewEncoder(f).Encode(s.serverList)
			s.lock.RUnlock()
		}
		if err != nil {
			log.Printf("Save: Config save error: %q\n", err.Error())
		}
	}
}

func checkSignature(obj signed, key *rsa.PublicKey, cType string) error {
	name, sig, t := obj.GetData()
	if time.Since(t).Seconds()+300 > 600 {
		logFile.Printf("%s: %q failed Time check with %q\n", cType, name, t.Format("2006-01-02 15:04:05"))
		return errorString("Failed Time check - check the Time!")
	}
	if key != nil {
		if err := asymmetric.SignCheck(obj, sig, key); err != nil {
			logFile.Printf("%s: %q failed Signature check\n", cType, name)
			return errorString("Failed Signature check: " + err.Error())
		}
	}
	return nil
}

func httpError(statusCode string, conn net.Conn) {
	logFile.Printf("Connect: Error %q from %q\n", statusCode, conn.RemoteAddr().String())
	if _, err := conn.Write([]byte("HTTP/1.0 " + statusCode + "\r\nDate: " + time.Now().Format(time.RFC1123) + "\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")); err != nil {
		logFile.Printf("Connect: Error sending error response %q to %q\n", statusCode, conn.RemoteAddr().String())
	}
}

func main() {
	flag.Parse()
	s := new(Servers)
	s.aliases = make(map[string]*server)
	if *configFilename != "" {
		f, err := os.Open(*configFilename)
		if err == nil {
			if err = gob.NewDecoder(f).Decode(&s.serverList); err == nil {
				for _, serv := range s.serverList {
					s.aliases[serv.ServerName] = serv
					for _, alias := range serv.Aliases {
						s.aliases[alias] = serv
					}
				}
			}
			f.Close()
		}
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("Could not read server configuration file %q: %q\n", *configFilename, err.Error())
				os.Exit(1)
			}
		}
	} else {
		s.serverList = make([]*server, 0)
	}
	if *keyFilename != "" {
		f, err := os.Open(*keyFilename)
		if err == nil {
			publicKey, err = asymmetric.PublicKey(f)
		}
		f.Close()
		if err != nil {
			log.Printf("Could not read public key file %q: %q\n", *configFilename, err.Error())
			os.Exit(1)
		}
	}
	var l, h net.Listener
	if *unixSocket != "" && *ipAddress != "" {
		log.Print("Currently only a single connection type for the RPC server is supported.\n")
		os.Exit(1)
	} else if *unixSocket != "" {
		if a, err := net.Listen("unix", *unixSocket); err != nil {
			log.Printf("Error creating unix Net connection for RPC server %q: %q\n", *unixSocket, err.Error())
			os.Exit(1)
		} else {
			l = a
		}
	} else if *ipAddress != "" {
		if a, err := net.Listen("tcp", *ipAddress); err != nil {
			log.Printf("Error creating tcp Net connection for RPC server %q: %q\n", *ipAddress, err.Error())
			os.Exit(1)
		} else {
			l = a
		}
	} else {
		log.Print("No connection specified for RPC server.\n")
		os.Exit(1)
	}
	if *httpAddress != "" {
		if a, err := net.Listen("tcp", *httpAddress); err != nil {
			log.Printf("Error creating tcp Net connection for HTTP proxy server %q: %q\n", *ipAddress, err.Error())
			os.Exit(1)
		} else {
			h = a
		}
	} else {
		log.Print("No http server Address specified.\n")
		os.Exit(1)
	}
	if *logFilename != "" {
		if f, err := os.Create(*logFilename); err == nil {
			log.Printf("Could not open log file for writing %q: %q\n", *logFilename, err.Error())
			os.Exit(1)
		} else {
			defer f.Close()
			logFile = log.New(f, "", log.LstdFlags)
		}
	} else {
		logFile = log.New(os.Stderr, "", log.LstdFlags)
	}
	logFile.Print("Starting RPC server")
	quit := make(chan bool, 0)
	go func(q chan<- bool, s *Servers, l net.Listener) {
		r := rpc.NewServer()
		r.Register(s)
		r.Accept(l)
		l.Close()
		logFile.Print("Unexpected shutdown of RPC Server.")
		q <- true
	}(quit, s, l)
	logFile.Print("Starting HTTP Proxy Server")
	go func(q chan<- bool, s *Servers, l net.Listener) {
		for {
			conn, err := l.Accept()
			if err != nil {
				logFile.Printf("Unexpected shutdown of HTTP Proxy Server: %q", err.Error())
				break
			}
			go s.serve(conn)
		}
		q <- true
	}(quit, s, h)
	<-quit
}
