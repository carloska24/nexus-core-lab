package subscriber

import (
	"context"
)

// ListFilter especifica critérios opcionais para listagem de assinantes.
type ListFilter struct {
	Status Status
}

// Repository define as operações de persistência exigidas pelo domínio Subscriber.
// Esta interface permite desacoplar os casos de uso da infraestrutura de armazenamento,
// viabilizando a implementação In-Memory no M1 e a implementação PostgreSQL no M6 sem refatorar o domínio.
type Repository interface {
	Save(ctx context.Context, sub *Subscriber) error
	Update(ctx context.Context, sub *Subscriber) error
	FindByID(ctx context.Context, id string) (*Subscriber, error)
	FindByIMSI(ctx context.Context, imsi string) (*Subscriber, error)
	List(ctx context.Context, filter ListFilter) ([]*Subscriber, error)
	ExistsByIMSIOrMSISDN(ctx context.Context, imsi, msisdn string) (bool, error)
}
