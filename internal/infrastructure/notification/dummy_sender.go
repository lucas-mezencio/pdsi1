package notification

import "context"

// DummySender captures notifications for tests or local runs.
type DummySender struct {
	Delivered []Notification
}

func (d *DummySender) Send(ctx context.Context, notification Notification) error {
	_ = ctx
	d.Delivered = append(d.Delivered, notification)
	return nil
}
