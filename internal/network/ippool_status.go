package network

import (
	"encoding/json"
	"net/http"
)

// IPPoolSnapshot is an atomic view of this allocator, not a count of Sessions.
type IPPoolSnapshot struct {
	CIDR               string  `json:"cidr"`
	Capacity           int     `json:"capacity"`
	Allocated          int     `json:"allocated"`
	Available          int     `json:"available"`
	UtilizationPercent float64 `json:"utilization_percent"`
}

// Snapshot is read-only. Released hosts are recycled and warm-up never skips holes,
// so every usable address absent from allocated remains eligible for Allocate.
func (p *IPPool) Snapshot() IPPoolSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	capacity := int(maxHost - minHost + 1)
	allocated := len(p.allocated)
	return IPPoolSnapshot{CIDR: baseIPv4Net, Capacity: capacity, Allocated: allocated, Available: capacity - allocated, UtilizationPercent: float64(allocated) / float64(capacity) * 100}
}

type IPPoolHandler struct{ pool *IPPool }

func NewIPPoolHandler(pool *IPPool) *IPPoolHandler { return &IPPoolHandler{pool: pool} }
func (h *IPPoolHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/network/ip-pool", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(h.pool.Snapshot())
	})
}
