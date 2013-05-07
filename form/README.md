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

#### func (Bool) String

```go
func (b Bool) String() string
```

#### type Errors

```go
type Errors map[string]errType
```

Map for returned errors

#### func  Validate

```go
func Validate(i interface{}, r *http.Request) Errors
```
Parses the request for the wanted fields and does validation.

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

#### func (Float) String

```go
func (f Float) String() string
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

#### func (Int) String

```go
func (i Int) String() string
```

#### type String

```go
type String struct {
}
```


```go
var (
	Email       *regexp.Regexp
	NotEmptyStr String
)
```
Precompiled regexes

#### func  NewString

```go
func NewString(equals []string, regex *regexp.Regexp) String
```

#### func (String) Get

```go
func (s String) Get() string
```

#### func (String) String

```go
func (s String) String() string
```
