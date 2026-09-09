package telemetry

import (
	"context"
	"sync"
)

const defaultBufferSize = 256

// Worker coordena o consumo assíncrono de eventos de conectividade através de um canal único.
type Worker struct {
	events    chan Event
	done      chan struct{}
	metrics   *Metrics
	mu        sync.RWMutex
	closed    bool
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewWorker cria uma nova instância de Worker de telemetria com canal bufferizado.
func NewWorker(bufferSize ...int) *Worker {
	size := defaultBufferSize
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		size = bufferSize[0]
	}

	return &Worker{
		events:  make(chan Event, size),
		done:    make(chan struct{}),
		metrics: &Metrics{},
	}
}

// Metrics expõe a estrutura de métricas do worker para inspeção e snapshots.
func (w *Worker) Metrics() *Metrics {
	return w.metrics
}

// Start inicia a execução da goroutine única do worker de forma segura e idempotente.
func (w *Worker) Start() {
	w.startOnce.Do(func() {
		go w.processLoop()
	})
}

// Emit envia um evento para o canal de forma não bloqueante.
// Se o canal estiver saturado, incrementa dropped_events_total sem bloquear o fluxo chamador.
// Se o worker já estiver em shutdown, descarta silenciosamente para prevenir send on closed channel.
func (w *Worker) Emit(evt Event) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.closed {
		return
	}

	select {
	case w.events <- evt:
	default:
		w.metrics.droppedEventsTotal.Add(1)
	}
}

// Shutdown encerra o worker de forma graciosa e idempotente:
// 1. Marca o worker como fechado sob lock e fecha o canal de eventos;
// 2. Drena e processa todos os eventos previamente enfileirados no buffer;
// 3. Aguarda a finalização do processamento respeitando o timeout do contexto.
func (w *Worker) Shutdown(ctx context.Context) error {
	w.stopOnce.Do(func() {
		w.mu.Lock()
		w.closed = true
		close(w.events)
		w.mu.Unlock()
	})

	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) processLoop() {
	defer close(w.done)

	for evt := range w.events {
		w.processEvent(evt)
	}
}

func (w *Worker) processEvent(evt Event) {
	switch evt.Type {
	case EventAttach:
		w.metrics.attachTotal.Add(1)
	case EventCellHandover:
		w.metrics.cellHandoverTotal.Add(1)
	case EventDetach:
		w.metrics.detachTotal.Add(1)
	case EventStaleDisconnect:
		w.metrics.staleDisconnectTotal.Add(1)
	}
}
