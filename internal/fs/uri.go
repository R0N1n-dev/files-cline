package fs

import (
	"net/url"
	"path/filepath"
	"strings"
)

// FileURI converts a filesystem path into a properly percent-encoded file:// URI
// (RFC 8089), which is what file managers, D-Bus, and drop targets expect.
func FileURI(path string) string {
	abs, err := filepath.Abs(ExpandPath(path))
	if err != nil {
		abs = path
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	return u.String()
}

// FileURIs maps a path list to file:// URIs.
func FileURIs(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, FileURI(p))
	}
	return out
}

// pathFromFileURI converts a file:// URI back into a local filesystem path.
func pathFromFileURI(uri string) string {
	s := strings.TrimSpace(uri)
	if !strings.HasPrefix(s, "file://") {
		return percentDecode(s)
	}
	if u, err := url.Parse(s); err == nil && u.Path != "" {
		return u.Path
	}
	rest := strings.TrimPrefix(s, "file://")
	if i := strings.Index(rest, "/"); i > 0 {
		rest = rest[i:]
	}
	return percentDecode(rest)
}
