package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRecentContract(t *testing.T) {
	w := NewWorker(256)
	mux := http.NewServeMux()
	w.RegisterRecentRoutes(mux)
	empty := httptest.NewRecorder()
	mux.ServeHTTP(empty, httptest.NewRequest("GET", "/api/v1/events/recent", nil))
	if empty.Code != 200 || empty.Body.String() != "[]\n" {
		t.Fatalf("empty: %d %s", empty.Code, empty.Body.String())
	}
	w.Start()
	types := []EventType{EventAttach, EventCellHandover, EventDetach, EventStaleDisconnect}
	// Equal timestamps deliberately do not determine the consumption order.
	stamp := time.Now().UTC()
	for i := 0; i < 104; i++ {
		w.Emit(Event{Type: types[i%4], DeviceID: strconv.Itoa(i), Timestamp: stamp})
	}
	if err := w.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	snapshot := w.Recent()
	if len(snapshot) != 100 {
		t.Fatalf("capacity: %d", len(snapshot))
	}
	for i, e := range snapshot {
		if e.DeviceID != strconv.Itoa(103-i) || e.ID != strconv.Itoa(104-i) {
			t.Fatalf("order/eviction: %+v", e)
		}
	}
	snapshot[0].DeviceID = "mutated"
	if w.Recent()[0].DeviceID != "103" {
		t.Fatal("snapshot aliases retention")
	}
	if len(NewWorker().Recent()) != 0 {
		t.Fatal("new instance retained history")
	}
	h := NewHandler(w.Metrics(), nil, nil)
	h.RegisterRoutes(mux)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/telemetry", nil))
	var counters TelemetryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &counters); err != nil {
		t.Fatal(err)
	}
	if counters.EventsTotal != (EventsTotalSnapshot{Attach: 26, CellHandover: 26, Detach: 26, StaleDisconnect: 26}) {
		t.Fatalf("counters: %+v", counters.EventsTotal)
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/events/recent", nil))
	var events []Event
	if err := json.Unmarshal(response.Body.Bytes(), &events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 100 || events[0].Type != EventStaleDisconnect {
		t.Fatalf("http: %s", response.Body.String())
	}
}

func TestRecentConcurrentSnapshots(t *testing.T) {
	w := NewWorker(4096)
	w.Start()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				w.Emit(Event{Type: EventAttach})
				snapshot := w.Recent()
				if len(snapshot) > 100 {
					t.Error("unbounded")
				}
				if len(snapshot) > 0 {
					snapshot[0].DeviceID = "local"
				}
			}
		}()
	}
	wg.Wait()
	if err := w.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(w.Recent()) != 100 || w.Metrics().attachTotal.Load() != 1600 {
		t.Fatal("events lost or counted twice")
	}
	for _, e := range w.Recent() {
		if e.DeviceID != "" {
			t.Fatal("snapshot mutation escaped")
		}
	}
}
