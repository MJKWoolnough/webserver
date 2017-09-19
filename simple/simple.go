package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"path"
	"strings"

	"github.com/MJKWoolnough/httpgzip"
	"github.com/MJKWoolnough/httplog"
	"github.com/MJKWoolnough/webserver/contact"
	"github.com/MJKWoolnough/webserver/proxy/client"
)

var (
	contactForm = flag.Bool("c", false, "enable a contact form at /contact.html")
	fileRoot    = flag.String("r", "", "root of http filesystem")
	logName     = flag.String("n", "", "name for logging")
	logFile     = flag.String("l", "", "filename for request logging")
	logger      *log.Logger
)

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
		http.Handle("/contact.html", &contact.Contact{
			Template: tmpl,
			From:     from,
			To:       to,
			Host:     addr,
			Auth:     smtp.PlainAuth("", username, password, addrMPort),
			Err:      ec,
		})
	}
	h := httpgzip.FileServer(http.Dir(*fileRoot))
	var lFile *os.File
	if *logFile != "" {
		var err error
		lFile, err = os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger.Printf("Error appending to log file: %s\n", err)
		} else {
			h = httplog.Wrap(h, httplog.NewWriteLogger(lFile, httplog.DefaultFormat))
		}

	}
	http.Handle("/", h)

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
