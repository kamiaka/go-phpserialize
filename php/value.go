package php

import (
	"math"
)

// Value represents PHP value
type Value struct {
	t Type
	i interface{}
}

// A ValueError occurs when a method is invoked on a Value that does not support it.
type ValueError struct {
	Method string
	Type   Type
}

func (e *ValueError) Error() string {
	if e.Type == 0 {
		return "php: call of " + e.Method + " on zero value"
	}
	return "php: call of " + e.Method + " on " + e.Type.String() + " Value"
}

func valueError(method string, t Type) {
	panic(&ValueError{
		Method: method,
		Type:   t,
	})
}

// Type returns PHP value type.
func (v *Value) Type() Type {
	return v.t
}

// Bool returns v's underlying value.
func (v *Value) Bool() bool {
	uv, ok := v.i.(bool)
	if !ok {
		valueError("php.Value.Bool", v.t)
	}
	return uv
}

// Int returns v's underlying value.
func (v *Value) Int() int64 {
	uv, ok := v.i.(int64)
	if !ok {
		valueError("php.Value.Int", v.t)
	}
	return uv
}

// Float returns v's underlying value.
func (v *Value) Float() float64 {
	uv, ok := v.i.(float64)
	if !ok {
		valueError("php.Value.Float", v.t)
	}
	return uv
}

// String returns v's underlying value.
// String is as special case because of Go's String method convention.
// Unlike the other getters, it does not panic if v's value is not String.
// Instead, it returns as string of the form "<T Value>" where T is v's type.
func (v *Value) String() string {
	uv, ok := v.i.(string)
	if !ok {
		return "<" + v.Type().String() + " value>"
	}
	return uv
}

// Array returns v's underlying value.
func (v *Value) Array() []*ArrayElement {
	uv, ok := v.i.([]*ArrayElement)
	if !ok {
		valueError("php.Value.Array", v.t)
	}
	return uv
}

// Keys returns v's array keys.
//  It panics if v's type is not array.
func (v *Value) Keys() []*Value {
	a := v.Array()
	keys := make([]*Value, len(a))
	for i, e := range a {
		keys[i] = e.Index
	}
	return keys
}

// Index returns v's element, returns nil if not found.
//  It panics if v's type is not array.
func (v *Value) Index(index *Value) *Value {
	for _, e := range v.Array() {
		if e.Index == index {
			return e.Value
		}
	}
	return nil
}

// IndexByName returns found v's element by index name, returns nil if not found.
func (v *Value) IndexByName(name string) *Value {
	for _, e := range v.Array() {
		if e.Index.Interface() == name {
			return e.Value
		}
	}
	return nil
}

// Object returns v's underlying value.
func (v *Value) Object() *Obj {
	uv, ok := v.i.(*Obj)
	if !ok {
		valueError("php.Value.Object", v.t)
	}
	return uv
}

// IsNil reports whether it's argument v is nil (PHP null)
func (v *Value) IsNil() bool {
	return v == nil || v.t == TypeNull
}

// Interface returns v's current value as an interface{}.
func (v *Value) Interface() interface{} {
	return v.i
}

// ArrayElement represents Array member.
//   array index must be int or string PHP value.
type ArrayElement struct {
	Index *Value
	Value *Value
}

// Obj ...
type Obj struct {
	Name   string
	Fields []*ObjField
}

// ObjField represents Array or Object member
type ObjField struct {
	Name       string
	Visibility Visibility
	Value      *Value
}

// Visibility for PHP class member
type Visibility uint

// Visibility list
const (
	VisibilityPublic Visibility = iota
	VisibilityProtected
	VisibilityPrivate
)

// Null returns null PHP Value
func Null() *Value {
	return &Value{
		t: TypeNull,
		i: nil,
	}
}

// Bool returns bool PHP Value.
func Bool(v bool) *Value {
	return &Value{
		t: TypeBool,
		i: v,
	}
}

// Int returns int PHP Value.
func Int(v int) *Value {
	return &Value{
		t: TypeInt,
		i: int64(v),
	}
}

// Float returns float PHP Value.
func Float(v float64) *Value {
	return &Value{
		t: TypeFloat,
		i: v,
	}
}

// NaN returns IEEE754 Not a number PHP Value.
func NaN() *Value {
	return Float(math.NaN())
}

// Inf returns positive infinity float PHP value if sign >= 0, negative infinity float PHP value if sign < 0.
func Inf(sign int) *Value {
	return Float(math.Inf(sign))
}

// String returns string PHP Value.
func String(v string) *Value {
	return &Value{
		t: TypeString,
		i: v,
	}
}

// Array returns array PHP Value.
func Array(v ...*ArrayElement) *Value {
	return &Value{
		t: TypeArray,
		i: v,
	}
}

// Append appends the values es to an array PHP value v.
//   v's value must be array PHP value.
func Append(v *Value, es ...*Value) *Value {
	ls := v.Array()
	next := 0
	for _, e := range ls {
		if e.Index.t == TypeInt && next <= int(e.Index.Int()) {
			next = int(e.Index.Int()) + 1
		}
	}
	for _, e := range es {
		ls = append(ls, Element(Int(next), e))
		next++
	}
	return Array(ls...)
}

// Element returns element of array PHP Value.
func Element(index, value *Value) *ArrayElement {
	return &ArrayElement{
		Index: index,
		Value: value,
	}
}

// Object returns object PHP Value.
func Object(name string, fields ...*ObjField) *Value {
	return &Value{
		t: TypeObject,
		i: &Obj{
			Name:   name,
			Fields: fields,
		},
	}
}

// Field returns PHP object field.
func Field(name string, v *Value, vis Visibility) *ObjField {
	return &ObjField{
		Name:       name,
		Visibility: vis,
		Value:      v,
	}
}

// PubField returns PHP object public field.
func PubField(name string, v *Value) *ObjField {
	return Field(name, v, VisibilityPublic)
}

// PrivField returns PHP object private field.
func PrivField(name string, v *Value) *ObjField {
	return Field(name, v, VisibilityPublic)
}

// ProtectedField returns PHP object protected field.
func ProtectedField(name string, v *Value) *ObjField {
	return Field(name, v, VisibilityPublic)
}
