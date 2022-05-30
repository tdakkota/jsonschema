// Package parser contains JSON Schema parsing utilities.
package jsonschema

import (
	"math/big"
	"regexp"

	"github.com/go-faster/errors"
)

// parser parses JSON schemas.
type parser struct {
	resolver rootResolver
	refcache map[string]*Schema
}

// newParser creates new parser.
func newParser(root []byte) *parser {
	return &parser{
		resolver: rootResolver(root),
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
	if ref := schema.Ref; ref != "" {
		s, err := p.resolve(ref, ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve %q", ref)
		}
		return s, nil
	}

	if f := schema.Format; f != "" {
		// TODO: support format validation
		return nil, errors.Errorf("unsupported format %q", f)
	}

	s := &Schema{
		types:                typeSet(0).set(schema.Type),
		format:               schema.Format,
		enum:                 schema.Enum,
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
		minItems:             parseMinMax(schema.MinItems),
		maxItems:             parseMinMax(schema.MaxItems),
		uniqueItems:          schema.UniqueItems,
		items:                nil,
		prefixItems:          nil,
		minimum:              nil,
		exclusiveMinimum:     schema.ExclusiveMinimum,
		maximum:              nil,
		exclusiveMaximum:     schema.ExclusiveMaximum,
		multipleOf:           nil,
		minLength:            parseMinMax(schema.MinLength),
		maxLength:            parseMinMax(schema.MaxLength),
		pattern:              nil,
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
		{"prefixItems", &s.prefixItems, schema.PrefixItems},
	} {
		*many.to, err = p.parseMany(many.schemas, ctx)
		if err != nil {
			return nil, errors.Wrap(err, many.name)
		}
	}

	for _, single := range []struct {
		name   string
		to     **Schema
		schema *RawSchema
	}{
		{"not", &s.not, schema.Not},
		{"items", &s.items, schema.Items},
	} {
		if single.schema == nil {
			continue
		}
		*single.to, err = p.parse(*single.schema, ctx)
		if err != nil {
			return nil, errors.Wrap(err, single.name)
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

	raw, err := p.resolver.ResolveReference(ref)
	if err != nil {
		return nil, errors.Wrap(err, "find schema")
	}

	return p.parse(raw, ctx)
}
