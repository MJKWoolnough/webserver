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

// Package log creates a http handler that will output a binary log of all connections.
package log

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	version   byte   = 1
	accessLog string = "access_log_"
	errorLog  string = "error_log_"
)

const (
	METHOD_OPTIONS method = iota
	METHOD_GET
	METHOD_HEAD
	METHOD_POST
	METHOD_PUT
	METHOD_DELETE
	METHOD_TRACE
	METHOD_CONNECT
	METHOD_UNKNOWN
)

var methods = []string{
	"OPTIONS",
	"GET",
	"HEAD",
	"POST",
	"PUT",
	"DELETE",
	"TRACE",
	"CONNECT",
	"UNKNOWN",
}

type method uint8

func (m method) String() string {
	return methods[m]
}

type wrappedLog struct {
	http.ResponseWriter
	*log
}

type log struct {
	logLength uint16
	version   byte
	sTime     int64
	eTime     int64
	addr      net.IP
	port      uint16
	httpData  byte
	status    uint16
	length    uint64
	dataLen   uint16
	textData  []byte
}

func (l *wrappedLog) Write(b []byte) (int, error) {
	c, err := l.ResponseWriter.Write(b)
	l.length += uint64(c)
	return c, err
}

func (l *wrappedLog) WriteHeader(status int) {
	l.status = uint16(status)
	l.ResponseWriter.WriteHeader(status)
}

type httpLog struct {
	handler http.Handler
	log     chan *log
	err     chan error
	dir     string
}

// NewHTTPLog will create the handler that will output to the logs in the given directory.
func NewHTTPLog(dir string, handler http.Handler) *httpLog {
	if handler == nil {
		handler = http.DefaultServeMux
	}
	h := &httpLog{handler, make(chan *log), make(chan error), dir}
	go h.logIt()
	return h
}

func (h *httpLog) setup() (*os.File, *os.File, <-chan time.Time, error) {
	var tErr error
	t := time.Now()
	y := t.Year()
	m := t.Month()
	m++
	if m > 12 {
		m = 1
		y++
	}
	c := time.After(time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).Sub(t))
	fpart := t.Format("2006-01")
	e, err := os.Create(h.dir + "/" + errorLog + fpart)
	if err != nil {
		tErr = errors.New("Failed to create error log file: " + err.Error())
		e, _ = os.Create(os.DevNull)
	} else {
		e.Seek(0, os.SEEK_END)
	}
	l, err := os.Create(h.dir + "/" + accessLog + fpart)
	if err != nil {
		e := "Failed to create access log file: " + err.Error()
		if tErr != nil {
			tErr = errors.New(tErr.Error() + "\n" + e)
		}
		tErr = errors.New(e)
		l, _ = os.Create(os.DevNull)
	} else {
		e.Seek(0, os.SEEK_END)
	}
	return l, e, c, tErr
}

func (h *httpLog) logIt() {
	logFile, errFile, timer, err := h.setup()
	defer logFile.Close()
	defer errFile.Close()
	if err != nil {
		go h.Error(fmt.Errorf("Error while trying to initialise logger: %q", err))
	}
	for {
		select {
		case s := <-h.log:
			if position, err := logFile.Seek(0, os.SEEK_CUR); err != nil {
				go h.Error(fmt.Errorf("Error while getting file position: %q", err))
			} else {
				if err := binary.Write(logFile, binary.LittleEndian, s.logLength); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.version); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.sTime); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.eTime); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.addr); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.port); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.httpData); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.status); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.length); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.dataLen); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
				if err := binary.Write(logFile, binary.LittleEndian, s.textData); err != nil {
					go h.Error(fmt.Errorf("Error while logging: %q", err))
					logFile.Seek(position, os.SEEK_SET)
					continue
				}
			}
		case e := <-h.err:
			if _, err := fmt.Fprintln(errFile, e); err != nil {
				fmt.Fprintf(os.Stderr, "Received error %q while trying to print the following the the error log: %q\n", err, e)
			}
		case <-timer:
			logFile.Close()
			errFile.Close()
			logFile, errFile, timer, err = h.setup()
			if err != nil {
				go h.Error(fmt.Errorf("Error while trying to rotate log: %q", err))
			}
		}
	}
}

// Error allows the webserver to print any errors to the rotatable error log.
func (h *httpLog) Error(err error) {
	if err != nil {
		h.err <- err
	}
}

func (h *httpLog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l := new(log)
	l.status = http.StatusOK
	l.sTime = time.Now().UnixNano()
	h.handler.ServeHTTP(&wrappedLog{w, l}, r)
	l.eTime = time.Now().UnixNano()
	l.logLength = 48 //1 + 8 + 8 + 16 + 2 + 1 + 2 + 8 + 2
	l.version = version
	host, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		h.Error(fmt.Errorf("Error splitting host/port %q: %q", r.RemoteAddr, err))
		return
	}
	addr := net.ParseIP(host)
	if addr == nil {
		h.Error(fmt.Errorf("Failed to parse host ip: %q", host))
		return
	}
	l.addr = addr.To16()
	fmt.Sscan(port, &l.port)
	m := METHOD_UNKNOWN
	for i, j := range methods {
		if j == r.Method {
			m = method(i)
			break
		}
	}
	l.httpData = uint8(r.ProtoMajor)&3 | (uint8(r.ProtoMinor) & 3 << 2) | (uint8(m) & 32 << 4)
	username, _ := Credentials(r)
	var b bytes.Buffer
	c := gzip.NewWriter(&b)
	fmt.Fprintf(c, "%s\x00%s\x00%s\x00%s", r.URL, r.Host, username, r.UserAgent())
	c.Close()
	l.dataLen = uint16(b.Len())
	l.logLength += l.dataLen
	l.textData = b.Bytes()
	h.log <- l
}

// Credentials gets any username and password used in Basic Authentication.
func Credentials(r *http.Request) (string, string) {
	var username, password string
	authStr := r.Header.Get("Authorization")
	if len(authStr) > 6 && authStr[:5] == "Basic" {
		if decoded, err := base64.StdEncoding.DecodeString(authStr[6:]); err == nil {
			colon := -1
			for i, c := range decoded {
				if c == ':' {
					colon = i
					break
				}
			}
			if colon != -1 {
				username = string(decoded[:colon])
				password = string(decoded[colon+1:])
			}
		}
	}
	return username, password
}
