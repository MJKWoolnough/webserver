package main

import (
	"html/template"
	"time"
)

type Treatment struct {
	ID          uint
	Name        string
	Description template.HTML
	Price       uint
	Duration    time.Duration
}
