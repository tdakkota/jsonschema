package jsonschema

import (
	"encoding/json"
	"math/big"
	"regexp"

	"github.com/go-faster/errors"
)

// compiler parses JSON schemas.
type compiler[V Value[V]] struct {
	doc         *document
	remote      RemoteResolver
	valueMapper func(json.RawMessage) (V, error)

	remotes  map[string]*document
	refcache map[string]*Schema[V]
}

// newCompiler creates new compiler.
func newCompiler[V Value[V]](root *document, mapper func(json.RawMessage) (V, error)) *compiler[V] {
	var loc string
	if root.id != nil {
		r := stripFragment(root.id)
		loc = r.String()
	}
	return &compiler[V]{
		doc:         root,
		remote:      Remote{},
		valueMapper: mapper,
		remotes: map[string]*document{
			"":  root,
			loc: root,
		},
		refcache: map[string]*Schema[V]{},
	}
}

// Compile compiles given RawSchema and returns compiled Schema.
//
// Do not modify RawSchema fields, Schema will reference them.
func (p *compiler[V]) Compile(schema RawSchema) (*Schema[V], error) {
	return p.compile(schema, newResolveCtx(p.doc.id))
}

func (p *compiler[V]) compile(schema RawSchema, ctx *resolveCtx) (_ *Schema[V], err error) {
	return p.compile1(schema, ctx, func(s *Schema[V]) {})
}

func (p *compiler[V]) compile1(schema RawSchema, ctx *resolveCtx, save func(s *Schema[V])) (_ *Schema[V], err error) {
	if ref := schema.Ref; ref != "" {
		s, err := p.resolve(ref, ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve %q", ref)
		}
		return s, nil
	}
	if id := schema.ID; id != "" {
		idURL, err := ctx.parseURL(id)
		if err != nil {
			return nil, errors.Wrap(err, "parse $id")
		}
		ctx = ctx.child(idURL)
	}

	if f := schema.Format; f != "" {
		// TODO: support format validation
		schema.Format = ""
	}
	enum := make([]V, len(schema.Enum))
	for i, val := range schema.Enum {
		mapped, err := p.valueMapper(val)
		if err != nil {
			return nil, errors.Wrapf(err, "parse enum[%d]", i)
		}
		enum[i] = mapped
	}

	s := &Schema[V]{
		types:                typeSet(0).set(schema.Type),
		format:               schema.Format,
		enum:                 enum,
		allOf:                nil,
		anyOf:                nil,
		oneOf:                nil,
		not:                  nil,
		minProperties:        parseMinMax(schema.MinProperties),
		maxProperties:        parseMinMax(schema.MaxProperties),
		required:             make(map[string]struct{}, len(schema.Required)),
		properties:           make(map[string]*Schema[V], len(schema.Properties)),
		patternProperties:    make([]patternProperty[V], 0, len(schema.PatternProperties)),
		additionalProperties: additional[V]{},
		dependentRequired:    nil,
		dependentSchemas:     nil,
		minItems:             parseMinMax(schema.MinItems),
		maxItems:             parseMinMax(schema.MaxItems),
		uniqueItems:          schema.UniqueItems,
		items:                schemaItems[V]{},
		additionalItems:      additional[V]{},
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
		s.properties[field.Name], err = p.compile(field.Schema, ctx)
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

			item, err := p.compile(field.Schema, ctx)
			if err != nil {
				return err
			}

			s.patternProperties = append(s.patternProperties, patternProperty[V]{
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
			s.items.Array, err = p.compileMany(it.Schemas, ctx)
		} else {
			s.items.Object, err = p.compile(it.Schema, ctx)
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
			s.additionalProperties.Schema, err = p.compile(ap.Schema, ctx)
			if err != nil {
				return nil, errors.Wrap(err, "additionalProperties")
			}
		}
	}

	{
		dep := schema.Dependencies
		if len(dep.Schemas) > 0 {
			s.dependentSchemas = make(map[string]*Schema[V], len(dep.Schemas))
			for field, schema := range dep.Schemas {
				s.dependentSchemas[field], err = p.compile(schema, ctx)
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
			s.additionalItems.Schema, err = p.compile(ai.Schema, ctx)
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
		to      *[]*Schema[V]
		schemas []RawSchema
	}{
		{"allOf", &s.allOf, schema.AllOf},
		{"anyOf", &s.anyOf, schema.AnyOf},
		{"oneOf", &s.oneOf, schema.OneOf},
	} {
		*many.to, err = p.compileMany(many.schemas, ctx)
		if err != nil {
			return nil, errors.Wrap(err, many.name)
		}
	}

	if sch := schema.Not; sch != nil {
		s.not, err = p.compile(*sch, ctx)
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

func (p *compiler[V]) compileMany(schemas []RawSchema, ctx *resolveCtx) ([]*Schema[V], error) {
	result := make([]*Schema[V], 0, len(schemas))
	for i, schema := range schemas {
		s, err := p.compile(schema, ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "[%d]", i)
		}

		result = append(result, s)
	}

	return result, nil
}
