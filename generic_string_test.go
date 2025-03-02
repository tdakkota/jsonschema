package jsonschema

import (
	"regexp"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func Test_runeCount(t *testing.T) {
	tests := []string{
		"",
		"\n",
		"\xff",
		"foo",
		"🤡",
	}
	for _, test := range tests {
		a := require.New(t)

		expected := utf8.RuneCountInString(test)
		testByte := []byte(test)

		// Ensure zero allocation.
		a.Zero(testing.AllocsPerRun(100, func() {
			runeCount(test)
		}))
		a.Zero(testing.AllocsPerRun(100, func() {
			runeCount(testByte)
		}))

		a.Equal(expected, runeCount(test))
		a.Equal(expected, runeCount(testByte))
	}
}

func Test_regexpMatch(t *testing.T) {
	a := require.New(t)

	r := regexp.MustCompile(`\d+`)
	tests := []struct {
		s        string
		expected bool
	}{
		{"", false},
		{"foo", false},
		{"123", true},
		{"123foo", true},
	}
	for _, tt := range tests {
		test := tt.s
		testByte := []byte(test)

		// Ensure zero allocation.
		a.Zero(testing.AllocsPerRun(100, func() {
			regexpMatch(r, test)
		}))
		a.Zero(testing.AllocsPerRun(100, func() {
			regexpMatch(r, testByte)
		}))

		a.Equal(tt.expected, regexpMatch(r, test))
		a.Equal(tt.expected, regexpMatch(r, testByte))
	}
}
