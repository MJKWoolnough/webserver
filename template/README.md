# template
--
    import "github.com/MJKWoolnough/webserver/template"

Package template watches template files and updates in memory templates and
static generated files on disk.

## Usage

```go
var (
	Templates = make(map[string]*templateInfo)
)
```

#### func  Menu

```go
func Menu(menu []string)
```
Menu allows the menu list for the header to be set.

#### func  NewStatic

```go
func NewStatic(input, output string, headfooter bool) error
```
NewStatic will generate a new static page from the given template and watch for
updates. If headfooter is true then the static page will also be updated when
the special 'header' and 'footer' templates are updated.

#### func  NewTemplate

```go
func NewTemplate(name, filename string) error
```
NewTemplate will register a new template to be watched for updates. Templates
named 'header' or 'footer' are considered special with regards to the static
templates and all other templates.
