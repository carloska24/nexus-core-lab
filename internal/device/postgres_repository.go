package device

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

// NewPostgresRepository instancia um novo repositório PostgreSQL para Device.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Save persiste um novo dispositivo garantindo unicidade estrita de IMEI e integridade referencial com Subscriber.
func (r *PostgresRepository) Save(ctx context.Context, dev *Device) error {
	query := `
		INSERT INTO devices (
			id, subscriber_id, imei, technology, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		dev.ID,
		dev.SubscriberID,
		dev.IMEI,
		string(dev.Technology),
		string(dev.Status),
		dev.CreatedAt,
		dev.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			if strings.Contains(pgErr.ConstraintName, "imei") || strings.Contains(pgErr.Detail, "imei") {
				return ErrDuplicateIMEI
			}
			return ErrDuplicateIMEI
		}
		return err
	}

	return nil
}

// FindByID busca um dispositivo pelo seu UUID interno.
func (r *PostgresRepository) FindByID(ctx context.Context, id string) (*Device, error) {
	query := `
		SELECT id, subscriber_id, imei, technology, status, created_at, updated_at
		FROM devices
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanDevice(row)
}

// FindByIMEI busca um dispositivo pelo seu identificador de hardware IMEI de 15 dígitos.
func (r *PostgresRepository) FindByIMEI(ctx context.Context, imei string) (*Device, error) {
	query := `
		SELECT id, subscriber_id, imei, technology, status, created_at, updated_at
		FROM devices
		WHERE imei = $1
	`
	row := r.db.QueryRowContext(ctx, query, imei)
	return scanDevice(row)
}

// ListBySubscriber recupera todos os dispositivos vinculados ao assinante informado.
func (r *PostgresRepository) ListBySubscriber(ctx context.Context, subscriberID string) ([]*Device, error) {
	query := `
		SELECT id, subscriber_id, imei, technology, status, created_at, updated_at
		FROM devices
		WHERE subscriber_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, subscriberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Device
	for rows.Next() {
		dev, err := scanDeviceRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, dev)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDevice(scanner rowScanner) (*Device, error) {
	var dev Device
	var techStr, statusStr string

	err := scanner.Scan(
		&dev.ID,
		&dev.SubscriberID,
		&dev.IMEI,
		&techStr,
		&statusStr,
		&dev.CreatedAt,
		&dev.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	dev.Technology = Technology(techStr)
	dev.Status = Status(statusStr)
	return &dev, nil
}

func scanDeviceRow(rows *sql.Rows) (*Device, error) {
	var dev Device
	var techStr, statusStr string

	err := rows.Scan(
		&dev.ID,
		&dev.SubscriberID,
		&dev.IMEI,
		&techStr,
		&statusStr,
		&dev.CreatedAt,
		&dev.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	dev.Technology = Technology(techStr)
	dev.Status = Status(statusStr)
	return &dev, nil
}
