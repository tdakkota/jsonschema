package jsonschema

import (
	"encoding/json"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Num represents JSON number.
type Num jx.Num

// MarshalJSON implements json.Marshaler.
func (n Num) MarshalJSON() ([]byte, error) {
	return json.Marshal(json.RawMessage(n))
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *Num) UnmarshalJSON(data []byte) error {
	j, err := jx.DecodeBytes(data).Num()
	if err != nil {
		return errors.Wrapf(err, "invalid number %s", data)
	}
	if j.Str() {
		return errors.Errorf("invalid number %s", data)
	}

	*n = Num(j)
	return nil
}

// SchemaType represents JSON Schema type list.
type SchemaType []string

func (r *SchemaType) UnmarshalJSON(data []byte) error {
	parseSingle := func(d *jx.Decoder) (string, error) {
		val, err := d.StrBytes()
		if err != nil {
			return "", err
		}
		switch string(val) {
		case "array":
			return "array", nil
		case "boolean":
			return "boolean", nil
		case "integer":
			return "integer", nil
		case "null":
			return "null", nil
		case "number":
			return "number", nil
		case "object":
			return "object", nil
		case "string":
			return "string", nil
		default:
			return "", errors.Errorf("unexpected type %q", val)
		}
	}

	d := jx.DecodeBytes(data)
	switch tt := d.Next(); tt {
	case jx.Array:
		return d.Arr(func(d *jx.Decoder) error {
			val, err := parseSingle(d)
			if err != nil {
				return err
			}
			*r = append(*r, val)
			return nil
		})
	case jx.String:
		val, err := parseSingle(d)
		if err != nil {
			return err
		}
		*r = []string{val}
		return nil
	default:
		return errors.Errorf("unexpected type: %q", tt)
	}
}

// RawSchema is unparsed JSON Schema.
type RawSchema struct {
	Ref    string            `json:"$ref,omitempty"`
	Type   SchemaType        `json:"type,omitempty"`
	Format string            `json:"format,omitempty"`
	Enum   []json.RawMessage `json:"enum,omitempty"`

	AllOf []RawSchema `json:"allOf,omitempty"`
	AnyOf []RawSchema `json:"anyOf,omitempty"`
	OneOf []RawSchema `json:"oneOf,omitempty"`
	Not   *RawSchema  `json:"not,omitempty"`

	MinProperties        *uint64               `json:"minProperties,omitempty"`
	MaxProperties        *uint64               `json:"maxProperties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	Properties           RawProperties         `json:"properties,omitempty"`
	PatternProperties    RawPatternProperties  `json:"patternProperties,omitempty"`
	AdditionalProperties *AdditionalProperties `json:"additionalProperties,omitempty"`

	MinItems    *uint64     `json:"minItems,omitempty"`
	MaxItems    *uint64     `json:"maxItems,omitempty"`
	UniqueItems bool        `json:"uniqueItems,omitempty"`
	Items       *RawSchema  `json:"items,omitempty"`
	PrefixItems []RawSchema `json:"prefixItems,omitempty"`

	Minimum          Num  `json:"minimum,omitempty"`
	ExclusiveMinimum bool `json:"exclusiveMinimum,omitempty"`
	Maximum          Num  `json:"maximum,omitempty"`
	ExclusiveMaximum bool `json:"exclusiveMaximum,omitempty"`
	MultipleOf       Num  `json:"multipleOf,omitempty"`

	MaxLength *uint64 `json:"maxLength,omitempty"`
	MinLength *uint64 `json:"minLength,omitempty"`
	Pattern   string  `json:"pattern,omitempty"`
}

// RawProperty is item of RawProperties.
type RawProperty struct {
	Name   string
	Schema RawSchema
}

// RawProperties is unparsed JSON Schema properties validator description.
type RawProperties []RawProperty

// MarshalJSON implements json.Marshaler.
func (p RawProperties) MarshalJSON() ([]byte, error) {
	var e jx.Encoder
	e.ObjStart()
	for _, prop := range p {
		e.FieldStart(prop.Name)
		b, err := json.Marshal(prop.Schema)
		if err != nil {
			return nil, errors.Wrap(err, "marshal")
		}
		e.Raw(b)
	}
	e.ObjEnd()
	return e.Bytes(), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *RawProperties) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return d.Obj(func(d *jx.Decoder, key string) error {
		var s RawSchema
		b, err := d.Raw()
		if err != nil {
			return err
		}

		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}

		*p = append(*p, RawProperty{
			Name:   key,
			Schema: s,
		})
		return nil
	})
}

// AdditionalProperties is JSON Schema additionalProperties validator description.
type AdditionalProperties struct {
	Bool   *bool
	Schema RawSchema
}

// MarshalJSON implements json.Marshaler.
func (p AdditionalProperties) MarshalJSON() ([]byte, error) {
	if p.Bool != nil {
		return json.Marshal(p.Bool)
	}
	return json.Marshal(p.Schema)
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *AdditionalProperties) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	switch tt := d.Next(); tt {
	case jx.Object:
	case jx.Bool:
		val, err := d.Bool()
		if err != nil {
			return err
		}
		p.Bool = &val
		return nil
	default:
		return errors.Errorf("unexpected type %s", tt.String())
	}

	s := RawSchema{}
	b, err := d.Raw()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	p.Schema = s
	return nil
}

// RawPatternProperty is item of RawPatternProperties.
type RawPatternProperty struct {
	Pattern string
	Schema  RawSchema
}

// RawPatternProperties is unparsed JSON Schema patternProperties validator description.
type RawPatternProperties []RawPatternProperty

// MarshalJSON implements json.Marshaler.
func (r RawPatternProperties) MarshalJSON() ([]byte, error) {
	var e jx.Encoder
	e.ObjStart()
	for _, prop := range r {
		e.FieldStart(prop.Pattern)
		b, err := json.Marshal(prop.Schema)
		if err != nil {
			return nil, errors.Wrap(err, "marshal")
		}
		e.Raw(b)
	}
	e.ObjEnd()
	return e.Bytes(), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *RawPatternProperties) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return d.Obj(func(d *jx.Decoder, key string) error {
		var s RawSchema
		b, err := d.Raw()
		if err != nil {
			return err
		}

		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}

		*r = append(*r, RawPatternProperty{
			Pattern: key,
			Schema:  s,
		})
		return nil
	})
}

// Discriminator is JSON Schema discriminator description.
type Discriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty"`
}
