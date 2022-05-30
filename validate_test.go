package jsonschema

import (
	_ "embed"
	"testing"
)

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
