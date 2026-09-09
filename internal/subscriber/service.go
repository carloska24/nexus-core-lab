package subscriber

import (
	"context"
	"fmt"
)

// Service coordena os casos de uso de negócio do domínio Subscriber.
type Service struct {
	repo Repository
}

// NewService cria uma nova instância do serviço de assinantes.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ProvisionRequest representa os dados necessários para provisionar um novo assinante.
type ProvisionRequest struct {
	IMSI   string `json:"imsi"`
	MSISDN string `json:"msisdn"`
}

// Provision executa o provisionamento de um novo assinante garantindo unicidade estrita.
func (s *Service) Provision(ctx context.Context, req ProvisionRequest) (*Subscriber, error) {
	sub, err := New(req.IMSI, req.MSISDN)
	if err != nil {
		return nil, err
	}

	exists, err := s.repo.ExistsByIMSIOrMSISDN(ctx, req.IMSI, req.MSISDN)
	if err != nil {
		return nil, fmt.Errorf("failed to check subscriber uniqueness: %w", err)
	}
	if exists {
		return nil, ErrDuplicateIMSI
	}

	if err := s.repo.Save(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to save subscriber: %w", err)
	}

	return sub, nil
}

// Activate transiciona o assinante para o estado ACTIVE.
func (s *Service) Activate(ctx context.Context, id string) (*Subscriber, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := sub.Activate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscriber: %w", err)
	}

	return sub, nil
}

// Suspend bloqueia temporariamente o assinante registrando o motivo.
func (s *Service) Suspend(ctx context.Context, id string, reason string) (*Subscriber, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := sub.Suspend(reason); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscriber: %w", err)
	}

	return sub, nil
}

// Deactivate encerra permanentemente o contrato do assinante (estado terminal).
func (s *Service) Deactivate(ctx context.Context, id string, reason string) (*Subscriber, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := sub.Deactivate(reason); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update subscriber: %w", err)
	}

	return sub, nil
}

// FindByID recupera um assinante pelo ID interno (UUID).
func (s *Service) FindByID(ctx context.Context, id string) (*Subscriber, error) {
	return s.repo.FindByID(ctx, id)
}

// FindByIMSI recupera um assinante pelo seu IMSI de rede.
func (s *Service) FindByIMSI(ctx context.Context, imsi string) (*Subscriber, error) {
	return s.repo.FindByIMSI(ctx, imsi)
}

// List lista assinantes cadastrados, opcionalmente filtrados por Status.
func (s *Service) List(ctx context.Context, status Status) ([]*Subscriber, error) {
	return s.repo.List(ctx, ListFilter{Status: status})
}
