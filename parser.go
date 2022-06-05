package jsonschema

import (
	"encoding/json"
	"math/big"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-faster/errors"

	"github.com/tdakkota/jsonschema/internal/jsonpointer"
)

// parser parses JSON schemas.
type parser struct {
	doc      *document
	refcache map[string]*Schema
}

// newParser creates new parser.
func newParser(root *document) *parser {
	return &parser{
		doc:      root,
		refcache: map[string]*Schema{},
	}
}

// Parse parses given RawSchema and returns parsed Schema.
//
// Do not modify RawSchema fields, Schema will reference them.
func (p *parser) Parse(schema RawSchema) (*Schema, error) {
	return p.parse(schema, resolveCtx{})
}

func (p *parser) parse(schema RawSchema, ctx resolveCtx) (_ *Schema, err error) {
	return p.parse1(schema, ctx, func(s *Schema) {})
}

func (p *parser) parse1(schema RawSchema, ctx resolveCtx, save func(s *Schema)) (_ *Schema, err error) {
	if ref := schema.Ref; ref != "" {
		s, err := p.resolve(ref, ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve %q", ref)
		}
		return s, nil
	}

	if f := schema.Format; f != "" {
		// TODO: support format validation
		schema.Format = ""
	}

	s := &Schema{
		types:                typeSet(0).set(schema.Type),
		format:               schema.Format,
		enum:                 schema.Enum,
		enumMap:              make(map[string]struct{}, len(schema.Enum)),
		allOf:                nil,
		anyOf:                nil,
		oneOf:                nil,
		not:                  nil,
		minProperties:        parseMinMax(schema.MinProperties),
		maxProperties:        parseMinMax(schema.MaxProperties),
		required:             map[string]struct{}{},
		properties:           map[string]*Schema{},
		patternProperties:    nil,
		additionalProperties: additionalProperties{},
		dependentRequired:    nil,
		dependentSchemas:     nil,
		minItems:             parseMinMax(schema.MinItems),
		maxItems:             parseMinMax(schema.MaxItems),
		uniqueItems:          schema.UniqueItems,
		items:                items{},
		additionalItems:      additionalItems{},
		minimum:              nil,
		exclusiveMinimum:     schema.ExclusiveMinimum,
		maximum:              nil,
		exclusiveMaximum:     schema.ExclusiveMaximum,
		multipleOf:           nil,
		minLength:            parseMinMax(schema.MinLength),
		maxLength:            parseMinMax(schema.MaxLength),
		pattern:              nil,
	}
	save(s)

	for _, value := range schema.Enum {
		s.enumMap[string(value)] = struct{}{}
	}

	for _, field := range schema.Required {
		// See https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.4.3.
		//
		// Elements of this array MUST be strings, and MUST be unique.
		if _, ok := s.required[field]; ok {
			return nil, errors.Errorf(`"required" list must be unique, duplicate %q`, field)
		}
		s.required[field] = struct{}{}
	}

	for _, field := range schema.Properties {
		s.properties[field.Name], err = p.parse(field.Schema, ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "property %q", field.Name)
		}
	}

	for _, field := range schema.PatternProperties {
		if err := func() error {
			pattern, err := regexp.Compile(field.Pattern)
			if err != nil {
				return err
			}

			item, err := p.parse(field.Schema, ctx)
			if err != nil {
				return err
			}

			s.patternProperties = append(s.patternProperties, patternProperty{
				Regexp: pattern,
				Schema: item,
			})
			return nil
		}(); err != nil {
			return nil, errors.Wrapf(err, "patternProperty %q", field.Pattern)
		}
	}

	if it := schema.Items; it != nil {
		s.items.Set = true
		if it.Array {
			s.items.Array, err = p.parseMany(it.Schemas, ctx)
		} else {
			s.items.Object, err = p.parse(it.Schema, ctx)
		}
		if err != nil {
			return nil, errors.Wrap(err, "items")
		}
	}

	if ap := schema.AdditionalProperties; ap != nil {
		s.additionalProperties.Set = true
		if val := ap.Bool; val != nil {
			s.additionalProperties.Bool = *val
		} else {
			s.additionalProperties.Schema, err = p.parse(ap.Schema, ctx)
			if err != nil {
				return nil, errors.Wrap(err, "additionalProperties")
			}
		}
	}

	{
		dep := schema.Dependencies
		if len(dep.Schemas) > 0 {
			s.dependentSchemas = make(map[string]*Schema, len(dep.Schemas))
			for field, schema := range dep.Schemas {
				s.dependentSchemas[field], err = p.parse(schema, ctx)
				if err != nil {
					return nil, errors.Wrapf(err, "dependent schema %q", field)
				}
			}
		}
		s.dependentRequired = dep.Required
	}

	if ai := schema.AdditionalItems; ai != nil {
		s.additionalItems.Set = true
		if val := ai.Bool; val != nil {
			s.additionalItems.Bool = *val
		} else {
			s.additionalItems.Schema, err = p.parse(ai.Schema, ctx)
			if err != nil {
				return nil, errors.Wrap(err, "additionalItems")
			}
		}
	}

	if pattern := schema.Pattern; len(pattern) > 0 {
		s.pattern, err = regexp.Compile(pattern)
		if err != nil {
			return nil, errors.Wrap(err, "pattern")
		}
	}

	// TODO: how does it affect performance?
	for _, many := range []struct {
		name    string
		to      *[]*Schema
		schemas []RawSchema
	}{
		{"allOf", &s.allOf, schema.AllOf},
		{"anyOf", &s.anyOf, schema.AnyOf},
		{"oneOf", &s.oneOf, schema.OneOf},
	} {
		*many.to, err = p.parseMany(many.schemas, ctx)
		if err != nil {
			return nil, errors.Wrap(err, many.name)
		}
	}

	if sch := schema.Not; sch != nil {
		s.not, err = p.parse(*sch, ctx)
		if err != nil {
			return nil, errors.Wrap(err, "not")
		}
	}

	for _, v := range []struct {
		name string
		to   **big.Rat
		num  Num
	}{
		{"minimum", &s.minimum, schema.Minimum},
		{"maximum", &s.maximum, schema.Maximum},
		{"multipleOf", &s.multipleOf, schema.MultipleOf},
	} {
		if len(v.num) == 0 {
			// Value is not set.
			continue
		}
		val := new(big.Rat)
		// TODO: more efficient way?
		if err := val.UnmarshalText(v.num); err != nil {
			return nil, errors.Wrap(err, v.name)
		}
		*v.to = val
	}

	return s, nil
}

func (p *parser) parseMany(schemas []RawSchema, ctx resolveCtx) ([]*Schema, error) {
	result := make([]*Schema, 0, len(schemas))
	for i, schema := range schemas {
		s, err := p.parse(schema, ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "[%d]", i)
		}

		result = append(result, s)
	}

	return result, nil
}

type resolveCtx map[string]struct{}

func (p *parser) resolve(ref string, ctx resolveCtx) (*Schema, error) {
	if s, ok := p.refcache[ref]; ok {
		return s, nil
	}

	if _, ok := ctx[ref]; ok {
		// TODO: better error?
		return nil, errors.New("infinite recursion")
	}
	ctx[ref] = struct{}{}
	defer func() {
		// Drop the resolved ref to prevent false-positive infinite recursion detection.
		delete(ctx, ref)
	}()

	root, err := p.resolveURL(ref)
	if err != nil {
		return nil, errors.Wrap(err, "resolve URL")
	}

	var raw RawSchema
	if err := json.Unmarshal(root, &raw); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return p.parse1(raw, ctx, func(s *Schema) {
		p.refcache[ref] = s
	})
}

func (p *parser) resolveURL(ref string) ([]byte, error) {
	if ref == "" {
		return nil, errors.New("empty ref")
	}

	u, err := url.Parse(ref)
	if err != nil {
		return nil, err
	}

	parsedRef := ref
	if id := p.doc.id; id != nil {
		parsedRef = id.ResolveReference(u).String()
	}
	if root, ok := p.doc.ids[parsedRef]; ok {
		return root, nil
	}

	frag := ref
	if !strings.HasPrefix(ref, "#/") {
		frag = u.Fragment
		if u.Scheme != "" || u.Host != "" || u.Path != "" {
			return nil, errors.New("invalid or unsupported ref")
		}
	}

	return jsonpointer.Resolve(frag, p.doc.data)
}
