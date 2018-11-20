package php

import "strconv"

// Type represents the PHP type
type Type uint

// types
const (
	TypeInvalid Type = iota
	TypeNull
	TypeBool
	TypeInt
	TypeFloat
	TypeString
	TypeArray
	TypeObject
)

var typeNames = []string{
	TypeInvalid: "invalid",
	TypeNull:    "null",
	TypeBool:    "bool",
	TypeInt:     "int",
	TypeFloat:   "float",
	TypeString:  "string",
	TypeArray:   "array",
	TypeObject:  "object",
}

func (t Type) String() string {
	if int(t) < len(typeNames) {
		return typeNames[t]
	}
	return "type" + strconv.Itoa(int(t))
}
