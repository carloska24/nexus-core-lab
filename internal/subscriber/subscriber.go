package subscriber

import (
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Status representa o estado no ciclo de vida de um assinante.
type Status string

const (
	StatusPendingActivation Status = "PENDING_ACTIVATION"
	StatusActive            Status = "ACTIVE"
	StatusSuspended         Status = "SUSPENDED"
	StatusDeactivated       Status = "DEACTIVATED"
)

var (
	ErrInvalidIMSI        = errors.New("invalid IMSI: must contain exactly 15 numeric digits")
	ErrInvalidMSISDN      = errors.New("invalid MSISDN: must contain between 10 and 15 digits, optionally prefixed with '+'")
	ErrInvalidTransition  = errors.New("invalid state transition")
	ErrAlreadyDeactivated = errors.New("subscriber is already deactivated; state is terminal")
	ErrSubscriberNotFound = errors.New("subscriber not found")
	ErrDuplicateIMSI      = errors.New("subscriber with this IMSI already exists")
	ErrDuplicateMSISDN    = errors.New("subscriber with this MSISDN already exists")
)

var (
	imsiRegex   = regexp.MustCompile(`^\d{15}$`)
	msisdnRegex = regexp.MustCompile(`^\+?[1-9]\d{9,14}$`)
)

// Subscriber representa a entidade de domínio do assinante de rede.
type Subscriber struct {
	ID                 string    `json:"id"`
	IMSI               string    `json:"imsi"`
	MSISDN             string    `json:"msisdn"`
	Status             Status    `json:"status"`
	SuspensionReason   string    `json:"suspension_reason,omitempty"`
	DeactivationReason string    `json:"deactivation_reason,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// New cria um novo assinante com status inicial PENDING_ACTIVATION.
func New(imsi, msisdn string) (*Subscriber, error) {
	if !imsiRegex.MatchString(imsi) {
		return nil, ErrInvalidIMSI
	}
	if !msisdnRegex.MatchString(msisdn) {
		return nil, ErrInvalidMSISDN
	}

	id, err := generateUUIDv4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate subscriber id: %w", err)
	}

	now := time.Now().UTC()
	return &Subscriber{
		ID:        id,
		IMSI:      imsi,
		MSISDN:    msisdn,
		Status:    StatusPendingActivation,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Activate transiciona o assinante para o estado ACTIVE.
// Permitido a partir de PENDING_ACTIVATION ou SUSPENDED.
func (s *Subscriber) Activate() error {
	switch s.Status {
	case StatusDeactivated:
		return ErrAlreadyDeactivated
	case StatusActive:
		return nil // idempotente
	case StatusPendingActivation, StatusSuspended:
		s.Status = StatusActive
		s.SuspensionReason = ""
		s.UpdatedAt = time.Now().UTC()
		return nil
	default:
		return fmt.Errorf("%w: cannot activate from %s", ErrInvalidTransition, s.Status)
	}
}

// Suspend transiciona o assinante para o estado SUSPENDED.
// Permitido exclusivamente a partir de ACTIVE.
func (s *Subscriber) Suspend(reason string) error {
	switch s.Status {
	case StatusDeactivated:
		return ErrAlreadyDeactivated
	case StatusSuspended:
		s.SuspensionReason = reason
		s.UpdatedAt = time.Now().UTC()
		return nil // idempotente com atualização de motivo
	case StatusActive:
		s.Status = StatusSuspended
		s.SuspensionReason = reason
		s.UpdatedAt = time.Now().UTC()
		return nil
	default:
		return fmt.Errorf("%w: cannot suspend from %s", ErrInvalidTransition, s.Status)
	}
}

// Deactivate transiciona o assinante para o estado terminal DEACTIVATED.
// Permitido a partir de PENDING_ACTIVATION, ACTIVE ou SUSPENDED.
func (s *Subscriber) Deactivate(reason string) error {
	if s.Status == StatusDeactivated {
		return ErrAlreadyDeactivated
	}

	s.Status = StatusDeactivated
	s.DeactivationReason = reason
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// generateUUIDv4 gera um identificador único universal em conformidade com RFC 4122.
func generateUUIDv4() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", err
	}

	// Define versão 4
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Define variante RFC 4122
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16],
	), nil
}
