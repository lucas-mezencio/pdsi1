package notification

import (
	"context"
	"errors"
	"fmt"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FirebaseSender struct {
	client *messaging.Client
}

func NewFirebaseSender(ctx context.Context, credentialsFile string) (*FirebaseSender, error) {
	if credentialsFile == "" {
		return nil, errors.New("firebase credentials file is required")
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("firebase init failed: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase messaging init failed: %w", err)
	}

	return &FirebaseSender{client: client}, nil
}

func (f *FirebaseSender) Send(ctx context.Context, notification Notification) error {
	if f.client == nil {
		return errors.New("firebase client is not initialized")
	}

	msg := &messaging.Message{
		Token: notification.FirebaseToken,
		Notification: &messaging.Notification{
			Title: "Medication Reminder",
			Body:  fmt.Sprintf("Time to take %s (%s)", notification.MedicamentName, notification.Dosage),
		},
		Data: map[string]string{
			"user_id":         notification.UserID,
			"prescription_id": notification.PrescriptionID,
			"medicament_name": notification.MedicamentName,
			"dosage":          notification.Dosage,
			"scheduled_at":    notification.ScheduledAt,
		},
	}

	_, err := f.client.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("firebase send failed: %w", err)
	}

	return nil
}
