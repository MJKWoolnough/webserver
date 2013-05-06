// Copyright (c) 2013 - Michael Woolnough <michael.woolnough@gmail.com>
// 
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met: 
// 
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer. 
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution. 
// 
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Package form makes validating web inputs simpler.
package form

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"strconv"
)

// Various defined errors
const (
	FieldNotFound errType = iota
	FieldNotMatch
	FieldNotEqual
	FieldNotMatchEqual
	FieldWrongType
	FieldIntTooLarge
	InputWrongType
	InputWrongKey
	noError
)

// Precompiled regex for an E-mail Address
var Email *regexp.Regexp

type number interface {
	data()    interface{}
	inRange() bool
}

type Bool struct {
	d, required bool
}

func NewBool(required bool) Bool {
	return Bool { false, required }
}

func (b Bool) Get() bool {
	return b.d
}

type Int struct {
	d, min, max int
}

func NewInt(min, max int) Int {
	return Int { 0, min, max }
}

func (i *Int) data() interface{} {
	return &i.d
}

func (i Int) inRange() bool {
	return i.d >= i.min && i.d <= i.max
}

func (i Int) Get() int {
	return i.d
}

type Float struct {
	d, min, max float64
}

func NewFloat(min, max float64) Float {
	return Float { 0, min, max }
}

func (f *Float) data() interface{} {
	return &f.d
}

func (f Float) inRange() bool {
	return f.d >= f.min && f.d <= f.max
}

func (f Float) Get() float64 {
	return f.d
}

type String struct {
	d string
	match []string
	regex *regexp.Regexp
}

func NewString(equals []string, regex *regexp.Regexp) String {
	return String { "", equals, regex }
}

func (s String) Get() string {
	return s.d
}

type errType int8

func (e errType) Error() string {
	switch e {
		case FieldNotFound:
			return "The requested field was not found"
		case FieldNotEqual:
			return "The field did not match the regular expression"
		case FieldNotMatch:
			return "The field did not equal the values given"
		case FieldNotMatchEqual:
			return "The field did not equal the values given or match the regular expression"
		case FieldWrongType:
			return "The field could not be marshalled into the requested type"
		case FieldIntTooLarge:
			return "The number given was too large to fit into the int"
		case InputWrongType:
			return "The data given was not of a recognised type"
		case InputWrongKey:
			return "Map Key needs to be a string"
	}
	return "Unknown"
}

// Map for returned errors
type Errors map[string]errType

func (e Errors) Error() string {
	err := "The following errors were found: -"
	for name, errT := range e {
		err += fmt.Sprintf("\n%q: %q", name, errT.Error())
	}
	return err
}

func init() {
	Email, _ = regexp.Compile("^[a-zA-Z0-9][a-zA-Z0-9_.-+]*@[a-zA-Z0-9.-]+.[a-zA-Z]{2,4}$")
}

// Parses the request for the wanted fields and does validation.
func Validate(i interface{}, r *http.Request) error {
	r.FormValue("")
	v := reflect.ValueOf(i)
	var errors Errors = make(map[string]errType)
	if v.Kind() == reflect.Struct {
		for j := 0; j < v.NumField(); j++ {
			fieldV := v.Field(j)
			if !fieldV.CanInterface() {
				continue
			}
			fieldT := v.Type().Field(j)
			name := fieldT.Name
			if fieldT.Tag != "" {
				name = string(fieldT.Tag)
			}
			data := fieldV.Interface()
			err := process(r, name, data)
			if err != noError {
				errors[name] = err
			}
		}
	} else if v.Kind() == reflect.Map {
		if v.Type().Key().Kind() != reflect.String {
			return InputWrongKey
		}
		for _, key := range v.MapKeys() {
			name := key.String()
			data := v.MapIndex(key).Interface()
			err := process(r, name, data)
			if err != noError {
				errors[name] = err
			}
		}
	} else {
		return InputWrongType
	}
	if len(errors) == 0 {
		return nil
	}
	return errors
}

func process(r *http.Request, name string, data interface{}) errType {
	var value string
	if v, ok := r.Form[name]; !ok {
		return FieldNotFound
	} else if len(v) == 0 {
		return FieldNotFound
	} else {
		value = v[0]
	}
	
	switch d := data.(type) {
		case *int8, *uint8, *int16, *uint16, *int32, *uint32, *int64, *uint64, *int, *uint, *float32, *float64, number:
			var v interface{}
			n, ok := d.(number); 
			if ok {
				v = n.data()
			} else {
				v = d
			}
			_, err := fmt.Sscan(value, v)
			if e, ok := err.(*strconv.NumError); ok {
				if e.Err == strconv.ErrRange {
					return FieldIntTooLarge
				} else if e.Err == strconv.ErrSyntax {
					return FieldWrongType
				}
			}
			if ok {
				if n.inRange() {
					return FieldNotMatch
				}
			}
			return noError
		case *string:
			*d = value
			return noError
		case *String:
			d.d = value
			if d.match != nil {
				for _, match := range d.match {
					if value == match {
						return noError
					}
				}
			}
			if d.regex == nil {
				if d.match == nil {
					return FieldNotMatch
				}
				return FieldNotEqual
			} else {
				if d.regex.Match([]byte(value)) {
					return noError
				}
			}
			return FieldNotMatchEqual
		case *bool:
			value = strings.ToLower(value)
			*d = value != "" && value != "false" && value != "0" && value != "off" && value != "no" && value != "\000"
			return noError
		case *Bool:
			value = strings.ToLower(value)
			d.d = value != "" && value != "false" && value != "0" && value != "off" && value != "no" && value != "\000"
			if d.d != d.required {
				return FieldNotEqual
			}
			return noError
	}
	return InputWrongType
}