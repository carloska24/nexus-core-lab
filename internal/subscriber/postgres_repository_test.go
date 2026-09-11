package subscriber

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

func TestPostgresRepository_SubscriberPersistence(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewPostgresRepository(db)

	imsi := "724991234567890"
	msisdn := "+5511999990001"

	// Cleanup anterior eventual
	_, _ = db.ExecContext(ctx, "DELETE FROM sessions WHERE subscriber_id IN (SELECT id FROM subscribers WHERE imsi = $1)", imsi)
	_, _ = db.ExecContext(ctx, "DELETE FROM devices WHERE subscriber_id IN (SELECT id FROM subscribers WHERE imsi = $1)", imsi)
	_, _ = db.ExecContext(ctx, "DELETE FROM subscribers WHERE imsi = $1 OR msisdn = $2", imsi, msisdn)

	sub, err := New(imsi, msisdn)
	if err != nil {
		t.Fatalf("unexpected error creating subscriber: %v", err)
	}

	// 1. Save
	if err := repo.Save(ctx, sub); err != nil {
		t.Fatalf("failed to save subscriber: %v", err)
	}

	// 2. FindByID
	found, err := repo.FindByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("failed to find subscriber by ID: %v", err)
	}
	if found.IMSI != imsi || found.MSISDN != msisdn || found.Status != StatusPendingActivation {
		t.Errorf("mismatched subscriber data: %+v", found)
	}

	// 3. FindByIMSI
	foundIMSI, err := repo.FindByIMSI(ctx, imsi)
	if err != nil {
		t.Fatalf("failed to find by IMSI: %v", err)
	}
	if foundIMSI.ID != sub.ID {
		t.Errorf("expected ID %s, got %s", sub.ID, foundIMSI.ID)
	}

	// 4. Update
	if err := sub.Activate(); err != nil {
		t.Fatalf("failed to activate: %v", err)
	}
	if err := repo.Update(ctx, sub); err != nil {
		t.Fatalf("failed to update subscriber: %v", err)
	}

	updated, err := repo.FindByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("failed to find updated subscriber: %v", err)
	}
	if updated.Status != StatusActive {
		t.Errorf("expected ACTIVE status, got %s", updated.Status)
	}

	// 5. ExistsByIMSIOrMSISDN
	exists, err := repo.ExistsByIMSIOrMSISDN(ctx, imsi, "random")
	if err != nil || !exists {
		t.Errorf("expected IMSI to exist, got %v, err: %v", exists, err)
	}
	exists, err = repo.ExistsByIMSIOrMSISDN(ctx, "random", msisdn)
	if err != nil || !exists {
		t.Errorf("expected MSISDN to exist, got %v, err: %v", exists, err)
	}

	// 6. Unique IMSI
	dupIMSI, _ := New(imsi, "+5511999990002")
	err = repo.Save(ctx, dupIMSI)
	if !errors.Is(err, ErrDuplicateIMSI) {
		t.Errorf("expected ErrDuplicateIMSI, got %v", err)
	}

	// 7. Unique MSISDN
	dupMSISDN, _ := New("724991234567891", msisdn)
	err = repo.Save(ctx, dupMSISDN)
	if !errors.Is(err, ErrDuplicateMSISDN) {
		t.Errorf("expected ErrDuplicateMSISDN, got %v", err)
	}

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM subscribers WHERE id = $1", sub.ID)
}
