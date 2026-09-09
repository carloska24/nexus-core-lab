package httpserver

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func TestTelemetryMiddleware_RequestIDGenerated(t *testing.T) {
	var capturedID string
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	var counter atomic.Uint64
	handler := TelemetryMiddleware(&counter)(dummy)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	respID := w.Header().Get(RequestIDHeader)
	if respID == "" {
		t.Fatalf("expected X-Request-ID header in response")
	}
	if capturedID != respID {
		t.Errorf("expected context request_id %s to match response header %s", capturedID, respID)
	}
	if counter.Load() != 1 {
		t.Errorf("expected requests counter to be 1, got %d", counter.Load())
	}
}

func TestTelemetryMiddleware_RequestIDPreserved(t *testing.T) {
	expectedID := "custom-client-req-id-12345"
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID := GetRequestID(r.Context())
		if ctxID != expectedID {
			t.Errorf("expected context request_id %s, got %s", expectedID, ctxID)
		}
		w.WriteHeader(http.StatusAccepted)
	})

	var counter atomic.Uint64
	handler := TelemetryMiddleware(&counter)(dummy)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set(RequestIDHeader, expectedID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	respID := w.Header().Get(RequestIDHeader)
	if respID != expectedID {
		t.Errorf("expected preserved X-Request-ID %s, got %s", expectedID, respID)
	}
}

func TestTelemetryMiddleware_ConcurrentRequestsCounter(t *testing.T) {
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var counter atomic.Uint64
	handler := TelemetryMiddleware(&counter)(dummy)

	var wg sync.WaitGroup
	requests := 100

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/concurrent", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}()
	}

	wg.Wait()

	if counter.Load() != uint64(requests) {
		t.Errorf("expected %d requests total, got %d", requests, counter.Load())
	}
}
