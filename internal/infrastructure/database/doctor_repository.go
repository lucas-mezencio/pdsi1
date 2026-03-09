package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

// DoctorRepository implements doctor.Repository using PostgreSQL.
type DoctorRepository struct {
	db *sql.DB
}

// NewDoctorRepository creates a new DoctorRepository.
func NewDoctorRepository(db *sql.DB) *DoctorRepository {
	return &DoctorRepository{db: db}
}

// Save creates or updates a doctor.
func (r *DoctorRepository) Save(ctx context.Context, entity *doctor.Doctor) error {
	query := `
		INSERT INTO doctors (id, name, email, phone, firebase_id, specialty, license_number, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			firebase_id = EXCLUDED.firebase_id,
			specialty = EXCLUDED.specialty,
			license_number = EXCLUDED.license_number,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		entity.ID,
		entity.Name,
		entity.Email,
		entity.Phone,
		entity.FirebaseID,
		entity.Specialty,
		entity.LicenseNumber,
		entity.CreatedAt,
		entity.UpdatedAt,
	)
	return err
}

// FindByID retrieves a doctor by ID.
func (r *DoctorRepository) FindByID(ctx context.Context, id string) (*doctor.Doctor, error) {
	query := `
		SELECT id, name, email, phone, firebase_id, specialty, license_number, created_at, updated_at
		FROM doctors
		WHERE id = $1
	`

	var entity doctor.Doctor
	var firebaseID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&firebaseID,
		&entity.Specialty,
		&entity.LicenseNumber,
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

// FindByEmail retrieves a doctor by email.
func (r *DoctorRepository) FindByEmail(ctx context.Context, email string) (*doctor.Doctor, error) {
	query := `
		SELECT id, name, email, phone, firebase_id, specialty, license_number, created_at, updated_at
		FROM doctors
		WHERE email = $1
	`

	var entity doctor.Doctor
	var firebaseID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, email).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&firebaseID,
		&entity.Specialty,
		&entity.LicenseNumber,
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
		SELECT id, name, email, phone, firebase_id, specialty, license_number, created_at, updated_at
		FROM doctors
		WHERE firebase_id = $1
	`

	var entity doctor.Doctor
	var firebaseIDValue sql.NullString
	if err := r.db.QueryRowContext(ctx, query, firebaseID).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&firebaseIDValue,
		&entity.Specialty,
		&entity.LicenseNumber,
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

// FindByLicenseNumber retrieves a doctor by license number.
func (r *DoctorRepository) FindByLicenseNumber(ctx context.Context, licenseNumber string) (*doctor.Doctor, error) {
	query := `
		SELECT id, name, email, phone, firebase_id, specialty, license_number, created_at, updated_at
		FROM doctors
		WHERE license_number = $1
	`

	var entity doctor.Doctor
	var firebaseID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, licenseNumber).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Email,
		&entity.Phone,
		&firebaseID,
		&entity.Specialty,
		&entity.LicenseNumber,
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
		SELECT id, name, email, phone, firebase_id, specialty, license_number, created_at, updated_at
		FROM doctors
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
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
			&firebaseID,
			&entity.Specialty,
			&entity.LicenseNumber,
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
