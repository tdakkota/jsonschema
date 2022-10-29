package jsonschema

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/go-faster/errors"
	"github.com/stretchr/testify/require"
)

type testingT interface {
	require.TestingT
	Skip(...interface{})
}

var (
	//go:embed _draft/draft4.json
	draft4Raw []byte
	draft4    = errors.Must(Parse(draft4Raw))
)

func mustDir(t testingT, fsys embed.FS, p string) []fs.DirEntry {
	entries, err := fsys.ReadDir(p)
	require.NoError(t, err)
	return entries
}

func mustFile(t testingT, fsys embed.FS, p string) []byte {
	entries, err := fsys.ReadFile(p)
	require.NoError(t, err)
	return entries
}

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

func runTests(t *testing.T, setName string, tests []Test) {
	type caseID struct {
		setName string
		testN   int
		caseN   int
	}
	skipYAML := map[caseID]string{
		{"maxLength", 1, 5}: "Test data contains invalid UTF-8, but yamlx returns error if data is not valid UTF-8",
	}

	for testN, test := range tests {
		testN := testN
		test := test
		t.Run(fmt.Sprintf("Test%d", testN+1), func(t *testing.T) {
			require.NoError(t, ValidateJSON(draft4, test.Schema))

			sch, err := Parse(test.Schema)
			require.NoError(t, err)

			for caseN, cse := range test.Tests {
				caseN := caseN
				cse := cse
				t.Run(fmt.Sprintf("Case%d", caseN+1), func(t *testing.T) {
					check := func(t *testing.T, err error) {
						f := "Schema: %s,\nData: %s,\nDescription: %s"
						args := []interface{}{
							test.Schema,
							cse.Data,
							cse.Description,
						}
						if cse.Valid {
							require.NoErrorf(t, err, f, args...)
						} else {
							require.Errorf(t, err, f, args...)
						}
					}
					t.Run("JSON", func(t *testing.T) {
						check(t, ValidateJSON(sch, cse.Data))
					})
					t.Run("YAML", func(t *testing.T) {
						if reason, ok := skipYAML[caseID{setName, testN + 1, caseN + 1}]; ok {
							t.Skip(reason)
							return
						}
						check(t, ValidateYAML(sch, cse.Data))
					})
				})
			}
		})
	}
}

func runSuite(t *testing.T, suite embed.FS, suiteRoot string) {
	drafts := mustDir(t, suite, suiteRoot)

	for _, draft := range drafts {
		draftName := draft.Name()
		t.Run(draftName, func(t *testing.T) {
			draftPath := path.Join(suiteRoot, draftName)
			sets := mustDir(t, suite, draftPath)

			skipSet := map[string]struct{}{
				"format": {},
			}

			for _, set := range sets {
				setName := set.Name()
				testName := strings.TrimSuffix(setName, ".json")
				t.Run(testName, func(t *testing.T) {
					if _, ok := skipSet[testName]; ok {
						t.Skipf("%s not supported yet", testName)
						return
					}
					data := mustFile(t, suite, path.Join(draftPath, setName))

					var tests []Test
					require.NoError(t, json.Unmarshal(data, &tests))

					runTests(t, testName, tests)
				})
			}
		})
	}
}

func TestParse(t *testing.T) {
	const veryBad = `{
  "allOf": [
    {
      "patternProperties": {
        "foo$": {
          "dependencies": {
            "foo": {
              "additionalProperties": {
                "additionalItems": {
                  "properties": {
                    "foo": {
                      "items": {
                        "required": [
                          "foo",
                          "foo"
                        ]
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  ]
}`

	tests := []struct {
		data    string
		want    *Schema
		wantErr bool
	}{
		// Invalid JSON handling.
		{"", nil, true},
		{"{", nil, true},
		{"[]", nil, true},
		// Invalid structure handling.
		{`{"type":{}}`, nil, true},
		{`{"id":{}}`, nil, true},
		{`{"items":10}`, nil, true},
		{`{"minimum":"10"}`, nil, true},
		{`{"minimum":true}`, nil, true},
		{`{"properties":["foobar"]}`, nil, true},
		{`{"additionalProperties":{"type":1}}`, nil, true},
		{`{"additionalProperties":[]}`, nil, true},
		{`{"patternProperties":{"foo":[]}}`, nil, true},
		{`{"dependencies":{"foo":1}}`, nil, true},
		{`{"dependencies":{"foo":[1]}}`, nil, true},
		{`{"dependencies":{"foo":{"type":1}}}`, nil, true},
		// Invalid "type".
		{`{"type":["foobar"]}`, nil, true},
		// Invalid "id".
		{`{"dependencies":{"id":":"}}`, nil, true},
		{`{"definitions":{"foo":{"id":":"}}}`, nil, true},
		{`{"items":[{"id":":"}]}`, nil, true},
		{`{"items":{"id":":"}}`, nil, true},
		// Invalid "ref".
		{`{"$ref":":"}`, nil, true},
		// Invalid "required".
		{veryBad, nil, true},
		// Bad regex.
		{`{"pattern":"\\"}`, nil, true},
		{`{"patternProperties":{"\\":{}}}`, nil, true},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			a := require.New(t)
			got, err := Parse([]byte(tt.data))
			if tt.wantErr {
				a.Error(err)
				return
			}
			a.NoError(err)
			a.Equal(tt.want, got)
		})
	}
}
