package jsonschema

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/go-faster/errors"
)

const maxResolveDepth = 1000

func stripFragment(u *url.URL) (loc url.URL) {
	// Make copy.
	loc = *u
	loc.Fragment = ""
	return loc
}

type resolveCtx struct {
	depth  int
	parent *url.URL
}

func newResolveCtx(parent *url.URL) *resolveCtx {
	return &resolveCtx{
		parent: parent,
	}
}

func (r *resolveCtx) child(newParent *url.URL) *resolveCtx {
	return &resolveCtx{
		parent: newParent,
	}
}

func (r *resolveCtx) add() error {
	if r.depth+1 >= maxResolveDepth {
		return errors.New("resolve depth exceeded")
	}
	r.depth++
	return nil
}

func (r *resolveCtx) delete() {
	r.depth--
}

func (r *resolveCtx) parseURL(ref string) (*url.URL, error) {
	if r.parent != nil {
		return r.parent.Parse(ref)
	}
	return url.Parse(ref)
}

func (p *compiler[V]) resolve(ref string, ctx *resolveCtx) (*Schema[V], error) {
	if s, ok := p.refcache[ref]; ok {
		return s, nil
	}

	u, err := ctx.parseURL(ref)
	if err != nil {
		return nil, errors.Wrap(err, "parse ref")
	}
	locURL := stripFragment(u)

	if err := ctx.add(); err != nil {
		return nil, err
	}
	defer func() {
		// Drop the resolved ref to prevent false-positive infinite recursion detection.
		ctx.delete()
	}()

	newURL, root, err := p.resolveURL(u, locURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "resolve URL")
	}
	if newURL != nil {
		locURL = stripFragment(newURL)
	}

	var raw RawSchema
	if err := json.Unmarshal(root, &raw); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return p.compile1(raw, ctx.child(&locURL), func(s *Schema[V]) {
		p.refcache[ref] = s
	})
}

func (p *compiler[V]) resolveURL(u *url.URL, loc string) (*url.URL, []byte, error) {
	if val, ok := p.doc.resolveID(u); ok {
		return u, val, nil
	}
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
