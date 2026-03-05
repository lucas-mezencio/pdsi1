package prescription

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// DoseStatus represents the status of a scheduled dose.
type DoseStatus string

const (
	DoseStatusPending DoseStatus = "PENDING"
	DoseStatusTaken   DoseStatus = "TAKEN"
	DoseStatusMissed  DoseStatus = "MISSED"
)

// DoseRecord tracks a single scheduled dose and whether it was taken.
type DoseRecord struct {
	ID             string     `json:"id"`
	PrescriptionID string     `json:"prescription_id"`
	UserID         string     `json:"user_id"`
	MedicamentName string     `json:"medicament_name"`
	Dosage         string     `json:"dosage"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
	Status         DoseStatus `json:"status"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// NewDoseRecord creates a new PENDING dose record.
func NewDoseRecord(id, prescriptionID, userID, medicamentName, dosage string, scheduledAt time.Time) *DoseRecord {
	if id == "" {
		id = uuid.New().String()
	}
	now := time.Now()
	return &DoseRecord{
		ID:             id,
		PrescriptionID: prescriptionID,
		UserID:         userID,
		MedicamentName: medicamentName,
		Dosage:         dosage,
		ScheduledAt:    scheduledAt,
		Status:         DoseStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// MarkTaken marks the dose as taken at the given time.
func (d *DoseRecord) MarkTaken(at time.Time) {
	d.Status = DoseStatusTaken
	d.ConfirmedAt = &at
	d.UpdatedAt = time.Now()
}

// MarkMissed marks the dose as missed.
func (d *DoseRecord) MarkMissed() {
	d.Status = DoseStatusMissed
	d.UpdatedAt = time.Now()
}

// DoseRecordRepository defines persistence for dose records.
type DoseRecordRepository interface {
	Save(ctx context.Context, record *DoseRecord) error
	FindByID(ctx context.Context, id string) (*DoseRecord, error)
	FindByUserID(ctx context.Context, userID string) ([]*DoseRecord, error)
	FindByPrescriptionID(ctx context.Context, prescriptionID string) ([]*DoseRecord, error)
	FindPendingBefore(ctx context.Context, before time.Time) ([]*DoseRecord, error)
}

// Domain errors
var (
	ErrDoseRecordNotFound = errors.New("dose record not found")
)
