package jsonschema

import (
	"testing"

	"github.com/go-faster/jx"
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

	d, err = collectIDs(nil, []byte(`{"definitions": null}`))
	a.NoError(err)
	a.Empty(d.ids)
}

func Test_document_findID(t *testing.T) {
	doc := &document{}
	require.Error(t, doc.findID(jx.DecodeStr(`{"id": null}`), nil))
}
