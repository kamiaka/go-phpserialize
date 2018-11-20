package phpserialize_test

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"

	phpserialize "github.com/kamiaka/go-phpserialize"
	"github.com/kamiaka/go-phpserialize/php"
)

func isNaN(v *php.Value) bool {
	return v != nil && v.Type() == php.TypeFloat && math.IsNaN(v.Float())
}

func TestUnmarshal(t *testing.T) {
	cases := []struct {
		bs         []byte
		want       *php.Value
		wantsError bool
	}{
		{
			bs:         []byte(``),
			wantsError: true,
		},
		{
			bs:   []byte(`N;`),
			want: php.Null(),
		},
		{
			bs:         []byte(`N;x`),
			wantsError: true,
		},
		{
			bs:   []byte(`b:1;`),
			want: php.Bool(true),
		},
		{
			bs:   []byte(`b:0;`),
			want: php.Bool(false),
		},
		{
			bs:         []byte(`b:2;`),
			wantsError: true,
		},
		{
			bs:   []byte(`d:3.14;`),
			want: php.Float(3.14),
		},
		{
			bs:   []byte(`d:NAN;`),
			want: php.NaN(),
		},
		{
			bs:   []byte(`d:INF;`),
			want: php.Inf(0),
		},
		{
			bs:   []byte(`d:-INF;`),
			want: php.Inf(-1),
		},
		{
			bs:   []byte(`s:5:"ss"ss";`),
			want: php.String(`ss"ss`),
		},
		{
			bs:   []byte("s:4:\"\n\n\n\n\";"),
			want: php.String("\n\n\n\n"),
		},
		{
			bs:   []byte("a:3:{i:0;i:1;i:1;i:2;i:2;i:3;}"),
			want: php.Append(php.Array(), php.Int(1), php.Int(2), php.Int(3)),
		},
	}
	for i, tc := range cases {
		got, err := phpserialize.Unmarshal(tc.bs)
		if err != nil {
			if !tc.wantsError {
				t.Fatalf("#%d: Unmarshal(...) returns error: %v", i, err)
			}
			continue
		}
		if tc.wantsError {
			t.Errorf("#%d: Unmarshal(...) wants error but no error occurred, return %#v", i, got)
		} else if !reflect.DeepEqual(tc.want, got) && !(isNaN(tc.want) && isNaN(got)) {
			t.Errorf("#%d: Unmarshal(...) == %#v, wants: %#v", i, got, tc.want)
			g, _ := json.Marshal(got)
			w, _ := json.Marshal(tc.want)
			fmt.Printf("got:  %s\nwant: %s\n", g, w)
		}
	}
}

func ExampleUnmarshal() {
	s := `a:2:{i:0;s:1:"a";i:1;s:3:"bbb";}`
	arr, _ := phpserialize.Unmarshal([]byte(s))

	for _, k := range arr.Keys() {
		fmt.Printf("%#v: %s\n", k.Interface(), arr.Index(k).String())
	}

	// Output:
	// 0: a
	// 1: bbb
}

func ExampleUnmarshal_map() {
	s := `a:2:{s:4:"key1";s:1:"a";s:4:"key2";s:3:"bbb";}`
	arr, _ := phpserialize.Unmarshal([]byte(s))

	fmt.Printf("%s\n", arr.IndexByName("key2").String())
	fmt.Printf("%s\n", arr.IndexByName("key1").String())

	// Output:
	// bbb
	// a
}

func ExampleUnmarshal_panic() {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*php.ValueError); ok {
				fmt.Println(e.Error())
			} else {
				panic(r)
			}
		}
	}()
	s := `a:4:{s:4:"key1";s:4:"aaaa";s:4:"key2";N;s:4:"key3";i:42;s:4:"key4";N;}`
	arr, _ := phpserialize.Unmarshal([]byte(s))

	fmt.Printf("%v\n", arr.IndexByName("key1").String())
	fmt.Printf("%v\n", arr.IndexByName("key2").String()) // no panic
	fmt.Printf("%v\n", arr.IndexByName("key3").Int())
	fmt.Printf("%v\n", arr.IndexByName("key4").Int()) // panic

	// Output:
	// aaaa
	// <null value>
	// 42
	// php: call of php.Value.Int on null Value
}
