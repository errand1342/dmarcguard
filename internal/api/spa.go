package api

// spa.go  – drop this file into internal/api/
//
// It replaces the bare http.FileServer with a wrapper that falls back to
// serving index.html for any path that is not a real file and does not
// start with /api or /metrics.  This lets Vue Router handle /login (and
// any other client-side route) instead of the Go server returning 404.

import (
	"io/fs"
	"net/http"
	"strings"
)

// spaHandler wraps a file system and serves index.html for any path that
// doesn't resolve to a real file, so Vue Router can handle the route.
type spaHandler struct {
	fs   fs.FS        // the embedded or OS file system
	base http.Handler // the underlying http.FileServer
}

// newSPAHandler constructs a handler for the given file system.
// Usage (replace your existing static file setup with this):
//
//	h := newSPAHandler(embeddedFS)    // pass your embed.FS sub-FS here
//	mux.Handle("/", h)
func newSPAHandler(fsys fs.FS) http.Handler {
	return &spaHandler{
		fs:   fsys,
		base: http.FileServer(http.FS(fsys)),
	}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Never intercept API or metrics paths – those are handled by their own
	// routes registered earlier in the mux.
	if strings.HasPrefix(r.URL.Path, "/api/") ||
		r.URL.Path == "/api" ||
		r.URL.Path == "/metrics" {
		http.NotFound(w, r)
		return
	}

	// Never fall back to index.html for assets – let 404 be returned instead
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		h.base.ServeHTTP(w, r)
		return
	}

	// Try to open the requested path in the embedded FS.
	// If it exists (JS bundle, CSS, favicon, etc.) serve it directly.
	f, err := h.fs.Open(strings.TrimPrefix(r.URL.Path, "/"))
	if err == nil {
		f.Close()
		h.base.ServeHTTP(w, r)
		return
	}

	// Path not found in FS → serve index.html so Vue Router takes over.
	r2 := r.Clone(r.Context())
	r2.URL.Path = "/"
	h.base.ServeHTTP(w, r2)
}
