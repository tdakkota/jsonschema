package jsonschema

import (
	"embed"
	"path"
	"testing"
)

//go:embed _testdata
var testdata embed.FS

func TestJSONSchemaSuite(t *testing.T) {
	runSuite(t, testdata, path.Join("_testdata", "suite"))
}

func TestCustomSuite(t *testing.T) {
	runSuite(t, testdata, path.Join("_testdata", "custom"))
}
