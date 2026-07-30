package scheduler

import (
	"context"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/notification"
)

// NotificationEvent outcome statuses. Empty Status is treated as "delivered"
// for backwards compatibility with rows written before the field existed.
const (
	StatusDelivered          = ""
	StatusSkippedNoTokens    = "skipped_no_tokens"
	StatusSkippedRetriesDone = "skipped_retries_exhausted"
)

type NotificationEvent struct {
	ID             string
	PrescriptionID string
	UserID         string
	MedicamentName string
	Dosage         string
	ScheduledAt    time.Time
	SentAt         time.Time
	Status         string
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

type noopLookup struct{}

func (n *noopLookup) ActiveTokens(ctx context.Context, userID string) ([]notification.Token, error) {
	_ = ctx
	_ = userID
	return nil, nil
}

func (n *noopLookup) TouchLastUsed(ctx context.Context, id string) error {
	_ = ctx
	_ = id
	return nil
}
