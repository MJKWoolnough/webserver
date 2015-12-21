package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/MJKWoolnough/webserver/proxy"
)

type Site struct {
	Name       string
	Default    bool
	Aliases    []string
	Cmd        string
	Arguments  []string
	WorkingDir string
	Env        []string
	Uid, Gid   uint32
}

type Config struct {
	HTTPAddr  string
	HTTPSAddr string
	Sites     []Site
}

var configFile = flag.String("c", "", "configuration file")

func main() {
	flag.Parse()
	logger := log.New(os.Stderr, "Proxy", log.LstdFlags)
	if *configFile == "" {
		logger.Println("no configuration file")
		return
	}
	f, err := os.Open(*configFile)
	if err != nil {
		logger.Println("error opening configuarion file: ", err)
		return
	}
	var config Config
	err = json.NewDecoder(f).Decode(&config)
	f.Close()
	if err != nil {
		logger.Println("error reading configuration file: ", err)
		return
	}
	if len(config.Sites) == 0 {
		logger.Println("no sites configured")
		return
	}
	var http, https net.Listener
	if config.HTTPAddr != "" {
		http, err = net.Listen("tcp", config.HTTPAddr)
		if err != nil {
			logger.Println("error opening HTTP listener: ", err)
		}
	}
	if config.HTTPSAddr != "" {
		https, err = net.Listen("tcp", config.HTTPSAddr)
		if err != nil {
			logger.Println("error opening HTTPS listener: ", err)
		}
	}
	if http == nil && https == nil {
		logger.Println("no working listeners")
		return
	}
	p := proxy.New(http, https)

	for _, site := range config.Sites {
		cmd := exec.Command(site.Cmd, site.Arguments...)
		cmd.Dir = site.WorkingDir
		cmd.Env = site.Env
		if site.Uid != 0 && site.Gid != 0 {
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Credential: &syscall.Credential{
					Uid: site.Uid,
					Gid: site.Gid,
				},
			}
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		host, err := p.NewHost(cmd)
		if err != nil {
			logger.Printf("error adding host %q: %s\n", site.Name, err)
			continue
		}
		if site.Default {
			p.Default(host)
		}
		host.AddAliases(site.Aliases...)
	}

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
		if http != nil {
			http.Close()
		}
		if https != nil {
			https.Close()
		}
		close(cc)
	}()

	err = p.Run()

	select {
	case <-cc:
	default:
		logger.Println(err)
		cc <- struct{}{}
	}
	<-cc
}
