package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/carloska24/nexus-core-lab/internal/device"
	"github.com/carloska24/nexus-core-lab/internal/network"
	"github.com/carloska24/nexus-core-lab/internal/platform/httpserver"
	"github.com/carloska24/nexus-core-lab/internal/session"
	"github.com/carloska24/nexus-core-lab/internal/subscriber"
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

func main() {
	// Composição de dependências do módulo Subscriber (Milestone 1)
	subscriberRepo := subscriber.NewMemoryRepository()
	subscriberService := subscriber.NewService(subscriberRepo)
	subscriberHandler := subscriber.NewHandler(subscriberService)

	// Composição de dependências do módulo Device (Milestone 2)
	deviceRepo := device.NewMemoryRepository()
	deviceService := device.NewService(deviceRepo, &subscriberCheckerAdapter{subService: subscriberService})
	deviceHandler := device.NewHandler(deviceService)

	// Composição de dependências dos módulos Network e Session (Milestone 3)
	ipPool := network.NewIPPool()
	sessionRepo := session.NewMemoryRepository()
	sessionService := session.NewService(sessionRepo, &deviceCheckerAdapter{deviceService: deviceService}, ipPool)
	sessionHandler := session.NewHandler(sessionService)

	handler := httpserver.New(
		subscriberHandler.RegisterRoutes,
		deviceHandler.RegisterRoutes,
		sessionHandler.RegisterRoutes,
	)

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

	log.Println("HTTP server stopped")
}
