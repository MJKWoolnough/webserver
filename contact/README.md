# contact
--
    import "vimagination.zapto.org/webserver/contact"


## Usage

#### type Contact

```go
type Contact struct {
	Template *template.Template
	From, To string
	Host     string
	Auth     smtp.Auth
	Err      chan<- error
}
```

Contact contains all of the variables required to create a 'Contact' form

#### func (*Contact) ServeHTTP

```go
func (c *Contact) ServeHTTP(w http.ResponseWriter, r *http.Request)
```
