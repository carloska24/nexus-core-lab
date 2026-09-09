package device

import (
	"context"
	"fmt"
)

// Service coordena os casos de uso de negócio do domínio Device.
type Service struct {
	repo       Repository
	subChecker SubscriberChecker
}

// NewService cria uma nova instância do serviço de dispositivos.
func NewService(repo Repository, subChecker SubscriberChecker) *Service {
	return &Service{
		repo:       repo,
		subChecker: subChecker,
	}
}

// RegisterRequest representa os dados necessários para registrar um equipamento no laboratório.
type RegisterRequest struct {
	SubscriberID string     `json:"subscriber_id"`
	IMEI         string     `json:"imei"`
	Technology   Technology `json:"technology"`
}

// Register valida o assinante e a unicidade de hardware antes de registrar o dispositivo.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*Device, error) {
	dev, err := New(req.SubscriberID, req.IMEI, req.Technology)
	if err != nil {
		return nil, err
	}

	// Valida se o assinante existe e está ativo via contrato consumidor
	if err := s.subChecker.CheckSubscriberActive(ctx, req.SubscriberID); err != nil {
		return nil, err
	}

	// Validação de unicidade estrita de IMEI
	if _, err := s.repo.FindByIMEI(ctx, req.IMEI); err == nil {
		return nil, ErrDuplicateIMEI
	}

	if err := s.repo.Save(ctx, dev); err != nil {
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	return dev, nil
}

// FindByID recupera um dispositivo pelo seu identificador UUID.
func (s *Service) FindByID(ctx context.Context, id string) (*Device, error) {
	return s.repo.FindByID(ctx, id)
}

// ListBySubscriber lista todos os equipamentos registrados para um assinante.
func (s *Service) ListBySubscriber(ctx context.Context, subscriberID string) ([]*Device, error) {
	if subscriberID == "" {
		return nil, ErrMissingSubscriber
	}
	return s.repo.ListBySubscriber(ctx, subscriberID)
}
