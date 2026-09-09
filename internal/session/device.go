package session

import (
	"context"
	"errors"
)

var (
	ErrDeviceNotFound    = errors.New("device not found")
	ErrDeviceNotEligible = errors.New("device is not eligible for attach")
)

// DeviceInfo carrega os dados essenciais retornados pelo contrato de verificação de dispositivo.
type DeviceInfo struct {
	DeviceID     string
	SubscriberID string
}

// DeviceChecker define o contrato consumidor mínimo exigido pelo pacote session
// para validar a elegibilidade do equipamento antes do Attach de rede.
type DeviceChecker interface {
	CheckDeviceAttachable(ctx context.Context, deviceID string) (*DeviceInfo, error)
}
