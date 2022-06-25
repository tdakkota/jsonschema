package jsonschema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_collectIDs(t *testing.T) {
	a := require.New(t)
	root := []byte(`{
            "id": "http://localhost:1234/",
            "items": {
                "id": "baseUriChange/",
                "items": {"$ref": "folderInteger.json"}
            }
        }`)

	d, err := collectIDs(nil, root)
	a.NoError(err)
	a.NotEmpty(d.ids)
	a.NotEmpty(d.ids["http://localhost:1234/baseUriChange/"])
}
