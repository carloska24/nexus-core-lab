package telemetry

import (
	"context"
	"sync/atomic"
	"time"
)

// ActiveSessionProvider define o contrato consumidor mínimo para recuperar o estado atual
// de sessões ativas diretamente da fonte de verdade (Session Repository).
type ActiveSessionProvider interface {
	ActiveCount(ctx context.Context) (int, error)
}

// Metrics armazena os contadores assíncronos protegidos por operações atômicas.
type Metrics struct {
	attachTotal          atomic.Uint64
	cellHandoverTotal    atomic.Uint64
	detachTotal          atomic.Uint64
	staleDisconnectTotal atomic.Uint64
	droppedEventsTotal   atomic.Uint64
}

// MetricsSnapshot reflete os gauges de estado atual e contadores de saturação.
type MetricsSnapshot struct {
	ActiveSessions     int64  `json:"active_sessions"`
	ConnectedDevices   int64  `json:"connected_devices"`
	DroppedEventsTotal uint64 `json:"dropped_events_total"`
}

// EventsTotalSnapshot reflete a contagem consolidada de eventos consumidos pelo worker.
type EventsTotalSnapshot struct {
	Attach          uint64 `json:"attach"`
	CellHandover    uint64 `json:"cell_handover"`
	Detach          uint64 `json:"detach"`
	StaleDisconnect uint64 `json:"stale_disconnect"`
}

// TelemetryResponse representa o payload JSON retornado pelo endpoint GET /telemetry.
type TelemetryResponse struct {
	Service       string              `json:"service"`
	Timestamp     time.Time           `json:"timestamp"`
	RequestsTotal uint64              `json:"requests_total"`
	Metrics       MetricsSnapshot     `json:"metrics"`
	EventsTotal   EventsTotalSnapshot `json:"events_total"`
}

// Snapshot constrói a resposta consolidada de telemetria lendo contadores atômicos e
// derivando os gauges de estado atual diretamente do provedor de sessões ativas.
func (m *Metrics) Snapshot(ctx context.Context, requestsTotal uint64, activeProvider ActiveSessionProvider) (TelemetryResponse, error) {
	var currentActive int64
	if activeProvider != nil {
		count, err := activeProvider.ActiveCount(ctx)
		if err != nil {
			return TelemetryResponse{}, err
		}
		currentActive = int64(count)
	}

	return TelemetryResponse{
		Service:       "nexus-core-lab",
		Timestamp:     time.Now().UTC(),
		RequestsTotal: requestsTotal,
		Metrics: MetricsSnapshot{
			ActiveSessions:     currentActive,
			ConnectedDevices:   currentActive, // Pela invariante 1:1, cada dispositivo conectado possui 1 sessão ativa
			DroppedEventsTotal: m.droppedEventsTotal.Load(),
		},
		EventsTotal: EventsTotalSnapshot{
			Attach:          m.attachTotal.Load(),
			CellHandover:    m.cellHandoverTotal.Load(),
			Detach:          m.detachTotal.Load(),
			StaleDisconnect: m.staleDisconnectTotal.Load(),
		},
	}, nil
}
