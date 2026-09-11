package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type migrationFile struct {
	version int
	name    string
	path    string
}

func main() {
	defaultURL := os.Getenv("DATABASE_URL")
	if defaultURL == "" {
		defaultURL = "postgres://nexus:nexus@localhost:5433/nexus_core_lab?sslmode=disable"
	}

	url := flag.String("url", defaultURL, "PostgreSQL connection URL")
	dir := flag.String("dir", "migrations", "Directory containing .sql migrations")
	up := flag.Bool("up", false, "Apply pending migrations")
	down := flag.Bool("down", false, "Rollback the latest applied migration")
	flag.Parse()

	if (!*up && !*down) || (*up && *down) {
		fmt.Println("Uso: go run ./cmd/migrate [-url <db_url>] -up | -down")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", *url)
	if err != nil {
		log.Fatalf("Erro ao abrir conexão com o banco: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Erro ao conectar ao PostgreSQL: %v", err)
	}

	if err := ensureMigrationTable(ctx, db); err != nil {
		log.Fatalf("Erro ao preparar tabela de schema_migrations: %v", err)
	}

	if *up {
		if err := runUp(ctx, db, *dir); err != nil {
			log.Fatalf("Falha ao aplicar migrações: %v", err)
		}
	} else if *down {
		if err := runDown(ctx, db, *dir); err != nil {
			log.Fatalf("Falha ao reverter migração: %v", err)
		}
	}
}

func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL
		);
	`
	_, err := db.ExecContext(ctx, query)
	return err
}

func runUp(ctx context.Context, db *sql.DB, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir %s: %w", dir, err)
	}

	var upMigrations []migrationFile
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".up.sql") {
			continue
		}
		parts := strings.SplitN(f.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		upMigrations = append(upMigrations, migrationFile{
			version: v,
			name:    f.Name(),
			path:    filepath.Join(dir, f.Name()),
		})
	}

	sort.Slice(upMigrations, func(i, j int) bool {
		return upMigrations[i].version < upMigrations[j].version
	})

	appliedMap := make(map[int]bool)
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return err
		}
		appliedMap[v] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	appliedCount := 0
	for _, m := range upMigrations {
		if appliedMap[m.version] {
			continue
		}

		content, err := os.ReadFile(m.path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", m.path, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", m.name, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)", m.version, time.Now().UTC()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", m.version, err)
		}

		fmt.Printf("[MIGRATE] Applied version %06d (%s)\n", m.version, m.name)
		appliedCount++
	}

	if appliedCount == 0 {
		fmt.Println("[MIGRATE] Database schema is up to date (no migrations to apply).")
	}

	return nil
}

func runDown(ctx context.Context, db *sql.DB, dir string) error {
	var maxVersion sql.NullInt64
	err := db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&maxVersion)
	if err != nil {
		return fmt.Errorf("failed to query current schema version: %w", err)
	}

	if !maxVersion.Valid {
		fmt.Println("[MIGRATE] No applied migrations to rollback.")
		return nil
	}

	currentVersion := int(maxVersion.Int64)
	pattern := fmt.Sprintf("%06d_*.down.sql", currentVersion)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(matches) == 0 {
		return fmt.Errorf("down migration file not found for version %06d in %s", currentVersion, dir)
	}

	downFile := matches[0]
	content, err := os.ReadFile(downFile)
	if err != nil {
		return fmt.Errorf("failed to read down migration %s: %w", downFile, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to execute down migration %s: %w", filepath.Base(downFile), err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = $1", currentVersion); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to remove version %d from schema_migrations: %w", currentVersion, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit down migration %d: %w", currentVersion, err)
	}

	fmt.Printf("[MIGRATE] Rolled back version %06d (%s)\n", currentVersion, filepath.Base(downFile))
	return nil
}
