# client
--
    import "github.com/MJKWoolnough/webserver/client"

Package client contains the registration methods to connect a client to a webserver/proxy.

## Usage

#### func  NewTCP4Socket

```go
func NewTCP4Socket(addr, port string) net.Addr
```

#### func  NewUnixSocket

```go
func NewUnixSocket(path string) net.Addr
```

#### func  Register

```go
func Register(serverName string, addr net.Addr, privateKey io.Reader) (net.Listener, error)
```
Register attempts to set-up the connection to the proxy server and returns a
Listener to pass to the http package.
