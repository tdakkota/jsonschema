// Package valueiter provides an interface of JSON Schema value to validate against.
package valueiter

import (
	"encoding/json"
	"math/big"

	"github.com/go-faster/jx"
)

// Number is parsed JSON number.
type Number struct {
	*big.Rat
}

// ByteSeq is parsed JSON string.
type ByteSeq interface {
	string | []byte
}

// Value is JSON Schema value to validate against.
type Value[V any, Str, Key ByteSeq] interface {
	// Type returns JSON type.
	Type() jx.Type
	// Bool parses value as bool.
	Bool() bool
	// Number parses value as number.
	Number() Number
	// Str parses value as string.
	//
	// Returned value may reference the buffer, do not modify or retain it.
	Str() Str
	// Array parses value as array and calls cb for each element.
	Array(cb func(value V) error) error
	// Object parses value as object and calls cb for each key-value pair.
	//
	// Key may reference the buffer, do not modify or retain it.
	Object(cb func(key Key, value V) error) error
}

// ValueComparator defines comparator for two values.
// Also, it defines enum comparator implementation interface.
type ValueComparator[V any] interface {
	// Contains returns true if value is in given json values slice.
	Contains(s []json.RawMessage, contains V) (bool, error)
	// Equal returns true if two values are equal.
	Equal(a, b V) (bool, error)
}
