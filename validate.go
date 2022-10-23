package jsonschema

import (
	"fmt"
	"unicode/utf8"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

func validate[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	if len(s.enum) > 0 || len(s.allOf) > 0 || len(s.oneOf) > 0 || len(s.anyOf) > 0 || s.not != nil {
		if err := validateEnum(s, val, cmp); err != nil {
			return errors.Wrap(err, "enum")
		}
		if err := validateAllOf(s, val, cmp); err != nil {
			return errors.Wrap(err, "allOf")
		}
		if err := validateOneOf(s, val, cmp); err != nil {
			return errors.Wrap(err, "oneOf")
		}
		if err := validateAnyOf(s, val, cmp); err != nil {
			return errors.Wrap(err, "anyOf")
		}
		if err := validateNot(s, val, cmp); err != nil {
			return errors.Wrap(err, "not")
		}
	}

	var (
		tt  = val.Type()
		err error
	)
	switch tt {
	case jx.String:
		err = validateString(s, val, cmp)
	case jx.Number:
		err = validateNumber(s, val, cmp)
	case jx.Null:
		err = validateNull(s, val, cmp)
	case jx.Bool:
		err = validateBool(s, val, cmp)
	case jx.Array:
		err = validateArray(s, val, cmp)
	case jx.Object:
		err = validateObject(s, val, cmp)
	default:
		panic(fmt.Sprintf("unreachable: %q", tt))
	}
	if err != nil {
		return errors.Wrap(err, tt.String())
	}
	return nil
}

func validateEnum[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	if len(s.enum) == 0 {
		return nil
	}

	for _, variant := range s.enum {
		ok, err := cmp.Equal(variant, val)
		if err != nil {
			return errors.Wrap(err, "compare")
		}
		if ok {
			return nil
		}
	}
	return errors.Errorf("%v is not present in enum", val)
}

func validateAllOf[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	for i, schema := range s.allOf {
		if err := validate(schema, val, cmp); err != nil {
			return errors.Wrapf(err, "[%d]", i)
		}
	}
	return nil
}

func validateOneOf[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	if len(s.oneOf) == 0 {
		return nil
	}

	counter := 0
	for _, schema := range s.oneOf {
		if err := validate(schema, val, cmp); err == nil {
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

func validateAnyOf[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	if len(s.anyOf) == 0 {
		return nil
	}

	for _, schema := range s.anyOf {
		if err := validate(schema, val, cmp); err == nil {
			return nil
		}
	}
	return errors.New("must match at least once")
}

func validateNot[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	if s := s.not; s != nil {
		if err := validate(s, val, cmp); err == nil {
			return errors.New("must not match")
		}
	}
	return nil
}

func checkType[V Value[V]](s *Schema[V], t typeSet) error {
	if !s.types.has(t) {
		return errors.New("type is not allowed")
	}
	return nil
}

func validateString[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	if err := checkType(s, stringType); err != nil {
		return err
	}

	if !(s.format != "" || s.minLength.IsSet() || s.maxLength.IsSet() || s.pattern != nil) {
		return nil
	}

	str := val.Str()
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

func validateNumber[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	hasNumber := s.types.has(numberType)

	if hasNumber && !(s.minimum != nil || s.maximum != nil || s.multipleOf != nil) {
		return nil
	}

	num := val.Number()
	if !hasNumber {
		isInt := num.IsInt()
		if isInt {
			if err := checkType(s, integerType); err != nil {
				return err
			}
		} else {
			return checkType(s, numberType)
		}
	}

	if s.minimum != nil || s.maximum != nil || s.multipleOf != nil {
		val := num.Rat
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

func validateNull[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	return checkType(s, nullType)
}

func validateBool[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	return checkType(s, booleanType)
}

func elemValidator[V Value[V]](s *Schema[V], idx int) (*Schema[V], error) {
	// 5.3.1.2.  Conditions for successful validation
	//
	// If "items" is not present, or its value is an object, validation
	// of the instance always succeeds, regardless of the value of
	// "additionalItems";
	if obj := s.items.Object; !s.items.Set || obj != nil {
		return obj, nil
	}

	if arr := s.items.Array; idx < len(arr) {
		return arr[idx], nil
	}

	ai := s.additionalItems
	if !ai.Set {
		return nil, nil
	}
	if ai.isSchema() {
		return ai.Schema, nil
	}
	if ai.Bool {
		return nil, nil
	}
	return nil, errors.New("schema does not allow additionalItems")
}

func validateArray[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	if err := checkType(s, arrayType); err != nil {
		return err
	}

	if !(s.minItems.IsSet() ||
		s.maxItems.IsSet() ||
		s.uniqueItems ||
		s.items.Set ||
		s.additionalItems.Set) {
		return nil
	}

	var (
		items []V
		i     = 0
	)
	if err := val.Array(func(val V) error {
		sch, err := elemValidator(s, i)
		if err != nil {
			return err
		}
		if sch != nil {
			if err := validate(sch, val, cmp); err != nil {
				return err
			}
		}
		if s.uniqueItems {
			items = append(items, val)
		}

		i++
		return nil
	}); err != nil {
		return err
	}

	if len(items) > 1 {
		for xi, x := range items {
			for yi, y := range items {
				if xi == yi {
					continue
				}
				if ok, _ := cmp.Equal(x, y); ok {
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

func validateObject[V Value[V], C ValueComparator[V]](s *Schema[V], val V, cmp C) error {
	if err := checkType(s, objectType); err != nil {
		return err
	}

	if !(s.minProperties.IsSet() ||
		s.maxProperties.IsSet() ||
		len(s.required) > 0 ||
		len(s.properties) > 0 ||
		len(s.patternProperties) > 0 ||
		s.additionalProperties.Set ||
		len(s.dependentSchemas) > 0 ||
		len(s.dependentRequired) > 0) {
		return nil
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
	if len(s.dependentRequired) > 0 || len(s.dependentSchemas) > 0 {
		if len(s.dependentRequired) > 0 && required == nil {
			required = map[string]struct{}{}
		}
		rootval := val
		if err := val.Object(func(k []byte, _ V) error {
			if r, ok := s.dependentRequired[string(k)]; ok {
				for _, value := range r {
					required[value] = struct{}{}
				}
			}
			if ds, ok := s.dependentSchemas[string(k)]; ok {
				if err := validate(ds, rootval, cmp); err != nil {
					return errors.Wrapf(err, "dependent %q", k)
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	if err := val.Object(func(k []byte, val V) (rerr error) {
		defer func() {
			i++
			if rerr != nil {
				rerr = errors.Wrapf(rerr, "%q", k)
			}
		}()
		delete(required, string(k))

		prop, ok := s.properties[string(k)]
		var matched bool
		for _, p := range s.patternProperties {
			if p.Regexp.Match(k) {
				matched = true
				if err := validate(p.Schema, val, cmp); err != nil {
					return errors.Wrapf(err, "pattern %q", p.Regexp)
				}
			}
		}
		if ok {
			return validate(prop, val, cmp)
		}

		if matched {
			return nil
		}

		ap := s.additionalProperties
		if ap.Set && ap.Schema == nil && !ap.Bool {
			return errors.New("additional properties are not allowed")
		}
		if s := ap.Schema; s != nil {
			if err := validate(s, val, cmp); err != nil {
				return errors.Wrap(err, "additionalProperties")
			}
			return validate(s, val, cmp)
		}

		return nil
	}); err != nil {
		return err
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
