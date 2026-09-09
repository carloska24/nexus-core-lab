package telemetry

import "time"

// EventType define os tipos de eventos suportados pelo pipeline assíncrono de telemetria.
type EventType string

const (
	EventAttach          EventType = "ATTACH"
	EventCellHandover    EventType = "CELL_HANDOVER"
	EventDetach          EventType = "DETACH"
	EventStaleDisconnect EventType = "STALE_DISCONNECT"
)

// Event encapsula os dados emitidos por transições de ciclo de vida de conectividade.
type Event struct {
	Type         EventType `json:"type"`
	SessionID    string    `json:"session_id"`
	DeviceID     string    `json:"device_id"`
	SubscriberID string    `json:"subscriber_id"`
	CellID       string    `json:"cell_id"`
	Timestamp    time.Time `json:"timestamp"`
}
