package session

import "context"

// SessionEventType representa o tipo semântico tipado de eventos emitidos pelo domínio Session.
type SessionEventType string

const (
	SessionEventAttach          SessionEventType = "ATTACH"
	SessionEventCellHandover    SessionEventType = "CELL_HANDOVER"
	SessionEventDetach          SessionEventType = "DETACH"
	SessionEventStaleDisconnect SessionEventType = "STALE_DISCONNECT"
)

// EventEmitter define o contrato consumidor tipado exigido pelo pacote session
// para despachar notificações de eventos de ciclo de vida sem acoplamento a infraestrutura.
type EventEmitter interface {
	EmitSessionEvent(
		ctx context.Context,
		eventType SessionEventType,
		s *Session,
	)
}
