package httpserver

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// WithSPA serves a compiled single-page application without weakening API
// routing. Operational paths always stay under the API handler; unknown
// extensionless GET/HEAD requests receive index.html for client-side routing.
func WithSPA(api http.Handler, root fs.FS) (http.Handler, error) {
	if api == nil {
		return nil, errors.New("SPA API handler is required")
	}
	if root == nil {
		return nil, errors.New("SPA filesystem is required")
	}
	if _, err := fs.Stat(root, "index.html"); err != nil {
		return nil, fmt.Errorf("SPA index.html: %w", err)
	}

	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isOperationalPath(r.URL.Path) {
			api.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			api.ServeHTTP(w, r)
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "." || name == "" {
			serveSPAIndex(w, r, root)
			return
		}
		if info, err := fs.Stat(root, name); err == nil && !info.IsDir() {
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "public, max-age=3600")
			}
			files.ServeHTTP(w, r)
			return
		}

		if name == "assets" || strings.HasPrefix(name, "assets/") || path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		serveSPAIndex(w, r, root)
	}), nil
}

func isOperationalPath(requestPath string) bool {
	return requestPath == "/health" || requestPath == "/telemetry" ||
		requestPath == "/api" || strings.HasPrefix(requestPath, "/api/")
}

func serveSPAIndex(w http.ResponseWriter, r *http.Request, root fs.FS) {
	contents, err := fs.ReadFile(root, "index.html")
	if err != nil {
		http.Error(w, "frontend unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(contents)
	}
}
