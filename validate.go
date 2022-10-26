package jsonschema

import (
	"fmt"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"

	"github.com/tdakkota/jsonschema/valueiter"
)

func validate[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
	if len(s.enum) > 0 || len(s.allOf) > 0 || len(s.oneOf) > 0 || len(s.anyOf) > 0 || s.not != nil {
		if err := validateEnum[V, C, Str, Key](s, val, cmp); err != nil {
			return errors.Wrap(err, "enum")
		}
		if err := validateAllOf[V, C, Str, Key](s, val, cmp); err != nil {
			return errors.Wrap(err, "allOf")
		}
		if err := validateOneOf[V, C, Str, Key](s, val, cmp); err != nil {
			return errors.Wrap(err, "oneOf")
		}
		if err := validateAnyOf[V, C, Str, Key](s, val, cmp); err != nil {
			return errors.Wrap(err, "anyOf")
		}
		if err := validateNot[V, C, Str, Key](s, val, cmp); err != nil {
			return errors.Wrap(err, "not")
		}
	}

	var (
		tt  = val.Type()
		err error
	)
	switch tt {
	case jx.String:
		err = validateString[V, C, Str, Key](s, val, cmp)
	case jx.Number:
		err = validateNumber[V, C, Str, Key](s, val, cmp)
	case jx.Null:
		err = validateNull[V, C, Str, Key](s, val, cmp)
	case jx.Bool:
		err = validateBool[V, C, Str, Key](s, val, cmp)
	case jx.Array:
		err = validateArray[V, C, Str, Key](s, val, cmp)
	case jx.Object:
		err = validateObject[V, C, Str, Key](s, val, cmp)
	default:
		panic(fmt.Sprintf("unreachable: %q", tt))
	}
	if err != nil {
		return errors.Wrap(err, tt.String())
	}
	return nil
}

func validateEnum[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
	if len(s.enum) == 0 {
		return nil
	}
	ok, err := cmp.Contains(s.enum, val)
	if err != nil {
		return errors.Wrap(err, "compare value")
	}
	if ok {
		return nil
	}
	return errors.Errorf("%v is not present in enum", val)
}

func validateAllOf[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
	for i, schema := range s.allOf {
		if err := validate[V, C, Str, Key](schema, val, cmp); err != nil {
			return errors.Wrapf(err, "[%d]", i)
		}
	}
	return nil
}

func validateOneOf[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
	if len(s.oneOf) == 0 {
		return nil
	}

	counter := 0
	for _, schema := range s.oneOf {
		if err := validate[V, C, Str, Key](schema, val, cmp); err == nil {
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

func validateAnyOf[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
	if len(s.anyOf) == 0 {
		return nil
	}

	for _, schema := range s.anyOf {
		if err := validate[V, C, Str, Key](schema, val, cmp); err == nil {
			return nil
		}
	}
	return errors.New("must match at least once")
}

func validateNot[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
	if s := s.not; s != nil {
		if err := validate[V, C, Str, Key](s, val, cmp); err == nil {
			return errors.New("must not match")
		}
	}
	return nil
}

func checkType(s *Schema, t typeSet) error {
	if !s.types.has(t) {
		return errors.New("type is not allowed")
	}
	return nil
}

func validateString[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
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
		count := runeCount(str)
		if s.minLength.IsSet() && count < int(s.minLength) {
			return errors.Errorf("length is smaller than %d", s.minLength)
		}
		if s.maxLength.IsSet() && count > int(s.maxLength) {
			return errors.Errorf("length is bigger than %d", s.maxLength)
		}
	}
	if pattern := s.pattern; pattern != nil && !regexpMatch(pattern, str) {
		return errors.Errorf("does not match pattern %s", pattern)
	}
	return nil
}

func validateNumber[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
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

func validateNull[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
	return checkType(s, nullType)
}

func validateBool[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
	return checkType(s, booleanType)
}

func elemValidator(s *Schema, idx int) (*Schema, error) {
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

func validateArray[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
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
			if err := validate[V, C, Str, Key](sch, val, cmp); err != nil {
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

func validateObject[
	V valueiter.Value[V, Str, Key],
	C valueiter.ValueComparator[V],
	Str, Key valueiter.ByteSeq,
](s *Schema, val V, cmp C) error {
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
		if err := val.Object(func(k Key, _ V) error {
			if r, ok := s.dependentRequired[string(k)]; ok {
				for _, value := range r {
					required[value] = struct{}{}
				}
			}
			if ds, ok := s.dependentSchemas[string(k)]; ok {
				if err := validate[V, C, Str, Key](ds, rootval, cmp); err != nil {
					return errors.Wrapf(err, "dependent %q", k)
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	if err := val.Object(func(k Key, val V) (rerr error) {
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
			if regexpMatch(p.Regexp, k) {
				matched = true
				if err := validate[V, C, Str, Key](p.Schema, val, cmp); err != nil {
					return errors.Wrapf(err, "pattern %q", p.Regexp)
				}
			}
		}
		if ok {
			return validate[V, C, Str, Key](prop, val, cmp)
		}

		if matched {
			return nil
		}

		ap := s.additionalProperties
		if ap.Set && ap.Schema == nil && !ap.Bool {
			return errors.New("additional properties are not allowed")
		}
		if s := ap.Schema; s != nil {
			if err := validate[V, C, Str, Key](s, val, cmp); err != nil {
				return errors.Wrap(err, "additionalProperties")
			}
			return validate[V, C, Str, Key](s, val, cmp)
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
