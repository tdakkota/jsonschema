package jsonschema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_collectIDs(t *testing.T) {
	a := require.New(t)
	d, err := collectIDs(nil, []byte(`
{
            "definitions": {
                "id_in_enum": {
                    "enum": [
                        {
                          "id": "https://localhost:1234/my_identifier.json",
                          "type": "null"
                        }
                    ]
                },
                "real_id_in_schema": {
                    "id": "https://localhost:1234/my_identifier.json",
                    "type": "string"
                },
                "zzz_id_in_const": {
                    "const": {
                        "id": "https://localhost:1234/my_identifier.json",
                        "type": "null"
                    }
                }
            },
            "anyOf": [
                { "$ref": "#/definitions/id_in_enum" },
                { "$ref": "https://localhost:1234/my_identifier.json" }
            ]
        }`))
	a.NoError(err)
	a.NotEmpty(d.ids)
}
