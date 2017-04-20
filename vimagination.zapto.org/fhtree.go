package main

import (
	"html/template"
	"net/http"

	"github.com/MJKWoolnough/form"
)

type Tree struct {
	HTMLTemplate, JSTemplate *template.Template
}

type TreeVars struct {
	ID uint
}

func (t *TreeVars) ParserList() form.ParserList {
	return form.ParserList{
		"id": form.Uint{&t.ID},
	}
}

type personHelpers struct {
	*Person
}

func (p personHelpers) SpousePos(spouse int) uint {
	if spouse < 0 {
		return 0
	}
	if spouse == 0 {
		return 1
	}
	if len(p.SpouseOf[spouse].Children) == 0 {
		return p.SpousePos(spouse-1) + 1
	}
	return p.ChildPos(spouse, 1)
}

func (p personHelpers) SiblingPos(sibling int, ignore uint) uint {
	pos := p.SpousePos(len(p.SpouseOf)-1) + 1
	for i := 0; i < sibling; i++ {
		if p.ChildOf.Children[i].ID != ignore {
			pos++
		}
	}
	return pos
}

func (p personHelpers) ChildPos(spouse int, child int) uint {
	var num uint
	for i := 0; i < spouse; i++ {
		num += uint(len(p.SpouseOf[i].Children))
	}
	return num + uint(child)
}

func (t *Tree) HTML(w http.ResponseWriter, r *http.Request) {
	var tv TreeVars
	r.ParseForm()
	form.Parse(&tv, r.Form)
	person := GedcomData.People[tv.ID]
	if person == nil {
		return
	}
	t.HTMLTemplate.Execute(w, personHelpers{person})
}

func (t *Tree) JS(w http.ResponseWriter, r *http.Request) {
	var tv TreeVars
	r.ParseForm()
	form.Parse(&tv, r.Form)
	w.Header().Set("Content-Type", "text/javascript")
	person := GedcomData.People[tv.ID]
	if person == nil {
		return
	}
	t.JSTemplate.Execute(w, person)
}
