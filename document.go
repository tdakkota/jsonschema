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

func (doc *document) resolveID(u *url.URL) ([]byte, bool) {
	parsedRef := u
	if doc.id != nil {
		parsedRef = doc.id.ResolveReference(u)
	}
	v, ok := doc.ids[parsedRef.String()]
	return v, ok
}

func (doc *document) resolve(u *url.URL) (*url.URL, []byte, error) {
	v, ok := doc.resolveID(u)
	if ok {
		return u, v, nil
	}
	return find(u, doc.data, false)
}

func (doc *document) findID(d *jx.Decoder, base *url.URL) error {
	return d.ObjBytes(func(d *jx.Decoder, key []byte) error {
		if string(key) != "id" {
			// TODO(tdakkota): get id field name from draft struct
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
	})
}

func collectIDs(base *url.URL, data []byte) (*document, error) {
	root := &document{
		id:   nil,
		data: data,
		ids:  map[string][]byte{},
	}

	rootd := jx.DecodeBytes(data)
	if err := root.findID(rootd, base); err != nil {
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

	rootd.ResetBytes(data)
	if err := rootd.ObjBytes(func(d *jx.Decoder, key []byte) error {
		switch string(key) {
		case "definitions", "properties", "patternProperties", "dependencies":
			return doObj(d)
		case "additionalItems", "additionalProperties", "not":
			return do(d)
		case "allOf", "anyOf", "oneOf":
			return doArr(d)
		case "items":
			switch d.Next() {
			case jx.Array:
				return doArr(d)
			case jx.Object:
				return do(d)
			}
		}
		return d.Skip()
	}); err != nil {
		return nil, errors.Wrap(err, "collect IDs")
	}

	return root, nil
}
