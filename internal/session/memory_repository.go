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

// Update atualiza uma sessão existente.
func (r *MemoryRepository) Update(ctx context.Context, s *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[s.ID]; !exists {
		return ErrSessionNotFound
	}

	r.byID[s.ID] = copySession(s)
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

// SetActive vincula a sessão informada como a sessão ativa do dispositivo.
func (r *MemoryRepository) SetActive(ctx context.Context, deviceID, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.activeByDevice[deviceID] = sessionID
	return nil
}

// ClearActive desvincula a sessão ativa do dispositivo se for a mesma informada.
func (r *MemoryRepository) ClearActive(ctx context.Context, deviceID, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if current, exists := r.activeByDevice[deviceID]; exists && current == sessionID {
		delete(r.activeByDevice, deviceID)
	}
	return nil
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

// ActiveCount retorna o número total de sessões ativas registradas (útil para testes).
func (r *MemoryRepository) ActiveCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.activeByDevice)
}
