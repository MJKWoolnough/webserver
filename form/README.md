# form
--
    import "github.com/MJKWoolnough/webserver/form"

Package form makes validating web inputs simpler.

## Usage

```go
const (
	FieldNotFound errType = iota
	FieldNotMatch
	FieldNotEqual
	FieldNotMatchEqual
	FieldWrongType
	FieldIntTooLarge
	InputWrongType
	InputWrongKey
)
```
Various defined errors

```go
var Email *regexp.Regexp
```
Precompiled regex for an E-mail Address

#### func  Validate

```go
func Validate(i interface{}, r *http.Request) error
```
Parses the request for the wanted fields and does validation.

#### type Bool

```go
type Bool struct {
}
```


#### func  NewBool

```go
func NewBool(required bool) Bool
```

#### func (Bool) Get

```go
func (b Bool) Get() bool
```

#### type Errors

```go
type Errors map[string]errType
```

Map for returned errors

#### func (Errors) Error

```go
func (e Errors) Error() string
```

#### type Float

```go
type Float struct {
}
```


#### func  NewFloat

```go
func NewFloat(min, max float64) Float
```

#### func (Float) Get

```go
func (f Float) Get() float64
```

#### type Int

```go
type Int struct {
}
```


#### func  NewInt

```go
func NewInt(min, max int) Int
```

#### func (Int) Get

```go
func (i Int) Get() int
```

#### type String

```go
type String struct {
}
```


#### func  NewString

```go
func NewString(equals []string, regex *regexp.Regexp) String
```

#### func (String) Get

```go
func (s String) Get() string
```
