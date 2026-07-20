package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
)

type DeviceTokenRepository struct {
	db *sql.DB
}

func NewDeviceTokenRepository(db *sql.DB) *DeviceTokenRepository {
	return &DeviceTokenRepository{db: db}
}

const pgErrUniqueViolation = "23505"

func (r *DeviceTokenRepository) Save(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error) {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	query := `
        INSERT INTO user_device_tokens
            (id, user_id, token, enabled, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (token) DO NOTHING
        RETURNING id
    `
	var id string
	err := r.db.QueryRowContext(ctx, query,
		t.ID, t.UserID, t.Token, t.Enabled, t.CreatedAt, t.UpdatedAt,
	).Scan(&id)
	if err == nil {
		return r.FindByID(ctx, id)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// No insert happened: token already exists. Determine whether this is a
	// same-user re-registration (allowed) or a cross-user collision (forbidden).
	existing, ferr := r.findByToken(ctx, t.Token)
	if ferr != nil {
		return nil, ferr
	}
	if existing.UserID != t.UserID {
		return nil, devicetoken.ErrConflict
	}
	return existing, nil
}

func (r *DeviceTokenRepository) FindByID(ctx context.Context, id string) (*devicetoken.DeviceToken, error) {
	return r.scanOne(ctx,
		`SELECT id, user_id, token, enabled, created_at, updated_at, last_used_at
         FROM user_device_tokens WHERE id = $1`, id)
}

func (r *DeviceTokenRepository) findByToken(ctx context.Context, token string) (*devicetoken.DeviceToken, error) {
	return r.scanOne(ctx,
		`SELECT id, user_id, token, enabled, created_at, updated_at, last_used_at
         FROM user_device_tokens WHERE token = $1`, token)
}

func (r *DeviceTokenRepository) FindByUserID(ctx context.Context, userID string) ([]*devicetoken.DeviceToken, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, token, enabled, created_at, updated_at, last_used_at
         FROM user_device_tokens WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*devicetoken.DeviceToken
	for rows.Next() {
		dt, err := scanDeviceTokenRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, dt)
	}
	return out, rows.Err()
}

func (r *DeviceTokenRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_device_tokens WHERE id = $1`, id)
	return err
}

func (r *DeviceTokenRepository) SetEnabled(ctx context.Context, id string, enabled bool) (*devicetoken.DeviceToken, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE user_device_tokens SET enabled = $1, updated_at = $2 WHERE id = $3`,
		enabled, time.Now(), id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, devicetoken.ErrNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *DeviceTokenRepository) scanOne(ctx context.Context, query string, args ...any) (*devicetoken.DeviceToken, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	dt, err := scanDeviceTokenRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, devicetoken.ErrNotFound
	}
	return dt, err
}

func scanDeviceTokenRow(row rowScanner) (*devicetoken.DeviceToken, error) {
	var dt devicetoken.DeviceToken
	var lastUsed sql.NullTime
	if err := row.Scan(
		&dt.ID, &dt.UserID, &dt.Token, &dt.Enabled,
		&dt.CreatedAt, &dt.UpdatedAt, &lastUsed,
	); err != nil {
		return nil, err
	}
	if lastUsed.Valid {
		dt.LastUsedAt = &lastUsed.Time
	}
	return &dt, nil
}
