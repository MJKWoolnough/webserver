# types
--
    import "github.com/MJKWoolnough/webserver/client/types"

Package types contains types and methods shared by the webserver/client and webserver/proxy packages.

## Usage

#### type ConnectServer

```go
type ConnectServer struct {
	ServerName string
	Net, Addr  string
	Time       time.Time
	Signature  []byte
}
```

ConnectServer contains the information needed to update the proxys information
about the client.

#### func (*ConnectServer) GetData

```go
func (c *ConnectServer) GetData() (serverName string, signature []byte, t time.Time)
```

#### func (*ConnectServer) Sign

```go
func (c *ConnectServer) Sign(key *rsa.PrivateKey) (err error)
```

#### type NewServer

```go
type NewServer struct {
	ServerName string
	Aliases    []string
	PublicKey  *rsa.PublicKey
	//sslCert
	Time      time.Time
	Signature []byte
}
```

NewServer contains the information needed to register a new client with the
proxy.

#### func (*NewServer) GetData

```go
func (n *NewServer) GetData() (serverName string, signature []byte, t time.Time)
```

#### func (*NewServer) Sign

```go
func (n *NewServer) Sign(key *rsa.PrivateKey) (err error)
```

#### type RemoveServer

```go
type RemoveServer struct {
	ServerName string
	Time       time.Time
	Signature  []byte
}
```

ConnectServer contains the information needed to remove the client.

#### func (*RemoveServer) GetData

```go
func (r *RemoveServer) GetData() (serverName string, signature []byte, t time.Time)
```

#### func (*RemoveServer) Sign

```go
func (r *RemoveServer) Sign(key *rsa.PrivateKey) (err error)
```
