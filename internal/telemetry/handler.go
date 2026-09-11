package telemetry

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// Handler expõe os dados de telemetria através de endpoints HTTP REST.
type Handler struct {
	metrics         *Metrics
	activeProvider  ActiveSessionProvider
	requestsCounter *atomic.Uint64
}

// NewHandler instancia o handler de telemetria.
func NewHandler(metrics *Metrics, activeProvider ActiveSessionProvider, requestsCounter *atomic.Uint64) *Handler {
	return &Handler{
		metrics:         metrics,
		activeProvider:  activeProvider,
		requestsCounter: requestsCounter,
	}
}

// RegisterRoutes acopla os endpoints de telemetria ao ServeMux informado.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /telemetry", h.handleGetTelemetry)
}

func (h *Handler) handleGetTelemetry(w http.ResponseWriter, r *http.Request) {
	var totalReqs uint64
	if h.requestsCounter != nil {
		totalReqs = h.requestsCounter.Load()
	}

	snapshot, err := h.metrics.Snapshot(r.Context(), totalReqs, h.activeProvider)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "internal_error",
			"message": "failed to retrieve active sessions from storage",
			"code":    "TELEMETRY_STORAGE_ERROR",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(snapshot)
}
