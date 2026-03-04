package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/scheduler"
)

type NotificationEventStore struct {
	db *sql.DB
}

func NewNotificationEventStore(db *sql.DB) *NotificationEventStore {
	return &NotificationEventStore{db: db}
}

func (s *NotificationEventStore) Save(ctx context.Context, event scheduler.NotificationEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	query := `
		INSERT INTO notification_events (
			id, prescription_id, user_id, medicament_name, dosage, scheduled_at, sent_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if _, err := s.db.ExecContext(
		ctx,
		query,
		event.ID,
		event.PrescriptionID,
		event.UserID,
		event.MedicamentName,
		event.Dosage,
		event.ScheduledAt,
		event.SentAt,
	); err != nil {
		return fmt.Errorf("save notification event: %w", err)
	}

	return nil
}
