package jsonschema

import "encoding/json"

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
