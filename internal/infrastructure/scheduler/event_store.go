package scheduler

import (
	"context"
	"time"
)

type NotificationEvent struct {
	ID             string
	PrescriptionID string
	UserID         string
	MedicamentName string
	Dosage         string
	ScheduledAt    time.Time
	SentAt         time.Time
}

type EventStore interface {
	Save(ctx context.Context, event NotificationEvent) error
}

// DoseRecordStore creates a pending dose record when a notification fires.
type DoseRecordStore interface {
	CreatePending(ctx context.Context, id, prescriptionID, userID, medicamentName, dosage string, scheduledAt time.Time) error
}

type CleanupStore interface {
	Delete(ctx context.Context, jobID string) error
}

type noopDoseRecordStore struct{}

func (n *noopDoseRecordStore) CreatePending(ctx context.Context, id, prescriptionID, userID, medicamentName, dosage string, scheduledAt time.Time) error {
	return nil
}

type noopEventStore struct{}

func (n *noopEventStore) Save(ctx context.Context, event NotificationEvent) error {
	_ = ctx
	_ = event
	return nil
}

type noopCleanup struct{}

func (n *noopCleanup) Delete(ctx context.Context, jobID string) error {
	_ = ctx
	_ = jobID
	return nil
}
