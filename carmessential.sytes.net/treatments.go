package main

import (
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/MJKWoolnough/bbcode"
)

type BBCodeExporter map[string]func(*bbcode.Tag, bool)

func (b BBCodeExpoerter) Open(t *bbcode.Tag) string {
	return b.export(t, false)
}

func (b BBCodeExporter) Close(t *bbcode.Tag) string {
	return b.export(t, true)
}

func (b BBCodeExporter) export(t *bbcode.Tag, close bool) string {
	if f, ok := b[strings.ToLower(t.Name)]; ok {
		return f(t, close)
	}
	return defaultOut(t)
}

var bbCodeExporter = BBCodeToHTML{
	"b": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</strong>"
		}
		return "<strong>"
	},
	"i": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</em>"
		}
		return "<em>"
	},
	"u": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</u>"
		}
		return "<u>"
	},
	"url": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</a>"
		}
		if !t.Closed {
			return defaultOut(t)
		}
		var (
			u   url.URL
			err error
		)
		if t.Attribute != "" {
			u, err = url.Parse(t.Attribute)
		} else if len(t.Inner) == 1 && t.Inner[0].Name == "@TEXT@" {
			u, err = url.Parse(t.Inner[0].Attribute)
		} else {
			return defaultOut(t)
		}
		if err != nil {
			return defaultOut(t)
		}
		return "<a href=\"" + template.HTMLEscapeString(u.String()) + "\">"
	},
	"center": func(t *bbcode.Tag, close bool) string {},
	"left":   func(t *bbcode.Tag, close bool) string {},
	"right":  func(t *bbcode.Tag, close bool) string {},
	"size":   func(t *bbcode.Tag, close bool) string {},
	"colour": func(t *bbcode.Tag, close bool) string {},
	"color":  func(t *bbcode.Tag, close bool) string { return bbCodeExporter["colour"](t, close) },
	"table":  func(t *bbcode.Tag, close bool) string {},
	"tr":     func(t *bbcode.Tag, close bool) string {},
	"td":     func(t *bbcode.Tag, close bool) string {},
	"th":     func(t *bbcode.Tag, close bool) string {},
	"img":    func(t *bbcode.Tag, close bool) string {},
	"font":   func(t *bbcode.Tag, close bool) string {},
	"list":   func(t *bbcode.Tag, close bool) string {},
	"*":      func(t *bbcode.Tag, close bool) string {},
}

func defaultOut(t *bbcode.Tag) string {
	out := t.BBCode()
	t.Inner = nil
	t.Closed = false
	return out
}

func bbCodeToHTML(text string) template.HTML {
	tags := bbcode.Parse(text)
	// walk tree and combine tags and remove invalid code
	return template.HTML(tags.Export(bbCodeToHTML))
}

type Treatment struct {
	ID          uint
	Name        string
	Description template.HTML
	Price       uint
	Duration    time.Duration
}
