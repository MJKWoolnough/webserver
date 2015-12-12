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
)

func main() {
	flag.Parse()
	if *contactForm {
		from := os.Getenv("contactFormFrom")
		os.Unsetenv("contactFormFrom")
		to := os.Genenv("contactFormTo")
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
		http.Handler("/contact", contact.New(tmpl, from, to, addr, smtp.PlainAuth("", username, password, addrMPort)))
	}
	http.Handle("/", http.FileServer(*fileRoot))

	cc := make(chan struct{})
	go func() {
		log.Println("Server Started")
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, os.Interrupt)
		select {
		case <-sc:
			log.Println("Closing")
		case <-cc:
		}
		signal.Stop(sc)
		close(sc)
		client.Stop()
		client.Wait()
		close(cc)
	}()

	err := client.Run()

	select {
	case <-cc:
	default:
		log.Println(err)
		cc <- struct{}{}
	}
	<-cc
}
