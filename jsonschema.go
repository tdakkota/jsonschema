package jsonschema

import (
	"encoding/json"

	"github.com/go-faster/jx"

	"github.com/tdakkota/jsonschema/valueiter/jxvalue"
)

// Parse parses given JSON and compiles JSON Schema validator.
func Parse(data []byte) (*Schema[jxvalue.Value], error) {
	var raw RawSchema
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	doc, err := collectIDs(nil, data)
	if err != nil {
		return nil, err
	}
	return newCompiler[jxvalue.Value](doc, func(raw json.RawMessage) (jxvalue.Value, error) {
		return jxvalue.Value{
			Raw: append(jx.Raw(nil), raw...),
		}, nil
	}).Compile(raw)
}

// ValidateJSON validates given JSON against given JSON Schema.
func ValidateJSON(s *Schema[jxvalue.Value], data []byte) error {
	raw, err := jx.DecodeBytes(data).Raw()
	if err != nil {
		return err
	}
	return validate(s, jxvalue.Value{Raw: raw}, jxvalue.Comparator{})
}
