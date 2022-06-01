package jsonschema

import (
	"encoding/json"
	"math/big"
	"regexp"
)

type patternProperty struct {
	Regexp *regexp.Regexp
	Schema *Schema
}

type minMax int

func (m minMax) IsSet() bool {
	return m >= 0
}

func parseMinMax(val *uint64) minMax {
	if val != nil {
		return minMax(*val)
	}
	return -1
}

type typeSet uint8

const (
	stringType typeSet = 1 << iota
	numberType
	integerType
	nullType
	booleanType
	arrayType
	objectType
)

func (t typeSet) set(types SchemaType) typeSet {
	for _, typ := range types {
		switch typ {
		case "array":
			t |= arrayType
		case "boolean":
			t |= booleanType
		case "integer":
			t |= integerType
		case "null":
			t |= nullType
		case "number":
			t |= numberType
		case "object":
			t |= objectType
		case "string":
			t |= stringType
		default:
			panic(typ)
		}
	}
	return t
}

func (t typeSet) has(typ typeSet) bool {
	return t == 0 || t&typ != 0
}

type (
	additional struct {
		Set    bool
		Bool   bool
		Schema *Schema
	}
	additionalProperties = additional
	additionalItems      = additional
)

func (a additional) isSchema() bool {
	return a.Set && a.Schema != nil
}

type items struct {
	Set    bool
	Object *Schema
	Array  []*Schema
}

// Schema is a parsed schema structure.
type Schema struct {
	types  typeSet
	format string
	enum   []json.RawMessage

	// Schema composition.
	allOf []*Schema
	anyOf []*Schema
	oneOf []*Schema
	not   *Schema

	// Object validators.
	minProperties        minMax
	maxProperties        minMax
	required             map[string]struct{}
	properties           map[string]*Schema
	patternProperties    []patternProperty
	additionalProperties additionalProperties

	// Array validators.
	minItems        minMax
	maxItems        minMax
	uniqueItems     bool
	items           items
	additionalItems additionalItems

	// Number validators.
	// TODO: try to store small numbers as int64
	minimum          *big.Rat
	exclusiveMinimum bool
	maximum          *big.Rat
	exclusiveMaximum bool
	multipleOf       *big.Rat

	// String validators.
	minLength minMax
	maxLength minMax
	pattern   *regexp.Regexp
}
