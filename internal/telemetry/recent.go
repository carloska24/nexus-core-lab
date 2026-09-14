package telemetry

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const RecentCapacity = 100

// retain shares the existing consumer: only events counted by this worker are retained.
// Order is consumption order, not wall-clock timestamp (which may tie or move backwards).
func (w *Worker) retain(evt Event) {
	w.recentMu.Lock()
	defer w.recentMu.Unlock()
	w.sequence++
	// Identity is scoped to this API execution; this is not persistent history.
	evt.ID = strconv.FormatUint(w.sequence, 10)
	if len(w.recent) == RecentCapacity {
		copy(w.recent, w.recent[1:])
		w.recent = w.recent[:RecentCapacity-1]
	}
	w.recent = append(w.recent, evt)
}

// Recent returns a defensive snapshot, newest consumed first. New workers start empty.
func (w *Worker) Recent() []Event {
	w.recentMu.RLock()
	defer w.recentMu.RUnlock()
	snapshot := make([]Event, len(w.recent))
	for i, evt := range w.recent {
		snapshot[len(snapshot)-1-i] = evt
	}
	return snapshot
}

func (w *Worker) RegisterRecentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/events/recent", func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		res.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(res).Encode(w.Recent())
	})
}
