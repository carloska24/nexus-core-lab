package httpserver

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestWithSPA(t *testing.T) {
	t.Parallel()
	root := fstest.MapFS{
		"index.html":             {Data: []byte("<main>NEXUS</main>")},
		"assets/index-abc123.js": {Data: []byte("console.log('nexus')")},
		"favicon.svg":            {Data: []byte("<svg></svg>")},
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/health" {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.NotFound(w, r)
	})
	handler, err := WithSPA(api, root)
	if err != nil {
		t.Fatalf("WithSPA: %v", err)
	}

	tests := []struct {
		name, method, target string
		status               int
		cache, content       string
	}{
		{"root", http.MethodGet, "/", http.StatusOK, "no-cache", "<main>NEXUS</main>"},
		{"frontend route", http.MethodGet, "/network", http.StatusOK, "no-cache", "<main>NEXUS</main>"},
		{"hashed asset", http.MethodGet, "/assets/index-abc123.js", http.StatusOK, "public, max-age=31536000, immutable", "console.log('nexus')"},
		{"ordinary asset", http.MethodGet, "/favicon.svg", http.StatusOK, "public, max-age=3600", "<svg></svg>"},
		{"missing asset", http.MethodGet, "/assets/missing.js", http.StatusNotFound, "", "404 page not found\n"},
		{"missing extensionless asset", http.MethodGet, "/assets/missing", http.StatusNotFound, "", "404 page not found\n"},
		{"health precedence", http.MethodGet, "/health", http.StatusOK, "", `{"status":"ok"}`},
		{"API 404 remains API 404", http.MethodGet, "/api/v1/missing", http.StatusNotFound, "", "404 page not found\n"},
		{"non-GET does not fall back", http.MethodPost, "/network", http.StatusNotFound, "", "404 page not found\n"},
		{"head fallback", http.MethodHead, "/network", http.StatusOK, "no-cache", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.target, nil)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != test.status {
				t.Fatalf("status = %d, want %d", res.Code, test.status)
			}
			if test.cache != "" && res.Header().Get("Cache-Control") != test.cache {
				t.Fatalf("Cache-Control = %q, want %q", res.Header().Get("Cache-Control"), test.cache)
			}
			if got := res.Body.String(); got != test.content {
				t.Fatalf("body = %q, want %q", got, test.content)
			}
		})
	}
}

func TestWithSPARequiresIndex(t *testing.T) {
	t.Parallel()
	_, err := WithSPA(http.NotFoundHandler(), fstest.MapFS{})
	if err == nil {
		t.Fatal("expected missing index error")
	}
	if !errorsIsPath(err, fs.ErrNotExist) {
		t.Fatalf("error = %v, want fs.ErrNotExist", err)
	}
}

func errorsIsPath(err error, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		unwrapped, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = unwrapped.Unwrap()
	}
	return false
}
