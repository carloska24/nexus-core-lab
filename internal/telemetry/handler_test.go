package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestHandler_GetTelemetry(t *testing.T) {
	metrics := &Metrics{}
	metrics.attachTotal.Store(5)
	metrics.cellHandoverTotal.Store(3)
	metrics.detachTotal.Store(2)
	metrics.staleDisconnectTotal.Store(1)
	metrics.droppedEventsTotal.Store(0)

	var reqCounter atomic.Uint64
	reqCounter.Store(42)

	activeProvider := &mockActiveProvider{count: 4}

	h := NewHandler(metrics, activeProvider, &reqCounter)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/telemetry", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", w.Code)
	}

	var resp TelemetryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Service != "nexus-core-lab" {
		t.Errorf("expected service nexus-core-lab, got %s", resp.Service)
	}
	if resp.RequestsTotal != 42 {
		t.Errorf("expected requests_total 42, got %d", resp.RequestsTotal)
	}
	if resp.Metrics.ActiveSessions != 4 {
		t.Errorf("expected active_sessions 4, got %d", resp.Metrics.ActiveSessions)
	}
	if resp.Metrics.ConnectedDevices != 4 {
		t.Errorf("expected connected_devices 4, got %d", resp.Metrics.ConnectedDevices)
	}
	if resp.EventsTotal.Attach != 5 {
		t.Errorf("expected attach 5, got %d", resp.EventsTotal.Attach)
	}
	if resp.EventsTotal.CellHandover != 3 {
		t.Errorf("expected cell_handover 3, got %d", resp.EventsTotal.CellHandover)
	}
	if resp.EventsTotal.Detach != 2 {
		t.Errorf("expected detach 2, got %d", resp.EventsTotal.Detach)
	}
	if resp.EventsTotal.StaleDisconnect != 1 {
		t.Errorf("expected stale_disconnect 1, got %d", resp.EventsTotal.StaleDisconnect)
	}
}
