package device

import (
	"context"
)

// Repository define as operações de persistência necessárias para o domínio Device.
// A interface viabiliza o repositório In-Memory no M2 e posterior persistência em PostgreSQL no M6.
type Repository interface {
	Save(ctx context.Context, dev *Device) error
	FindByID(ctx context.Context, id string) (*Device, error)
	FindByIMEI(ctx context.Context, imei string) (*Device, error)
	ListBySubscriber(ctx context.Context, subscriberID string) ([]*Device, error)
}
