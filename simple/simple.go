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

	"github.com/MJKWoolnough/webserver/contact"
	"github.com/MJKWoolnough/webserver/proxy/client"
)

var (
	contactForm = flag.Bool("c", false, "enable a contact form at /contact.html")
	fileRoot    = flag.String("r", "", "root of http filesystem")
	logName     = flag.String("n", "", "name for logging")
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
		http.Handle("/contact.html", &contact.Contact{
			Template: tmpl,
			From:     from,
			To:       to,
			Host:     addr,
			Auth:     smtp.PlainAuth("", username, password, addrMPort),
		})
	}
	http.Handle("/", http.FileServer(http.Dir(*fileRoot)))

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

	select {
	case <-cc:
	default:
		logger.Println(err)
		cc <- struct{}{}
	}
	<-cc
}
