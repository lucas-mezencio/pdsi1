package application

import (
	"context"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

// NotificationScheduler schedules and cancels prescription notifications.
type NotificationScheduler interface {
	Schedule(ctx context.Context, schedule prescription.NotificationSchedule, startDate time.Time) error
	CancelByPrescriptionID(ctx context.Context, prescriptionID string) error
}
