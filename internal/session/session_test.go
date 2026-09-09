package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/carloska24/nexus-core-lab/internal/network"
)

type mockDeviceChecker struct {
	mu      sync.Mutex
	devices map[string]*DeviceInfo
	allowed map[string]bool
}

func newMockDeviceChecker() *mockDeviceChecker {
	return &mockDeviceChecker{
		devices: make(map[string]*DeviceInfo),
		allowed: make(map[string]bool),
	}
}

func (m *mockDeviceChecker) addDevice(deviceID, subscriberID string, eligible bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[deviceID] = &DeviceInfo{DeviceID: deviceID, SubscriberID: subscriberID}
	m.allowed[deviceID] = eligible
}

func (m *mockDeviceChecker) CheckDeviceAttachable(ctx context.Context, deviceID string) (*DeviceInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dev, exists := m.devices[deviceID]
	if !exists {
		return nil, ErrDeviceNotFound
	}
	if !m.allowed[deviceID] {
		return nil, ErrDeviceNotEligible
	}
	return dev, nil
}

func TestAttach_SuccessAndValidations(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	checker := newMockDeviceChecker()
	checker.addDevice("DEV-001", "SUB-001", true)
	checker.addDevice("DEV-INELIGIBLE", "SUB-002", false)
	ipPool := network.NewIPPool()

	svc := NewService(repo, checker, ipPool)

	// Caso 1: Célula inválida
	_, err := svc.Attach(ctx, AttachRequest{DeviceID: "DEV-001", CellID: "CELL-UNKNOWN"})
	if !errors.Is(err, network.ErrCellNotFound) {
		t.Fatalf("expected ErrCellNotFound, got %v", err)
	}

	// Caso 2: Dispositivo inexistente
	_, err = svc.Attach(ctx, AttachRequest{DeviceID: "DEV-UNKNOWN", CellID: "CELL-SP-001"})
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}

	// Caso 3: Dispositivo inelegível
	_, err = svc.Attach(ctx, AttachRequest{DeviceID: "DEV-INELIGIBLE", CellID: "CELL-SP-001"})
	if !errors.Is(err, ErrDeviceNotEligible) {
		t.Fatalf("expected ErrDeviceNotEligible, got %v", err)
	}

	// Caso 4: Sucesso
	sess, err := svc.Attach(ctx, AttachRequest{DeviceID: "DEV-001", CellID: "CELL-SP-001"})
	if err != nil {
		t.Fatalf("unexpected attach error: %v", err)
	}

	if sess.Status != StatusConnected {
		t.Errorf("expected StatusConnected, got %s", sess.Status)
	}
	if sess.IPAddress != "10.45.0.2" {
		t.Errorf("expected IP 10.45.0.2, got %s", sess.IPAddress)
	}
	if sess.SubscriberID != "SUB-001" {
		t.Errorf("expected SUB-001, got %s", sess.SubscriberID)
	}
	if !ipPool.IsAllocated(sess.IPAddress) {
		t.Errorf("expected IP %s to be marked allocated in pool", sess.IPAddress)
	}

	active, err := svc.GetActiveByDevice(ctx, "DEV-001")
	if err != nil {
		t.Fatalf("unexpected error getting active session: %v", err)
	}
	if active.ID != sess.ID {
		t.Errorf("expected active session ID %s, got %s", sess.ID, active.ID)
	}
}

func TestReAttach_DeterministicSemantics(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	checker := newMockDeviceChecker()
	checker.addDevice("DEV-001", "SUB-001", true)
	ipPool := network.NewIPPool()

	svc := NewService(repo, checker, ipPool)

	// Primeiro Attach
	sess1, err := svc.Attach(ctx, AttachRequest{DeviceID: "DEV-001", CellID: "CELL-SP-001"})
	if err != nil {
		t.Fatalf("unexpected attach error: %v", err)
	}

	// Re-attach para o mesmo Device (outra célula)
	sess2, err := svc.Attach(ctx, AttachRequest{DeviceID: "DEV-001", CellID: "CELL-SP-002"})
	if err != nil {
		t.Fatalf("unexpected re-attach error: %v", err)
	}

	// A invariante exige novo Session ID
	if sess2.ID == sess1.ID {
		t.Fatalf("expected different session ID on re-attach")
	}
	if sess2.Status != StatusConnected {
		t.Errorf("expected sess2 to be CONNECTED, got %s", sess2.Status)
	}

	// Sessão 1 deve ter sido encerrada com STALE_DISCONNECT
	oldSess, err := svc.FindByID(ctx, sess1.ID)
	if err != nil {
		t.Fatalf("unexpected error finding sess1: %v", err)
	}
	if oldSess.Status != StatusDisconnected {
		t.Errorf("expected old session to be DISCONNECTED, got %s", oldSess.Status)
	}
	if oldSess.DisconnectReason != DisconnectReasonStale {
		t.Errorf("expected STALE_DISCONNECT reason, got %s", oldSess.DisconnectReason)
	}
	if oldSess.ClosedAt == nil {
		t.Errorf("expected ClosedAt timestamp to be populated on stale disconnect")
	}

	// Exatamente UMA sessão conectada para o Device
	active, err := svc.GetActiveByDevice(ctx, "DEV-001")
	if err != nil {
		t.Fatalf("failed to get active session: %v", err)
	}
	if active.ID != sess2.ID {
		t.Errorf("expected active session to be sess2 (%s), got %s", sess2.ID, active.ID)
	}

	// O pool de IP deve ter exatamente 1 IP alocado no total para este cenário
	if ipPool.AllocatedCount() != 1 {
		t.Errorf("expected exactly 1 allocated IP in pool, got %d", ipPool.AllocatedCount())
	}
	if !ipPool.IsAllocated(sess2.IPAddress) {
		t.Errorf("expected sess2 IP %s to be allocated in pool", sess2.IPAddress)
	}
}

func TestHandover_Lifecycle(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	checker := newMockDeviceChecker()
	checker.addDevice("DEV-001", "SUB-001", true)
	ipPool := network.NewIPPool()

	svc := NewService(repo, checker, ipPool)

	sess, err := svc.Attach(ctx, AttachRequest{DeviceID: "DEV-001", CellID: "CELL-SP-001"})
	if err != nil {
		t.Fatalf("unexpected attach error: %v", err)
	}
	originalIP := sess.IPAddress
	originalID := sess.ID

	// Handover para célula inválida
	_, err = svc.Handover(ctx, sess.ID, "CELL-UNKNOWN")
	if !errors.Is(err, network.ErrCellNotFound) {
		t.Fatalf("expected ErrCellNotFound, got %v", err)
	}

	// Handover para a mesma célula (no-op bem-sucedido)
	sameCellSess, err := svc.Handover(ctx, sess.ID, "CELL-SP-001")
	if err != nil {
		t.Fatalf("unexpected error on same-cell handover: %v", err)
	}
	if sameCellSess.CellID != "CELL-SP-001" {
		t.Errorf("expected CELL-SP-001, got %s", sameCellSess.CellID)
	}

	// Handover para CELL-SP-003
	hoSess, err := svc.Handover(ctx, sess.ID, "CELL-SP-003")
	if err != nil {
		t.Fatalf("unexpected handover error: %v", err)
	}
	if hoSess.ID != originalID {
		t.Errorf("handover must preserve session ID: expected %s, got %s", originalID, hoSess.ID)
	}
	if hoSess.IPAddress != originalIP {
		t.Errorf("handover must preserve IP: expected %s, got %s", originalIP, hoSess.IPAddress)
	}
	if hoSess.CellID != "CELL-SP-003" {
		t.Errorf("expected target cell CELL-SP-003, got %s", hoSess.CellID)
	}
	if hoSess.Status != StatusConnected {
		t.Errorf("expected StatusConnected, got %s", hoSess.Status)
	}

	// Detach e posterior tentativa de handover
	if _, err := svc.Detach(ctx, sess.ID); err != nil {
		t.Fatalf("unexpected detach error: %v", err)
	}

	_, err = svc.Handover(ctx, sess.ID, "CELL-SP-002")
	if !errors.Is(err, ErrSessionNotConnected) {
		t.Fatalf("expected ErrSessionNotConnected on disconnected session, got %v", err)
	}
}

func TestDetach_LifecycleAndIdempotency(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	checker := newMockDeviceChecker()
	checker.addDevice("DEV-001", "SUB-001", true)
	ipPool := network.NewIPPool()

	svc := NewService(repo, checker, ipPool)

	sess, err := svc.Attach(ctx, AttachRequest{DeviceID: "DEV-001", CellID: "CELL-SP-001"})
	if err != nil {
		t.Fatalf("unexpected attach error: %v", err)
	}
	allocatedIP := sess.IPAddress

	// Detach 1: Desconexão voluntária
	detached, err := svc.Detach(ctx, sess.ID)
	if err != nil {
		t.Fatalf("unexpected detach error: %v", err)
	}
	if detached.Status != StatusDisconnected {
		t.Errorf("expected StatusDisconnected, got %s", detached.Status)
	}
	if detached.DisconnectReason != DisconnectReasonVoluntary {
		t.Errorf("expected VOLUNTARY_DETACH reason, got %s", detached.DisconnectReason)
	}
	if detached.ClosedAt == nil {
		t.Errorf("expected ClosedAt timestamp to be set")
	}

	// IP deve ter sido liberado no pool
	if ipPool.IsAllocated(allocatedIP) {
		t.Errorf("expected IP %s to be released from pool", allocatedIP)
	}
	if ipPool.AllocatedCount() != 0 {
		t.Errorf("expected 0 allocated IPs in pool, got %d", ipPool.AllocatedCount())
	}

	// Consulta de sessão ativa para o device deve retornar ErrSessionNotFound
	_, err = svc.GetActiveByDevice(ctx, "DEV-001")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound for detached device, got %v", err)
	}

	// Detach 2: Idempotência (não altera motivo nem libera IP novamente)
	secondDetach, err := svc.Detach(ctx, sess.ID)
	if err != nil {
		t.Fatalf("unexpected error on second detach: %v", err)
	}
	if secondDetach.Status != StatusDisconnected {
		t.Errorf("expected StatusDisconnected, got %s", secondDetach.Status)
	}
	if secondDetach.DisconnectReason != DisconnectReasonVoluntary {
		t.Errorf("expected reason to remain VOLUNTARY_DETACH, got %s", secondDetach.DisconnectReason)
	}
	if ipPool.AllocatedCount() != 0 {
		t.Errorf("expected 0 allocated IPs in pool, got %d", ipPool.AllocatedCount())
	}
}

func TestConcurrentAttach_SameDevice_Deterministic(t *testing.T) {
	// Requisito 4: Teste determinístico com múltiplos ATTACH concorrentes para o MESMO Device
	ctx := context.Background()
	repo := NewMemoryRepository()
	checker := newMockDeviceChecker()
	targetDeviceID := "DEV-CONCURRENT-001"
	checker.addDevice(targetDeviceID, "SUB-CONCURRENT-001", true)
	ipPool := network.NewIPPool()

	svc := NewService(repo, checker, ipPool)

	concurrency := 20
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	createdSessions := make([]*Session, concurrency)
	errorsList := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-startBarrier // Aguarda largada simultânea de todas as goroutines

			cellID := fmt.Sprintf("CELL-SP-00%d", (index%3)+1)
			s, err := svc.Attach(ctx, AttachRequest{
				DeviceID: targetDeviceID,
				CellID:   cellID,
			})
			createdSessions[index] = s
			errorsList[index] = err
		}(i)
	}

	// Dispara todas as goroutines simultaneamente
	close(startBarrier)
	wg.Wait()

	// Valida se nenhuma goroutine retornou erro inesperado
	for i, err := range errorsList {
		if err != nil {
			t.Fatalf("goroutine %d failed unexpectedly: %v", i, err)
		}
	}

	// 1. Ao final, deve existir EXATAMENTE UMA sessão CONNECTED vinculada ao dispositivo
	activeSession, err := svc.GetActiveByDevice(ctx, targetDeviceID)
	if err != nil {
		t.Fatalf("failed to retrieve final active session: %v", err)
	}
	if activeSession.Status != StatusConnected {
		t.Fatalf("expected final active session to be CONNECTED, got %s", activeSession.Status)
	}
	if activeSession.DisconnectReason != "" {
		t.Fatalf("active session must not have disconnect reason, got %s", activeSession.DisconnectReason)
	}

	// 2. No repositório, o activeByDevice aponta para essa sessão única
	if repo.ActiveCount() != 1 {
		t.Fatalf("expected repo activeCount to be exactly 1, got %d", repo.ActiveCount())
	}

	// 3. Todas as outras sessões criadas devem ter sido encerradas com STALE_DISCONNECT
	connectedCount := 0
	staleCount := 0

	for _, created := range createdSessions {
		currentSess, err := svc.FindByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("failed to query session %s: %v", created.ID, err)
		}

		if currentSess.ID == activeSession.ID {
			connectedCount++
			if currentSess.Status != StatusConnected {
				t.Errorf("session %s should be CONNECTED, got %s", currentSess.ID, currentSess.Status)
			}
		} else {
			staleCount++
			if currentSess.Status != StatusDisconnected {
				t.Errorf("superseded session %s must be DISCONNECTED, got %s", currentSess.ID, currentSess.Status)
			}
			if currentSess.DisconnectReason != DisconnectReasonStale {
				t.Errorf("superseded session %s reason must be STALE_DISCONNECT, got %s", currentSess.ID, currentSess.DisconnectReason)
			}
			if currentSess.ClosedAt == nil {
				t.Errorf("superseded session %s ClosedAt must be populated", currentSess.ID)
			}
		}
	}

	if connectedCount != 1 {
		t.Fatalf("expected exactly 1 connected session, found %d", connectedCount)
	}
	if staleCount != concurrency-1 {
		t.Fatalf("expected %d superseded sessions, found %d", concurrency-1, staleCount)
	}

	// 4. Não existem IPs simultaneamente alocados indevidamente (exatamente 1 IP no pool)
	if ipPool.AllocatedCount() != 1 {
		t.Fatalf("expected exactly 1 allocated IP in IPPool, got %d", ipPool.AllocatedCount())
	}
	if !ipPool.IsAllocated(activeSession.IPAddress) {
		t.Fatalf("expected active session IP %s to be allocated in pool", activeSession.IPAddress)
	}
}
