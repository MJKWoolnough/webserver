package main

import (
	"html/template"
	"io"
	"net/http"
	"os"
)

type Templates struct {
	header, footer *template.Template
}

var templates Templates

func SetupTemplates(header, footer string) error {
	h, err := template.New("header").ParseFiles(header)
	if err != nil {
		return err
	}
	f, err := template.New("footer").ParseFiles(footer)
	if err != nil {
		return err
	}
	templates = Templates{
		header: h,
		footer: f,
	}
	return nil
}

type Fixed struct {
	Templates
	filename string
}

func (t Templates) Fixed(filename string) Fixed {
	return Fixed{
		t,
		filename,
	}
}

func (f Fixed) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data interface{}
	err := f.header.Execute(w, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	file, err := os.Open(f.filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	_, err = io.Copy(w, file)
	file.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	err = f.footer.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
}
