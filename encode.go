package phpserialize

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
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
	writeReflectValue(e, reflect.ValueOf(i))
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
			n = fmt.Sprintf("\x00%s\x00_%s", name, f.Name)
		} else {
			n = f.Name
		}
		writeString(w, n)
		writeReflectValue(w, v.Field(i))
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
