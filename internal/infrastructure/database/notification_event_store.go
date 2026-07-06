package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/scheduler"
)

// NotificationEventStore persists the audit trail of dispatched
// notification jobs. Medicament name and dosage are stored encrypted
// with pgcrypto using the master key passed to NewNotificationEventStore.
type NotificationEventStore struct {
	db  *sql.DB
	key string
}

// NewNotificationEventStore creates a new NotificationEventStore.
// key is the pgcrypto symmetric master key (DB_ENCRYPTION_KEY).
func NewNotificationEventStore(db *sql.DB, key string) *NotificationEventStore {
	return &NotificationEventStore{db: db, key: key}
}

func (s *NotificationEventStore) Save(ctx context.Context, event scheduler.NotificationEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	query := `
		INSERT INTO notification_events (
			id, prescription_id, user_id, medicament_name, dosage, scheduled_at, sent_at
		) VALUES (
			$1, $2, $3,
			pgp_sym_encrypt($4, $8),
			pgp_sym_encrypt($5, $8),
			$6, $7
		)
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
		s.key,
	); err != nil {
		return fmt.Errorf("save notification event: %w", err)
	}

	return nil
}
