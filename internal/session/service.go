package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/carloska24/nexus-core-lab/internal/network"
)

// AttachRequest define a carga útil necessária para solicitar a anexação de um dispositivo à rede.
type AttachRequest struct {
	DeviceID string `json:"device_id"`
	CellID   string `json:"cell_id"`
}

// HandoverRequest define os dados para realizar o handover de uma sessão ativa.
type HandoverRequest struct {
	TargetCellID string `json:"target_cell_id"`
}

// Service coordena os casos de uso do domínio Session, assegurando atomicidade
// estrita nas transições de conectividade por dispositivo.
type Service struct {
	repo          Repository
	deviceChecker DeviceChecker
	ipPool        *network.IPPool
	emitter       EventEmitter

	deviceLocksMu sync.Mutex
	deviceLocks   map[string]*sync.Mutex
}

// NewService instancia um novo serviço de sessões, com emitter de eventos opcional.
func NewService(repo Repository, deviceChecker DeviceChecker, ipPool *network.IPPool, emitter ...EventEmitter) *Service {
	var em EventEmitter
	if len(emitter) > 0 {
		em = emitter[0]
	}
	return &Service{
		repo:          repo,
		deviceChecker: deviceChecker,
		ipPool:        ipPool,
		emitter:       em,
		deviceLocks:   make(map[string]*sync.Mutex),
	}
}

// lockDevice obtém um mutex exclusivo por deviceID para serializar atomicamente
// as operações de ciclo de vida de conectividade de um mesmo equipamento.
func (s *Service) lockDevice(deviceID string) func() {
	s.deviceLocksMu.Lock()
	mu, exists := s.deviceLocks[deviceID]
	if !exists {
		mu = &sync.Mutex{}
		s.deviceLocks[deviceID] = mu
	}
	s.deviceLocksMu.Unlock()

	mu.Lock()
	return mu.Unlock
}

// Attach cria uma nova sessão conectada para um equipamento elegível.
// Se o dispositivo já possuir uma sessão ativa, aplica a semântica determinística de re-attach:
// encerra a sessão anterior com STALE_DISCONNECT, libera seu IP e ativa a nova sessão.
func (s *Service) Attach(ctx context.Context, req AttachRequest) (*Session, error) {
	if _, err := network.FindCell(req.CellID); err != nil {
		return nil, err
	}

	devInfo, err := s.deviceChecker.CheckDeviceAttachable(ctx, req.DeviceID)
	if err != nil {
		return nil, err
	}

	unlock := s.lockDevice(req.DeviceID)
	defer unlock()

	// 1. Aloca novo IP do pool
	newIP, err := s.ipPool.Allocate()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate ip from pool: %w", err)
	}

	// 2. Cria instância da nova sessão
	newSession, err := New(req.DeviceID, devInfo.SubscriberID, req.CellID, newIP)
	if err != nil {
		_ = s.ipPool.Release(newIP)
		return nil, err
	}

	// 3. Persiste atomicamente no repositório (substitui anterior se re-attach)
	staleSession, err := s.repo.AttachSession(ctx, newSession)
	if err != nil {
		// Compensação: em caso de falha de persistência, libera novo IP alocado
		_ = s.ipPool.Release(newIP)
		return nil, fmt.Errorf("failed to attach session: %w", err)
	}

	// 4. Somente após sucesso na persistência: se foi re-attach, libera o IP anterior e emite STALE_DISCONNECT
	if staleSession != nil {
		_ = s.ipPool.Release(staleSession.IPAddress)
		if s.emitter != nil {
			s.emitter.EmitSessionEvent(ctx, SessionEventStaleDisconnect, staleSession)
		}
	}

	// 5. Emite evento de ATTACH para a nova sessão
	if s.emitter != nil {
		s.emitter.EmitSessionEvent(ctx, SessionEventAttach, newSession)
	}

	return newSession, nil
}

// Handover altera a célula de fixação de uma sessão CONNECTED sem alterar seu IP ou identificador.
// Handover para a célula atual é tratado como no-op com sucesso (sem emissão de evento).
func (s *Service) Handover(ctx context.Context, sessionID, targetCellID string) (*Session, error) {
	if _, err := network.FindCell(targetCellID); err != nil {
		return nil, err
	}

	sess, err := s.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	unlock := s.lockDevice(sess.DeviceID)
	defer unlock()

	// Reavalia o estado da sessão sob lock do dispositivo
	sess, err = s.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if sess.Status != StatusConnected {
		return nil, ErrSessionNotConnected
	}

	if sess.CellID == targetCellID {
		return sess, nil // no-op idempotente (não emite evento)
	}

	if err := sess.Handover(targetCellID); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, sess); err != nil {
		return nil, fmt.Errorf("failed to update session handover: %w", err)
	}

	if s.emitter != nil {
		s.emitter.EmitSessionEvent(ctx, SessionEventCellHandover, sess)
	}

	return sess, nil
}

// Detach encerra a sessão ativa, liberando o IP de volta ao pool e desvinculando o dispositivo.
// Operação idempotente: se já desconectada, retorna sucesso sem alterar o motivo, sem liberar o IP novamente e sem emitir evento.
func (s *Service) Detach(ctx context.Context, sessionID string) (*Session, error) {
	sess, err := s.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	unlock := s.lockDevice(sess.DeviceID)
	defer unlock()

	// Reavalia sob lock
	sess, err = s.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Idempotência: se já desconectada, não repete liberação nem emite evento adicional
	if sess.Status == StatusDisconnected {
		return sess, nil
	}

	sess.Detach(DisconnectReasonVoluntary)
	if err := s.repo.Update(ctx, sess); err != nil {
		// Se persistência falhar, NÃO libera o IP
		return nil, fmt.Errorf("failed to update detached session: %w", err)
	}

	// Somente após persistência confirmada: libera o IP no pool
	_ = s.ipPool.Release(sess.IPAddress)

	if s.emitter != nil {
		s.emitter.EmitSessionEvent(ctx, SessionEventDetach, sess)
	}

	return sess, nil
}

// FindByID busca os dados de uma sessão por seu UUID.
func (s *Service) FindByID(ctx context.Context, id string) (*Session, error) {
	return s.repo.FindByID(ctx, id)
}

// GetActiveByDevice busca a sessão atualmente conectada de um dispositivo.
func (s *Service) GetActiveByDevice(ctx context.Context, deviceID string) (*Session, error) {
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}
	return s.repo.FindActiveByDevice(ctx, deviceID)
}
