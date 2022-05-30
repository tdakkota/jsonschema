package jsonschema

import (
	"fmt"
	"math/big"
	"unicode/utf8"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"

	"github.com/tdakkota/jsonschema/internal/jsonequal"
)

// Validate validates given data.
func (s *Schema) Validate(data []byte) error {
	d := jx.GetDecoder()
	defer jx.PutDecoder(d)
	// TODO: do not stop early, collect errors instead.
	d.ResetBytes(data)

	tt := d.Next()
	if tt == jx.Invalid {
		return errors.Wrap(d.Validate(), "invalid json")
	}

	if err := s.validateEnum(data); err != nil {
		return errors.Wrap(err, "enum")
	}
	if err := s.validateAllOf(data); err != nil {
		return errors.Wrap(err, "allOf")
	}
	if err := s.validateOneOf(data); err != nil {
		return errors.Wrap(err, "oneOf")
	}
	if err := s.validateAnyOf(data); err != nil {
		return errors.Wrap(err, "anyOf")
	}
	if err := s.validateNot(data); err != nil {
		return errors.Wrap(err, "not")
	}

	var err error
	switch tt {
	case jx.String:
		err = s.validateString(d)
	case jx.Number:
		err = s.validateNumber(d)
	case jx.Null:
		err = s.validateNull(d)
	case jx.Bool:
		err = s.validateBool(d)
	case jx.Array:
		err = s.validateArray(d)
	case jx.Object:
		err = s.validateObject(d)
	default:
		panic(fmt.Sprintf("unreachable: %q", tt))
	}
	if err != nil {
		return errors.Wrap(err, tt.String())
	}
	return nil
}

func (s *Schema) validateEnum(data []byte) error {
	if len(s.enum) == 0 {
		return nil
	}
	for _, variant := range s.enum {
		ok, err := jsonequal.Equal(variant, data)
		if err != nil {
			return errors.Wrap(err, "compare")
		}
		if ok {
			return nil
		}
	}
	return errors.Errorf("%q is not present in enum", data)
}

func (s *Schema) validateAllOf(data []byte) error {
	for i, schema := range s.allOf {
		if err := schema.Validate(data); err != nil {
			return errors.Wrapf(err, "[%d]", i)
		}
	}
	return nil
}

func (s *Schema) validateOneOf(data []byte) error {
	if len(s.oneOf) == 0 {
		return nil
	}
	counter := 0
	for _, schema := range s.oneOf {
		if err := schema.Validate(data); err == nil {
			if counter != 0 {
				return errors.New("must match exactly once")
			}
			counter++
		}
	}
	if counter != 0 {
		return nil
	}
	return errors.New("must match at least once")
}

func (s *Schema) validateAnyOf(data []byte) error {
	if len(s.anyOf) == 0 {
		return nil
	}
	for _, schema := range s.anyOf {
		if err := schema.Validate(data); err == nil {
			return nil
		}
	}
	return errors.New("must match at least once")
}

func (s *Schema) validateNot(data []byte) error {
	if s.not != nil {
		if err := s.not.Validate(data); err == nil {
			return errors.New("must not match")
		}
	}
	return nil
}

func (s *Schema) checkType(t typeSet) error {
	if !s.types.has(t) {
		return errors.New("type is not allowed")
	}
	return nil
}

func (s *Schema) skipType(d *jx.Decoder, t typeSet) error {
	if err := s.checkType(t); err != nil {
		return err
	}
	return d.Skip()
}

func (s *Schema) validateString(d *jx.Decoder) error {
	if err := s.checkType(stringType); err != nil {
		return err
	}

	if !(s.format != "" || s.minLength.IsSet() || s.maxLength.IsSet() || s.pattern != nil) {
		return d.Skip()
	}

	str, err := d.StrBytes()
	if err != nil {
		return errors.Wrap(err, "parse JSON")
	}
	if s.format != "" {
		panic("unreachable")
	}
	if s.minLength.IsSet() || s.maxLength.IsSet() {
		count := utf8.RuneCount(str)
		if s.minLength.IsSet() && count < int(s.minLength) {
			return errors.Errorf("length is smaller than %d", s.minLength)
		}
		if s.maxLength.IsSet() && count > int(s.maxLength) {
			return errors.Errorf("length is bigger than %d", s.maxLength)
		}
	}
	if s.pattern != nil && !s.pattern.Match(str) {
		return errors.Errorf("does not match pattern %s", s.pattern)
	}
	return nil
}

func (s *Schema) validateNumber(d *jx.Decoder) error {
	hasNumber := s.types.has(numberType)

	if hasNumber && !(s.minimum != nil || s.maximum != nil || s.multipleOf != nil) {
		return d.Skip()
	}

	num, err := d.Num()
	if err != nil {
		return errors.Wrap(err, "parse JSON")
	}

	if !hasNumber {
		isInt := num.IsInt()
		if isInt {
			if err := s.checkType(integerType); err != nil {
				return err
			}
		} else {
			return s.checkType(numberType)
		}
	}

	if s.minimum != nil || s.maximum != nil || s.multipleOf != nil {
		val := new(big.Rat)
		// TODO: more efficient way?
		if err := val.UnmarshalText(num); err != nil {
			return errors.Wrap(err, "parse")
		}
		if s.minimum != nil {
			cmp := val.Cmp(s.minimum)
			if (s.exclusiveMinimum && cmp <= 0) || cmp < 0 {
				return errors.Errorf("value %s is smaller than %s", val, s.minimum)
			}
		}
		if s.maximum != nil {
			cmp := val.Cmp(s.maximum)
			if (s.exclusiveMaximum && cmp >= 0) || cmp > 0 {
				return errors.Errorf("value %s is bigger than %s", val, s.maximum)
			}
		}
		if s.multipleOf != nil {
			if !val.Quo(val, s.multipleOf).IsInt() {
				return errors.Errorf("%s is not multiple of %s", val, s.multipleOf)
			}
		}
	}

	return nil
}

func (s *Schema) validateNull(d *jx.Decoder) error {
	return s.skipType(d, nullType)
}

func (s *Schema) validateBool(d *jx.Decoder) error {
	return s.skipType(d, booleanType)
}

func (s *Schema) validateArray(d *jx.Decoder) error {
	if err := s.checkType(arrayType); err != nil {
		return err
	}

	if !(s.minItems.IsSet() ||
		s.maxItems.IsSet() ||
		s.uniqueItems ||
		s.items != nil ||
		len(s.prefixItems) > 0) {
		return d.Skip()
	}

	iter, err := d.ArrIter()
	if err != nil {
		return errors.Wrap(err, "parse JSON")
	}
	var (
		i     = 0
		items []jx.Raw
	)
	for iter.Next() {
		sch := s.items
		if i < len(s.prefixItems) {
			sch = s.prefixItems[i]
		}

		if sch != nil || s.uniqueItems {
			if err := func() error {
				raw, err := d.Raw()
				if err != nil {
					return errors.Wrap(err, "parse JSON")
				}

				if sch != nil {
					if err := sch.Validate(raw); err != nil {
						return err
					}
				}

				if s.uniqueItems {
					items = append(items, raw)
				}

				return nil
			}(); err != nil {
				return errors.Wrapf(err, "[%d]", i)
			}
		} else {
			if err := d.Skip(); err != nil {
				return errors.Wrap(err, "parse JSON")
			}
		}
		i++
	}
	if err := iter.Err(); err != nil {
		return errors.Wrap(err, "parse JSON")
	}

	if len(items) > 1 {
		for xi, x := range items {
			for yi, y := range items {
				if xi == yi {
					continue
				}
				if ok, _ := jsonequal.Equal(x, y); ok {
					return errors.Errorf("items %d and %d are equal", xi, yi)
				}
			}
		}
	}

	if s.minItems.IsSet() && i < int(s.minItems) {
		return errors.Errorf("length is smaller than %d", s.minItems)
	}
	if s.maxItems.IsSet() && i > int(s.maxItems) {
		return errors.Errorf("length is bigger than %d", s.maxItems)
	}

	return nil
}

func (s *Schema) validateObject(d *jx.Decoder) error {
	if err := s.checkType(objectType); err != nil {
		return err
	}

	if !(s.minProperties.IsSet() ||
		s.maxProperties.IsSet() ||
		len(s.required) > 0 ||
		len(s.properties) > 0 ||
		len(s.patternProperties) > 0 ||
		s.additionalProperties.Set) {
		return d.Skip()
	}

	iter, err := d.ObjIter()
	if err != nil {
		return errors.Wrap(err, "parse JSON")
	}
	var (
		i        = 0
		required map[string]struct{}
	)
	if len(s.required) > 0 {
		required = make(map[string]struct{}, len(s.required))
		for k := range s.required {
			required[k] = struct{}{}
		}
	}
	for iter.Next() {
		k := iter.Key()
		delete(required, string(k))

		if prop, ok := s.properties[string(k)]; ok ||
			s.additionalProperties.Set ||
			len(s.patternProperties) > 0 {
			if err := func() error {
				item, err := d.Raw()
				if err != nil {
					return errors.Wrap(err, "parse JSON")
				}

				var matched bool
				for _, p := range s.patternProperties {
					if p.Regexp.Match(k) {
						matched = true
						if err := p.Schema.Validate(item); err != nil {
							return errors.Wrapf(err, "pattern %q", p.Regexp)
						}
					}
				}
				if ok {
					return prop.Validate(item)
				}

				if matched {
					return nil
				}

				ap := s.additionalProperties
				if ap.Set && ap.Schema == nil && !ap.Bool {
					return errors.New("additional properties are not allowed")
				}
				if sch := ap.Schema; sch != nil {
					if err := sch.Validate(item); err != nil {
						return errors.Wrap(err, "additionalProperties")
					}
				}

				return nil
			}(); err != nil {
				return errors.Wrapf(err, "%q", k)
			}
		} else {
			if err := d.Skip(); err != nil {
				return errors.Wrap(err, "parse JSON")
			}
		}
		i++
	}
	if err := iter.Err(); err != nil {
		return errors.Wrap(err, "parse JSON")
	}

	for k := range required {
		return errors.Errorf("required property %q is missing", k)
	}

	if s.minProperties.IsSet() && i < int(s.minProperties) {
		return errors.Errorf("length is smaller than %d", s.minProperties)
	}
	if s.maxProperties.IsSet() && i > int(s.maxProperties) {
		return errors.Errorf("length is bigger than %d", s.maxProperties)
	}

	return nil
}
