package jsonschema

import (
	"math/big"
	"regexp"

	"github.com/tdakkota/jsonschema/valueiter"
)

type patternProperty[V valueiter.Value[V]] struct {
	Regexp *regexp.Regexp
	Schema *Schema[V]
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
	additional[V valueiter.Value[V]] struct {
		Set    bool
		Bool   bool
		Schema *Schema[V]
	}
)

func (a additional[V]) isSchema() bool {
	return a.Set && a.Schema != nil
}

type schemaItems[V valueiter.Value[V]] struct {
	Set    bool
	Object *Schema[V]
	Array  []*Schema[V]
}

// Schema is a parsed schema structure.
type Schema[V valueiter.Value[V]] struct {
	types  typeSet
	format string

	enum []V

	// Schema composition.
	allOf []*Schema[V]
	anyOf []*Schema[V]
	oneOf []*Schema[V]
	not   *Schema[V]

	// Object validators.
	minProperties        minMax
	maxProperties        minMax
	required             map[string]struct{}
	properties           map[string]*Schema[V]
	patternProperties    []patternProperty[V]
	additionalProperties additional[V]
	dependentRequired    map[string][]string
	dependentSchemas     map[string]*Schema[V]

	// Array validators.
	minItems        minMax
	maxItems        minMax
	uniqueItems     bool
	items           schemaItems[V]
	additionalItems additional[V]

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
