package main // import "vimagination.zapto.org/webserver/forward"

import (
	"encoding/binary"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

var (
	httpAddr  = flag.String("h", ":8080", "address to proxy http to")
	httpsAddr = flag.String("s", ":8443", "address to proxy https to")
	logName   = flag.String("n", "", "name for logging")
	logger    *log.Logger
)

const (
	IPHeader   = "X-Forwarded-For"
	PortHeader = "X-Forwarded-Port"
)

func proxyConn(envName, toAddr string) error {
	if hfd, ok := os.LookupEnv(envName); ok {
		fd, _ := strconv.ParseUint(hfd, 10, 0)
		os.Unsetenv(envName)
		c, err := net.FileConn(os.NewFile(uintptr(fd), ""))
		if err != nil {
			return err
		}
		u, ok := c.(*net.UnixConn)
		if !ok {
			return errors.New("invalid socket type")
		}
		length := make([]byte, 4)
		oob := make([]byte, syscall.CmsgSpace(4))
		for {
			_, _, _, _, err = u.ReadMsgUnix(length, oob)
			if err != nil {
				return err
			}
			buf := make([]byte, binary.LittleEndian.Uint32(length))
			if len(buf) > 0 {
				_, err = u.Read(buf)
				if err != nil {
					return err
				}
			}
			msg, err := syscall.ParseSocketControlMessage(oob)
			if err != nil {
				return err
			}
			if len(msg) != 1 {
				return errors.New("invalid number of socket control messages")
			}
			fd, err := syscall.ParseUnixRights(&msg[0])
			if err != nil {
				return err
			}
			if len(fd) != 1 {
				return errors.New("invalid number of file descriptors")
			}
			c, err := net.FileConn(os.NewFile(uintptr(fd[0]), ""))
			if err != nil {
				return err
			}
			// add IPHeader and PortHeader
			go forward(buf, c, toAddr)
		}
	}
	return nil
}

func forward(buf []byte, c net.Conn, toAddr string) {
	f, err := net.Dial("tcp", toAddr)
	if err != nil {
		logger.Println("error connecting to host: ", err)
		return
	}
	_, err = f.Write(buf)
	if err != nil {
		logger.Println("error forwarding buffer: ", err)
		return
	}
	ec := make(chan error, 1)
	go copyConn(c, f, ec)
	go copyConn(f, c, ec)

	err = <-ec
	if err != nil && err != io.EOF {
		logger.Println("error forwarding conn: ", err)
	}
}

func copyConn(a, b net.Conn, ec chan error) {
	_, err := io.Copy(a, b)
	ec <- err
}

func main() {
	flag.Parse()
	logger = log.New(os.Stderr, *logName, log.LstdFlags)
	ec := make(chan error, 1)
	go func() {
		ec <- proxyConn("proxyHTTPSocket", *httpAddr)
	}()
	go func() {
		ec <- proxyConn("proxyHTTPSSocket", *httpsAddr)
	}()

	cc := make(chan struct{})
	go func() {
		logger.Println("Server Started")
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, os.Interrupt)
		select {
		case <-sc:
			logger.Println("Closing")
		case <-cc:
		}
		signal.Stop(sc)
		close(sc)
		close(cc)
	}()

	err := <-ec

	if err == nil {
		err = <-ec
	}

	select {
	case <-cc:
	default:
		if err == nil {
			logger.Println("no sockets")
		} else {
			logger.Println(err)
		}
		cc <- struct{}{}
	}
	<-cc
}
