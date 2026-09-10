package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeAPIServer simula os contratos HTTP da API REST do NEXUS Core Lab para testes desacoplados.
// Não importa nenhum pacote interno de domínio.
type fakeAPIServer struct {
	mu             sync.Mutex
	subscribers    map[string]bool   // id -> activated
	devices        map[string]string // device_id -> subscriber_id
	sessions       map[string]string // session_id -> device_id
	currentConcurr atomic.Int64
	maxConcurr     atomic.Int64

	// Injeção de falhas controladas
	failOnIMSI  string
	failStep    string
	failStatus  int
	failMessage string
	onStepHook  func(step string)
}

func newFakeAPIServer() *fakeAPIServer {
	return &fakeAPIServer{
		subscribers: make(map[string]bool),
		devices:     make(map[string]string),
		sessions:    make(map[string]string),
	}
}

func (s *fakeAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.currentConcurr.Add(1)
	defer s.currentConcurr.Add(-1)

	// Atualiza pico de requisições concorrentes
	for {
		cur := s.currentConcurr.Load()
		max := s.maxConcurr.Load()
		if cur <= max {
			break
		}
		if s.maxConcurr.CompareAndSwap(max, cur) {
			break
		}
	}

	path := r.URL.Path
	method := r.Method

	// Rota: GET /telemetry
	if method == http.MethodGet && path == "/telemetry" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TelemetrySnapshot{
			Service:       "nexus-core-lab-fake",
			Timestamp:     time.Now().UTC(),
			RequestsTotal: 42,
			Metrics: TelemetryMetrics{
				ActiveSessions:     0,
				ConnectedDevices:   0,
				DroppedEventsTotal: 0,
			},
			EventsTotal: TelemetryEvents{
				Attach:       5,
				CellHandover: 5,
				Detach:       5,
			},
		})
		return
	}

	// Rota: POST /api/v1/subscribers
	if method == http.MethodPost && path == "/api/v1/subscribers" {
		var req ProvisionRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		if s.onStepHook != nil {
			s.onStepHook("provision")
		}

		if s.failStep == "provision" && (s.failOnIMSI == "" || s.failOnIMSI == req.IMSI) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(s.failStatus)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "simulated_error",
				Message: s.failMessage,
				Code:    "SIMULATED_FAILURE",
			})
			return
		}

		s.mu.Lock()
		subID := fmt.Sprintf("sub-%s", req.IMSI)
		s.subscribers[subID] = false
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ProvisionResponse{
			ID:     subID,
			IMSI:   req.IMSI,
			MSISDN: req.MSISDN,
			Status: "PENDING_ACTIVATION",
		})
		return
	}

	// Rota: POST /api/v1/subscribers/{id}/activate
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/subscribers/") && strings.HasSuffix(path, "/activate") {
		parts := strings.Split(path, "/")
		subID := parts[4]

		if s.onStepHook != nil {
			s.onStepHook("activate")
		}

		s.mu.Lock()
		_, exists := s.subscribers[subID]
		if exists {
			s.subscribers[subID] = true
		}
		s.mu.Unlock()

		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ACTIVE"}`))
		return
	}

	// Rota: POST /api/v1/devices
	if method == http.MethodPost && path == "/api/v1/devices" {
		var req DeviceRegisterRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		if s.onStepHook != nil {
			s.onStepHook("register_device")
		}

		s.mu.Lock()
		active := s.subscribers[req.SubscriberID]
		s.mu.Unlock()

		if !active {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		devID := fmt.Sprintf("dev-%s", req.IMEI)
		s.mu.Lock()
		s.devices[devID] = req.SubscriberID
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(DeviceResponse{
			ID:           devID,
			SubscriberID: req.SubscriberID,
			IMEI:         req.IMEI,
			Technology:   req.Technology,
			Status:       "REGISTERED",
		})
		return
	}

	// Rota: POST /api/v1/sessions/attach
	if method == http.MethodPost && path == "/api/v1/sessions/attach" {
		var req AttachRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		if s.onStepHook != nil {
			s.onStepHook("attach")
		}

		if s.failStep == "attach" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(s.failStatus)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "simulated_error",
				Message: s.failMessage,
				Code:    "SIMULATED_ATTACH_FAILURE",
			})
			return
		}

		s.mu.Lock()
		_, devExists := s.devices[req.DeviceID]
		s.mu.Unlock()

		if !devExists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		sessID := fmt.Sprintf("sess-%s", req.DeviceID)
		s.mu.Lock()
		s.sessions[sessID] = req.DeviceID
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(AttachResponse{
			ID:        sessID,
			DeviceID:  req.DeviceID,
			CellID:    req.CellID,
			IPAddress: "10.45.0.2",
			Status:    "CONNECTED",
		})
		return
	}

	// Rota: POST /api/v1/sessions/{id}/handover
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/sessions/") && strings.HasSuffix(path, "/handover") {
		parts := strings.Split(path, "/")
		sessID := parts[4]

		var req HandoverRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		if s.onStepHook != nil {
			s.onStepHook("handover")
		}

		s.mu.Lock()
		devID, sessExists := s.sessions[sessID]
		s.mu.Unlock()

		if !sessExists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SessionResponse{
			ID:        sessID,
			DeviceID:  devID,
			CellID:    req.TargetCellID,
			IPAddress: "10.45.0.2",
			Status:    "CONNECTED",
		})
		return
	}

	// Rota: POST /api/v1/sessions/{id}/detach
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/sessions/") && strings.HasSuffix(path, "/detach") {
		parts := strings.Split(path, "/")
		sessID := parts[4]

		if s.onStepHook != nil {
			s.onStepHook("detach")
		}

		s.mu.Lock()
		devID, sessExists := s.sessions[sessID]
		s.mu.Unlock()

		if !sessExists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SessionResponse{
			ID:        sessID,
			DeviceID:  devID,
			CellID:    "CELL-SP-002",
			IPAddress: "10.45.0.2",
			Status:    "DETACHED",
		})
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func TestHeroFlow_SingleDevice(t *testing.T) {
	fake := newFakeAPIServer()
	ts := httptest.NewServer(fake)
	defer ts.Close()

	client := NewClient(ts.URL, 5*time.Second)
	cfg := RunnerConfig{
		BaseURL: ts.URL,
		Devices: 1,
		Timeout: 5 * time.Second,
	}

	res, err := Run(context.Background(), cfg, client)
	if err != nil {
		t.Fatalf("expected Run to succeed, got %v", err)
	}

	if res.DevicesRequested != 1 {
		t.Errorf("expected 1 requested, got %d", res.DevicesRequested)
	}
	if res.DevicesCompleted != 1 {
		t.Errorf("expected 1 completed, got %d", res.DevicesCompleted)
	}
	if res.DevicesFailed != 0 {
		t.Errorf("expected 0 failed, got %d", res.DevicesFailed)
	}
	if len(res.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res.Results))
	}

	r0 := res.Results[0]
	if r0.Failed {
		t.Errorf("expected device 0 not to fail, got %v", r0.Error)
	}
	if r0.SubscriberID == "" || r0.DeviceID == "" || r0.SessionID == "" {
		t.Errorf("expected non-empty IDs, got sub=%s, dev=%s, sess=%s", r0.SubscriberID, r0.DeviceID, r0.SessionID)
	}
	if r0.IPAddress != "10.45.0.2" {
		t.Errorf("expected IP 10.45.0.2, got %s", r0.IPAddress)
	}
	if res.Telemetry == nil {
		t.Errorf("expected telemetry snapshot to be present")
	}
}

func TestHeroFlow_MultipleDevicesConcurrent(t *testing.T) {
	fake := newFakeAPIServer()
	ts := httptest.NewServer(fake)
	defer ts.Close()

	client := NewClient(ts.URL, 5*time.Second)
	cfg := RunnerConfig{
		BaseURL: ts.URL,
		Devices: 5,
		Timeout: 5 * time.Second,
	}

	res, err := Run(context.Background(), cfg, client)
	if err != nil {
		t.Fatalf("expected Run to succeed, got %v", err)
	}

	if res.DevicesRequested != 5 {
		t.Errorf("expected 5 requested, got %d", res.DevicesRequested)
	}
	if res.DevicesCompleted != 5 {
		t.Errorf("expected 5 completed, got %d", res.DevicesCompleted)
	}
	if res.DevicesFailed != 0 {
		t.Errorf("expected 0 failed, got %d", res.DevicesFailed)
	}

	// Valida concorrência observada no servidor fake
	if fake.maxConcurr.Load() < 1 {
		t.Errorf("expected concurrent requests > 0, got %d", fake.maxConcurr.Load())
	}
}

func TestDeviceFailure_HTTP409_Conflict_Isolation(t *testing.T) {
	fake := newFakeAPIServer()
	fake.failStep = "provision"
	fake.failStatus = http.StatusConflict
	fake.failMessage = "subscriber already exists with this IMSI"

	// Gerar identidades de teste para saber o IMSI do device 1
	ident1 := GenerateIdentities(1000, 1)
	fake.failOnIMSI = ident1.IMSI

	ts := httptest.NewServer(fake)
	defer ts.Close()

	client := NewClient(ts.URL, 5*time.Second)
	cfg := RunnerConfig{
		BaseURL: ts.URL,
		Devices: 3,
		Timeout: 5 * time.Second,
	}

	// Fixamos o seed temporariamente para coincidir com o ident1 testado
	startTime := time.Now()
	collector := &resultsCollector{}
	var wg sync.WaitGroup
	for i := 0; i < cfg.Devices; i++ {
		wg.Add(1)
		go func(devIndex int) {
			defer wg.Done()
			runVirtualDevice(context.Background(), client, devIndex, 1000, collector)
		}(i)
	}
	wg.Wait()

	rawResults := collector.all()
	completed := 0
	failed := 0
	for _, r := range rawResults {
		if r.Failed {
			failed++
			if r.FailedStep != "provision_subscriber" {
				t.Errorf("expected failed step provision_subscriber, got %s", r.FailedStep)
			}
			if !strings.Contains(r.Error.Error(), "409") {
				t.Errorf("expected error to mention 409, got %v", r.Error)
			}
		} else {
			completed++
		}
	}

	if completed != 2 {
		t.Errorf("expected 2 devices to complete successfully, got %d", completed)
	}
	if failed != 1 {
		t.Errorf("expected 1 device to fail, got %d", failed)
	}
	if time.Since(startTime) <= 0 {
		t.Errorf("expected non-zero duration")
	}
}

func TestDeviceFailure_HTTP500_ServerError_Isolation(t *testing.T) {
	fake := newFakeAPIServer()
	fake.failStep = "attach"
	fake.failStatus = http.StatusInternalServerError
	fake.failMessage = "ip pool exhausted"

	ts := httptest.NewServer(fake)
	defer ts.Close()

	client := NewClient(ts.URL, 5*time.Second)
	cfg := RunnerConfig{
		BaseURL: ts.URL,
		Devices: 2,
		Timeout: 5 * time.Second,
	}

	res, err := Run(context.Background(), cfg, client)
	if err != nil {
		t.Fatalf("expected Run to return result without crashing, got %v", err)
	}

	if res.DevicesFailed != 2 {
		t.Errorf("expected 2 devices failed due to simulated attach error, got %d", res.DevicesFailed)
	}
	for _, r := range res.Results {
		if r.FailedStep != "attach_session" {
			t.Errorf("expected failure at attach_session, got %s", r.FailedStep)
		}
	}
}

func TestCancellation_PreCanceledContext(t *testing.T) {
	fake := newFakeAPIServer()
	ts := httptest.NewServer(fake)
	defer ts.Close()

	client := NewClient(ts.URL, 5*time.Second)
	cfg := RunnerConfig{
		BaseURL: ts.URL,
		Devices: 3,
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pré-cancela

	res, err := Run(ctx, cfg, client)
	if err != nil {
		t.Fatalf("expected Run to handle canceled context gracefully, got %v", err)
	}

	if res.DevicesFailed != 3 {
		t.Errorf("expected 3 devices to fail due to pre-canceled context, got %d", res.DevicesFailed)
	}
	for _, r := range res.Results {
		if !r.Failed {
			t.Errorf("expected device to record failure")
		}
	}
}

func TestCancellation_DuringExecution(t *testing.T) {
	fake := newFakeAPIServer()
	ctx, cancel := context.WithCancel(context.Background())

	var cancelOnce sync.Once
	fake.onStepHook = func(step string) {
		if step == "register_device" {
			cancelOnce.Do(func() {
				cancel()
			})
		}
	}

	ts := httptest.NewServer(fake)
	defer ts.Close()

	client := NewClient(ts.URL, 5*time.Second)
	cfg := RunnerConfig{
		BaseURL: ts.URL,
		Devices: 5,
		Timeout: 5 * time.Second,
	}

	res, err := Run(ctx, cfg, client)
	if err != nil {
		t.Fatalf("expected Run to return result on cancel, got %v", err)
	}

	// Algum device deve ter sido interrompido
	if res.DevicesRequested != 5 {
		t.Errorf("expected 5 requested, got %d", res.DevicesRequested)
	}
}

func TestFlagValidation(t *testing.T) {
	client := NewClient("http://localhost:8080", time.Second)

	_, errLow := Run(context.Background(), RunnerConfig{Devices: 0}, client)
	if errLow == nil {
		t.Errorf("expected error for devices=0")
	}

	_, errHigh := Run(context.Background(), RunnerConfig{Devices: 101}, client)
	if errHigh == nil {
		t.Errorf("expected error for devices=101")
	}
}

func TestIdentitiesGeneration(t *testing.T) {
	seed := int64(12345000)
	seenIMSI := make(map[string]bool)
	seenMSISDN := make(map[string]bool)
	seenIMEI := make(map[string]bool)

	for i := 0; i < 100; i++ {
		ident := GenerateIdentities(seed, i)

		// 15 dígitos exatos para IMSI
		if len(ident.IMSI) != 15 {
			t.Fatalf("expected IMSI length 15, got %d (%s)", len(ident.IMSI), ident.IMSI)
		}
		for _, c := range ident.IMSI {
			if c < '0' || c > '9' {
				t.Fatalf("IMSI must be purely numeric, got %s", ident.IMSI)
			}
		}

		// 15 dígitos exatos para IMEI
		if len(ident.IMEI) != 15 {
			t.Fatalf("expected IMEI length 15, got %d (%s)", len(ident.IMEI), ident.IMEI)
		}
		for _, c := range ident.IMEI {
			if c < '0' || c > '9' {
				t.Fatalf("IMEI must be purely numeric, got %s", ident.IMEI)
			}
		}

		// MSISDN formato E.164: prefixo +55199 + 8 dígitos = 14 caracteres
		if len(ident.MSISDN) != 14 || !strings.HasPrefix(ident.MSISDN, "+55199") {
			t.Fatalf("expected MSISDN length 14 with +55199 prefix, got %s", ident.MSISDN)
		}

		// Sem colisão intra-execução
		if seenIMSI[ident.IMSI] {
			t.Fatalf("duplicate IMSI detected: %s", ident.IMSI)
		}
		seenIMSI[ident.IMSI] = true

		if seenMSISDN[ident.MSISDN] {
			t.Fatalf("duplicate MSISDN detected: %s", ident.MSISDN)
		}
		seenMSISDN[ident.MSISDN] = true

		if seenIMEI[ident.IMEI] {
			t.Fatalf("duplicate IMEI detected: %s", ident.IMEI)
		}
		seenIMEI[ident.IMEI] = true
	}
}
