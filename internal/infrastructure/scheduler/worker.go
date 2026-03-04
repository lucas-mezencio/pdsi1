package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/notification"
)

const (
	defaultPollInterval = 250 * time.Millisecond
)

type SchedulerWorker struct {
	client       redis.UniversalClient
	publisher    message.Publisher
	keyPrefix    string
	pollInterval time.Duration
}

func NewSchedulerWorker(client redis.UniversalClient, publisher message.Publisher, keyPrefix string) *SchedulerWorker {
	if keyPrefix == "" {
		keyPrefix = defaultKeyPrefix
	}

	return &SchedulerWorker{
		client:       client,
		publisher:    publisher,
		keyPrefix:    keyPrefix,
		pollInterval: defaultPollInterval,
	}
}

func (w *SchedulerWorker) SetPollInterval(interval time.Duration) {
	if interval > 0 {
		w.pollInterval = interval
	}
}

func (w *SchedulerWorker) Run(ctx context.Context) error {
	if w.client == nil {
		return errors.New("redis client is required")
	}
	if w.publisher == nil {
		return errors.New("publisher is required")
	}
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.dispatchDue(ctx); err != nil {
				return err
			}
		}
	}
}

func (w *SchedulerWorker) dispatchDue(ctx context.Context) error {
	deadline := time.Now().UnixNano()
	jobIDs, err := w.client.ZRangeByScore(ctx, w.scheduleKey(), &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", deadline),
	}).Result()
	if err != nil {
		return fmt.Errorf("load due jobs: %w", err)
	}

	for _, jobID := range jobIDs {
		payload, err := w.client.HGet(ctx, w.jobsKey(), jobID).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				_ = w.client.ZRem(ctx, w.scheduleKey(), jobID).Err()
				continue
			}
			return fmt.Errorf("load job payload: %w", err)
		}

		msg := message.NewMessage(jobID, []byte(payload))
		if err := w.publisher.Publish(NotificationTopic, msg); err != nil {
			return fmt.Errorf("publish notification: %w", err)
		}

		pipeline := w.client.TxPipeline()
		pipeline.ZRem(ctx, w.scheduleKey(), jobID)
		pipeline.HDel(ctx, w.jobsKey(), jobID)
		if _, err := pipeline.Exec(ctx); err != nil {
			return fmt.Errorf("cleanup job: %w", err)
		}
	}

	return nil
}

func (w *SchedulerWorker) scheduleKey() string {
	return fmt.Sprintf("%s:notification_schedule", w.keyPrefix)
}

func (w *SchedulerWorker) jobsKey() string {
	return fmt.Sprintf("%s:notification_jobs", w.keyPrefix)
}

func StartNotificationConsumer(ctx context.Context, subscriber message.Subscriber, sender notification.Sender, userRepo user.Repository) error {
	if subscriber == nil {
		return errors.New("subscriber is required")
	}
	if sender == nil {
		return errors.New("sender is required")
	}
	if userRepo == nil {
		return errors.New("user repository is required")
	}

	messages, err := subscriber.Subscribe(ctx, NotificationTopic)
	if err != nil {
		return fmt.Errorf("subscribe notifications: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-messages:
			if !ok {
				return nil
			}

			var job NotificationJob
			if err := json.Unmarshal(msg.Payload, &job); err != nil {
				msg.Nack()
				return fmt.Errorf("decode notification job: %w", err)
			}

			userEntity, err := userRepo.FindByID(ctx, job.UserID)
			if err != nil {
				msg.Nack()
				return fmt.Errorf("load user: %w", err)
			}

			note := notification.Notification{
				UserID:         job.UserID,
				PrescriptionID: job.PrescriptionID,
				MedicamentName: job.MedicamentName,
				Dosage:         job.Dosage,
				ScheduledAt:    job.ScheduledAt.Format(time.RFC3339),
				FirebaseToken:  userEntity.FirebaseToken,
			}
			if err := sender.Send(ctx, note); err != nil {
				msg.Nack()
				return fmt.Errorf("send notification: %w", err)
			}

			msg.Ack()
		}
	}
}
