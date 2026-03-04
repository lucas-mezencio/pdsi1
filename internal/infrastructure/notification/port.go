package notification

import "context"

// Sender delivers notifications to users.
type Sender interface {
	Send(ctx context.Context, notification Notification) error
}

type Notification struct {
	UserID         string
	PrescriptionID string
	MedicamentName string
	Dosage         string
	ScheduledAt    string
	FirebaseToken  string
}
