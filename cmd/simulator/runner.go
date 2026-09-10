package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TelecomIdentity encapsula os identificadores telecom gerados para um dispositivo virtual.
type TelecomIdentity struct {
	IMSI   string
	MSISDN string
	IMEI   string
}

// GenerateIdentities produz identidades válidas e determinísticas para o índice do dispositivo,
// garantindo ausência de colisão intra-execução e conformidade exata de formato e tamanho.
func GenerateIdentities(seed int64, devIndex int) TelecomIdentity {
	seq := seed + int64(devIndex)
	return TelecomIdentity{
		IMSI:   fmt.Sprintf("72499%010d", seq),
		MSISDN: fmt.Sprintf("+55199%08d", seq%100000000),
		IMEI:   fmt.Sprintf("86000%010d", seq),
	}
}

// RunnerConfig parametriza a execução do simulador.
type RunnerConfig struct {
	BaseURL string
	Devices int
	Timeout time.Duration
}

// DeviceResult armazena o resultado e telemetria pontual da execução de um dispositivo virtual.
type DeviceResult struct {
	DeviceIndex  int
	IMSI         string
	MSISDN       string
	IMEI         string
	SubscriberID string
	DeviceID     string
	SessionID    string
	IPAddress    string
	Failed       bool
	FailedStep   string
	Error        error
	Duration     time.Duration
}

// SimulationResult agrega os resultados consolidados da execução de todos os Virtual Devices.
type SimulationResult struct {
	DevicesRequested int
	DevicesCompleted int
	DevicesFailed    int
	Duration         time.Duration
	Results          []DeviceResult
	Telemetry        *TelemetrySnapshot
}

// resultsCollector provê agregação thread-safe dos resultados dos dispositivos virtuais.
type resultsCollector struct {
	mu      sync.Mutex
	results []DeviceResult
}

func (c *resultsCollector) record(res DeviceResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = append(c.results, res)
}

func (c *resultsCollector) all() []DeviceResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]DeviceResult, len(c.results))
	copy(out, c.results)
	return out
}

// runVirtualDevice orquestra o ciclo de vida completo (Hero Flow) de um dispositivo virtual.
func runVirtualDevice(ctx context.Context, client *Client, devIndex int, seed int64, collector *resultsCollector) {
	startTime := time.Now()
	ident := GenerateIdentities(seed, devIndex)

	res := DeviceResult{
		DeviceIndex: devIndex,
		IMSI:        ident.IMSI,
		MSISDN:      ident.MSISDN,
		IMEI:        ident.IMEI,
	}

	fail := func(step string, err error) {
		res.Failed = true
		res.FailedStep = step
		res.Error = err
		res.Duration = time.Since(startTime)
		collector.record(res)
		fmt.Printf("[Device %02d] FAILED at %s: %v\n", devIndex, step, err)
	}

	// 1. Provision Subscriber
	subResp, err := client.ProvisionSubscriber(ctx, ident.IMSI, ident.MSISDN)
	if err != nil {
		fail("provision_subscriber", err)
		return
	}
	res.SubscriberID = subResp.ID
	fmt.Printf("[Device %02d] Provisioned subscriber %s (IMSI: %s)\n", devIndex, subResp.ID, ident.IMSI)

	// 2. Activate Subscriber
	if err := client.ActivateSubscriber(ctx, subResp.ID); err != nil {
		fail("activate_subscriber", err)
		return
	}
	fmt.Printf("[Device %02d] Activated subscriber %s\n", devIndex, subResp.ID)

	// 3. Register Device
	devResp, err := client.RegisterDevice(ctx, subResp.ID, ident.IMEI, "5G")
	if err != nil {
		fail("register_device", err)
		return
	}
	res.DeviceID = devResp.ID
	fmt.Printf("[Device %02d] Registered device %s (IMEI: %s)\n", devIndex, devResp.ID, ident.IMEI)

	// 4. Attach Session (CELL-SP-001)
	attachResp, err := client.AttachSession(ctx, devResp.ID, "CELL-SP-001")
	if err != nil {
		fail("attach_session", err)
		return
	}
	res.SessionID = attachResp.ID
	res.IPAddress = attachResp.IPAddress
	fmt.Printf("[Device %02d] Attached session %s on CELL-SP-001 (IP: %s)\n", devIndex, attachResp.ID, attachResp.IPAddress)

	// 5. Handover Session (CELL-SP-002)
	if err := client.HandoverSession(ctx, attachResp.ID, "CELL-SP-002"); err != nil {
		fail("handover_session", err)
		return
	}
	fmt.Printf("[Device %02d] Handover session %s to CELL-SP-002\n", devIndex, attachResp.ID)

	// 6. Detach Session
	if err := client.DetachSession(ctx, attachResp.ID); err != nil {
		fail("detach_session", err)
		return
	}
	fmt.Printf("[Device %02d] Detached session %s\n", devIndex, attachResp.ID)

	res.Duration = time.Since(startTime)
	collector.record(res)
}

// Run executa a simulação concorrente com N Virtual Devices e consolida os resultados.
func Run(ctx context.Context, cfg RunnerConfig, client *Client) (*SimulationResult, error) {
	if cfg.Devices < 1 || cfg.Devices > 100 {
		return nil, fmt.Errorf("devices must be between 1 and 100, got %d", cfg.Devices)
	}

	startTime := time.Now()
	seed := (time.Now().Unix() % 800000) * 1000
	collector := &resultsCollector{}

	var wg sync.WaitGroup
	for i := 0; i < cfg.Devices; i++ {
		wg.Add(1)
		go func(devIndex int) {
			defer wg.Done()
			runVirtualDevice(ctx, client, devIndex, seed, collector)
		}(i)
	}

	// Aguarda o término de todos os Virtual Devices
	wg.Wait()
	duration := time.Since(startTime)

	// Consulta de telemetria observacional ao término
	var teleSnapshot *TelemetrySnapshot
	teleCtx, teleCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer teleCancel()
	if tele, err := client.GetTelemetry(teleCtx); err == nil {
		teleSnapshot = tele
	}

	rawResults := collector.all()
	completed := 0
	failed := 0
	for _, r := range rawResults {
		if r.Failed {
			failed++
		} else {
			completed++
		}
	}

	return &SimulationResult{
		DevicesRequested: cfg.Devices,
		DevicesCompleted: completed,
		DevicesFailed:    failed,
		Duration:         duration,
		Results:          rawResults,
		Telemetry:        teleSnapshot,
	}, nil
}
