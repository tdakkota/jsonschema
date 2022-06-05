package jsonschema

import (
	"embed"
	"path"
	"strings"
	"testing"
)

var (
	//go:embed _bench
	bench embed.FS
)

func BenchmarkValidate(b *testing.B) {
	const root = "_bench"
	for _, e := range mustDir(b, bench, root) {
		schemaDirPath := path.Join(root, e.Name())
		b.Run(e.Name(), func(b *testing.B) {
			benchSchema := mustFile(b, bench, path.Join(schemaDirPath, "schema.json"))
			sch, err := Parse(benchSchema)
			if err != nil {
				b.Skip("Unsupported yet")
				return
			}

			dataDirPath := path.Join(schemaDirPath, "data")
			for _, f := range mustDir(b, bench, dataDirPath) {
				benchData := mustFile(b, bench, path.Join(dataDirPath, f.Name()))
				testName := strings.TrimSuffix(f.Name(), ".json")
				b.Run(testName, func(b *testing.B) {
					b.SetBytes(int64(len(benchData)))
					b.ReportAllocs()
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						if err := sch.Validate(benchData); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
