// Package jxvalue provides an implementation of Value interface using github.com/go-faster/jx package.
package jxvalue

import (
	"encoding/json"
	"math/big"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"

	"github.com/tdakkota/jsonschema/internal/jsonequal"
	"github.com/tdakkota/jsonschema/valueiter"
)

var _ valueiter.Value[Value] = Value{}

// Value is valueiter.Value implementation for jx.
type Value struct {
	Raw jx.Raw
}

// Type implements valueiter.Value.
func (v Value) Type() jx.Type {
	return v.Raw.Type()
}

// Bool implements valueiter.Value.
func (v Value) Bool() bool {
	dec := jx.GetDecoder()
	dec.ResetBytes(v.Raw)
	defer jx.PutDecoder(dec)

	return errors.Must(dec.Bool())
}

// Number implements valueiter.Value.
func (v Value) Number() valueiter.Number {
	dec := jx.GetDecoder()
	dec.ResetBytes(v.Raw)
	defer jx.PutDecoder(dec)

	n := errors.Must(dec.Num())
	var rat big.Rat
	if err := rat.UnmarshalText(n); err != nil {
		panic(err)
	}
	return valueiter.Number{
		Rat: &rat,
	}
}

// Str implements valueiter.Value.
func (v Value) Str() []byte {
	// Do not use pool here, because StrBytes() may return a slice that references the buffer.
	return errors.Must(jx.DecodeBytes(v.Raw).StrBytes())
}

// Array implements valueiter.Value.
func (v Value) Array(cb func(Value) error) error {
	dec := jx.GetDecoder()
	dec.ResetBytes(v.Raw)
	defer jx.PutDecoder(dec)

	iter := errors.Must(dec.ArrIter())
	for iter.Next() {
		raw := errors.Must(dec.Raw())
		if err := cb(Value{Raw: raw}); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Object implements valueiter.Value.
func (v Value) Object(cb func(key []byte, value Value) error) error {
	dec := jx.GetDecoder()
	dec.ResetBytes(v.Raw)
	defer jx.PutDecoder(dec)

	iter := errors.Must(dec.ObjIter())
	for iter.Next() {
		key := iter.Key()
		raw := errors.Must(dec.Raw())
		if err := cb(key, Value{Raw: raw}); err != nil {
			return err
		}
	}
	return iter.Err()
}

var _ valueiter.ValueComparator[Value] = Comparator{}

// Comparator is Value comparator.
type Comparator struct{}

func (c Comparator) Contains(enum []json.RawMessage, val Value) (bool, error) {
	// FIXME(tdakkota): this is dramatically slow.
	for _, e := range enum {
		ok, err := jsonequal.Equal(val.Raw, e)
		if err != nil {
			return true, errors.Wrapf(err, "compare %q and %v", e, val)
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
}

// Equal implements ValueComparator interface.
func (c Comparator) Equal(a, b Value) (bool, error) {
	return jsonequal.Equal(a.Raw, b.Raw)
}
