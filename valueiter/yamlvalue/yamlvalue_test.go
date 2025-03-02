package yamlvalue

import (
	"fmt"
	"testing"

	"github.com/go-faster/yaml"
	"github.com/stretchr/testify/require"
)

func Test_yamlEqual(t *testing.T) {
	tests := []struct {
		l, r string
		want bool
	}{
		// Null.
		{`null`, `null`, true},
		// Bool.
		{`false`, `false`, true},
		{`true`, `true`, true},
		{`false`, `true`, false},
		// String.
		{`"foo"`, `"foo"`, true},
		{`"foo"`, `"foo" `, true},
		{`"foo\u000a"`, `"foo\n"`, true},
		{`"foo"`, `"foo\n"`, false},
		{`"foo"`, `"foo "`, false},
		// Number.
		{`0`, `0`, true},
		{`-0`, `-0`, true},
		{`-0`, `0`, true},
		{`1`, `1`, true},
		{`10`, `10`, true},
		{`0.0`, `0.0`, true},
		{`10`, `1e1`, true},
		{`1000000000000000000000000000000`, `1000000000000000000000000000000`, true},
		{`10`, `1.0e1`, true},
		{`0`, `1`, false},
		{`-1`, `1`, false},
		{`1e1`, `100`, false},
		// Array.
		{`[]`, `[]`, true},
		{`[]`, `[ ]`, true},
		{`[[]]`, `[[] ]`, true},
		{`["a", "b"]`, `["a", "b"]`, true},
		{`["a"]`, `[]`, false},
		{`[1,2,3]`, `[1,2]`, false},
		{`[[]]`, `[[1]]`, false},
		{`["b","a"]`, `["a","b"]`, false},
		// Object.
		{`{}`, `{}`, true},
		{`{}`, `{ }`, true},
		{`{"a":"b"}`, `{"a":"b"}`, true},
		{`{"a":"b","b":"a"}`, `{"b":"a", "a":"b"}`, true},
		{`{}`, `{"a":"b"}`, false},
		{`{"b":"a"}`, `{"a":"b"}`, false},
		{`{"a":10}`, `{"a":"b"}`, false},
		// Type comparison.
		{`{}`, `[]`, false},
		{`{}`, `0`, false},
		{`{}`, `null`, false},
		{`{}`, `false`, false},
		{`{}`, `""`, false},
	}

	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			a := require.New(t)
			parseNode := func(s string) *yaml.Node {
				var n yaml.Node
				a.NoError(yaml.Unmarshal([]byte(s), &n))
				return &n
			}

			got, err := yamlEqual(parseNode(tt.l), parseNode(tt.r))
			a.NoError(err)
			a.Equal(tt.want, got)
		})
	}
}
