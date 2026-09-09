package device

import (
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Technology representa a tecnologia de rádio suportada pelo equipamento simulado.
type Technology string

const (
	TechLTE Technology = "LTE"
	Tech5G  Technology = "5G"
)

// Status representa o estado cadastral do dispositivo na rede.
type Status string

const (
	StatusRegistered Status = "REGISTERED"
	StatusInactive   Status = "INACTIVE"
)

var (
	ErrInvalidIMEI       = errors.New("invalid IMEI: must contain exactly 15 numeric digits")
	ErrInvalidTechnology = errors.New("invalid technology: must be either LTE or 5G")
	ErrMissingSubscriber = errors.New("subscriber_id is required")
	ErrDeviceNotFound    = errors.New("device not found")
	ErrDuplicateIMEI     = errors.New("device with this IMEI already exists")
)

var imeiRegex = regexp.MustCompile(`^\d{15}$`)

// Device representa a entidade de domínio do equipamento de rede.
type Device struct {
	ID           string     `json:"id"`
	SubscriberID string     `json:"subscriber_id"`
	IMEI         string     `json:"imei"`
	Technology   Technology `json:"technology"`
	Status       Status     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// New cria uma nova entidade Device com validações estritas de IMEI e tecnologia.
func New(subscriberID, imei string, tech Technology) (*Device, error) {
	if subscriberID == "" {
		return nil, ErrMissingSubscriber
	}
	if !imeiRegex.MatchString(imei) {
		return nil, ErrInvalidIMEI
	}
	if tech != TechLTE && tech != Tech5G {
		return nil, ErrInvalidTechnology
	}

	id, err := generateUUIDv4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device id: %w", err)
	}

	now := time.Now().UTC()
	return &Device{
		ID:           id,
		SubscriberID: subscriberID,
		IMEI:         imei,
		Technology:   tech,
		Status:       StatusRegistered,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// generateUUIDv4 gera um identificador único universal em conformidade com RFC 4122.
func generateUUIDv4() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", err
	}

	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16],
	), nil
}
