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

type noopEventStore struct{}

func (n *noopEventStore) Save(ctx context.Context, event NotificationEvent) error {
	_ = ctx
	_ = event
	return nil
}
