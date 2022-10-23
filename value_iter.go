package jsonschema

import (
	"math/big"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"

	"github.com/tdakkota/jsonschema/internal/jsonequal"
)

// Number is parsed JSON number.
type Number struct {
	*big.Rat
}

// Value represents JSON Schema value to validate against.
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

var _ Value[jsonValue] = jsonValue{}

type jsonValue struct {
	raw jx.Raw
}

func (j jsonValue) Type() jx.Type {
	return j.raw.Type()
}

func (j jsonValue) Bool() bool {
	dec := jx.GetDecoder()
	dec.ResetBytes(j.raw)
	defer jx.PutDecoder(dec)

	return errors.Must(dec.Bool())
}

func (j jsonValue) Number() Number {
	dec := jx.GetDecoder()
	dec.ResetBytes(j.raw)
	defer jx.PutDecoder(dec)

	n := errors.Must(dec.Num())
	var rat big.Rat
	if err := rat.UnmarshalText(n); err != nil {
		panic(err)
	}
	return Number{
		Rat: &rat,
	}
}

func (j jsonValue) Str() []byte {
	// Do not use pool here, because StrBytes() may return a slice that references the buffer.
	return errors.Must(jx.DecodeBytes(j.raw).StrBytes())
}

func (j jsonValue) Array(cb func(jsonValue) error) error {
	dec := jx.GetDecoder()
	dec.ResetBytes(j.raw)
	defer jx.PutDecoder(dec)

	iter := errors.Must(dec.ArrIter())
	for iter.Next() {
		raw := errors.Must(dec.Raw())
		if err := cb(jsonValue{raw: raw}); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (j jsonValue) Object(cb func(key []byte, value jsonValue) error) error {
	dec := jx.GetDecoder()
	dec.ResetBytes(j.raw)
	defer jx.PutDecoder(dec)

	iter := errors.Must(dec.ObjIter())
	for iter.Next() {
		key := iter.Key()
		raw := errors.Must(dec.Raw())
		if err := cb(key, jsonValue{raw: raw}); err != nil {
			return err
		}
	}
	return iter.Err()
}

var _ ValueComparator[jsonValue] = jsonComparator{}

type jsonComparator struct{}

func (c jsonComparator) Equal(a, b jsonValue) (bool, error) {
	return jsonequal.Equal(a.raw, b.raw)
}

func ValidateJSON(s *Schema[jsonValue], data []byte) error {
	raw, err := jx.DecodeBytes(data).Raw()
	if err != nil {
		return err
	}
	return validate[jsonValue, jsonComparator](s, jsonValue{raw: raw}, jsonComparator{})
}
