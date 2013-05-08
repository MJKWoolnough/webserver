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

// Package template watches template files and updates in memory templates and
// static generated files on disk.
package template

import (
	"bytes"
	"github.com/MJKWoolnough/io-watcher"
	"html/template"
	"os"
	"path/filepath"
)

type headerData struct {
	Title string
	CSS   string
	Menu  map[string]bool
}

type static struct {
	inFilename  string
	outFilename string
}

// NewStatic will generate a new static page from the given template and watch for updates.
// If headfooter is true then the static page will also be updated when the special 'header'
// and 'footer' templates are updated.
func NewStatic(input, output string, headfooter bool) error {
	s := static{input, output}
	if err := s.update(); err != nil {
		return err
	}
	if err := watcher.Watch(input, s); err != nil {
		return err
	}
	if headfooter {
		statics = append(statics, s)
	}
	return nil
}

func (s static) Update(filename string, code uint8) {
	if code != watcher.WATCH_MODIFY {
		return
	}
	s.update()
}

func (s static) update() error {
	t, err := template.New(filepath.Base(s.inFilename)).Funcs(templateFuncList).ParseFiles(s.inFilename)
	if err != nil {
		return err
	}
	f, err := os.Create(s.outFilename)
	if err != nil {
		return err
	}
	t.Execute(f, nil)
	f.Close()
	return nil
}

type templateInfo struct {
	*template.Template
}

// NewTemplate will register a new template to be watched for updates.
// Templates named 'header' or 'footer' are considered special with regards
// to the static templates and all other templates.
func NewTemplate(name, filename string) error {
	if name == "" {
		name = filepath.Base(filename)
	}
	t := new(templateInfo)
	t.Template = template.New(name)
	if err := t.update(filename); err != nil {
		return err
	}
	if err := watcher.Watch(filename, t); err != nil {
		return err
	}
	Templates[name] = t
	return nil
}

func (t *templateInfo) Update(filename string, code uint8) {
	if code != watcher.WATCH_MODIFY {
		return
	}
	t.update(filename)
}

func (t *templateInfo) update(filename string) error {
	name := t.Name()
	if name == "header" || name == "footer" {
		buf := new(bytes.Buffer)
		if f, err := os.Open(filename); err != nil {
			return err
		} else {
			buf.ReadFrom(f)
			f.Close()
		}
		temp, err := template.New(name).Parse(string(buf.Bytes()))
		if err != nil {
			return err
		}
		t.Template = temp
		for _, s := range statics {
			s.update()
		}
	} else {
		temp, err := template.New(name).Funcs(templateFuncList).ParseFiles(filename)
		if err != nil {
			return err
		}
		t.Template = temp
	}
	return nil
}

func header(title, css, menu string) (template.HTML, error) {
	head := new(headerData)
	if title != "" {
		head.Title = " - " + title
	}
	head.CSS = css
	head.Menu = make(map[string]bool)
	for _, m := range menuList {
		head.Menu[m] = m == menu
	}
	buf := new(bytes.Buffer)
	Templates["header"].Execute(buf, head)
	return template.HTML(buf.String()), nil
}

func footer() (template.HTML, error) {
	var buf bytes.Buffer
	Templates["footer"].Execute(&buf, nil)
	return template.HTML(buf.String()), nil
}

// Menu allows the menu list for the header to be set.
func Menu(menu []string) {
	menuList = menu
}

var (
	statics          []static
	Templates        = make(map[string]*templateInfo)
	menuList         []string
	templateFuncList = template.FuncMap{
		"header": header,
		"footer": footer,
	}
)
