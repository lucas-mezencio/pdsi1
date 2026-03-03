package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com/lib/pq"
)

// PrescriptionRepository implements prescription.Repository using PostgreSQL.
type PrescriptionRepository struct {
	db *sql.DB
}

// NewPrescriptionRepository creates a new PrescriptionRepository.
func NewPrescriptionRepository(db *sql.DB) *PrescriptionRepository {
	return &PrescriptionRepository{db: db}
}

// Save creates or updates a prescription and its medicaments.
func (r *PrescriptionRepository) Save(ctx context.Context, entity *prescription.Prescription) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := r.savePrescription(ctx, tx, entity); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := r.replaceMedicaments(ctx, tx, entity); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// FindByID retrieves a prescription by ID.
func (r *PrescriptionRepository) FindByID(ctx context.Context, id string) (*prescription.Prescription, error) {
	query := `
		SELECT id, user_id, medic_id, active, created_at, updated_at
		FROM prescriptions
		WHERE id = $1
	`

	var entity prescription.Prescription
	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entity.ID,
		&entity.UserID,
		&entity.MedicID,
		&entity.Active,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, prescription.ErrPrescriptionNotFound
		}
		return nil, err
	}

	medicaments, err := r.findMedicaments(ctx, entity.ID)
	if err != nil {
		return nil, err
	}
	entity.Medicaments = medicaments

	return &entity, nil
}

// FindAll retrieves all prescriptions.
func (r *PrescriptionRepository) FindAll(ctx context.Context) ([]*prescription.Prescription, error) {
	query := `
		SELECT id, user_id, medic_id, active, created_at, updated_at
		FROM prescriptions
		ORDER BY created_at DESC
	`

	return r.findWithQuery(ctx, query)
}

// FindByUserID retrieves prescriptions for a user.
func (r *PrescriptionRepository) FindByUserID(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
	query := `
		SELECT id, user_id, medic_id, active, created_at, updated_at
		FROM prescriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	return r.findWithQuery(ctx, query, userID)
}

// FindByMedicID retrieves prescriptions for a doctor.
func (r *PrescriptionRepository) FindByMedicID(ctx context.Context, medicID string) ([]*prescription.Prescription, error) {
	query := `
		SELECT id, user_id, medic_id, active, created_at, updated_at
		FROM prescriptions
		WHERE medic_id = $1
		ORDER BY created_at DESC
	`

	return r.findWithQuery(ctx, query, medicID)
}

// FindActive retrieves all active prescriptions.
func (r *PrescriptionRepository) FindActive(ctx context.Context) ([]*prescription.Prescription, error) {
	query := `
		SELECT id, user_id, medic_id, active, created_at, updated_at
		FROM prescriptions
		WHERE active = TRUE
		ORDER BY created_at DESC
	`

	return r.findWithQuery(ctx, query)
}

// FindActiveByUserID retrieves active prescriptions for a user.
func (r *PrescriptionRepository) FindActiveByUserID(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
	query := `
		SELECT id, user_id, medic_id, active, created_at, updated_at
		FROM prescriptions
		WHERE user_id = $1 AND active = TRUE
		ORDER BY created_at DESC
	`

	return r.findWithQuery(ctx, query, userID)
}

// Delete removes a prescription by ID.
func (r *PrescriptionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM prescriptions WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return prescription.ErrPrescriptionNotFound
	}

	return nil
}

// Exists checks if a prescription exists by ID.
func (r *PrescriptionRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM prescriptions WHERE id = $1)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *PrescriptionRepository) savePrescription(ctx context.Context, tx *sql.Tx, entity *prescription.Prescription) error {
	query := `
		INSERT INTO prescriptions (id, user_id, medic_id, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			medic_id = EXCLUDED.medic_id,
			active = EXCLUDED.active,
			updated_at = EXCLUDED.updated_at
	`

	_, err := tx.ExecContext(ctx, query,
		entity.ID,
		entity.UserID,
		entity.MedicID,
		entity.Active,
		entity.CreatedAt,
		entity.UpdatedAt,
	)
	return err
}

func (r *PrescriptionRepository) replaceMedicaments(ctx context.Context, tx *sql.Tx, entity *prescription.Prescription) error {
	deleteQuery := `DELETE FROM medicaments WHERE prescription_id = $1`
	if _, err := tx.ExecContext(ctx, deleteQuery, entity.ID); err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO medicaments (prescription_id, name, dosage, frequency, times, doses)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, m := range entity.Medicaments {
		if _, err := tx.ExecContext(ctx, insertQuery,
			entity.ID,
			m.Name,
			m.Dosage,
			m.Frequency,
			pq.Array(m.Times),
			m.Doses,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *PrescriptionRepository) findMedicaments(ctx context.Context, prescriptionID string) ([]prescription.Medicament, error) {
	query := `
		SELECT name, dosage, frequency, times, doses
		FROM medicaments
		WHERE prescription_id = $1
		ORDER BY id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, prescriptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []prescription.Medicament
	for rows.Next() {
		var entity prescription.Medicament
		if err := rows.Scan(
			&entity.Name,
			&entity.Dosage,
			&entity.Frequency,
			pq.Array(&entity.Times),
			&entity.Doses,
		); err != nil {
			return nil, err
		}
		result = append(result, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *PrescriptionRepository) findWithQuery(ctx context.Context, query string, args ...any) ([]*prescription.Prescription, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*prescription.Prescription
	for rows.Next() {
		var entity prescription.Prescription
		if err := rows.Scan(
			&entity.ID,
			&entity.UserID,
			&entity.MedicID,
			&entity.Active,
			&entity.CreatedAt,
			&entity.UpdatedAt,
		); err != nil {
			return nil, err
		}

		medicaments, err := r.findMedicaments(ctx, entity.ID)
		if err != nil {
			return nil, err
		}
		entity.Medicaments = medicaments
		result = append(result, &entity)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
