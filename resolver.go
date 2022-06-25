package jsonschema

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"

	"github.com/go-faster/errors"
)

// RemoteResolver resolves remote references.
type RemoteResolver interface {
	Resolve(ctx context.Context, loc string) ([]byte, error)
}

// NoRemote is no-op implementation of RemoteResolver.
// Always returns error.
type NoRemote struct {
}

// Resolve implements RemoteResolver.
func (n NoRemote) Resolve(ctx context.Context, loc string) ([]byte, error) {
	return nil, errors.New("remote references are not allowed")
}

var _ RemoteResolver = Remote{}

// Remote is built-in implementation of RemoteResolver.
type Remote struct {
	HTTPClient    *http.Client
	AllowRelative bool
}

func (n Remote) getClient() *http.Client {
	if c := n.HTTPClient; c != nil {
		return c
	}
	return http.DefaultClient
}

func (n Remote) getHTTP(ctx context.Context, u *url.URL) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}
	if pass, ok := u.User.Password(); ok && u.User != nil {
		req.SetBasicAuth(u.User.Username(), pass)
	}

	resp, err := n.getClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "do")
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if code := resp.StatusCode; code >= 299 {
		text := http.StatusText(code)
		return nil, errors.Errorf("bad HTTP code %d (%s)", code, text)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read data")
	}

	return data, nil
}

// Resolve implements RemoteResolver.
func (n Remote) Resolve(ctx context.Context, loc string) ([]byte, error) {
	u, err := url.Parse(loc)
	if err != nil {
		return nil, errors.Wrap(err, "parse location")
	}

	switch u.Scheme {
	case "http", "https":
		return n.getHTTP(ctx, u)
	case "file", "":
		if !n.AllowRelative && !fs.ValidPath(u.Path) {
			return nil, errors.New("relative paths are not allowed")
		}

		return os.ReadFile(u.Path)
	default:
		return nil, errors.Errorf("unknown scheme %q", u.Scheme)
	}
}
