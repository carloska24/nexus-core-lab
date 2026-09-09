package subscriber

import (
	"context"
	"sync"
)

// MemoryRepository implementa Repository em memória RAM utilizando sync.RWMutex.
// É seguro para acesso concorrente e realiza cópias defensivas dos dados.
type MemoryRepository struct {
	mu       sync.RWMutex
	byID     map[string]*Subscriber
	byIMSI   map[string]string // imsi -> id
	byMSISDN map[string]string // msisdn -> id
}

// NewMemoryRepository instancia um repositório thread-safe em memória.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:     make(map[string]*Subscriber),
		byIMSI:   make(map[string]string),
		byMSISDN: make(map[string]string),
	}
}

// Save persiste um novo assinante garantindo unicidade estrita de IMSI e MSISDN.
func (r *MemoryRepository) Save(ctx context.Context, sub *Subscriber) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[sub.ID]; exists {
		return ErrDuplicateIMSI
	}
	if _, exists := r.byIMSI[sub.IMSI]; exists {
		return ErrDuplicateIMSI
	}
	if _, exists := r.byMSISDN[sub.MSISDN]; exists {
		return ErrDuplicateMSISDN
	}

	clone := copySubscriber(sub)
	r.byID[sub.ID] = clone
	r.byIMSI[sub.IMSI] = sub.ID
	r.byMSISDN[sub.MSISDN] = sub.ID

	return nil
}

// Update atualiza os dados mutáveis de um assinante existente (status, motivos, timestamps).
func (r *MemoryRepository) Update(ctx context.Context, sub *Subscriber) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.byID[sub.ID]
	if !exists {
		return ErrSubscriberNotFound
	}

	// Não permitimos alteração de IMSI ou MSISDN via Update
	clone := copySubscriber(sub)
	clone.IMSI = existing.IMSI
	clone.MSISDN = existing.MSISDN

	r.byID[sub.ID] = clone
	return nil
}

// FindByID busca um assinante pelo ID interno (UUID).
func (r *MemoryRepository) FindByID(ctx context.Context, id string) (*Subscriber, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sub, exists := r.byID[id]
	if !exists {
		return nil, ErrSubscriberNotFound
	}

	return copySubscriber(sub), nil
}

// FindByIMSI busca um assinante pelo seu IMSI de 15 dígitos.
func (r *MemoryRepository) FindByIMSI(ctx context.Context, imsi string) (*Subscriber, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.byIMSI[imsi]
	if !exists {
		return nil, ErrSubscriberNotFound
	}

	sub, exists := r.byID[id]
	if !exists {
		return nil, ErrSubscriberNotFound
	}

	return copySubscriber(sub), nil
}

// List retorna os assinantes cadastrados, opcionalmente filtrados por Status.
func (r *MemoryRepository) List(ctx context.Context, filter ListFilter) ([]*Subscriber, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Subscriber, 0, len(r.byID))
	for _, sub := range r.byID {
		if filter.Status != "" && sub.Status != filter.Status {
			continue
		}
		result = append(result, copySubscriber(sub))
	}

	return result, nil
}

// ExistsByIMSIOrMSISDN verifica se já existe algum assinante com o IMSI ou MSISDN fornecido.
func (r *MemoryRepository) ExistsByIMSIOrMSISDN(ctx context.Context, imsi, msisdn string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.byIMSI[imsi]; exists {
		return true, nil
	}
	if _, exists := r.byMSISDN[msisdn]; exists {
		return true, nil
	}

	return false, nil
}

// copySubscriber cria uma cópia defensiva para garantir imutabilidade fora do lock do repositório.
func copySubscriber(s *Subscriber) *Subscriber {
	if s == nil {
		return nil
	}
	clone := *s
	return &clone
}
