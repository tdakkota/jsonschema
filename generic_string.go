package jsonschema

import (
	"regexp"

	"github.com/tdakkota/jsonschema/valueiter"
)

func runeCount[S valueiter.ByteSeq](s S) int {
	return len([]rune(string(s)))
}

func regexpMatch[S valueiter.ByteSeq](r *regexp.Regexp, s S) bool {
	switch s := any(s).(type) {
	case string:
		return r.MatchString(s)
	case []byte:
		return r.Match(s)
	default:
		panic("unreachable")
	}
}
