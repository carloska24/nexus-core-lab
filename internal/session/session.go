package session

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

// Status representa o estado da sessão no ciclo de conexão de rede.
type Status string

const (
	StatusConnected    Status = "CONNECTED"
	StatusDisconnected Status = "DISCONNECTED"
)

const (
	DisconnectReasonVoluntary = "VOLUNTARY_DETACH"
	DisconnectReasonStale     = "STALE_DISCONNECT"
)

var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionNotConnected = errors.New("session is not connected")
	ErrMissingDeviceID     = errors.New("device_id is required")
	ErrMissingSubscriberID = errors.New("subscriber_id is required")
	ErrMissingCellID       = errors.New("cell_id is required")
	ErrMissingIPAddress    = errors.New("ip_address is required")
)

// Session representa a entidade de domínio de uma sessão de conectividade.
type Session struct {
	ID               string     `json:"id"`
	DeviceID         string     `json:"device_id"`
	SubscriberID     string     `json:"subscriber_id"`
	CellID           string     `json:"cell_id"`
	IPAddress        string     `json:"ip_address"`
	Status           Status     `json:"status"`
	DisconnectReason string     `json:"disconnect_reason,omitempty"`
	AttachedAt       time.Time  `json:"attached_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ClosedAt         *time.Time `json:"closed_at,omitempty"`
}

// New instancia uma nova entidade Session no estado CONNECTED com identificador UUID v4.
func New(deviceID, subscriberID, cellID, ipAddress string) (*Session, error) {
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}
	if subscriberID == "" {
		return nil, ErrMissingSubscriberID
	}
	if cellID == "" {
		return nil, ErrMissingCellID
	}
	if ipAddress == "" {
		return nil, ErrMissingIPAddress
	}

	id, err := generateUUIDv4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session id: %w", err)
	}

	now := time.Now().UTC()
	return &Session{
		ID:           id,
		DeviceID:     deviceID,
		SubscriberID: subscriberID,
		CellID:       cellID,
		IPAddress:    ipAddress,
		Status:       StatusConnected,
		AttachedAt:   now,
		UpdatedAt:    now,
	}, nil
}

// Handover transiciona a sessão ativa para uma nova célula sem alterar o IP alocado.
func (s *Session) Handover(targetCellID string) error {
	if s.Status != StatusConnected {
		return ErrSessionNotConnected
	}
	if s.CellID == targetCellID {
		return nil // no-op idempotente
	}

	s.CellID = targetCellID
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// Detach encerra a sessão ativa, registrando o motivo e timestamp de fechamento.
func (s *Session) Detach(reason string) {
	if s.Status == StatusDisconnected {
		return
	}

	now := time.Now().UTC()
	s.Status = StatusDisconnected
	s.DisconnectReason = reason
	s.ClosedAt = &now
	s.UpdatedAt = now
}

// generateUUIDv4 gera um identificador único universal RFC 4122 sem dependências externas.
func generateUUIDv4() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", err
	}

	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16],
	), nil
}
