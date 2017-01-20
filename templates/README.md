# templates
--
    import "github.com/MJKWoolnough/webserver/templates"


## Usage

#### func  Unwatch

```go
func Unwatch(t *Template)
```
Unwatch stops all updates to the given template

#### type Template

```go
type Template struct {
}
```

Template represents a *template.Template which will be automatically updated
when a file it relies on is updated.

Only a sucessful processing of the updated files will update the template. It
will only be left in an unseable state if it cannot be sucessfully processed
when first created by New.

#### func  New

```go
func New(gen func() (*template.Template, error), files ...string) (t *Template, err error)
```
New creates a new watched Template.

All files should use a consistent naming scheme across Templates.

If an error occurs processing the template it will still return an actively
watched Template, but also return an error. The template will not be useable
until it is sucessfully generated after a file change.

#### func (*Template) Get

```go
func (t *Template) Get() *template.Template
```
Get safely gets the underlying *template.Template

#### func (*Template) Regen

```go
func (t *Template) Regen() error
```
Regen regenerates the underlying *template.Template using the original gen func
