package jsonschema

import (
	"encoding/json"

	"github.com/go-faster/jx"
	"github.com/go-faster/yaml"

	"github.com/tdakkota/jsonschema/valueiter/jxvalue"
	"github.com/tdakkota/jsonschema/valueiter/yamlxvalue"
)

// Parse parses given JSON and compiles JSON Schema validator.
func Parse(data []byte) (*Schema, error) {
	var raw RawSchema
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	doc, err := collectIDs(nil, data)
	if err != nil {
		return nil, err
	}
	return newCompiler(doc).Compile(raw)
}

// ValidateJSON validates given JSON against given JSON Schema.
func ValidateJSON(s *Schema, data []byte) error {
	raw, err := jx.DecodeBytes(data).Raw()
	if err != nil {
		return err
	}
	type (
		Value      = jxvalue.Value
		Comparator = jxvalue.Comparator
		Str        = []byte
		Key        = []byte
	)
	return validate[Value, Comparator, Str, Key](s, Value{Raw: raw}, Comparator{})
}

// ValidateYAML validates given YAML against given JSON Schema.
func ValidateYAML(s *Schema, data []byte) error {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return err
	}
	type (
		Value      = yamlxvalue.Value
		Comparator = yamlxvalue.Comparator
		Str        = string
		Key        = string
	)
	return validate[Value, Comparator, Str, Key](s, Value{Node: &node}, Comparator{})
}
