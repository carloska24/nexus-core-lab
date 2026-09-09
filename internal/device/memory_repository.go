package device

import (
	"context"
	"sync"
)

// MemoryRepository implementa Repository em memória RAM utilizando sync.RWMutex.
// É thread-safe e realiza cópias defensivas dos dados.
type MemoryRepository struct {
	mu           sync.RWMutex
	byID         map[string]*Device
	byIMEI       map[string]string   // imei -> id
	bySubscriber map[string][]string // subscriber_id -> []id
}

// NewMemoryRepository instancia um repositório thread-safe em memória para Device.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:         make(map[string]*Device),
		byIMEI:       make(map[string]string),
		bySubscriber: make(map[string][]string),
	}
}

// Save persiste um novo dispositivo garantindo unicidade estrita de IMEI.
func (r *MemoryRepository) Save(ctx context.Context, dev *Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[dev.ID]; exists {
		return ErrDuplicateIMEI
	}
	if _, exists := r.byIMEI[dev.IMEI]; exists {
		return ErrDuplicateIMEI
	}

	clone := copyDevice(dev)
	r.byID[dev.ID] = clone
	r.byIMEI[dev.IMEI] = dev.ID
	r.bySubscriber[dev.SubscriberID] = append(r.bySubscriber[dev.SubscriberID], dev.ID)

	return nil
}

// FindByID busca um dispositivo pelo seu ID interno (UUID).
func (r *MemoryRepository) FindByID(ctx context.Context, id string) (*Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dev, exists := r.byID[id]
	if !exists {
		return nil, ErrDeviceNotFound
	}

	return copyDevice(dev), nil
}

// FindByIMEI busca um dispositivo pelo seu IMEI de 15 dígitos.
func (r *MemoryRepository) FindByIMEI(ctx context.Context, imei string) (*Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.byIMEI[imei]
	if !exists {
		return nil, ErrDeviceNotFound
	}

	dev, exists := r.byID[id]
	if !exists {
		return nil, ErrDeviceNotFound
	}

	return copyDevice(dev), nil
}

// ListBySubscriber lista todos os dispositivos vinculados ao assinante informado.
func (r *MemoryRepository) ListBySubscriber(ctx context.Context, subscriberID string) ([]*Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.bySubscriber[subscriberID]
	result := make([]*Device, 0, len(ids))

	for _, id := range ids {
		if dev, exists := r.byID[id]; exists {
			result = append(result, copyDevice(dev))
		}
	}

	return result, nil
}

// copyDevice cria uma cópia defensiva para garantir imutabilidade fora do lock do repositório.
func copyDevice(d *Device) *Device {
	if d == nil {
		return nil
	}
	clone := *d
	return &clone
}
