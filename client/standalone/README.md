# client
--
    import "github.com/MJKWoolnough/webserver/client/standalone"

Package client is a drop-in replacement webserver/client to create a standalone
webserver without the need of a proxy - useful for testing.

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
func Register(serverName string, addr net.Addr, privateKey io.Reader) (conn net.Listener, err error)
```
Register creates a tcp connection on port 12346 and returns the Listener.
