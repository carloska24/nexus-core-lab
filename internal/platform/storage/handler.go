package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

const pingTimeout = 2 * time.Second

type Snapshot struct {
	Mode               string `json:"mode"`
	DatabaseConfigured bool   `json:"database_configured"`
	DatabaseStatus     string `json:"database_status"`
}

// Handler borrows the application's DB handle; it never opens or closes a pool.
type Handler struct{ ping func(context.Context) error }

func NewHandler(db *sql.DB) *Handler {
	h := &Handler{}
	if db != nil {
		h.ping = db.PingContext
	}
	return h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/system/storage", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	s := Snapshot{Mode: "MEMORY", DatabaseStatus: "NOT_APPLICABLE"}
	if h.ping != nil {
		s.Mode, s.DatabaseConfigured, s.DatabaseStatus = "POSTGRESQL", true, "AVAILABLE"
		ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
		defer cancel()
		if err := h.ping(ctx); err != nil {
			s.DatabaseStatus = "UNAVAILABLE"
		}
	}
	// Connection errors deliberately never cross the HTTP or logging boundary.
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s)
}
