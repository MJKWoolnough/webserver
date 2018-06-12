package templates // import "vimagination.zapto.org/webserver/templates"

import (
	"html/template"
	"sync"

	"gopkg.in/fsnotify.v1"
)

type templateFiles struct {
	*Template
	files []string
}

var (
	addTemplate    = make(chan templateFiles)
	removeTemplate = make(chan *Template)
)

func init() {
	go watchFiles()
}

// Template represents a *template.Template which will be automatically
// updated when a file it relies on is updated.
//
// Only a sucessful processing of the updated files will update the template.
// It will only be left in an unseable state if it cannot be sucessfully
// processed when first created by New.
type Template struct {
	gen func() (*template.Template, error)

	mu sync.RWMutex
	t  *template.Template
}

// New creates a new watched Template.
//
// All files should use a consistent naming scheme across Templates.
//
// If an error occurs processing the template it will still return an actively
// watched Template, but also return an error. The template will not be useable
// until it is sucessfully generated after a file change.
func New(gen func() (*template.Template, error), files ...string) (t *Template, err error) {
	t = new(Template)
	t.t, err = gen()
	t.gen = gen
	if len(files) > 0 {
		addTemplate <- templateFiles{
			t,
			files,
		}
	}
	return t, err
}

// Get safely gets the underlying *template.Template
func (t *Template) Get() *template.Template {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.t
}

// Regen regenerates the underlying *template.Template using the original gen
// func
func (t *Template) Regen() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	nt, err := t.gen()
	if err != nil {
		return err
	}
	t.t = nt
	return nil
}

// runs in its own goroutine
func watchFiles() {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	files := make(map[string][]*Template)
	for {
		select {
		case tf := <-addTemplate:
			for _, file := range tf.files {
				fs := files[file]
				if len(fs) == 0 {
					fsw.Add(file)
				}
				fs = append(fs, tf.Template)
				files[file] = fs
			}
		case t := <-removeTemplate:
			for name, ts := range files {
				for i := 0; i < len(ts); i++ {
					if ts[i] == t {
						ts[i] = ts[len(ts)-1]
						ts = ts[:len(ts)-1]
						i--
					}
				}
				if len(ts) == 0 {
					delete(files, name)
					fsw.Remove(name)
				} else {
					files[name] = ts
				}
			}
		case e := <-fsw.Events:
			ts := files[e.Name]
			for _, t := range ts {
				t.Regen()
			}
		case <-fsw.Errors:

		}
	}
}

// Unwatch stops all updates to the given template
func Unwatch(t *Template) {
	removeTemplate <- t
}
