package phpserialize

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"

	"github.com/kamiaka/go-phpserialize/php"
)

// Marshaler is the interface implemented by types that can marshal themselves
//  into valid PHP serialize.
type Marshaler interface {
	MarshalPHPSerialize() ([]byte, error)
}

// Marshal returns the PHP serialized bytes of i.
func Marshal(i interface{}) ([]byte, error) {
	e := newEncodeState()

	err := e.marshal(i)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), e.Bytes()...), nil
}

type encodeState struct {
	bytes.Buffer
}

func newEncodeState() *encodeState {
	return new(encodeState)
}

type serializeErr struct {
	error
}

func (e *encodeState) marshal(i interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(serializeErr); ok {
				err = e.error
			} else {
				panic(r)
			}
		}
	}()
	writeInterface(e, i)
	return nil
}

// UnsupportedTypeError is returned when attempting to encode an unsupported value.
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) Error() string {
	return "PHP serialize: unsupported type: " + e.Type.String()
}

// UnsupportedMapKeyTypeError is returned when attempting to encode an unsupported map key.
type UnsupportedMapKeyTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedMapKeyTypeError) Error() string {
	return "PHP serialize: unsupported map key type: " + e.Type.String()
}

// fixed serialized values
var (
	sNil    = []byte("N;")
	sTrue   = []byte("b:1;")
	sFalse  = []byte("b:0;")
	sNAN    = []byte("d:NAN;")
	sInf    = []byte("d:INF;")
	sNegInf = []byte("d:-INF;")
)

func writeNil(w io.Writer) {
	w.Write(sNil)
}

func writeBool(w io.Writer, b bool) {
	if b {
		w.Write(sTrue)
	} else {
		w.Write(sFalse)
	}
}

func writeInt(w io.Writer, v int64) {
	fmt.Fprintf(w, "i:%d;", v)
}

func writeUint(w io.Writer, v uint64) {
	fmt.Fprintf(w, "i:%d;", v)
}

func writeFloat(w io.Writer, f float64) {
	if math.IsNaN(f) {
		w.Write(sNAN)
	} else if math.IsInf(f, -1) {
		w.Write(sNegInf)
	} else if math.IsInf(f, 1) {
		w.Write(sInf)
	} else {
		fmt.Fprintf(w, "d:%v;", f)
	}
}

func writeString(w io.Writer, s string) {
	fmt.Fprintf(w, `s:%d:"%s";`, len(s), s)
}

func writeArray(w io.Writer, v reflect.Value) {
	l := v.Len()
	fmt.Fprintf(w, "a:%d:{", l)
	for i := 0; i < l; i++ {
		writeInt(w, int64(i))
		writeReflectValue(w, v.Index(i))
	}
	w.Write([]byte{'}'})
}

func intVal(v reflect.Value) (i int64, ok bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(v.Uint()), true
	case reflect.Interface:
		return intVal(reflect.ValueOf(v.Interface()))
	default:
		return 0, false
	}
}

func sortKeys(keys []reflect.Value) {
	sort.Slice(keys, func(i, j int) bool {
		a, ak := intVal(keys[i])
		b, bk := intVal(keys[j])
		if ak && bk {
			return a < b
		}
		if ak && !bk {
			return true
		}
		if bk {
			return false
		}
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
}

func writeMap(w io.Writer, v reflect.Value) {
	keys := v.MapKeys()
	sortKeys(keys)
	fmt.Fprintf(w, "a:%d:{", len(keys))
	for _, k := range keys {
		writeMapKey(w, k)
		writeReflectValue(w, v.MapIndex(k))
	}
	w.Write([]byte{'}'})
}

func writeMapKey(w io.Writer, v reflect.Value) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		writeInt(w, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		writeUint(w, v.Uint())
	case reflect.String:
		writeString(w, v.String())
	case reflect.Interface:
		writeMapKey(w, reflect.ValueOf(v.Interface()))
	default:
		raiseError(&UnsupportedMapKeyTypeError{v.Type()})
	}
}

func writeStruct(w io.Writer, v reflect.Value) {
	name := v.Type().Name()
	t := v.Type()
	num := t.NumField()
	fmt.Fprintf(w, `O:%d:"%s":%d:{`, len(name), name, num)

	for i := 0; i < num; i++ {
		f := t.Field(i)
		var n string
		if 'a' <= f.Name[0] && f.Name[0] <= 'z' {
			n = fmt.Sprintf("\x00%s\x00%s", name, f.Name)
		} else {
			n = f.Name
		}
		writeString(w, n)
		writeReflectValue(w, v.Field(i))
	}
	w.Write([]byte{'}'})
}

func writeInterface(w io.Writer, i interface{}) {
	if v, ok := i.(Marshaler); ok {
		bs, err := v.MarshalPHPSerialize()
		if err != nil {
			panic(serializeErr{err})
		}
		w.Write(bs)
		return
	}
	if v, ok := i.(*php.Value); ok {
		writePHPValue(w, v)
		return
	}
	writeReflectValue(w, reflect.ValueOf(i))
}

func writePHPValue(w io.Writer, v *php.Value) {
	if v.IsNil() {
		writeNil(w)
		return
	}
	switch v.Type() {
	case php.TypeBool:
		writeBool(w, v.Bool())
	case php.TypeInt:
		writeInt(w, v.Int())
	case php.TypeFloat:
		writeFloat(w, v.Float())
	case php.TypeString:
		writeString(w, v.String())
	case php.TypeArray:
		writePHPArray(w, v.Array())
	case php.TypeObject:
		writePHPObject(w, v.Object())
	default:
		panic(serializeErr{fmt.Errorf("invalid PHPValue Type: %v", v.Type())})
	}
}

func writePHPArray(w io.Writer, arr []*php.ArrayElement) {
	fmt.Fprintf(w, "a:%d:{", len(arr))
	for _, val := range arr {
		writePHPValue(w, val.Index)
		writePHPValue(w, val.Value)
	}
	w.Write([]byte{'}'})
}

func writePHPObject(w io.Writer, obj *php.Obj) {
	fmt.Fprintf(w, `O:%d:"%s":%d:{`, len(obj.Name), obj.Name, len(obj.Fields))
	for _, f := range obj.Fields {
		var name string
		switch f.Visibility {
		case php.VisibilityProtected:
			name = fmt.Sprintf("*%s", f.Name)
		case php.VisibilityPrivate:
			name = fmt.Sprintf("\x00%s\x00%s", obj.Name, f.Name)
		default: // public
			name = f.Name
		}
		writeString(w, name)
		writePHPValue(w, f.Value)
	}
	w.Write([]byte{'}'})
}

func writeReflectValue(w io.Writer, v reflect.Value) {
	if !v.IsValid() {
		writeNil(w)
		return
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			writeNil(w)
			return
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Bool:
		writeBool(w, v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		writeInt(w, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		writeUint(w, v.Uint())
	case reflect.Float32, reflect.Float64:
		writeFloat(w, v.Float())
	case reflect.String:
		writeString(w, v.String())
	case reflect.Array, reflect.Slice:
		writeArray(w, v)
	case reflect.Map:
		writeMap(w, v)
	case reflect.Struct:
		writeStruct(w, v)
	case reflect.Interface:
		writeReflectValue(w, reflect.ValueOf(v.Interface()))
	default:
		raiseError(&UnsupportedTypeError{v.Type()})
	}
}

func raiseError(e error) {
	panic(serializeErr{e})
}
