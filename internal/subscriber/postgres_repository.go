package subscriber

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresRepository implementa Repository utilizando PostgreSQL via database/sql.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository instancia um repositório PostgreSQL para Subscriber.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Save persiste um novo assinante com garantia de unicidade de IMSI e MSISDN.
func (r *PostgresRepository) Save(ctx context.Context, sub *Subscriber) error {
	query := `
		INSERT INTO subscribers (
			id, imsi, msisdn, status, suspension_reason, deactivation_reason, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	var suspReason, deactReason *string
	if sub.SuspensionReason != "" {
		suspReason = &sub.SuspensionReason
	}
	if sub.DeactivationReason != "" {
		deactReason = &sub.DeactivationReason
	}

	_, err := r.db.ExecContext(ctx, query,
		sub.ID,
		sub.IMSI,
		sub.MSISDN,
		string(sub.Status),
		suspReason,
		deactReason,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			if strings.Contains(pgErr.ConstraintName, "imsi") || strings.Contains(pgErr.Detail, "imsi") {
				return ErrDuplicateIMSI
			}
			if strings.Contains(pgErr.ConstraintName, "msisdn") || strings.Contains(pgErr.Detail, "msisdn") {
				return ErrDuplicateMSISDN
			}
			return ErrDuplicateIMSI
		}
		return err
	}

	return nil
}

// Update atualiza status, motivos de suspensão/desativação e timestamps.
func (r *PostgresRepository) Update(ctx context.Context, sub *Subscriber) error {
	query := `
		UPDATE subscribers
		SET status = $1, suspension_reason = $2, deactivation_reason = $3, updated_at = $4
		WHERE id = $5
	`
	var suspReason, deactReason *string
	if sub.SuspensionReason != "" {
		suspReason = &sub.SuspensionReason
	}
	if sub.DeactivationReason != "" {
		deactReason = &sub.DeactivationReason
	}

	res, err := r.db.ExecContext(ctx, query,
		string(sub.Status),
		suspReason,
		deactReason,
		sub.UpdatedAt,
		sub.ID,
	)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSubscriberNotFound
	}

	return nil
}

// FindByID busca um assinante pelo ID interno UUID.
func (r *PostgresRepository) FindByID(ctx context.Context, id string) (*Subscriber, error) {
	query := `
		SELECT id, imsi, msisdn, status, suspension_reason, deactivation_reason, created_at, updated_at
		FROM subscribers
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanSubscriber(row)
}

// FindByIMSI busca um assinante pelo seu IMSI.
func (r *PostgresRepository) FindByIMSI(ctx context.Context, imsi string) (*Subscriber, error) {
	query := `
		SELECT id, imsi, msisdn, status, suspension_reason, deactivation_reason, created_at, updated_at
		FROM subscribers
		WHERE imsi = $1
	`
	row := r.db.QueryRowContext(ctx, query, imsi)
	return scanSubscriber(row)
}

// List retorna os assinantes cadastrados, com filtro opcional por status.
func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]*Subscriber, error) {
	var query string
	var args []any

	if filter.Status != "" {
		query = `
			SELECT id, imsi, msisdn, status, suspension_reason, deactivation_reason, created_at, updated_at
			FROM subscribers
			WHERE status = $1
			ORDER BY created_at ASC
		`
		args = append(args, string(filter.Status))
	} else {
		query = `
			SELECT id, imsi, msisdn, status, suspension_reason, deactivation_reason, created_at, updated_at
			FROM subscribers
			ORDER BY created_at ASC
		`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Subscriber
	for rows.Next() {
		sub, err := scanSubscriberRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ExistsByIMSIOrMSISDN verifica se já existe algum assinante com o IMSI ou MSISDN fornecido.
func (r *PostgresRepository) ExistsByIMSIOrMSISDN(ctx context.Context, imsi, msisdn string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM subscribers WHERE imsi = $1 OR msisdn = $2
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, imsi, msisdn).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSubscriber(scanner rowScanner) (*Subscriber, error) {
	var sub Subscriber
	var statusStr string
	var suspReason, deactReason sql.NullString

	err := scanner.Scan(
		&sub.ID,
		&sub.IMSI,
		&sub.MSISDN,
		&statusStr,
		&suspReason,
		&deactReason,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubscriberNotFound
		}
		return nil, err
	}

	sub.Status = Status(statusStr)
	if suspReason.Valid {
		sub.SuspensionReason = suspReason.String
	}
	if deactReason.Valid {
		sub.DeactivationReason = deactReason.String
	}

	return &sub, nil
}

func scanSubscriberRow(rows *sql.Rows) (*Subscriber, error) {
	var sub Subscriber
	var statusStr string
	var suspReason, deactReason sql.NullString

	err := rows.Scan(
		&sub.ID,
		&sub.IMSI,
		&sub.MSISDN,
		&statusStr,
		&suspReason,
		&deactReason,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	sub.Status = Status(statusStr)
	if suspReason.Valid {
		sub.SuspensionReason = suspReason.String
	}
	if deactReason.Valid {
		sub.DeactivationReason = deactReason.String
	}

	return &sub, nil
}
