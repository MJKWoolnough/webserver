# cache
--
    import "github.com/MJKWoolnough/webserver/cache"

Package cache allows clients to store resources that expire after a given duration.

## Usage

#### func  Insert

```go
func Insert(name string, resource Resource, duration time.Duration)
```
Insert adds a new named resource into the cache which will expire after the
given duration. If a resource with the same name already exists it will Remove
that resource, without executing the Expire function, and replace it with the
new resource.

#### func  Remove

```go
func Remove(name string) error
```
Remove will delete the named resource from the cache. The Expiry method will not
be executed.

#### type ErrNotFound

```go
type ErrNotFound struct {
}
```

ErrNotFound is returned when trying to Read or Remove a name that does not
exist.

#### func (ErrNotFound) Error

```go
func (e ErrNotFound) Error() string
```

#### type Resource

```go
type Resource interface {
	// Expire will be executed after the given duration has passed allowing the data
	// processed or save in another format.
	Expire(time.Time)
}
```

Resource represents the clients data.

#### func  Read

```go
func Read(name string) (Resource, error)
```
Read allows a client to retrive a named resource from the cache.
