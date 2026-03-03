package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// UserRepository implements user.Repository using PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Save creates or updates a user.
func (r *UserRepository) Save(ctx context.Context, entity *user.User) error {
	query := `
		INSERT INTO users (id, name, email, phone, firebase_token, notifications_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			firebase_token = EXCLUDED.firebase_token,
			notifications_enabled = EXCLUDED.notifications_enabled,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		entity.ID,
		entity.Name,
		entity.Email,
		entity.Phone,
		entity.FirebaseToken,
		entity.NotificationsEnabled,
		entity.CreatedAt,
		entity.UpdatedAt,
	)
	return err
}

// FindByID retrieves a user by ID.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	query := `
		SELECT id, name, email, phone, firebase_token, notifications_enabled, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var entity user.User
	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&entity.FirebaseToken,
		&entity.NotificationsEnabled,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}

	return &entity, nil
}

// FindByEmail retrieves a user by email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT id, name, email, phone, firebase_token, notifications_enabled, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var entity user.User
	if err := r.db.QueryRowContext(ctx, query, email).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&entity.FirebaseToken,
		&entity.NotificationsEnabled,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}

	return &entity, nil
}

// FindAll retrieves all users.
func (r *UserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	query := `
		SELECT id, name, email, phone, firebase_token, notifications_enabled, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*user.User
	for rows.Next() {
		var entity user.User
		if err := rows.Scan(
			&entity.ID,
			&entity.Name,
			&entity.Email,
			&entity.Phone,
			&entity.FirebaseToken,
			&entity.NotificationsEnabled,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, &entity)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Delete removes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

// Exists checks if a user exists by ID.
func (r *UserRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
