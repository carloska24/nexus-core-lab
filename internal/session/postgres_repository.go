package session

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// PostgresRepository implementa Repository utilizando PostgreSQL via database/sql.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository instancia um novo repositório PostgreSQL para Session.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Save persiste uma nova sessão.
func (r *PostgresRepository) Save(ctx context.Context, s *Session) error {
	query := `
		INSERT INTO sessions (
			id, device_id, subscriber_id, cell_id, ip_address, status,
			disconnect_reason, attached_at, updated_at, closed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	var discReason *string
	if s.DisconnectReason != "" {
		discReason = &s.DisconnectReason
	}

	_, err := r.db.ExecContext(ctx, query,
		s.ID,
		s.DeviceID,
		s.SubscriberID,
		s.CellID,
		s.IPAddress,
		string(s.Status),
		discReason,
		s.AttachedAt,
		s.UpdatedAt,
		s.ClosedAt,
	)
	return err
}

// Update atualiza célula, status, motivo de desconexão e timestamps de encerramento da sessão.
func (r *PostgresRepository) Update(ctx context.Context, s *Session) error {
	query := `
		UPDATE sessions
		SET cell_id = $1, status = $2, disconnect_reason = $3, updated_at = $4, closed_at = $5
		WHERE id = $6
	`
	var discReason *string
	if s.DisconnectReason != "" {
		discReason = &s.DisconnectReason
	}

	res, err := r.db.ExecContext(ctx, query,
		s.CellID,
		string(s.Status),
		discReason,
		s.UpdatedAt,
		s.ClosedAt,
		s.ID,
	)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// FindByID busca uma sessão por seu UUID.
func (r *PostgresRepository) FindByID(ctx context.Context, id string) (*Session, error) {
	query := `
		SELECT id, device_id, subscriber_id, cell_id, ip_address, status,
		       disconnect_reason, attached_at, updated_at, closed_at
		FROM sessions
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanSession(row)
}

// FindActiveByDevice recupera a sessão atualmente CONNECTED vinculada ao dispositivo informado.
func (r *PostgresRepository) FindActiveByDevice(ctx context.Context, deviceID string) (*Session, error) {
	query := `
		SELECT id, device_id, subscriber_id, cell_id, ip_address, status,
		       disconnect_reason, attached_at, updated_at, closed_at
		FROM sessions
		WHERE device_id = $1 AND status = 'CONNECTED'
	`
	row := r.db.QueryRowContext(ctx, query, deviceID)
	return scanSession(row)
}

// AttachSession persiste a nova sessão e substitui atomicamente qualquer sessão CONNECTED
// anterior do dispositivo utilizando transação SQL local com bloqueio FOR UPDATE.
func (r *PostgresRepository) AttachSession(ctx context.Context, newSession *Session) (*Session, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Busca e bloqueia eventual sessão CONNECTED ativa para o mesmo equipamento
	selectQuery := `
		SELECT id, device_id, subscriber_id, cell_id, ip_address, status,
		       disconnect_reason, attached_at, updated_at, closed_at
		FROM sessions
		WHERE device_id = $1 AND status = 'CONNECTED'
		FOR UPDATE
	`
	row := tx.QueryRowContext(ctx, selectQuery, newSession.DeviceID)
	var staleSession *Session
	old, err := scanSession(row)
	if err != nil && !errors.Is(err, ErrSessionNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	if old != nil {
		// 2. Marca a sessão anterior como DISCONNECTED por STALE_DISCONNECT
		updateQuery := `
			UPDATE sessions
			SET status = 'DISCONNECTED', disconnect_reason = $1, updated_at = $2, closed_at = $3
			WHERE id = $4
		`
		reason := DisconnectReasonStale
		if _, err := tx.ExecContext(ctx, updateQuery, reason, now, now, old.ID); err != nil {
			return nil, err
		}
		old.Status = StatusDisconnected
		old.DisconnectReason = reason
		old.UpdatedAt = now
		old.ClosedAt = &now
		staleSession = old
	}

	// 3. Insere a nova sessão CONNECTED
	insertQuery := `
		INSERT INTO sessions (
			id, device_id, subscriber_id, cell_id, ip_address, status,
			disconnect_reason, attached_at, updated_at, closed_at
		) VALUES ($1, $2, $3, $4, $5, $6, NULL, $7, $8, NULL)
	`
	_, err = tx.ExecContext(ctx, insertQuery,
		newSession.ID,
		newSession.DeviceID,
		newSession.SubscriberID,
		newSession.CellID,
		newSession.IPAddress,
		string(newSession.Status),
		newSession.AttachedAt,
		newSession.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// 4. Efetiva transação atomicamente
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return staleSession, nil
}

// ActiveCount retorna a contagem exata de sessões ativas persistidas com status CONNECTED.
// Propaga explicitamente qualquer falha de SQL sem mascaramento.
func (r *PostgresRepository) ActiveCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE status = 'CONNECTED'`
	var count int
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(scanner rowScanner) (*Session, error) {
	var s Session
	var statusStr string
	var discReason sql.NullString
	var closedAt sql.NullTime

	err := scanner.Scan(
		&s.ID,
		&s.DeviceID,
		&s.SubscriberID,
		&s.CellID,
		&s.IPAddress,
		&statusStr,
		&discReason,
		&s.AttachedAt,
		&s.UpdatedAt,
		&closedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	s.Status = Status(statusStr)
	if discReason.Valid {
		s.DisconnectReason = discReason.String
	}
	if closedAt.Valid {
		t := closedAt.Time.UTC()
		s.ClosedAt = &t
	}

	return &s, nil
}
