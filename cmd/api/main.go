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

	"github.com/carloska24/nexus-core-lab/internal/platform/httpserver"
)

func main() {
	handler := httpserver.New()

	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Println("NEXUS Core Lab API listening on :8080")

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
