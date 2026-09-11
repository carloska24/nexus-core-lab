package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config define os parâmetros de conexão e pool para o PostgreSQL.
type Config struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DefaultConfig retorna parâmetros seguros de conexão padrão.
func DefaultConfig(url string) Config {
	return Config{
		URL:             url,
		MaxOpenConns:    25,
		MaxIdleConns:    25,
		ConnMaxLifetime: 5 * time.Minute,
	}
}

// Open inicializa o pool de conexões com PostgreSQL e valida conectividade via PingContext.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("database url cannot be empty")
	}

	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}

// ValidateSchema verifica se o banco de dados possui o esquema de migrações atualizado.
func ValidateSchema(ctx context.Context, db *sql.DB, expectedVersion int) error {
	var exists bool
	checkTableQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'schema_migrations'
		);
	`
	if err := db.QueryRowContext(ctx, checkTableQuery).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check schema_migrations table: %w", err)
	}
	if !exists {
		return fmt.Errorf("schema_migrations table does not exist")
	}

	var maxVersion sql.NullInt64
	err := db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&maxVersion)
	if err != nil {
		return fmt.Errorf("failed to query max schema version: %w", err)
	}

	if !maxVersion.Valid || int(maxVersion.Int64) < expectedVersion {
		current := 0
		if maxVersion.Valid {
			current = int(maxVersion.Int64)
		}
		return fmt.Errorf("database schema is out of date (current version: %d, expected: %d)", current, expectedVersion)
	}

	return nil
}
