package phpserialize_test

import (
	"bytes"
	"fmt"
	"testing"

	phpserialize "github.com/kamiaka/go-phpserialize"
	"github.com/kamiaka/go-phpserialize/php"
)

func intPtr(i int) *int {
	return &i
}

type testVal struct {
	First  string
	Second int
	Third  bool
	fourth int
}

func TestMarshals(t *testing.T) {
	var nilPtr *int

	cases := []struct {
		val  interface{}
		want []byte
	}{
		{
			val:  nil,
			want: []byte("N;"),
		},
		{
			val:  true,
			want: []byte("b:1;"),
		},
		{
			val:  false,
			want: []byte("b:0;"),
		},
		{
			val:  42,
			want: []byte("i:42;"),
		},
		{
			val:  1234.5,
			want: []byte("d:1234.5;"),
		},
		{
			val:  12345678901234567890.0,
			want: []byte("d:1.2345678901234567e+19;"),
		},
		{
			val:  "日本語",
			want: []byte(`s:9:"日本語";`),
		},
		{
			val:  uint8(42),
			want: []byte("i:42;"),
		},
		{
			val:  intPtr(42),
			want: []byte("i:42;"),
		},
		{
			val:  nilPtr,
			want: []byte("N;"),
		},
		{
			val:  []int{1, 3, 5},
			want: []byte("a:3:{i:0;i:1;i:1;i:3;i:2;i:5;}"),
		},
		{
			val:  []int(nil),
			want: []byte(`a:0:{}`),
		},
		{
			val: map[interface{}]int{
				"a":  0,
				0:    1,
				"bb": 2,
				3:    3,
				11:   4,
			},
			want: []byte(`a:5:{i:0;i:1;i:3;i:3;i:11;i:4;s:1:"a";i:0;s:2:"bb";i:2;}`),
		},
		{
			val: testVal{
				First:  "f\nval",
				Second: 42,
				Third:  true,
				fourth: 3,
			},
			want: []byte(`O:7:"testVal":4:{s:5:"First";s:5:"f` + "\n" + `val";s:6:"Second";i:42;s:5:"Third";b:1;s:15:"` + "\x00testVal\x00fourth" + `";i:3;}`),
		},
		{
			val: php.Array([]*php.ArrayElement{
				{Index: php.Int(0), Value: php.Int(1)},
				{Index: php.String("a"), Value: php.String("aa")},
			}...),
			want: []byte(`a:2:{i:0;i:1;s:1:"a";s:2:"aa";}`),
		},
		{
			val: php.Object(
				"Foo",
				[]*php.ObjField{
					php.Field("a", php.Int(42), php.VisibilityPublic),
					php.Field("b", php.String("aaa"), php.VisibilityProtected),
					php.Field("c", php.Bool(true), php.VisibilityPrivate),
				}...,
			),
			want: []byte(`O:3:"Foo":3:{s:1:"a";i:42;s:2:"*b";s:3:"aaa";s:6:"` + "\x00Foo\x00c" + `";b:1;}`),
		},
	}

	for i, tc := range cases {
		got, err := phpserialize.Marshal(tc.val)
		if err != nil {
			t.Fatalf("#%d: Marshal(...) returns error: %v", i, err)
		}
		if !bytes.Equal(got, tc.want) {
			t.Errorf("#%d: Marshal(...) == %s\nwant: %s", i, got, tc.want)
		}
	}
}

func ExampleMarshal() {
	bs, _ := phpserialize.Marshal([]string{"a", "bbb"})
	fmt.Println(string(bs))

	// Output:
	// a:2:{i:0;s:1:"a";i:1;s:3:"bbb";}
}
