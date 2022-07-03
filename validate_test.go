package jsonschema

import (
	"embed"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	//go:embed _bench
	bench embed.FS
)

type benchData struct {
	Name string
	Data []byte
}

type benchSchema struct {
	Name   string
	Schema *Schema
	Data   []benchData
	Skip   bool
}

func collectBench(b testingT) (r []benchSchema) {
	const root = "_bench"
	for _, e := range mustDir(b, bench, root) {
		schemaDirPath := path.Join(root, e.Name())

		schema := mustFile(b, bench, path.Join(schemaDirPath, "schema.json"))
		sch, err := Parse(schema)
		if err != nil {
			b.Errorf("Cannot generate %s: %s", schemaDirPath, err)
			continue
		}

		dataDirPath := path.Join(schemaDirPath, "data")
		var datas []benchData
		for _, f := range mustDir(b, bench, dataDirPath) {
			datas = append(datas, benchData{
				Name: strings.TrimSuffix(f.Name(), ".json"),
				Data: mustFile(b, bench, path.Join(dataDirPath, f.Name())),
			})
		}

		r = append(r, benchSchema{
			Name:   e.Name(),
			Schema: sch,
			Data:   datas,
		})
	}
	return r
}

func TestBenchSuite(t *testing.T) {
	for _, s := range collectBench(t) {
		s := s
		t.Run(s.Name, func(t *testing.T) {
			if s.Skip {
				t.Skip("Unsupported yet")
			}

			for _, data := range s.Data {
				data := data
				t.Run(data.Name, func(t *testing.T) {
					require.NoError(t, s.Schema.Validate(data.Data))
				})
			}
		})
	}
}

func BenchmarkValidate(b *testing.B) {
	for _, s := range collectBench(b) {
		s := s
		b.Run(s.Name, func(b *testing.B) {
			if s.Skip {
				b.Skip("Unsupported yet")
			}

			for _, data := range s.Data {
				data := data
				b.Run(data.Name, func(b *testing.B) {
					b.SetBytes(int64(len(data.Data)))
					b.ReportAllocs()
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						if err := s.Schema.Validate(data.Data); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
