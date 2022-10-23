// Package valueiter provides an interface of JSON Schema value to validate against.
package valueiter

import (
	"math/big"

	"github.com/go-faster/jx"
)

// Number is parsed JSON number.
type Number struct {
	*big.Rat
}

// Value is JSON Schema value to validate against.
type Value[V any] interface {
	// Type returns JSON type.
	Type() jx.Type
	// Bool parses value as bool.
	Bool() bool
	// Number parses value as number.
	Number() Number
	// Str parses value as string.
	//
	// Returned value may reference the buffer, do not modify or retain it.
	Str() []byte
	// Array parses value as array and calls cb for each element.
	Array(cb func(value V) error) error
	// Object parses value as object and calls cb for each key-value pair.
	//
	// Key may reference the buffer, do not modify or retain it.
	Object(cb func(key []byte, value V) error) error
}

// ValueComparator compares two values.
type ValueComparator[V Value[V]] interface {
	Equal(a, b V) (bool, error)
}
