package session

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrDuplicateSession = errors.New("session already exists")
)

// MemoryRepository implementa Repository em memória RAM utilizando sync.RWMutex.
// É thread-safe e realiza cópias defensivas dos dados.
type MemoryRepository struct {
	mu             sync.RWMutex
	byID           map[string]*Session
	activeByDevice map[string]string // deviceID -> sessionID
}

// NewMemoryRepository instancia um repositório thread-safe em memória para Session.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:           make(map[string]*Session),
		activeByDevice: make(map[string]string),
	}
}

// Save persiste uma nova sessão.
func (r *MemoryRepository) Save(ctx context.Context, s *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[s.ID]; exists {
		return ErrDuplicateSession
	}

	r.byID[s.ID] = copySession(s)
	return nil
}

// Update atualiza uma sessão existente e sincroniza o mapa de sessões ativas por dispositivo.
func (r *MemoryRepository) Update(ctx context.Context, s *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[s.ID]; !exists {
		return ErrSessionNotFound
	}

	r.byID[s.ID] = copySession(s)

	if s.Status == StatusDisconnected {
		if activeID, exists := r.activeByDevice[s.DeviceID]; exists && activeID == s.ID {
			delete(r.activeByDevice, s.DeviceID)
		}
	}

	return nil
}

// FindByID recupera uma cópia da sessão por seu UUID.
func (r *MemoryRepository) FindByID(ctx context.Context, id string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, exists := r.byID[id]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return copySession(s), nil
}

// FindActiveByDevice recupera a sessão atualmente ativa vinculada a um dispositivo.
func (r *MemoryRepository) FindActiveByDevice(ctx context.Context, deviceID string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessionID, exists := r.activeByDevice[deviceID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	s, exists := r.byID[sessionID]
	if !exists || s.Status != StatusConnected {
		return nil, ErrSessionNotFound
	}

	return copySession(s), nil
}

// AttachSession persiste a nova sessão e substitui atomicamente qualquer sessão ativa anterior do mesmo dispositivo.
func (r *MemoryRepository) AttachSession(ctx context.Context, newSession *Session) (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[newSession.ID]; exists {
		return nil, ErrDuplicateSession
	}

	var staleSession *Session
	if oldSessionID, exists := r.activeByDevice[newSession.DeviceID]; exists {
		if old, ok := r.byID[oldSessionID]; ok && old.Status == StatusConnected {
			old.Detach(DisconnectReasonStale)
			staleSession = copySession(old)
		}
	}

	r.byID[newSession.ID] = copySession(newSession)
	r.activeByDevice[newSession.DeviceID] = newSession.ID

	return staleSession, nil
}

// ActiveCount retorna o número total de sessões ativas registradas no repositório.
func (r *MemoryRepository) ActiveCount(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.activeByDevice), nil
}

func copySession(s *Session) *Session {
	if s == nil {
		return nil
	}

	clone := *s
	if s.ClosedAt != nil {
		closedAtCopy := *s.ClosedAt
		clone.ClosedAt = &closedAtCopy
	}
	return &clone
}
