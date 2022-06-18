package jsonschema

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"testing"

	"github.com/go-faster/errors"
)

var (
	//go:embed _testdata
	testdata embed.FS
	remotes  = errors.Must(fs.Sub(testdata, path.Join("_testdata", "remotes")))
)

func TestJSONSchemaSuite(t *testing.T) {
	h := http.Server{
		Addr:    "localhost:1234",
		Handler: http.FileServer(http.FS(remotes)),
	}
	defer h.Close()

	go func() {
		if err := h.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Error(err)
		}
	}()
	runSuite(t, testdata, path.Join("_testdata", "suite"))
}

func TestCustomSuite(t *testing.T) {
	runSuite(t, testdata, path.Join("_testdata", "custom"))
}
