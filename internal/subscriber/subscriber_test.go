package subscriber

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestSubscriberCreation(t *testing.T) {
	t.Run("successful creation with valid IMSI and MSISDN", func(t *testing.T) {
		sub, err := New("724991234567890", "+5519998765432")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if sub.ID == "" {
			t.Error("expected non-empty ID")
		}
		if sub.Status != StatusPendingActivation {
			t.Errorf("expected status %s, got %s", StatusPendingActivation, sub.Status)
		}
		if sub.IMSI != "724991234567890" {
			t.Errorf("expected IMSI 724991234567890, got %s", sub.IMSI)
		}
		if sub.MSISDN != "+5519998765432" {
			t.Errorf("expected MSISDN +5519998765432, got %s", sub.MSISDN)
		}
		if sub.CreatedAt.IsZero() || sub.UpdatedAt.IsZero() {
			t.Error("expected timestamps to be initialized")
		}
	})

	t.Run("invalid IMSI length or characters", func(t *testing.T) {
		invalidIMSIs := []string{
			"123",              // curto
			"1234567890123456", // 16 dígitos
			"72499123456789A",  // contém letra
			"",                 // vazio
		}

		for _, imsi := range invalidIMSIs {
			_, err := New(imsi, "+5519998765432")
			if !errors.Is(err, ErrInvalidIMSI) {
				t.Errorf("for IMSI %q, expected ErrInvalidIMSI, got %v", imsi, err)
			}
		}
	})

	t.Run("invalid MSISDN format", func(t *testing.T) {
		invalidMSISDNs := []string{
			"123",          // curto
			"+0123456789",  // prefixo zero inválido
			"not-a-number", // letras
			"",             // vazio
		}

		for _, msisdn := range invalidMSISDNs {
			_, err := New("724991234567890", msisdn)
			if !errors.Is(err, ErrInvalidMSISDN) {
				t.Errorf("for MSISDN %q, expected ErrInvalidMSISDN, got %v", msisdn, err)
			}
		}
	})
}

func TestSubscriberStateTransitions(t *testing.T) {
	t.Run("valid cycle: PENDING -> ACTIVE -> SUSPENDED -> ACTIVE -> DEACTIVATED", func(t *testing.T) {
		sub, err := New("724991234567890", "+5519998765432")
		if err != nil {
			t.Fatalf("failed to create subscriber: %v", err)
		}

		// Activate from PENDING
		if err := sub.Activate(); err != nil {
			t.Fatalf("failed to activate: %v", err)
		}
		if sub.Status != StatusActive {
			t.Errorf("expected status %s, got %s", StatusActive, sub.Status)
		}

		// Suspend from ACTIVE
		if err := sub.Suspend("billing_hold"); err != nil {
			t.Fatalf("failed to suspend: %v", err)
		}
		if sub.Status != StatusSuspended {
			t.Errorf("expected status %s, got %s", StatusSuspended, sub.Status)
		}
		if sub.SuspensionReason != "billing_hold" {
			t.Errorf("expected reason 'billing_hold', got %q", sub.SuspensionReason)
		}

		// Re-activate from SUSPENDED
		if err := sub.Activate(); err != nil {
			t.Fatalf("failed to reactivate: %v", err)
		}
		if sub.Status != StatusActive {
			t.Errorf("expected status %s, got %s", StatusActive, sub.Status)
		}
		if sub.SuspensionReason != "" {
			t.Errorf("expected cleared suspension reason, got %q", sub.SuspensionReason)
		}

		// Deactivate terminal
		if err := sub.Deactivate("customer_request"); err != nil {
			t.Fatalf("failed to deactivate: %v", err)
		}
		if sub.Status != StatusDeactivated {
			t.Errorf("expected status %s, got %s", StatusDeactivated, sub.Status)
		}
		if sub.DeactivationReason != "customer_request" {
			t.Errorf("expected reason 'customer_request', got %q", sub.DeactivationReason)
		}
	})

	t.Run("invalid transition: cannot suspend from PENDING", func(t *testing.T) {
		sub, _ := New("724991234567890", "+5519998765432")
		err := sub.Suspend("test")
		if !errors.Is(err, ErrInvalidTransition) {
			t.Errorf("expected ErrInvalidTransition, got %v", err)
		}
	})

	t.Run("terminal state: cannot perform actions once DEACTIVATED", func(t *testing.T) {
		sub, _ := New("724991234567890", "+5519998765432")
		_ = sub.Deactivate("cancel")

		if err := sub.Activate(); !errors.Is(err, ErrAlreadyDeactivated) {
			t.Errorf("expected ErrAlreadyDeactivated on Activate, got %v", err)
		}
		if err := sub.Suspend("hold"); !errors.Is(err, ErrAlreadyDeactivated) {
			t.Errorf("expected ErrAlreadyDeactivated on Suspend, got %v", err)
		}
		if err := sub.Deactivate("cancel_again"); !errors.Is(err, ErrAlreadyDeactivated) {
			t.Errorf("expected ErrAlreadyDeactivated on Deactivate, got %v", err)
		}
	})
}

func TestMemoryRepositoryConcurrency(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	var wg sync.WaitGroup
	workers := 20
	subsPerWorker := 10

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < subsPerWorker; i++ {
				imsi := fmt.Sprintf("72499%02d%08d", workerID, i)
				msisdn := fmt.Sprintf("+5519%02d%07d", workerID, i)

				sub, err := New(imsi, msisdn)
				if err != nil {
					t.Errorf("worker %d failed to create subscriber: %v", workerID, err)
					return
				}

				if err := repo.Save(ctx, sub); err != nil {
					t.Errorf("worker %d failed to save subscriber: %v", workerID, err)
					return
				}

				found, err := repo.FindByIMSI(ctx, imsi)
				if err != nil || found.ID != sub.ID {
					t.Errorf("worker %d failed to find subscriber by IMSI: %v", workerID, err)
					return
				}
			}
		}(w)
	}

	wg.Wait()

	all, err := repo.List(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("failed to list subscribers: %v", err)
	}

	expectedCount := workers * subsPerWorker
	if len(all) != expectedCount {
		t.Fatalf("expected %d subscribers, got %d", expectedCount, len(all))
	}
}
