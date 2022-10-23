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
	dec := j.dec()
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
	dec := j.dec()
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
