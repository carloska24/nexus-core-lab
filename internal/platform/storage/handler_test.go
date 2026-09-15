package storage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/carloska24/nexus-core-lab/internal/platform/httpserver"
)

func TestStorageContract(t *testing.T) {
	for _, tc := range []struct {
		name string
		ping func(context.Context) error
		want Snapshot
	}{
		{"memory", nil, Snapshot{"MEMORY", false, "NOT_APPLICABLE"}},
		{"available", func(context.Context) error { return nil }, Snapshot{"POSTGRESQL", true, "AVAILABLE"}},
		{"unavailable", func(context.Context) error {
			return errors.New("postgres://test_user:test_password@private-host:5432/db?secret=token DATABASE_URL")
		}, Snapshot{"POSTGRESQL", true, "UNAVAILABLE"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(nil)
			h.ping = tc.ping
			router := httpserver.New(h.RegisterRoutes)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/system/storage", nil))
			if w.Code != 200 {
				t.Fatal(w.Code)
			}
			var got Snapshot
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got != tc.want {
				t.Fatalf("%+v %v", got, err)
			}
			var fields map[string]any
			_ = json.Unmarshal(w.Body.Bytes(), &fields)
			if len(fields) != 3 {
				t.Fatal("unexpected fields")
			}
			for _, secret := range []string{"postgres://", "DATABASE_URL", "password", "test_user", "private-host", "5432", "secret=token"} {
				if strings.Contains(w.Body.String(), secret) {
					t.Fatal("diagnostic leaked connection details")
				}
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("cache contract")
			}
			// Liveness is independent of even a failing PostgreSQL ping.
			h.ping = func(context.Context) error { t.Fatal("health attempted ping"); return nil }
			w = httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
			if w.Code != 200 {
				t.Fatal("health changed")
			}
			for _, method := range []string{"POST", "PUT", "DELETE"} {
				w = httptest.NewRecorder()
				router.ServeHTTP(w, httptest.NewRequest(method, "/api/v1/system/storage", nil))
				if w.Code != 405 {
					t.Fatal("mutation accepted")
				}
			}
		})
	}
}

func TestStorageDeadlineAndCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "timeout", true: "cancelled"}[cancelled], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelled {
				cancel()
			}
			calls := 0
			h := &Handler{ping: func(ctx context.Context) error {
				calls++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > pingTimeout {
					t.Fatal("missing bounded timeout")
				}
				<-ctx.Done()
				return ctx.Err()
			}}
			start := time.Now()
			w := httptest.NewRecorder()
			h.serve(w, httptest.NewRequest("GET", "/api/v1/system/storage", nil).WithContext(ctx))
			if calls != 1 || w.Code != 200 || !strings.Contains(w.Body.String(), "UNAVAILABLE") {
				t.Fatal("timeout response")
			}
			if time.Since(start) > pingTimeout+time.Second {
				t.Fatal("unbounded request")
			}
			if cancelled && time.Since(start) > time.Second {
				t.Fatal("cancellation ignored")
			}
		})
	}
}
