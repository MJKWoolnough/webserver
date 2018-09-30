package contact // import "vimagination.zapto.org/webserver/contact"

import (
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"

	"vimagination.zapto.org/form"
)

type Contact struct {
	Template *template.Template
	From, To string
	Host     string
	Auth     smtp.Auth
	Err      chan<- error
}

type values struct {
	Name, Email, Phone, Subject, Message string
	Errors                               form.Errors
	Done                                 bool
}

func (v *values) ParserList() form.ParserList {
	return form.ParserList{
		"name":    form.RequiredString{&v.Name},
		"email":   form.RequiredString{&v.Email},
		"phone":   form.String{&v.Phone},
		"subject": form.String{&v.Subject},
		"message": form.String{&v.Message},
	}
}

func (c *Contact) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	var v values
	if r.Method == http.MethodPost {
		r.ParseForm()
		if r.Form.Get("submit") != "" {
			err := form.Parse(&v, r.PostForm)
			if err == nil {
				err = smtp.SendMail(c.Host, c.Auth, c.From, []string{c.To}, []byte(fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: Message Received\r\n\r\nName: %s\nEmail: %s\nPhone: %s\nSubject: %s\nMessage: %s", c.To, c.From, v.Name, v.Email, v.Phone, v.Subject, v.Message)))
				if c.Err != nil && err != nil {
					c.Err <- err
				}
				v.Done = true
			} else {
				v.Errors = err.(form.Errors)
			}
		}
	}
	c.Template.Execute(w, &v)
}
