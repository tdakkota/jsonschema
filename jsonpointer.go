package jsonschema

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

func splitFunc(s string, sep byte, cb func(s string) error) error {
	for {
		idx := strings.IndexByte(s, sep)
		if idx < 0 {
			break
		}
		if err := cb(s[:idx]); err != nil {
			return err
		}
		s = s[idx+1:]
	}
	return cb(s)
}

func find(u *url.URL, buf []byte, validate bool) (*url.URL, []byte, error) {
	d := jx.GetDecoder()
	defer jx.PutDecoder(d)

	ptr := u.Fragment
	if ptr == "" {
		if validate {
			d.ResetBytes(buf)
			return u, buf, d.Validate()
		}
		return u, buf, nil
	}

	if ptr[0] != '/' {
		return nil, nil, errors.Errorf("invalid pointer %q: pointer must start with '/'", ptr)
	}
	// Cut first /.
	ptr = ptr[1:]

	err := splitFunc(ptr, '/', func(part string) (err error) {
		part = unescape(part)
		var (
			result []byte
			ok     bool
		)
		d.ResetBytes(buf)
		switch tt := d.Next(); tt {
		case jx.Object:
			r, err := findKey(u, d, part)
			if err != nil {
				return errors.Wrapf(err, "find key %q", part)
			}
			u, result, ok = r.u, r.result, r.ok
		case jx.Array:
			result, ok, err = findIdx(d, part)
			if err != nil {
				return errors.Wrapf(err, "find index %q", part)
			}
		default:
			return errors.Errorf("unexpected type %q", tt)
		}
		if !ok {
			return errors.Errorf("pointer %q not found", ptr)
		}

		buf = result
		return err
	})
	return u, buf, err
}

func findIdx(d *jx.Decoder, part string) (result []byte, ok bool, _ error) {
	index, err := strconv.ParseUint(part, 10, 64)
	if err != nil {
		return nil, false, errors.Wrap(err, "index")
	}

	counter := uint64(0)

	iter, err := d.ArrIter()
	if err != nil {
		return nil, false, err
	}
	for iter.Next() {
		if index == counter {
			raw, err := d.Raw()
			if err != nil {
				return nil, false, errors.Wrapf(err, "parse %d", counter)
			}
			result = raw
			ok = true
			break
		}
		if err := d.Skip(); err != nil {
			return nil, false, err
		}
		counter++
	}
	return result, ok, iter.Err()
}

type findKeyResult struct {
	u      *url.URL
	result []byte
	ok     bool
}

func findKey(base *url.URL, d *jx.Decoder, part string) (r findKeyResult, _ error) {
	iter, err := d.ObjIter()
	if err != nil {
		return r, err
	}

	for iter.Next() {
		if r.ok && r.u != nil {
			// We found "id" and needed key, return.
			break
		}
		switch key := iter.Key(); string(key) {
		case part:
			raw, err := d.Raw()
			if err != nil {
				return r, errors.Wrapf(err, "parse %q", key)
			}
			r.result = raw
			r.ok = true
		case "id":
			if d.Next() != jx.String {
				if err := d.Skip(); err != nil {
					return r, err
				}
				continue
			}
			// TODO(tdakkota): get id field name from draft struct
			id, err := d.Str()
			if err != nil {
				return r, errors.Wrapf(err, "parse %q", key)
			}

			parser := url.Parse
			if base != nil {
				parser = base.Parse
			}

			u, err := parser(id)
			if err != nil {
				return r, errors.Wrapf(err, "parse id")
			}
			r.u = u
		default:
			if err := d.Skip(); err != nil {
				return r, err
			}
		}
	}
	if r.u == nil {
		r.u = base
	}
	return r, iter.Err()
}

var (
	unescapeReplacer = strings.NewReplacer(
		"~1", "/",
		"~0", "~",
	)
)

func unescape(part string) string {
	// Replacer always creates new string, check that unescape is really necessary.
	if !strings.Contains(part, "~1") && !strings.Contains(part, "~0") {
		return part
	}
	return unescapeReplacer.Replace(part)
}
