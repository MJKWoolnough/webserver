# log
--
    import "github.com/MJKWoolnough/webserver/log"

Package log creates a http handler that will output a binary log of all connections.

## Usage

```go
const (
	METHOD_OPTIONS method = iota
	METHOD_GET
	METHOD_HEAD
	METHOD_POST
	METHOD_PUT
	METHOD_DELETE
	METHOD_TRACE
	METHOD_CONNECT
	METHOD_UNKNOWN
)
```

#### func  Credentials

```go
func Credentials(r *http.Request) (string, string)
```
Credentials gets any username and password used in Basic Authentication.

#### func  NewHTTPLog

```go
func NewHTTPLog(dir string, handler http.Handler) *httpLog
```
NewHTTPLog will create the handler that will output to the logs in the given
directory.
