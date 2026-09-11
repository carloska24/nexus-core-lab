package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/carloska24/nexus-core-lab/internal/device"
	"github.com/carloska24/nexus-core-lab/internal/network"
	"github.com/carloska24/nexus-core-lab/internal/platform/httpserver"
	"github.com/carloska24/nexus-core-lab/internal/platform/postgres"
	"github.com/carloska24/nexus-core-lab/internal/session"
	"github.com/carloska24/nexus-core-lab/internal/subscriber"
	"github.com/carloska24/nexus-core-lab/internal/telemetry"
)

// subscriberCheckerAdapter conecta o serviço de Subscriber ao contrato consumidor de Device
// sem vazar tipos ou strings de status do domínio Subscriber para o pacote Device.
type subscriberCheckerAdapter struct {
	subService *subscriber.Service
}

func (a *subscriberCheckerAdapter) CheckSubscriberActive(ctx context.Context, subscriberID string) error {
	sub, err := a.subService.FindByID(ctx, subscriberID)
	if err != nil {
		if errors.Is(err, subscriber.ErrSubscriberNotFound) {
			return device.ErrSubscriberNotFound
		}
		return err
	}

	if sub.Status != subscriber.StatusActive {
		return device.ErrSubscriberNotActive
	}

	return nil
}

// deviceCheckerAdapter conecta o serviço de Device ao contrato consumidor de Session
// sem vazar structs ou tipos internos de Device para o pacote Session.
type deviceCheckerAdapter struct {
	deviceService *device.Service
}

func (a *deviceCheckerAdapter) CheckDeviceAttachable(ctx context.Context, deviceID string) (*session.DeviceInfo, error) {
	dev, err := a.deviceService.FindByID(ctx, deviceID)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			return nil, session.ErrDeviceNotFound
		}
		return nil, err
	}

	if dev.Status != device.StatusRegistered {
		return nil, session.ErrDeviceNotEligible
	}

	return &session.DeviceInfo{
		DeviceID:     dev.ID,
		SubscriberID: dev.SubscriberID,
	}, nil
}

// telemetrySessionAdapter conecta o domínio Session ao pipeline assíncrono de Telemetria
// sem que o domínio de sessões precise conhecer o pacote de telemetria.
type telemetrySessionAdapter struct {
	worker *telemetry.Worker
}

func (a *telemetrySessionAdapter) EmitSessionEvent(ctx context.Context, eventType session.SessionEventType, s *session.Session) {
	var tType telemetry.EventType
	switch eventType {
	case session.SessionEventAttach:
		tType = telemetry.EventAttach
	case session.SessionEventCellHandover:
		tType = telemetry.EventCellHandover
	case session.SessionEventDetach:
		tType = telemetry.EventDetach
	case session.SessionEventStaleDisconnect:
		tType = telemetry.EventStaleDisconnect
	default:
		return
	}

	a.worker.Emit(telemetry.Event{
		Type:         tType,
		SessionID:    s.ID,
		DeviceID:     s.DeviceID,
		SubscriberID: s.SubscriberID,
		CellID:       s.CellID,
		Timestamp:    time.Now().UTC(),
	})
}

// warmUpIPPool carrega todas as sessões CONNECTED e pré-aloca os IPs no IPPool antes de aceitar requisições.
func warmUpIPPool(ctx context.Context, db *sql.DB, ipPool *network.IPPool) error {
	rows, err := db.QueryContext(ctx, "SELECT ip_address FROM sessions WHERE status = 'CONNECTED'")
	if err != nil {
		return fmt.Errorf("failed to query connected sessions for IPPool warm-up: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return fmt.Errorf("failed to scan session ip: %w", err)
		}
		if err := ipPool.MarkAllocated(ip); err != nil {
			return fmt.Errorf("failed to mark IP %s as allocated during warm-up: %w", ip, err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating connected sessions during warm-up: %w", err)
	}

	log.Printf("IPPool warm-up completed: %d active session IP(s) pre-allocated", count)
	return nil
}

func main() {
	ctx := context.Background()

	// Infraestrutura de Telemetria e Observabilidade (Milestone 4)
	var requestsCounter atomic.Uint64
	telemetryWorker := telemetry.NewWorker()
	telemetryWorker.Start()

	// Inicialização do pool de IPs em memória
	ipPool := network.NewIPPool()

	// Resolução e injeção de persistência (In-Memory vs PostgreSQL)
	databaseURL := os.Getenv("DATABASE_URL")

	var (
		subscriberRepo subscriber.Repository
		deviceRepo     device.Repository
		sessionRepo    session.Repository
		sqlDB          *sql.DB
	)

	if databaseURL == "" {
		log.Println("storage_backend=memory")
		subscriberRepo = subscriber.NewMemoryRepository()
		deviceRepo = device.NewMemoryRepository()
		sessionRepo = session.NewMemoryRepository()
	} else {
		log.Println("storage_backend=postgres")
		var err error
		sqlDB, err = postgres.Open(ctx, postgres.DefaultConfig(databaseURL))
		if err != nil {
			log.Fatalf("fatal: database connection failed: %v", err)
		}
		defer sqlDB.Close()

		// Valida se as migrações esperadas foram executadas
		const expectedSchemaVersion = 3
		if err := postgres.ValidateSchema(ctx, sqlDB, expectedSchemaVersion); err != nil {
			log.Fatalf("fatal: database schema validation failed: %v. Please run: go run ./cmd/migrate -up", err)
		}

		// Reconstituição obrigatória do estado do IPPool a partir das sessões ativas
		if err := warmUpIPPool(ctx, sqlDB, ipPool); err != nil {
			log.Fatalf("fatal: IPPool warm-up failed: %v", err)
		}

		subscriberRepo = subscriber.NewPostgresRepository(sqlDB)
		deviceRepo = device.NewPostgresRepository(sqlDB)
		sessionRepo = session.NewPostgresRepository(sqlDB)
	}

	// Composição de dependências do módulo Subscriber (Milestone 1)
	subscriberService := subscriber.NewService(subscriberRepo)
	subscriberHandler := subscriber.NewHandler(subscriberService)

	// Composição de dependências do módulo Device (Milestone 2)
	deviceService := device.NewService(deviceRepo, &subscriberCheckerAdapter{subService: subscriberService})
	deviceHandler := device.NewHandler(deviceService)

	// Composição de dependências dos módulos Network e Session (Milestone 3 & 6)
	sessionAdapter := &telemetrySessionAdapter{worker: telemetryWorker}
	sessionService := session.NewService(sessionRepo, &deviceCheckerAdapter{deviceService: deviceService}, ipPool, sessionAdapter)
	sessionHandler := session.NewHandler(sessionService)

	// Composição de dependências do módulo Telemetry (Milestone 4 & 6)
	telemetryHandler := telemetry.NewHandler(telemetryWorker.Metrics(), sessionRepo, &requestsCounter)

	router := httpserver.New(
		subscriberHandler.RegisterRoutes,
		deviceHandler.RegisterRoutes,
		sessionHandler.RegisterRoutes,
		telemetryHandler.RegisterRoutes,
	)

	// Envolve o roteador com middleware de Request ID, log/slog e contagem de requisições
	handler := httpserver.TelemetryMiddleware(&requestsCounter)(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	address := ":" + port

	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("NEXUS Core Lab API listening on %s", address)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatal(err)

	case sig := <-stop:
		log.Printf("received signal: %s", sig)
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("shutting down HTTP server")

	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("shutting down telemetry worker")
	if err := telemetryWorker.Shutdown(shutdownContext); err != nil {
		log.Printf("telemetry worker shutdown error: %v", err)
	}

	log.Println("HTTP server stopped")
}
