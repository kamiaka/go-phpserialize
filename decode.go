package phpserialize

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/kamiaka/go-phpserialize/php"
)

// Unmarshal returns the PHP unserialized Value of bs.
func Unmarshal(data []byte) (*php.Value, error) {
	s := newDecodeState(data)

	return s.unmarshal()
}

type decodeState struct {
	data []byte
	off  int
}

func newDecodeState(data []byte) *decodeState {
	return &decodeState{
		data: data,
	}
}

func (d *decodeState) error(format string, args ...interface{}) error {
	panic(serializeErr{fmt.Errorf("php serialize: %v", fmt.Sprintf(format, args...))})
}

func (d *decodeState) unmarshal() (v *php.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(serializeErr); ok {
				err = e.error
			} else {
				panic(r)
			}
		}
	}()

	v = d.readValue()
	if !d.isEOF() {
		d.error("unexpected token: %s, position: %d", []byte{d.data[d.off]}, d.off)
	}
	return
}

func (d *decodeState) isEOF() bool {
	return len(d.data) <= d.off
}

func (d *decodeState) skipEq(str string) {
	bs := []byte(str)
	l := len(bs)
	end := d.off + l
	if len(d.data) < end {
		d.error("cannot read byte: %v", io.EOF)
		return
	}
	got := d.data[d.off:end]
	for i := 0; i < l; i++ {
		if bs[i] != got[i] {
			d.error("unexpected token %s, position: %d", []byte{got[i]}, end)
			return
		}
	}
	d.off = end
}

func (d *decodeState) readBytes(delim byte) []byte {
	i := bytes.IndexByte(d.data[d.off:], delim)
	end := d.off + i
	if i < 0 {
		d.error("unexpected EOF, want: %s, from position: %d", []byte{delim}, d.off)
		return nil
	}
	data := d.data[d.off:end]
	d.off = end + 1

	return data
}

func (d *decodeState) readValue() *php.Value {
	if d.isEOF() {
		d.error("unexpected EOF in read value type, position: %d", d.off)
		return nil
	}
	switch d.data[d.off] {
	case 'N':
		return d.readNil()
	case 'b':
		return d.readBool()
	case 'i':
		return d.readInt()
	case 's':
		return d.readString()
	case 'd':
		return d.readFloat()
	case 'a':
		return d.readArray()
	case 'O':
		return d.readObject()
	default:
		d.error("unexpected token %s at position: %d", []byte{d.data[d.off]}, d.off)
		return nil
	}
}

func (d *decodeState) readNil() *php.Value {
	d.skipEq("N;")
	return php.Null()
}

func (d *decodeState) readBool() *php.Value {
	d.skipEq("b:")
	bs := d.readBytes(';')
	fmt.Printf("bytes: %s\n", bs)

	var b bool
	if bytes.Equal(bs, []byte{'1'}) {
		b = true
	} else if !bytes.Equal(bs, []byte{'0'}) {
		d.error("cannot convert `%s` to bool", string(bs))
		return nil
	}

	return php.Bool(b)
}

func (d *decodeState) readInt() *php.Value {
	d.skipEq("i:")
	return php.Int(d.readIntBody(';'))
}

func (d *decodeState) readIntBody(delim byte) int {
	bs := d.readBytes(delim)
	i, err := strconv.Atoi(string(bs))
	if err != nil {
		d.error("cannot convert `%s` to int: %v", bs, err)
		return 0
	}
	return i
}

func (d *decodeState) readFloat() *php.Value {
	d.skipEq("d:")
	bs := d.readBytes(';')
	var f float64
	var err error
	if bytes.Equal(bs, []byte("NAN")) {
		f = math.NaN()
	} else if bytes.Equal(bs, []byte("INF")) {
		f = math.Inf(0)
	} else if bytes.Equal(bs, []byte("-INF")) {
		f = math.Inf(-1)
	} else {
		f, err = strconv.ParseFloat(string(bs), 64)
		if err != nil {
			d.error("cannot convert `%v` to float: %v", bs, err)
			return nil
		}
	}
	return php.Float(f)
}

func (d *decodeState) readString() *php.Value {
	str := d.readStringLiteral()
	d.skipEq(";")
	return php.String(str)
}

func (d *decodeState) readStringLiteral() string {
	d.skipEq("s:")
	l := d.readIntBody(':')
	str := d.readStrBody(l)
	return str
}

func (d *decodeState) readStrBody(length int) string {
	d.skipEq(`"`)
	end := d.off + length
	if len(d.data) < end {
		d.error("unexpected EOF in string body, from: %d, length: %d", d.off, length)
		return ""
	}
	str := d.data[d.off:end]
	d.off = end
	d.skipEq(`"`)
	return string(str)
}

func (d *decodeState) readArray() *php.Value {
	d.skipEq("a:")
	l := d.readIntBody(':')
	d.skipEq("{")
	ls := make([]*php.ArrayElement, l)
	for i := 0; i < l; i++ {
		k := d.readKey()
		v := d.readValue()
		ls[i] = php.Element(k, v)
	}
	d.skipEq("}")
	return php.Array(ls...)
}

func (d *decodeState) readKey() *php.Value {
	v := d.readValue()
	switch v.Type() {
	case php.TypeInt, php.TypeString:
		return v
	default:
		d.error("invalid array key type: %s", v.Type)
		return nil
	}
}

func (d *decodeState) readObject() *php.Value {
	d.skipEq("O:")
	name := d.readStrBody(d.readIntBody(':'))
	d.skipEq(":")

	l := d.readIntBody(':')

	fields := make([]*php.ObjField, l)
	for i := 0; i < l; i++ {
		name := d.readStringLiteral()
		vis := php.VisibilityPublic
		if name[0] == '*' {
			name = name[1:]
			vis = php.VisibilityProtected
		} else if name[0] == '\x00' {
			i := bytes.IndexByte([]byte(name[1:]), '\x00')
			if i == -1 {
				d.error("invalid field name: %s", name)
				return nil
			}
			name = name[i+2:]
			vis = php.VisibilityPrivate
		}
		fields[i] = php.Field(name, d.readValue(), vis)
	}

	return php.Object(name, fields...)
}
