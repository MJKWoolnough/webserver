package contact

import (
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"

	"github.com/MJKWoolnough/form"
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
	var v values
	r.ParseForm()
	if r.Form.Get("submit") != "" {
		err := form.Parse(&v, r.Form)
		if err == nil {
			err = smtp.SendMail(c.Host, c.Auth, c.From, []string{c.To}, []byte(fmt.Sprintf("To: %s\nFrom: %s\nSubject: Message Received\n\nName: %s\nEmail: %s\nPhone: %s\nSubject: %s\nMessage: %s", c.To, c.From, v.Name, v.Email, v.Phone, v.Subject, v.Message)))
			if c.Err != nil && err != nil {
				c.Err <- err
			}
			v.Done = true
		} else {
			v.Errors = err.(form.Errors)
		}
	}
	c.Template.Execute(w, &v)
}
