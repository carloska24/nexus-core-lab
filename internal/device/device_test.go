package device

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

// mockSubscriberChecker implementa SubscriberChecker em memória para testes unitários.
type mockSubscriberChecker struct {
	subscribers map[string]string // id -> status
}

func (m *mockSubscriberChecker) CheckSubscriberActive(ctx context.Context, subscriberID string) error {
	status, exists := m.subscribers[subscriberID]
	if !exists {
		return ErrSubscriberNotFound
	}
	if status != "ACTIVE" {
		return ErrSubscriberNotActive
	}
	return nil
}

func TestDeviceEntityValidations(t *testing.T) {
	t.Run("valid creation with LTE and 5G", func(t *testing.T) {
		devLTE, err := New("sub-123", "356938035643803", TechLTE)
		if err != nil {
			t.Fatalf("expected no error for LTE, got %v", err)
		}
		if devLTE.Technology != TechLTE || devLTE.Status != StatusRegistered {
			t.Errorf("unexpected device state: %+v", devLTE)
		}

		dev5G, err := New("sub-123", "864275039281745", Tech5G)
		if err != nil {
			t.Fatalf("expected no error for 5G, got %v", err)
		}
		if dev5G.Technology != Tech5G || dev5G.Status != StatusRegistered {
			t.Errorf("unexpected device state: %+v", dev5G)
		}
	})

	t.Run("invalid IMEI formats", func(t *testing.T) {
		invalidIMEIs := []string{
			"123",              // curto
			"1234567890123456", // 16 dígitos
			"35693803564380A",  // contém letra
			"",                 // vazio
		}

		for _, imei := range invalidIMEIs {
			_, err := New("sub-123", imei, TechLTE)
			if !errors.Is(err, ErrInvalidIMEI) {
				t.Errorf("for IMEI %q, expected ErrInvalidIMEI, got %v", imei, err)
			}
		}
	})

	t.Run("invalid technology", func(t *testing.T) {
		invalidTechs := []Technology{"3G", "WIFI", "EDGE", ""}
		for _, tech := range invalidTechs {
			_, err := New("sub-123", "356938035643803", tech)
			if !errors.Is(err, ErrInvalidTechnology) {
				t.Errorf("for tech %q, expected ErrInvalidTechnology, got %v", tech, err)
			}
		}
	})

	t.Run("missing subscriber id", func(t *testing.T) {
		_, err := New("", "356938035643803", TechLTE)
		if !errors.Is(err, ErrMissingSubscriber) {
			t.Errorf("expected ErrMissingSubscriber, got %v", err)
		}
	})
}

func TestDeviceRegistrationRules(t *testing.T) {
	mockChecker := &mockSubscriberChecker{
		subscribers: map[string]string{
			"active-sub":    "ACTIVE",
			"pending-sub":   "PENDING_ACTIVATION",
			"suspended-sub": "SUSPENDED",
			"deact-sub":     "DEACTIVATED",
		},
	}

	repo := NewMemoryRepository()
	service := NewService(repo, mockChecker)
	ctx := context.Background()

	t.Run("successful registration for ACTIVE subscriber", func(t *testing.T) {
		req := RegisterRequest{
			SubscriberID: "active-sub",
			IMEI:         "356938035643801",
			Technology:   Tech5G,
		}

		dev, err := service.Register(ctx, req)
		if err != nil {
			t.Fatalf("expected registration to succeed, got %v", err)
		}

		if dev.ID == "" || dev.Status != StatusRegistered {
			t.Errorf("unexpected registered device: %+v", dev)
		}
	})

	t.Run("rejection for nonexistent subscriber", func(t *testing.T) {
		req := RegisterRequest{
			SubscriberID: "non-existent-sub",
			IMEI:         "356938035643802",
			Technology:   TechLTE,
		}

		_, err := service.Register(ctx, req)
		if !errors.Is(err, ErrSubscriberNotFound) {
			t.Errorf("expected ErrSubscriberNotFound, got %v", err)
		}
	})

	t.Run("rejection for PENDING_ACTIVATION subscriber", func(t *testing.T) {
		req := RegisterRequest{
			SubscriberID: "pending-sub",
			IMEI:         "356938035643803",
			Technology:   TechLTE,
		}

		_, err := service.Register(ctx, req)
		if !errors.Is(err, ErrSubscriberNotActive) {
			t.Errorf("expected ErrSubscriberNotActive, got %v", err)
		}
	})

	t.Run("rejection for SUSPENDED subscriber", func(t *testing.T) {
		req := RegisterRequest{
			SubscriberID: "suspended-sub",
			IMEI:         "356938035643804",
			Technology:   Tech5G,
		}

		_, err := service.Register(ctx, req)
		if !errors.Is(err, ErrSubscriberNotActive) {
			t.Errorf("expected ErrSubscriberNotActive, got %v", err)
		}
	})

	t.Run("rejection for DEACTIVATED subscriber", func(t *testing.T) {
		req := RegisterRequest{
			SubscriberID: "deact-sub",
			IMEI:         "356938035643805",
			Technology:   TechLTE,
		}

		_, err := service.Register(ctx, req)
		if !errors.Is(err, ErrSubscriberNotActive) {
			t.Errorf("expected ErrSubscriberNotActive, got %v", err)
		}
	})

	t.Run("rejection for duplicate IMEI", func(t *testing.T) {
		req := RegisterRequest{
			SubscriberID: "active-sub",
			IMEI:         "356938035643801", // já registrado no primeiro subteste
			Technology:   TechLTE,
		}

		_, err := service.Register(ctx, req)
		if !errors.Is(err, ErrDuplicateIMEI) {
			t.Errorf("expected ErrDuplicateIMEI, got %v", err)
		}
	})

	t.Run("query by ID and list by Subscriber", func(t *testing.T) {
		// Registra um segundo dispositivo para active-sub
		req2 := RegisterRequest{
			SubscriberID: "active-sub",
			IMEI:         "356938035643899",
			Technology:   TechLTE,
		}
		dev2, err := service.Register(ctx, req2)
		if err != nil {
			t.Fatalf("failed to register second device: %v", err)
		}

		// Busca por ID
		found, err := service.FindByID(ctx, dev2.ID)
		if err != nil || found.IMEI != req2.IMEI {
			t.Errorf("failed to find device by ID: %v, %+v", err, found)
		}

		// Listagem por assinante (deve conter os 2 dispositivos registrados)
		list, err := service.ListBySubscriber(ctx, "active-sub")
		if err != nil {
			t.Fatalf("failed to list by subscriber: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 devices for active-sub, got %d", len(list))
		}
	})
}

func TestDeviceRepositoryConcurrency(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	var wg sync.WaitGroup
	workers := 20
	devsPerWorker := 10

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			subID := fmt.Sprintf("sub-worker-%02d", workerID)
			for i := 0; i < devsPerWorker; i++ {
				imei := fmt.Sprintf("35693%02d%08d", workerID, i)
				dev, err := New(subID, imei, TechLTE)
				if err != nil {
					t.Errorf("worker %d failed to create device: %v", workerID, err)
					return
				}

				if err := repo.Save(ctx, dev); err != nil {
					t.Errorf("worker %d failed to save device: %v", workerID, err)
					return
				}

				found, err := repo.FindByIMEI(ctx, imei)
				if err != nil || found.ID != dev.ID {
					t.Errorf("worker %d failed to find device by IMEI: %v", workerID, err)
					return
				}
			}
		}(w)
	}

	wg.Wait()

	// Valida listagem para cada worker
	for w := 0; w < workers; w++ {
		subID := fmt.Sprintf("sub-worker-%02d", w)
		list, err := repo.ListBySubscriber(ctx, subID)
		if err != nil {
			t.Fatalf("failed to list devices for %s: %v", subID, err)
		}
		if len(list) != devsPerWorker {
			t.Errorf("expected %d devices for %s, got %d", devsPerWorker, subID, len(list))
		}
	}
}
