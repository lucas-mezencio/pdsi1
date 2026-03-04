package notification

import (
	"context"
	"fmt"
)

// DummySender captures notifications for tests or local runs.
type DummySender struct {
	Delivered []Notification
}

func (d *DummySender) Send(ctx context.Context, notification Notification) error {
	_ = ctx
	d.Delivered = append(d.Delivered, notification)
	fmt.Printf("dummy notifier: user=%s med=%s dosage=%s at=%s\n", notification.UserID, notification.MedicamentName, notification.Dosage, notification.ScheduledAt)
	return nil
}
