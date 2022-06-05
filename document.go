package jsonschema

import (
	"net/url"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

type document struct {
	id   *url.URL
	data []byte
	ids  map[string][]byte
}

func (doc *document) findID(base *url.URL, d *jx.Decoder, key []byte) error {
	if string(key) != "id" {
		return d.Skip()
	}

	val, err := d.Str()
	if err != nil {
		return err
	}

	id, err := url.Parse(val)
	if err != nil {
		return err
	}

	doc.id = id
	if base != nil {
		doc.id = base.ResolveReference(id)
	}
	return nil
}

func collectIDs(base *url.URL, data []byte) (*document, error) {
	root := &document{
		id:   nil,
		data: data,
		ids:  map[string][]byte{},
	}

	d := jx.DecodeBytes(data)
	if err := d.ObjBytes(func(d *jx.Decoder, key []byte) error {
		return root.findID(base, d, key)
	}); err != nil {
		return nil, errors.Wrap(err, "find ID")
	}
	if root.id != nil {
		root.ids[root.id.String()] = root.data
	}

	do := func(d *jx.Decoder) error {
		if d.Next() != jx.Object {
			return d.Skip()
		}
		raw, err := d.Raw()
		if err != nil {
			return err
		}
		b := root.id
		if b == nil {
			b = base
		}
		sub, err := collectIDs(b, raw)
		if err != nil {
			return err
		}

		if sub.id != nil {
			root.ids[sub.id.String()] = sub.data
		}
		for k, v := range sub.ids {
			root.ids[k] = v
		}
		return nil
	}
	doObj := func(d *jx.Decoder) error {
		if d.Next() != jx.Object {
			return d.Skip()
		}
		return d.ObjBytes(func(d *jx.Decoder, key []byte) error {
			return do(d)
		})
	}
	doArr := func(d *jx.Decoder) error {
		return d.Arr(func(d *jx.Decoder) error {
			return do(d)
		})
	}

	d.ResetBytes(data)
	if err := d.ObjBytes(func(r *jx.Decoder, key []byte) error {
		switch string(key) {
		case "definitions", "properties", "patternProperties", "dependencies":
			return doObj(r)
		case "additionalItems", "additionalProperties", "not":
			return do(r)
		case "allOf", "anyOf", "oneOf":
			return doArr(r)
		case "items":
			switch d.Next() {
			case jx.Array:
				return doArr(r)
			case jx.Object:
				return do(r)
			}
		}
		return d.Skip()
	}); err != nil {
		return nil, errors.Wrap(err, "collect IDs")
	}

	return root, nil
}
