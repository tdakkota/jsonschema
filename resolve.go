package jsonschema

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/go-faster/errors"
)

type refKey struct {
	loc string
	ref string
}

func (r *refKey) fromURL(u *url.URL) (loc url.URL) {
	{
		// Make copy.
		loc = *u
		loc.Fragment = ""
		r.loc = loc.String()
	}
	r.ref = "#" + u.Fragment
	return loc
}

type resolveCtx struct {
	parent *url.URL
	// Store references to detect infinite recursive references.
	refs map[refKey]struct{}
}

func newResolveCtx(parent *url.URL) *resolveCtx {
	return &resolveCtx{
		parent: parent,
		refs:   map[refKey]struct{}{},
	}
}

func (r *resolveCtx) add(key refKey) error {
	if _, ok := r.refs[key]; ok {
		// TODO: better error?
		return errors.New("infinite recursion")
	}
	r.refs[key] = struct{}{}
	return nil
}

func (r *resolveCtx) delete(key refKey) {
	delete(r.refs, key)
}

func (r *resolveCtx) parseURL(ref string) (*url.URL, error) {
	if r.parent != nil {
		return r.parent.Parse(ref)
	}
	return url.Parse(ref)
}

func (p *compiler) resolve(ref string, ctx *resolveCtx) (*Schema, error) {
	if s, ok := p.refcache[ref]; ok {
		return s, nil
	}

	u, err := ctx.parseURL(ref)
	if err != nil {
		return nil, errors.Wrap(err, "parse ref")
	}
	var key refKey
	locURL := key.fromURL(u)

	if err := ctx.add(key); err != nil {
		return nil, err
	}
	defer func() {
		// Drop the resolved ref to prevent false-positive infinite recursion detection.
		ctx.delete(key)
	}()

	newURL, root, err := p.resolveURL(u, key)
	if err != nil {
		return nil, errors.Wrap(err, "resolve URL")
	}
	if newURL != nil {
		cpy := *newURL
		locURL = cpy
		locURL.Fragment = ""
	}

	var raw RawSchema
	if err := json.Unmarshal(root, &raw); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return p.compile1(raw, &resolveCtx{
		parent: &locURL,
		refs:   ctx.refs,
	}, func(s *Schema) {
		p.refcache[ref] = s
	})
}

func (p *compiler) resolveURL(u *url.URL, key refKey) (*url.URL, []byte, error) {
	if val, ok := p.doc.resolveID(u); ok {
		return u, val, nil
	}
	loc := key.loc
	doc, ok := p.remotes[loc]
	if !ok {
		var err error
		data, err := p.remote.Resolve(context.TODO(), loc)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "remote %q", loc)
		}

		doc, err = collectIDs(nil, data)
		if err != nil {
			return nil, nil, err
		}
		p.remotes[loc] = doc
	}
	return doc.resolve(u)
}
