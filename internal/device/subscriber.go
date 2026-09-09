package device

import (
	"context"
	"errors"
)

var (
	ErrSubscriberNotFound  = errors.New("subscriber not found")
	ErrSubscriberNotActive = errors.New("subscriber is not active")
)

// SubscriberChecker define o contrato consumidor mínimo exigido pelo pacote device
// para validar a elegibilidade do assinante antes do registro de um equipamento.
// Retorna ErrSubscriberNotFound se o assinante não existir ou ErrSubscriberNotActive se não estiver ativo.
type SubscriberChecker interface {
	CheckSubscriberActive(ctx context.Context, subscriberID string) error
}
