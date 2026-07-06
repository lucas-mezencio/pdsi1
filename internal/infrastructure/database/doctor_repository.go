package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

// DoctorRepository implements doctor.Repository using PostgreSQL.
//
// PII columns (email, phone, license_number) are stored encrypted with
// pgcrypto (pgp_sym_encrypt / pgp_sym_decrypt) using the master key
// passed to NewDoctorRepository. Email and license_number uniqueness is
// preserved via SHA-256 hash sidecar columns (email_hash,
// license_number_hash).
type DoctorRepository struct {
	db  *sql.DB
	key string
}

// NewDoctorRepository creates a new DoctorRepository.
// key is the pgcrypto symmetric master key (DB_ENCRYPTION_KEY). It must
// be the same value used by every replica and by backup restores.
func NewDoctorRepository(db *sql.DB, key string) *DoctorRepository {
	return &DoctorRepository{db: db, key: key}
}

// Save creates or updates a doctor.
func (r *DoctorRepository) Save(ctx context.Context, entity *doctor.Doctor) error {
	query := `
		INSERT INTO doctors (
			id, name, email, phone, firebase_id, specialty, license_number,
			created_at, updated_at, email_hash, license_number_hash
		) VALUES (
			$1, $2,
			pgp_sym_encrypt($3, $10),
			pgp_sym_encrypt($4, $10),
			$5, $6,
			pgp_sym_encrypt($7, $10),
			$8, $9,
			digest(lower($3), 'sha256'),
			digest(lower($7), 'sha256')
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			email = EXCLUDED.email,
			email_hash = EXCLUDED.email_hash,
			phone = EXCLUDED.phone,
			firebase_id = EXCLUDED.firebase_id,
			specialty = EXCLUDED.specialty,
			license_number = EXCLUDED.license_number,
			license_number_hash = EXCLUDED.license_number_hash,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		entity.ID,
		entity.Name,
		entity.Email,
		entity.Phone,
		sql.NullString{String: entity.FirebaseID, Valid: entity.FirebaseID != ""},
		entity.Specialty,
		entity.LicenseNumber,
		entity.CreatedAt,
		entity.UpdatedAt,
		r.key,
	)
	return err
}

// FindByID retrieves a doctor by ID.
func (r *DoctorRepository) FindByID(ctx context.Context, id string) (*doctor.Doctor, error) {
	query := `
		SELECT
			id, name,
			pgp_sym_decrypt(email,          $2) AS email,
			pgp_sym_decrypt(phone,          $2) AS phone,
			pgp_sym_decrypt(license_number, $2) AS license_number,
			firebase_id, specialty, created_at, updated_at
		FROM doctors
		WHERE id = $1
	`

	var entity doctor.Doctor
	var firebaseID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, id, r.key).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&entity.LicenseNumber,
		&firebaseID,
		&entity.Specialty,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, doctor.ErrDoctorNotFound
		}
		return nil, err
	}
	if firebaseID.Valid {
		entity.FirebaseID = firebaseID.String
	}

	return &entity, nil
}

// FindByEmail retrieves a doctor by email using the email_hash sidecar.
func (r *DoctorRepository) FindByEmail(ctx context.Context, email string) (*doctor.Doctor, error) {
	query := `
		SELECT
			id, name,
			pgp_sym_decrypt(email,          $2) AS email,
			pgp_sym_decrypt(phone,          $2) AS phone,
			pgp_sym_decrypt(license_number, $2) AS license_number,
			firebase_id, specialty, created_at, updated_at
		FROM doctors
		WHERE email_hash = digest(lower($1), 'sha256')
	`

	var entity doctor.Doctor
	var firebaseID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, email, r.key).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&entity.LicenseNumber,
		&firebaseID,
		&entity.Specialty,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, doctor.ErrDoctorNotFound
		}
		return nil, err
	}
	if firebaseID.Valid {
		entity.FirebaseID = firebaseID.String
	}

	return &entity, nil
}

// FindByFirebaseID retrieves a doctor by firebase auth UID.
func (r *DoctorRepository) FindByFirebaseID(ctx context.Context, firebaseID string) (*doctor.Doctor, error) {
	query := `
		SELECT
			id, name,
			pgp_sym_decrypt(email,          $2) AS email,
			pgp_sym_decrypt(phone,          $2) AS phone,
			pgp_sym_decrypt(license_number, $2) AS license_number,
			firebase_id, specialty, created_at, updated_at
		FROM doctors
		WHERE firebase_id = $1
	`

	var entity doctor.Doctor
	var firebaseIDValue sql.NullString
	if err := r.db.QueryRowContext(ctx, query, firebaseID, r.key).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&entity.LicenseNumber,
		&firebaseIDValue,
		&entity.Specialty,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, doctor.ErrDoctorNotFound
		}
		return nil, err
	}
	if firebaseIDValue.Valid {
		entity.FirebaseID = firebaseIDValue.String
	}

	return &entity, nil
}

// FindByLicenseNumber retrieves a doctor by license number using the
// license_number_hash sidecar.
func (r *DoctorRepository) FindByLicenseNumber(ctx context.Context, licenseNumber string) (*doctor.Doctor, error) {
	query := `
		SELECT
			id, name,
			pgp_sym_decrypt(email,          $2) AS email,
			pgp_sym_decrypt(phone,          $2) AS phone,
			pgp_sym_decrypt(license_number, $2) AS license_number,
			firebase_id, specialty, created_at, updated_at
		FROM doctors
		WHERE license_number_hash = digest(lower($1), 'sha256')
	`

	var entity doctor.Doctor
	var firebaseID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, licenseNumber, r.key).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&entity.LicenseNumber,
		&firebaseID,
		&entity.Specialty,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, doctor.ErrDoctorNotFound
		}
		return nil, err
	}
	if firebaseID.Valid {
		entity.FirebaseID = firebaseID.String
	}

	return &entity, nil
}

// FindAll retrieves all doctors.
func (r *DoctorRepository) FindAll(ctx context.Context) ([]*doctor.Doctor, error) {
	query := `
		SELECT
			id, name,
			pgp_sym_decrypt(email,          $1) AS email,
			pgp_sym_decrypt(phone,          $1) AS phone,
			pgp_sym_decrypt(license_number, $1) AS license_number,
			firebase_id, specialty, created_at, updated_at
		FROM doctors
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, r.key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*doctor.Doctor
	for rows.Next() {
		var entity doctor.Doctor
		var firebaseID sql.NullString
		if err := rows.Scan(
			&entity.ID,
			&entity.Name,
			&entity.Email,
			&entity.Phone,
			&entity.LicenseNumber,
			&firebaseID,
			&entity.Specialty,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if firebaseID.Valid {
			entity.FirebaseID = firebaseID.String
		}
		result = append(result, &entity)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Delete removes a doctor by ID.
func (r *DoctorRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM doctors WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return doctor.ErrDoctorNotFound
	}

	return nil
}

// Exists checks if a doctor exists by ID.
func (r *DoctorRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM doctors WHERE id = $1)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
