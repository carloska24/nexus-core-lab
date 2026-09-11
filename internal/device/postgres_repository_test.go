package device

import (
	"context"
	"database/sql"
	"errors"
	"os"
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

func TestPostgresRepository_DevicePersistence(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	subID := "11111111-1111-4111-8111-111111111111"
	imei1 := "490154203237510"
	imei2 := "490154203237520"

	// Insere subscriber pré-requisito
	now := time.Now().UTC()
	_, _ = db.ExecContext(ctx, "DELETE FROM sessions WHERE subscriber_id = $1", subID)
	_, _ = db.ExecContext(ctx, "DELETE FROM devices WHERE subscriber_id = $1", subID)
	_, _ = db.ExecContext(ctx, "DELETE FROM subscribers WHERE id = $1", subID)

	_, err := db.ExecContext(ctx, `
		INSERT INTO subscribers (id, imsi, msisdn, status, created_at, updated_at)
		VALUES ($1, '724991111111111', '+5511999991111', 'ACTIVE', $2, $2)
	`, subID, now)
	if err != nil {
		t.Fatalf("failed to insert prerequisite subscriber: %v", err)
	}

	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM sessions WHERE subscriber_id = $1", subID)
		_, _ = db.ExecContext(ctx, "DELETE FROM devices WHERE subscriber_id = $1", subID)
		_, _ = db.ExecContext(ctx, "DELETE FROM subscribers WHERE id = $1", subID)
	}()

	dev1, err := New(subID, imei1, TechLTE)
	if err != nil {
		t.Fatalf("unexpected error creating device: %v", err)
	}

	// 1. Save
	if err := repo.Save(ctx, dev1); err != nil {
		t.Fatalf("failed to save device: %v", err)
	}

	// 2. FindByID
	found, err := repo.FindByID(ctx, dev1.ID)
	if err != nil {
		t.Fatalf("failed to find device by ID: %v", err)
	}
	if found.IMEI != imei1 || found.Technology != TechLTE || found.Status != StatusRegistered {
		t.Errorf("unexpected device data: %+v", found)
	}

	// 3. FindByIMEI
	foundIMEI, err := repo.FindByIMEI(ctx, imei1)
	if err != nil {
		t.Fatalf("failed to find device by IMEI: %v", err)
	}
	if foundIMEI.ID != dev1.ID {
		t.Errorf("expected ID %s, got %s", dev1.ID, foundIMEI.ID)
	}

	// 4. ListBySubscriber
	dev2, _ := New(subID, imei2, Tech5G)
	if err := repo.Save(ctx, dev2); err != nil {
		t.Fatalf("failed to save second device: %v", err)
	}

	list, err := repo.ListBySubscriber(ctx, subID)
	if err != nil {
		t.Fatalf("failed to list devices by subscriber: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 devices, got %d", len(list))
	}

	// 5. Unique IMEI
	dupDev, _ := New(subID, imei1, Tech5G)
	err = repo.Save(ctx, dupDev)
	if !errors.Is(err, ErrDuplicateIMEI) {
		t.Errorf("expected ErrDuplicateIMEI, got %v", err)
	}

	// 6. FK constraint: dispositivo apontando para subscriber inexistente
	invalidSubDev, _ := New("99999999-9999-4999-8999-999999999999", "490154203237530", TechLTE)
	err = repo.Save(ctx, invalidSubDev)
	if err == nil {
		t.Errorf("expected FK error when inserting device for non-existent subscriber")
	}
}
