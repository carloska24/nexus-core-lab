package telemetry

import (
	"context"
	"sync"
	"testing"
	"time"
)

type mockActiveProvider struct {
	count int
	err   error
}

func (m *mockActiveProvider) ActiveCount(ctx context.Context) (int, error) {
	return m.count, m.err
}

func TestWorker_ProcessEventTypes(t *testing.T) {
	w := NewWorker(64)
	w.Start()

	w.Emit(Event{Type: EventAttach})
	w.Emit(Event{Type: EventAttach})
	w.Emit(Event{Type: EventCellHandover})
	w.Emit(Event{Type: EventStaleDisconnect})
	w.Emit(Event{Type: EventDetach})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := w.Shutdown(ctx); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}

	snap, err := w.Metrics().Snapshot(context.Background(), 10, &mockActiveProvider{count: 1})
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}

	if snap.EventsTotal.Attach != 2 {
		t.Errorf("expected 2 attach events, got %d", snap.EventsTotal.Attach)
	}
	if snap.EventsTotal.CellHandover != 1 {
		t.Errorf("expected 1 cell handover event, got %d", snap.EventsTotal.CellHandover)
	}
	if snap.EventsTotal.StaleDisconnect != 1 {
		t.Errorf("expected 1 stale disconnect event, got %d", snap.EventsTotal.StaleDisconnect)
	}
	if snap.EventsTotal.Detach != 1 {
		t.Errorf("expected 1 detach event, got %d", snap.EventsTotal.Detach)
	}
	if snap.RequestsTotal != 10 {
		t.Errorf("expected 10 requests total, got %d", snap.RequestsTotal)
	}
	if snap.Metrics.ActiveSessions != 1 || snap.Metrics.ConnectedDevices != 1 {
		t.Errorf("expected 1 active session/connected device, got %d / %d",
			snap.Metrics.ActiveSessions, snap.Metrics.ConnectedDevices)
	}
	if snap.Metrics.DroppedEventsTotal != 0 {
		t.Errorf("expected 0 dropped events, got %d", snap.Metrics.DroppedEventsTotal)
	}
}

func TestWorker_BufferFullNonBlocking(t *testing.T) {
	// Worker com buffer pequeno de tamanho 2 sem iniciar o loop para encher o canal
	w := NewWorker(2)

	w.Emit(Event{Type: EventAttach})
	w.Emit(Event{Type: EventAttach})

	// Terceiro e quarto eventos devem ser descartados sem travar
	w.Emit(Event{Type: EventAttach})
	w.Emit(Event{Type: EventAttach})

	if w.Metrics().droppedEventsTotal.Load() != 2 {
		t.Fatalf("expected 2 dropped events, got %d", w.Metrics().droppedEventsTotal.Load())
	}
}

func TestWorker_GracefulDrain(t *testing.T) {
	w := NewWorker(100)
	w.Start()

	totalEvents := 50
	for i := 0; i < totalEvents; i++ {
		w.Emit(Event{Type: EventAttach})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := w.Shutdown(ctx); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}

	// Todos os 50 eventos enfileirados devem ter sido processados no drain
	if w.Metrics().attachTotal.Load() != uint64(totalEvents) {
		t.Fatalf("expected %d attach events processed after drain, got %d",
			totalEvents, w.Metrics().attachTotal.Load())
	}
}

func TestWorker_EmitConcurrentWithShutdown(t *testing.T) {
	w := NewWorker(128)
	w.Start()

	var wg sync.WaitGroup
	workers := 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				w.Emit(Event{Type: EventAttach})
				time.Sleep(10 * time.Microsecond)
			}
		}()
	}

	// Dispara shutdown enquanto emissões estão ocorrendo
	time.Sleep(5 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := w.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}

	// Idempotência do Shutdown
	if err := w.Shutdown(ctx); err != nil {
		t.Fatalf("second shutdown must be idempotent, got error: %v", err)
	}

	wg.Wait()
}

func TestWorker_StartIdempotent(t *testing.T) {
	w := NewWorker(16)
	w.Start()
	w.Start() // Segunda chamada não deve instanciar segunda goroutine

	w.Emit(Event{Type: EventAttach})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := w.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}

	if w.Metrics().attachTotal.Load() != 1 {
		t.Errorf("expected exactly 1 attach event, got %d", w.Metrics().attachTotal.Load())
	}
}
