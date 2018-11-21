go-phpserialize
===============
[![GoDoc](https://godoc.org/github.com/kamiaka/go-phpserialize?status.svg)](https://godoc.org/github.com/kamiaka/go-phpserialize)

Go PHP serialize library.

## Examples

### Serialize Example

```go
func ExampleMarshal() {
  bs, _ := phpserialize.Marshal([]string{"a", "bbb"})
  fmt.Println(string(bs))

  // Output:
  // a:2:{i:0;s:1:"a";i:1;s:3:"bbb";}
}
```

### Unserialize Example

#### Array

```go
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
```

#### Assoc

```go
func ExampleUnmarshal_map() {
  s := `a:2:{s:4:"key1";s:1:"a";s:4:"key2";s:3:"bbb";}`
  arr, _ := phpserialize.Unmarshal([]byte(s))

  fmt.Printf("%s\n", arr.IndexByName("key2").String())
  fmt.Printf("%s\n", arr.IndexByName("key1").String())

  // Output:
  // bbb
  // a
}
```

#### Assoc (panic)

```go
func ExampleUnmarshal_panic() (err error) {
  defer func() {
    if r := recover(); r != nil {
      if e, ok := r.(*php.ValueError); ok {
        fmt.Println(e.Error())
        err = e
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
  return
}
```