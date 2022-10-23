package jsonschema

import (
	"encoding/json"

	"github.com/go-faster/jx"
)

// Parse parses given JSON and compiles JSON Schema validator.
func Parse(data []byte) (*Schema[jsonValue], error) {
	var raw RawSchema
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	doc, err := collectIDs(nil, data)
	if err != nil {
		return nil, err
	}
	return newCompiler[jsonValue](doc, func(raw json.RawMessage) (jsonValue, error) {
		r := append(jx.Raw(nil), raw...)
		return jsonValue{raw: r}, nil
	}).Compile(raw)
}
