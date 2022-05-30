package jsonschema

import (
	"embed"
	"io/fs"

	"github.com/stretchr/testify/require"
)

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
