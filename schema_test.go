package jsonschema

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed _testdata
var suite embed.FS

type Case struct {
	Description string          `json:"description"`
	Data        json.RawMessage `json:"data"`
	Valid       bool            `json:"valid"`
}

type Test struct {
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
	Tests       []Case          `json:"tests"`
}

func runTests(t *testing.T, tests []Test) {
	parse := func(data []byte) (*Schema, error) {
		var raw RawSchema
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		return NewParser(rootResolver(data)).Parse(raw)
	}

	for i, test := range tests {
		test := test
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			for i, cse := range test.Tests {
				cse := cse
				t.Run(fmt.Sprintf("Case%d", i+1), func(t *testing.T) {
					a := require.New(t)
					sch, err := parse(test.Schema)
					if err != nil {
						t.Skip(err)
						return
					}

					f := "Schema: %s,\nData: %s,\nDescription: %s"
					args := []interface{}{
						test.Schema,
						cse.Data,
						cse.Description,
					}
					if err := sch.Validate(cse.Data); cse.Valid {
						a.NoErrorf(err, f, args...)
					} else {
						a.Errorf(err, f, args...)
					}
				})
			}
		})
	}
}

func TestJSONSchemaSuite(t *testing.T) {
	drafts, err := fs.ReadDir(suite, "_testdata")
	require.NoError(t, err)

	for _, draft := range drafts {
		draftName := draft.Name()
		t.Run(draftName, func(t *testing.T) {
			p := path.Join("_testdata", draftName)
			sets, err := fs.ReadDir(suite, p)
			require.NoError(t, err)

			skipSet := map[string]struct{}{
				"id":           {},
				"definitions":  {},
				"dependencies": {},
				"refRemote":    {},
			}

			for _, set := range sets {
				setName := set.Name()
				testName := strings.TrimSuffix(setName, ".json")
				t.Run(strings.TrimSuffix(setName, ".json"), func(t *testing.T) {
					if _, ok := skipSet[testName]; ok {
						t.Skipf("%s not supported yet", testName)
						return
					}
					data, err := fs.ReadFile(suite, path.Join(p, setName))
					require.NoError(t, err)

					var tests []Test
					require.NoError(t, json.Unmarshal(data, &tests))

					runTests(t, tests)
				})
			}
		})
	}
}
