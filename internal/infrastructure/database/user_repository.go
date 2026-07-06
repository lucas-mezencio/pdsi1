package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// UserRepository implements user.Repository using PostgreSQL.
//
// PII columns (email, phone, cpf) are stored encrypted with pgcrypto
// (pgp_sym_encrypt / pgp_sym_decrypt) using the master key passed to
// NewUserRepository. Email uniqueness is preserved via the SHA-256 hash
// sidecar column email_hash.
type UserRepository struct {
	db  *sql.DB
	key string
}

// NewUserRepository creates a new UserRepository.
// key is the pgcrypto symmetric master key (DB_ENCRYPTION_KEY). It must
// be the same value used by every replica and by backup restores.
func NewUserRepository(db *sql.DB, key string) *UserRepository {
	return &UserRepository{db: db, key: key}
}

// Save creates or updates a user.
func (r *UserRepository) Save(ctx context.Context, entity *user.User) error {
	query := `
		INSERT INTO users (
			id, name, email, phone, cpf, firebase_id, firebase_token,
			notifications_enabled, role, created_at, updated_at, email_hash
		) VALUES (
			$1, $2,
			pgp_sym_encrypt($3, $12),
			pgp_sym_encrypt($4, $12),
			CASE WHEN $5 = '' THEN NULL ELSE pgp_sym_encrypt($5, $12) END,
			$6, $7, $8, $9, $10, $11,
			digest(lower($3), 'sha256')
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			email = EXCLUDED.email,
			email_hash = EXCLUDED.email_hash,
			phone = EXCLUDED.phone,
			cpf = EXCLUDED.cpf,
			firebase_id = EXCLUDED.firebase_id,
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
		entity.CPF,
		sql.NullString{String: entity.FirebaseID, Valid: entity.FirebaseID != ""},
		entity.FirebaseToken,
		entity.NotificationsEnabled,
		string(entity.Role),
		entity.CreatedAt,
		entity.UpdatedAt,
		r.key,
	)
	return err
}

// FindByID retrieves a user by ID.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	query := `
		SELECT
			id, name,
			pgp_sym_decrypt(email, $2) AS email,
			pgp_sym_decrypt(phone, $2) AS phone,
			pgp_sym_decrypt(cpf,   $2) AS cpf,
			firebase_id, firebase_token, notifications_enabled, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var entity user.User
	var role string
	var cpf []byte
	var firebaseID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, id, r.key).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&cpf,
		&firebaseID,
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

	if len(cpf) > 0 {
		// Decryption succeeded and returned a non-empty payload. Convert via
		// string to preserve the original semantic (we treat all-null / all-zero
		// bytea as "no cpf").
		entity.CPF = string(cpf)
	}
	if firebaseID.Valid {
		entity.FirebaseID = firebaseID.String
	}
	entity.Role = user.Role(role)
	return &entity, nil
}

// FindByEmail retrieves a user by email using the email_hash sidecar.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT
			id, name,
			pgp_sym_decrypt(email, $2) AS email,
			pgp_sym_decrypt(phone, $2) AS phone,
			pgp_sym_decrypt(cpf,   $2) AS cpf,
			firebase_id, firebase_token, notifications_enabled, role, created_at, updated_at
		FROM users
		WHERE email_hash = digest(lower($1), 'sha256')
	`

	var entity user.User
	var role string
	var cpf []byte
	var firebaseID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, email, r.key).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&cpf,
		&firebaseID,
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

	if len(cpf) > 0 {
		entity.CPF = string(cpf)
	}
	if firebaseID.Valid {
		entity.FirebaseID = firebaseID.String
	}
	entity.Role = user.Role(role)
	return &entity, nil
}

// FindByFirebaseID retrieves a user by firebase auth UID.
func (r *UserRepository) FindByFirebaseID(ctx context.Context, firebaseID string) (*user.User, error) {
	query := `
		SELECT
			id, name,
			pgp_sym_decrypt(email, $2) AS email,
			pgp_sym_decrypt(phone, $2) AS phone,
			pgp_sym_decrypt(cpf,   $2) AS cpf,
			firebase_id, firebase_token, notifications_enabled, role, created_at, updated_at
		FROM users
		WHERE firebase_id = $1
	`

	var entity user.User
	var role string
	var cpf []byte
	var firebaseIDValue sql.NullString
	if err := r.db.QueryRowContext(ctx, query, firebaseID, r.key).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&cpf,
		&firebaseIDValue,
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

	if len(cpf) > 0 {
		entity.CPF = string(cpf)
	}
	if firebaseIDValue.Valid {
		entity.FirebaseID = firebaseIDValue.String
	}
	entity.Role = user.Role(role)
	return &entity, nil
}

// FindAll retrieves all users.
func (r *UserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	query := `
		SELECT
			id, name,
			pgp_sym_decrypt(email, $1) AS email,
			pgp_sym_decrypt(phone, $1) AS phone,
			pgp_sym_decrypt(cpf,   $1) AS cpf,
			firebase_id, firebase_token, notifications_enabled, role, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, r.key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*user.User
	for rows.Next() {
		entity, err := scanUser(rows, r.key)
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
		SELECT
			u.id, u.name,
			pgp_sym_decrypt(u.email, $2) AS email,
			pgp_sym_decrypt(u.phone, $2) AS phone,
			pgp_sym_decrypt(u.cpf,   $2) AS cpf,
			u.firebase_id, u.firebase_token, u.notifications_enabled, u.role, u.created_at, u.updated_at
		FROM users u
		INNER JOIN user_links ul ON ul.caregiver_id = u.id
		WHERE ul.elderly_id = $1
		ORDER BY u.name
	`
	return r.queryUsers(ctx, query, elderlyID, r.key)
}

// FindCharges retrieves all elderly users linked to a caregiver.
func (r *UserRepository) FindCharges(ctx context.Context, caregiverID string) ([]*user.User, error) {
	query := `
		SELECT
			u.id, u.name,
			pgp_sym_decrypt(u.email, $2) AS email,
			pgp_sym_decrypt(u.phone, $2) AS phone,
			pgp_sym_decrypt(u.cpf,   $2) AS cpf,
			u.firebase_id, u.firebase_token, u.notifications_enabled, u.role, u.created_at, u.updated_at
		FROM users u
		INNER JOIN user_links ul ON ul.elderly_id = u.id
		WHERE ul.caregiver_id = $1
		ORDER BY u.name
	`
	return r.queryUsers(ctx, query, caregiverID, r.key)
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

func (r *UserRepository) queryUsers(ctx context.Context, query string, args ...any) ([]*user.User, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*user.User
	for rows.Next() {
		entity, err := scanUser(rows, r.key)
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

func scanUser(row rowScanner, _ string) (*user.User, error) {
	var entity user.User
	var role string
	var cpf []byte
	var firebaseID sql.NullString
	if err := row.Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&cpf,
		&firebaseID,
		&entity.FirebaseToken,
		&entity.NotificationsEnabled,
		&role,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(cpf) > 0 {
		entity.CPF = string(cpf)
	}
	if firebaseID.Valid {
		entity.FirebaseID = firebaseID.String
	}
	entity.Role = user.Role(role)
	return &entity, nil
}
