package session

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func getTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	return db
}

func setupSubscriberAndDevice(t *testing.T, db *sql.DB, ctx context.Context, subID, devID, imsi, msisdn, imei string) func() {
	now := time.Now().UTC()

	_, _ = db.ExecContext(ctx, "DELETE FROM sessions WHERE device_id = $1", devID)
	_, _ = db.ExecContext(ctx, "DELETE FROM devices WHERE id = $1", devID)
	_, _ = db.ExecContext(ctx, "DELETE FROM subscribers WHERE id = $1", subID)

	_, err := db.ExecContext(ctx, `
		INSERT INTO subscribers (id, imsi, msisdn, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'ACTIVE', $4, $4)
	`, subID, imsi, msisdn, now)
	if err != nil {
		t.Fatalf("failed to insert test subscriber: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO devices (id, subscriber_id, imei, technology, status, created_at, updated_at)
		VALUES ($1, $2, $3, '5G', 'REGISTERED', $4, $4)
	`, devID, subID, imei, now)
	if err != nil {
		t.Fatalf("failed to insert test device: %v", err)
	}

	return func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM sessions WHERE device_id = $1", devID)
		_, _ = db.ExecContext(ctx, "DELETE FROM devices WHERE id = $1", devID)
		_, _ = db.ExecContext(ctx, "DELETE FROM subscribers WHERE id = $1", subID)
	}
}

func TestPostgresRepository_SessionLifecycle(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	subID := "22222222-2222-4222-8222-222222222222"
	devID := "33333333-3333-4333-8333-333333333333"
	cleanup := setupSubscriberAndDevice(t, db, ctx, subID, devID, "724992222222222", "+5511999992222", "490154203237666")
	defer cleanup()

	// 1. First Attach via AttachSession
	s1, err := New(devID, subID, "CELL-SP-001", "10.45.0.10")
	if err != nil {
		t.Fatalf("unexpected error creating session 1: %v", err)
	}

	stale, err := repo.AttachSession(ctx, s1)
	if err != nil {
		t.Fatalf("failed to attach first session: %v", err)
	}
	if stale != nil {
		t.Errorf("expected nil stale session on first attach, got %+v", stale)
	}

	// Consulta sessão ativa
	active, err := repo.FindActiveByDevice(ctx, devID)
	if err != nil {
		t.Fatalf("failed to find active session: %v", err)
	}
	if active.ID != s1.ID || active.Status != StatusConnected {
		t.Errorf("unexpected active session: %+v", active)
	}

	// 2. Handover
	if err := active.Handover("CELL-SP-002"); err != nil {
		t.Fatalf("failed to prepare handover: %v", err)
	}
	if err := repo.Update(ctx, active); err != nil {
		t.Fatalf("failed to update session on handover: %v", err)
	}

	ho, err := repo.FindByID(ctx, s1.ID)
	if err != nil {
		t.Fatalf("failed to query after handover: %v", err)
	}
	if ho.CellID != "CELL-SP-002" || ho.IPAddress != "10.45.0.10" {
		t.Errorf("handover data mismatch: %+v", ho)
	}

	// 3. Re-attach (novo attach para o mesmo device)
	s2, err := New(devID, subID, "CELL-SP-003", "10.45.0.11")
	if err != nil {
		t.Fatalf("unexpected error creating session 2: %v", err)
	}

	stale, err = repo.AttachSession(ctx, s2)
	if err != nil {
		t.Fatalf("failed to attach session 2 (re-attach): %v", err)
	}
	if stale == nil {
		t.Fatalf("expected superseded stale session, got nil")
	}
	if stale.ID != s1.ID || stale.DisconnectReason != DisconnectReasonStale || stale.Status != StatusDisconnected {
		t.Errorf("unexpected stale session state: %+v", stale)
	}

	// Verifica se s1 está DISCONNECTED no banco
	oldS1, err := repo.FindByID(ctx, s1.ID)
	if err != nil {
		t.Fatalf("failed to find s1 after re-attach: %v", err)
	}
	if oldS1.Status != StatusDisconnected || oldS1.DisconnectReason != DisconnectReasonStale {
		t.Errorf("expected old session to be STALE_DISCONNECT, got %+v", oldS1)
	}

	// Verifica se s2 é a nova ativa
	active2, err := repo.FindActiveByDevice(ctx, devID)
	if err != nil {
		t.Fatalf("failed to find active session after re-attach: %v", err)
	}
	if active2.ID != s2.ID {
		t.Errorf("expected s2 to be active, got %s", active2.ID)
	}

	// 4. Detach voluntário na s2
	active2.Detach(DisconnectReasonVoluntary)
	if err := repo.Update(ctx, active2); err != nil {
		t.Fatalf("failed to update detached session: %v", err)
	}

	detachedS2, err := repo.FindByID(ctx, s2.ID)
	if err != nil {
		t.Fatalf("failed to find detached s2: %v", err)
	}
	if detachedS2.Status != StatusDisconnected || detachedS2.DisconnectReason != DisconnectReasonVoluntary {
		t.Errorf("expected voluntary detach on s2, got %+v", detachedS2)
	}

	// Não deve mais haver sessão CONNECTED para o dispositivo
	_, err = repo.FindActiveByDevice(ctx, devID)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound for active session, got %v", err)
	}
}

func TestPostgresRepository_ActiveCountAndError(t *testing.T) {
	db := getTestDB(t)

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	count, err := repo.ActiveCount(ctx)
	if err != nil {
		t.Fatalf("unexpected error on ActiveCount: %v", err)
	}
	if count < 0 {
		t.Errorf("expected non-negative count, got %d", count)
	}

	// Simula erro de banco fechando a conexão
	_ = db.Close()
	_, err = repo.ActiveCount(ctx)
	if err == nil {
		t.Fatalf("expected error on closed db, got nil")
	}
}

func TestPostgresRepository_PartialUniqueConstraints(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	subID := "44444444-4444-4444-8444-444444444444"
	dev1ID := "55555555-5555-4555-8555-555555555555"
	dev2ID := "66666666-6666-4666-8666-666666666666"

	// Insere subscriber e 2 devices
	now := time.Now().UTC()
	_, _ = db.ExecContext(ctx, "DELETE FROM sessions WHERE subscriber_id = $1", subID)
	_, _ = db.ExecContext(ctx, "DELETE FROM devices WHERE subscriber_id = $1", subID)
	_, _ = db.ExecContext(ctx, "DELETE FROM subscribers WHERE id = $1", subID)

	_, _ = db.ExecContext(ctx, `INSERT INTO subscribers (id, imsi, msisdn, status, created_at, updated_at) VALUES ($1, '724994444444444', '+5511999994444', 'ACTIVE', $2, $2)`, subID, now)
	_, _ = db.ExecContext(ctx, `INSERT INTO devices (id, subscriber_id, imei, technology, status, created_at, updated_at) VALUES ($1, $2, '490154203237771', 'LTE', 'REGISTERED', $3, $3)`, dev1ID, subID, now)
	_, _ = db.ExecContext(ctx, `INSERT INTO devices (id, subscriber_id, imei, technology, status, created_at, updated_at) VALUES ($1, $2, '490154203237772', 'LTE', 'REGISTERED', $3, $3)`, dev2ID, subID, now)

	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM sessions WHERE subscriber_id = $1", subID)
		_, _ = db.ExecContext(ctx, "DELETE FROM devices WHERE subscriber_id = $1", subID)
		_, _ = db.ExecContext(ctx, "DELETE FROM subscribers WHERE id = $1", subID)
	}()

	s1, _ := New(dev1ID, subID, "CELL-SP-001", "10.45.0.50")
	if err := repo.Save(ctx, s1); err != nil {
		t.Fatalf("failed to save s1: %v", err)
	}

	// 1. Violação de partial unique device_id: inserir outra sessão CONNECTED diretamente via Save para dev1
	dupDevSession, _ := New(dev1ID, subID, "CELL-SP-002", "10.45.0.51")
	err := repo.Save(ctx, dupDevSession)
	if err == nil {
		t.Errorf("expected partial unique index violation for duplicate CONNECTED device_id, got nil")
	}

	// 2. Violação de partial unique ip_address: inserir outra sessão CONNECTED com o mesmo IP (10.45.0.50) para dev2
	dupIPSession, _ := New(dev2ID, subID, "CELL-SP-001", "10.45.0.50")
	err = repo.Save(ctx, dupIPSession)
	if err == nil {
		t.Errorf("expected partial unique index violation for duplicate CONNECTED ip_address, got nil")
	}
}

func TestPostgresRepository_ConcurrentAttach(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	subID := "77777777-7777-4777-8777-777777777777"
	devID := "88888888-8888-4888-8888-888888888888"
	cleanup := setupSubscriberAndDevice(t, db, ctx, subID, devID, "724997777777777", "+5511999997777", "490154203237888")
	defer cleanup()

	concurrency := 10
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startBarrier

			ip := fmt.Sprintf("10.45.1.%d", idx+2)
			s, err := New(devID, subID, "CELL-SP-001", ip)
			if err != nil {
				return
			}
			_, _ = repo.AttachSession(ctx, s)
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	// Ao final, no PostgreSQL deve haver exatamente UMA sessão CONNECTED para devID
	query := `SELECT COUNT(*) FROM sessions WHERE device_id = $1 AND status = 'CONNECTED'`
	var connectedCount int
	if err := db.QueryRowContext(ctx, query, devID).Scan(&connectedCount); err != nil {
		t.Fatalf("failed to query connected count: %v", err)
	}
	if connectedCount != 1 {
		t.Errorf("expected exactly 1 CONNECTED session for device after concurrent attach, got %d", connectedCount)
	}
}
