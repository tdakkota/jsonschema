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
	for i, test := range tests {
		test := test
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
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

func mustDir(t require.TestingT, fsys embed.FS, p string) []fs.DirEntry {
	entries, err := fsys.ReadDir(p)
	require.NoError(t, err)
	return entries
}

func mustFile(t require.TestingT, fsys embed.FS, p string) []byte {
	entries, err := fsys.ReadFile(p)
	require.NoError(t, err)
	return entries
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

var (
	//go:embed _bench/geojson.json
	benchSchema []byte
	//go:embed _bench/canada.json
	benchData []byte
)

func BenchmarkValidate(b *testing.B) {
	sch, err := Parse(benchSchema)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(benchData)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := sch.Validate(benchData); err != nil {
			b.Fatal(err)
		}
	}
}
