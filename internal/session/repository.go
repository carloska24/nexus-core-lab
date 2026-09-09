package session

import (
	"context"
)

// Repository define as operações de persistência necessárias para o domínio Session.
type Repository interface {
	Save(ctx context.Context, s *Session) error
	Update(ctx context.Context, s *Session) error
	FindByID(ctx context.Context, id string) (*Session, error)
	FindActiveByDevice(ctx context.Context, deviceID string) (*Session, error)
	SetActive(ctx context.Context, deviceID, sessionID string) error
	ClearActive(ctx context.Context, deviceID, sessionID string) error
}
