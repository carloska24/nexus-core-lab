package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	apiURL := flag.String("api", "http://localhost:8080", "Base URL da API HTTP do NEXUS Core Lab")
	devices := flag.Int("devices", 5, "Quantidade de Virtual Devices concorrentes (1 a 100)")
	flag.Parse()

	if *devices < 1 || *devices > 100 {
		fmt.Fprintf(os.Stderr, "Erro de configuração: o parâmetro -devices deve estar entre 1 e 100 (recebido: %d)\n", *devices)
		os.Exit(1)
	}

	cfg := RunnerConfig{
		BaseURL: *apiURL,
		Devices: *devices,
		Timeout: 10 * time.Second,
	}

	client := NewClient(cfg.BaseURL, cfg.Timeout)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println("======================================================================")
	fmt.Println("           NEXUS CORE LAB — SIMULATION RUNNER CLI                    ")
	fmt.Println("======================================================================")
	fmt.Printf("API Target:        %s\n", cfg.BaseURL)
	fmt.Printf("Virtual Devices:   %d\n", cfg.Devices)
	fmt.Printf("Hero Flow:         Provision -> Activate -> Register -> Attach -> Handover -> Detach\n")
	fmt.Println("----------------------------------------------------------------------")
	fmt.Println("Iniciando execução concorrente...")

	res, err := Run(ctx, cfg, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao executar simulação: %v\n", err)
		os.Exit(1)
	}

	printReport(res, cfg.BaseURL)

	if res.DevicesFailed > 0 {
		os.Exit(1)
	}
}

func printReport(res *SimulationResult, apiTarget string) {
	var successRate float64
	if res.DevicesRequested > 0 {
		successRate = (float64(res.DevicesCompleted) / float64(res.DevicesRequested)) * 100.0
	}

	fmt.Println()
	fmt.Println("======================================================================")
	fmt.Println("               NEXUS CORE LAB — SIMULATION RUNNER REPORT              ")
	fmt.Println("======================================================================")
	fmt.Printf("API Target:           %s\n", apiTarget)
	fmt.Printf("Devices Requested:    %d\n", res.DevicesRequested)
	fmt.Printf("Devices Completed:    %d\n", res.DevicesCompleted)
	fmt.Printf("Devices Failed:       %d\n", res.DevicesFailed)
	fmt.Printf("Total Duration:       %s\n", res.Duration.Round(time.Millisecond))
	fmt.Printf("Success Rate:         %.1f%%\n", successRate)
	fmt.Println("----------------------------------------------------------------------")

	if res.Telemetry != nil {
		fmt.Println("Server Telemetry Snapshot (GET /telemetry):")
		fmt.Printf("  Requests Total:     %d\n", res.Telemetry.RequestsTotal)
		fmt.Printf("  Active Sessions:    %d\n", res.Telemetry.Metrics.ActiveSessions)
		fmt.Printf("  Connected Devices:  %d\n", res.Telemetry.Metrics.ConnectedDevices)
		fmt.Printf("  Attaches:           %d\n", res.Telemetry.EventsTotal.Attach)
		fmt.Printf("  Cell Handovers:     %d\n", res.Telemetry.EventsTotal.CellHandover)
		fmt.Printf("  Detaches:           %d\n", res.Telemetry.EventsTotal.Detach)
		fmt.Printf("  Stale Disconnects:  %d\n", res.Telemetry.EventsTotal.StaleDisconnect)
		fmt.Printf("  Dropped Events:     %d\n", res.Telemetry.Metrics.DroppedEventsTotal)
	} else {
		fmt.Println("Server Telemetry Snapshot: (Não disponível / timeout na consulta final)")
	}
	fmt.Println("======================================================================")
}
