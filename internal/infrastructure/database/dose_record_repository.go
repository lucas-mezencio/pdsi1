package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

// DoseRecordRepository implements prescription.DoseRecordRepository using
// PostgreSQL.
//
// Medicament name and dosage are stored encrypted with pgcrypto
// (pgp_sym_encrypt / pgp_sym_decrypt) using the master key passed to
// NewDoseRecordRepository.
type DoseRecordRepository struct {
	db  *sql.DB
	key string
}

// NewDoseRecordRepository creates a new DoseRecordRepository.
// key is the pgcrypto symmetric master key (DB_ENCRYPTION_KEY).
func NewDoseRecordRepository(db *sql.DB, key string) *DoseRecordRepository {
	return &DoseRecordRepository{db: db, key: key}
}

// Save creates or updates a dose record.
func (r *DoseRecordRepository) Save(ctx context.Context, record *prescription.DoseRecord) error {
	query := `
		INSERT INTO dose_records (
			id, prescription_id, user_id, medicament_name, dosage,
			scheduled_at, status, confirmed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3,
			pgp_sym_encrypt($4, $11),
			pgp_sym_encrypt($5, $11),
			$6, $7, $8, $9, $10
		)
		ON CONFLICT (id) DO UPDATE SET
			status       = EXCLUDED.status,
			confirmed_at = EXCLUDED.confirmed_at,
			updated_at   = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		record.ID,
		record.PrescriptionID,
		record.UserID,
		record.MedicamentName,
		record.Dosage,
		record.ScheduledAt,
		string(record.Status),
		record.ConfirmedAt,
		record.CreatedAt,
		record.UpdatedAt,
		r.key,
	)
	return err
}

// CreatePending inserts a new PENDING dose record. Implements scheduler.DoseRecordStore.
func (r *DoseRecordRepository) CreatePending(ctx context.Context, id, prescriptionID, userID, medicamentName, dosage string, scheduledAt time.Time) error {
	record := prescription.NewDoseRecord(id, prescriptionID, userID, medicamentName, dosage, scheduledAt)
	return r.Save(ctx, record)
}

// FindByID retrieves a dose record by ID.
func (r *DoseRecordRepository) FindByID(ctx context.Context, id string) (*prescription.DoseRecord, error) {
	query := `
		SELECT
			id, prescription_id, user_id,
			pgp_sym_decrypt(medicament_name, $2) AS medicament_name,
			pgp_sym_decrypt(dosage,          $2) AS dosage,
			scheduled_at, status, confirmed_at, created_at, updated_at
		FROM dose_records
		WHERE id = $1
	`

	var record prescription.DoseRecord
	var status string
	var confirmedAt sql.NullTime

	if err := r.db.QueryRowContext(ctx, query, id, r.key).Scan(
		&record.ID,
		&record.PrescriptionID,
		&record.UserID,
		&record.MedicamentName,
		&record.Dosage,
		&record.ScheduledAt,
		&status,
		&confirmedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, prescription.ErrDoseRecordNotFound
		}
		return nil, err
	}

	record.Status = prescription.DoseStatus(status)
	if confirmedAt.Valid {
		record.ConfirmedAt = &confirmedAt.Time
	}
	return &record, nil
}

// FindByUserID retrieves all dose records for a user.
func (r *DoseRecordRepository) FindByUserID(ctx context.Context, userID string) ([]*prescription.DoseRecord, error) {
	query := `
		SELECT
			id, prescription_id, user_id,
			pgp_sym_decrypt(medicament_name, $2) AS medicament_name,
			pgp_sym_decrypt(dosage,          $2) AS dosage,
			scheduled_at, status, confirmed_at, created_at, updated_at
		FROM dose_records
		WHERE user_id = $1
		ORDER BY scheduled_at DESC
	`
	return r.queryDoseRecords(ctx, query, userID, r.key)
}

// FindByPrescriptionID retrieves all dose records for a prescription.
func (r *DoseRecordRepository) FindByPrescriptionID(ctx context.Context, prescriptionID string) ([]*prescription.DoseRecord, error) {
	query := `
		SELECT
			id, prescription_id, user_id,
			pgp_sym_decrypt(medicament_name, $2) AS medicament_name,
			pgp_sym_decrypt(dosage,          $2) AS dosage,
			scheduled_at, status, confirmed_at, created_at, updated_at
		FROM dose_records
		WHERE prescription_id = $1
		ORDER BY scheduled_at DESC
	`
	return r.queryDoseRecords(ctx, query, prescriptionID, r.key)
}

// FindPendingBefore retrieves all PENDING dose records scheduled before a given time.
func (r *DoseRecordRepository) FindPendingBefore(ctx context.Context, before time.Time) ([]*prescription.DoseRecord, error) {
	query := `
		SELECT
			id, prescription_id, user_id,
			pgp_sym_decrypt(medicament_name, $2) AS medicament_name,
			pgp_sym_decrypt(dosage,          $2) AS dosage,
			scheduled_at, status, confirmed_at, created_at, updated_at
		FROM dose_records
		WHERE status = 'PENDING' AND scheduled_at < $1
		ORDER BY scheduled_at ASC
	`
	return r.queryDoseRecords(ctx, query, before.Format(time.RFC3339Nano), r.key)
}

func (r *DoseRecordRepository) queryDoseRecords(ctx context.Context, query string, args ...any) ([]*prescription.DoseRecord, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*prescription.DoseRecord
	for rows.Next() {
		var record prescription.DoseRecord
		var status string
		var confirmedAt sql.NullTime

		if err := rows.Scan(
			&record.ID,
			&record.PrescriptionID,
			&record.UserID,
			&record.MedicamentName,
			&record.Dosage,
			&record.ScheduledAt,
			&status,
			&confirmedAt,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, err
		}

		record.Status = prescription.DoseStatus(status)
		if confirmedAt.Valid {
			record.ConfirmedAt = &confirmedAt.Time
		}
		result = append(result, &record)
	}

	return result, rows.Err()
}
