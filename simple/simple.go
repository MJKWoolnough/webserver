package main // import "vimagination.zapto.org/webserver/simple"

import (
	"crypto/tls"
	"flag"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"path"
	"strings"

	"golang.org/x/crypto/acme/autocert"
	"vimagination.zapto.org/httpbuffer"
	_ "vimagination.zapto.org/httpbuffer/deflate"
	_ "vimagination.zapto.org/httpbuffer/gzip"
	"vimagination.zapto.org/httpgzip"
	"vimagination.zapto.org/httplog"
	"vimagination.zapto.org/webserver/contact"
	"vimagination.zapto.org/webserver/proxy/client"
)

var (
	contactForm = flag.Bool("c", false, "enable a contact form at /contact.html")
	fileRoot    = flag.String("r", "", "root of http filesystem")
	logName     = flag.String("n", "", "name for logging")
	logFile     = flag.String("l", "", "filename for request logging")
	serverName  = flag.String("s", "", "server name for HTTPS")
	logger      *log.Logger
)

type http2https struct {
	http.Handler
}

func (hh http2https) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil {
		var url = "https://" + r.Host + r.URL.Path
		if len(r.URL.RawQuery) != 0 {
			url += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, url, http.StatusMovedPermanently)
		return
	}
	hh.Handler.ServeHTTP(w, r)
}

func main() {
	flag.Parse()
	logger = log.New(os.Stderr, *logName, log.LstdFlags)
	if *contactForm {
		from := os.Getenv("contactFormFrom")
		os.Unsetenv("contactFormFrom")
		to := os.Getenv("contactFormTo")
		os.Unsetenv("contactFormTo")
		addr := os.Getenv("contactFormAddr")
		os.Unsetenv("contactFormAddr")
		username := os.Getenv("contactFormUsername")
		os.Unsetenv("contactFormUsername")
		password := os.Getenv("contactFormPassword")
		os.Unsetenv("contactFormPassword")
		p := strings.IndexByte(addr, ':')
		addrMPort := addr
		if p > 0 {
			addrMPort = addrMPort[:p]
		}
		tmpl := template.Must(template.ParseFiles(path.Join(*fileRoot, "contact.html")))
		ec := make(chan error)
		go func() {
			for {
				logger.Println(<-ec)
			}
		}()
		http.Handle("/contact.html", &httpbuffer.Handler{
			&contact.Contact{
				Template: tmpl,
				From:     from,
				To:       to,
				Host:     addr,
				Auth:     smtp.PlainAuth("", username, password, addrMPort),
				Err:      ec,
			},
		})
	}
	http.Handle("/", httpgzip.FileServer(http.Dir(*fileRoot)))
	var (
		lFile  *os.File
		server = &http.Server{
			Handler:  http.DefaultServeMux,
			ErrorLog: logger,
		}
	)
	if *serverName != "" {
		leManager := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache("./certcache/"),
			HostPolicy: autocert.HostWhitelist(*serverName),
		}
		server.Handler = leManager.HTTPHandler(http2https{server.Handler})
		server.TLSConfig = &tls.Config{
			GetCertificate: leManager.GetCertificate,
			NextProtos:     []string{"h2", "http/1.1"},
		}
	}
	if *logFile != "" {
		var err error
		lFile, err = os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger.Printf("Error appending to log file: %s\n", err)
		} else {
			lr, err := httplog.NewWriteLogger(lFile, httplog.DefaultFormat)
			if err != nil {
				logger.Fatalf("error starting request logging: %s\n", err)
			}
			server.Handler = httplog.Wrap(http.DefaultServeMux, lr)
		}

	}
	if err := client.Setup(server); err != nil {
		logger.Fatalf("error setting up server: %s\n", err)
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
		client.Close()
		client.Wait()
		close(cc)
	}()
	err := client.Run()
	if lFile != nil {
		lFile.Close()
	}

	select {
	case <-cc:
	default:
		logger.Println(err)
		cc <- struct{}{}
	}
	<-cc
}
