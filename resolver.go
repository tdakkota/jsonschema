package jsonschema

import (
	"encoding/json"

	"github.com/tdakkota/jsonschema/internal/jsonpointer"
)

type rootResolver []byte

func (r rootResolver) ResolveReference(ref string) (s RawSchema, _ error) {
	data, err := jsonpointer.Resolve(ref, r)
	if err != nil {
		return RawSchema{}, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}
