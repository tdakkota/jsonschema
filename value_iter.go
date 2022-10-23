package jsonschema

import (
	"math/big"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"

	"github.com/tdakkota/jsonschema/internal/jsonequal"
)

type Number struct {
	*big.Rat
}

// Value represents JSON value.
type Value[V any] interface {
	Type() jx.Type
	Bool() bool
	Number() Number
	Str() string
	Array(cb func(value V) error) error
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

func (j jsonValue) dec() *jx.Decoder {
	return jx.DecodeBytes(j.raw)
}

func (j jsonValue) Bool() bool {
	return errors.Must(j.dec().Bool())
}

func (j jsonValue) Number() Number {
	n := errors.Must(j.dec().Num())
	var rat big.Rat
	if err := rat.UnmarshalText(n); err != nil {
		panic(err)
	}
	return Number{
		Rat: &rat,
	}
}

func (j jsonValue) Str() string {
	return errors.Must(j.dec().Str())
}

func (j jsonValue) Array(cb func(jsonValue) error) error {
	return j.dec().Arr(func(d *jx.Decoder) error {
		raw := errors.Must(d.Raw())
		return cb(jsonValue{raw: raw})
	})
}

func (j jsonValue) Object(cb func(key []byte, value jsonValue) error) error {
	return j.dec().ObjBytes(func(d *jx.Decoder, key []byte) error {
		raw := errors.Must(d.Raw())
		return cb(key, jsonValue{raw: raw})
	})
}

var _ ValueComparator[jsonValue] = jsonComparator{}

type jsonComparator struct{}

func (c jsonComparator) Equal(a, b jsonValue) (bool, error) {
	return jsonequal.Equal(a.raw, b.raw)
}

func ValidateJSON(s *Schema[jsonValue], data []byte) error {
	return validate[jsonValue, jsonComparator](s, jsonValue{raw: data}, jsonComparator{})
}
