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

type Bool struct {
	data, required bool
}

func NewBool(required bool) Bool {
	return Bool { false, required }
}

func (b Bool) Get() bool {
	return b.data
}

type Int struct {
	data, min, max int
}

func NewInt(min, max int) Int {
	return Int { 0, min, max }
}

func (i Int) Get() int {
	return i.data
}

type Float struct {
	data, min, max float64
}

func NewFloat(min, max float64) Float {
	return Float { 0, min, max }
}

func (f Float) Get() float64 {
	return f.data
}

type String struct {
	data string
	equal []string
	regex *regexp.Regexp
}

func NewString(equals []string, regex *regexp.Regexp) String {
	return String { "", equals, regex }
}

func (s String) Get() string {
	return s.data
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
func Validate(i *interface{}, r *http.Request) error {
	if r.Form == nil {
		r.FormValue("")
	}
	v := reflect.ValueOf(i)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	var errors Errors = make(map[string]errType)
	if v.Kind() == reflect.Struct {
		for j := 0; j < v.NumField(); j++ {
			fieldV := v.Field(j)
			if !fieldV.CanInterface() || !fieldV.CanSet() {
				continue
			}
			fieldT := v.Type().Field(j)
			name := fieldT.Name
			if fieldT.Tag != "" {
				name = string(fieldT.Tag)
			}
			data, err := process(r, name, fieldV.Interface())
			if data != nil {
				fieldV.Set(reflect.ValueOf(data))
			}
			if err != noError {
				errors[name] = err
			}
		}
	} else if v.Kind() == reflect.Map {
		if v.Type().Key().Kind() != reflect.String {
			return InputWrongKey
		}
		for _, key := range v.MapKeys() {
			elm := v.MapIndex(key)
			if !elm.CanInterface() || !elm.CanSet() {
				continue
			}
			data, err := process(r, key.String(), v.MapIndex(key).Interface())
			if data != nil {
				elm.Set(reflect.ValueOf(data))
			}
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

func process(r *http.Request, name string, data interface{}) (interface{}, errType) {
	var value string
	if v, ok := r.Form[name]; !ok {
		return nil, FieldNotFound
	} else if len(v) == 0 {
		return nil, FieldNotFound
	} else {
		value = v[0]
	}
	
	switch d := (data).(type) {
		case int8:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case uint8:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case int16:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case uint16:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case int32:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case uint32:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case int64:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case uint64:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case int:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case uint:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case Int:
			_, err := fmt.Sscan(value, &d.data)
			if e := checkErr(err); e != noError {
				return d, e
			}
			if d.data < d.min || d.data > d.max {
				return d, FieldNotMatch
			}
			return d, noError
		case float32:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case float64:
			_, err := fmt.Sscan(value, &d)
			return d, checkErr(err)
		case Float:
			_, err := fmt.Sscan(value, &d.data)
			if e := checkErr(err); e != noError {
				return d, e
			}
			if d.data < d.min || d.data > d.max {
				return d, FieldNotMatch
			}
			return d, noError
		case string:
			return value, noError
		case String:
			d.data = value
			if d.regex == nil && d.equal == nil {
				return d, noError
			}
			if d.equal != nil {
				for _, equal := range d.equal {
					if value == equal {
						return d, noError
					}
				}
				if d.regex == nil {
					return d, FieldNotEqual
				}
			}
			if d.regex != nil {
				if d.regex.Match([]byte(value)) {
					return d, noError
				}
				if d.equal == nil {
					return d, FieldNotMatch
				}
			}
			return d, FieldNotMatchEqual
		case bool:
			value = strings.ToLower(value)
			d = value != "" && value != "false" && value != "0" && value != "off" && value != "no" && value != "\000"
			return d, noError
		case Bool:
			value = strings.ToLower(value)
			d.data = value != "" && value != "false" && value != "0" && value != "off" && value != "no" && value != "\000"
			if d.data != d.required {
				return d, FieldNotEqual
			}
			return d, noError
	}
	return nil, InputWrongType
}

func checkErr(err error) errType {
	if e, ok := err.(*strconv.NumError); ok {
		if e.Err == strconv.ErrRange {
			return FieldIntTooLarge
		} else if e.Err == strconv.ErrSyntax {
			return FieldWrongType
		}
	}
	return noError
}