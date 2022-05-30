package jsonschema

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/go-faster/errors"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed _testdata
	suite embed.FS

	//go:embed _draft/draft4.json
	draft4Raw []byte
	draft4    = errors.Must(Parse(draft4Raw))
)

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
	for i, test := range tests {
		test := test
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			require.NoError(t, draft4.Validate(test.Schema))

			sch, err := Parse(test.Schema)
			if err != nil {
				t.Skipf("Schema: %s,\nError: %s", test.Schema, err)
				return
			}
			for i, cse := range test.Tests {
				cse := cse
				t.Run(fmt.Sprintf("Case%d", i+1), func(t *testing.T) {
					a := require.New(t)

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
	suiteRoot := path.Join("_testdata", "suite")
	drafts := mustDir(t, suite, suiteRoot)

	for _, draft := range drafts {
		draftName := draft.Name()
		t.Run(draftName, func(t *testing.T) {
			draftPath := path.Join(suiteRoot, draftName)
			sets := mustDir(t, suite, draftPath)

			skipSet := map[string]struct{}{
				"id":           {},
				"definitions":  {},
				"dependencies": {},
				"format":       {},
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
					data := mustFile(t, suite, path.Join(draftPath, setName))

					var tests []Test
					require.NoError(t, json.Unmarshal(data, &tests))

					runTests(t, tests)
				})
			}
		})
	}
}
