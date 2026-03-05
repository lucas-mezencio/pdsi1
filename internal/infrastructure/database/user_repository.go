package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

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
		INSERT INTO users (id, name, email, phone, firebase_token, notifications_enabled, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			firebase_token = EXCLUDED.firebase_token,
			notifications_enabled = EXCLUDED.notifications_enabled,
			role = EXCLUDED.role,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		entity.ID,
		entity.Name,
		entity.Email,
		entity.Phone,
		entity.FirebaseToken,
		entity.NotificationsEnabled,
		string(entity.Role),
		entity.CreatedAt,
		entity.UpdatedAt,
	)
	return err
}

// FindByID retrieves a user by ID.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	query := `
		SELECT id, name, email, phone, firebase_token, notifications_enabled, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var entity user.User
	var role string
	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&entity.FirebaseToken,
		&entity.NotificationsEnabled,
		&role,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}

	entity.Role = user.Role(role)
	return &entity, nil
}

// FindByEmail retrieves a user by email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT id, name, email, phone, firebase_token, notifications_enabled, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var entity user.User
	var role string
	if err := r.db.QueryRowContext(ctx, query, email).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&entity.FirebaseToken,
		&entity.NotificationsEnabled,
		&role,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}

	entity.Role = user.Role(role)
	return &entity, nil
}

// FindAll retrieves all users.
func (r *UserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	query := `
		SELECT id, name, email, phone, firebase_token, notifications_enabled, role, created_at, updated_at
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
		entity, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, entity)
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

// FindCaregivers retrieves all caregivers linked to an elderly user.
func (r *UserRepository) FindCaregivers(ctx context.Context, elderlyID string) ([]*user.User, error) {
	query := `
		SELECT u.id, u.name, u.email, u.phone, u.firebase_token, u.notifications_enabled, u.role, u.created_at, u.updated_at
		FROM users u
		INNER JOIN user_links ul ON ul.caregiver_id = u.id
		WHERE ul.elderly_id = $1
		ORDER BY u.name
	`
	return r.queryUsers(ctx, query, elderlyID)
}

// FindCharges retrieves all elderly users linked to a caregiver.
func (r *UserRepository) FindCharges(ctx context.Context, caregiverID string) ([]*user.User, error) {
	query := `
		SELECT u.id, u.name, u.email, u.phone, u.firebase_token, u.notifications_enabled, u.role, u.created_at, u.updated_at
		FROM users u
		INNER JOIN user_links ul ON ul.elderly_id = u.id
		WHERE ul.caregiver_id = $1
		ORDER BY u.name
	`
	return r.queryUsers(ctx, query, caregiverID)
}

// IsLinked checks if a caregiver is linked to an elderly user.
func (r *UserRepository) IsLinked(ctx context.Context, caregiverID, elderlyID string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM user_links WHERE caregiver_id = $1 AND elderly_id = $2)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, caregiverID, elderlyID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// LinkUsers creates a caregiver-elderly link.
func (r *UserRepository) LinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	query := `
		INSERT INTO user_links (caregiver_id, elderly_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (caregiver_id, elderly_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, caregiverID, elderlyID, time.Now())
	return err
}

// UnlinkUsers removes a caregiver-elderly link.
func (r *UserRepository) UnlinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	query := `DELETE FROM user_links WHERE caregiver_id = $1 AND elderly_id = $2`
	_, err := r.db.ExecContext(ctx, query, caregiverID, elderlyID)
	return err
}

func (r *UserRepository) queryUsers(ctx context.Context, query string, arg string) ([]*user.User, error) {
	rows, err := r.db.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*user.User
	for rows.Next() {
		entity, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, entity)
	}

	return result, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*user.User, error) {
	var entity user.User
	var role string
	if err := row.Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&entity.FirebaseToken,
		&entity.NotificationsEnabled,
		&role,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		return nil, err
	}
	entity.Role = user.Role(role)
	return &entity, nil
}
