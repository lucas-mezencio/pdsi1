package scheduler

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com.br/lucas-mezencio/pdsi1/tests/testcontainers"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

func openTestRedis(t *testing.T) redis.UniversalClient {
	t.Helper()

	ctx := context.Background()
	client, _, cleanup := testcontainers.StartRedisContainer(ctx)
	if client == nil {
		t.Skip("docker not available")
	}

	_ = client.FlushDB(ctx)
	t.Cleanup(cleanup)
	return client
}

func TestRedisScheduler_SchedulesNotifications(t *testing.T) {
	client := openTestRedis(t)

	logger := watermill.NopLogger{}
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: client,
	}, &logger)
	if err != nil {
		t.Fatalf("publisher init failed: %v", err)
	}
	defer publisher.Close()

	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        client,
		ConsumerGroup: "tests",
		Consumer:      "test-consumer",
		BlockTime:     50 * time.Millisecond,
	}, &logger)
	if err != nil {
		t.Fatalf("subscriber init failed: %v", err)
	}
	defer subscriber.Close()

	sched, err := NewRedisScheduler(RedisSchedulerConfig{Client: client})
	if err != nil {
		t.Fatalf("scheduler init failed: %v", err)
	}

	start := time.Now().Add(3 * time.Second).Truncate(time.Second).Add(1 * time.Second)
	startTime := start.Format("15:04:05")

	schedule := prescription.NotificationSchedule{
		PrescriptionID: uuid.New().String(),
		UserID:         uuid.New().String(),
		MedicamentName: "TestMed",
		Dosage:         "1",
		Time:           startTime,
		Frequency:      "00:00:01",
		TotalDoses:     8,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	if err := sched.Schedule(ctx, schedule, start); err != nil {
		t.Fatalf("schedule failed: %v", err)
	}

	worker := NewSchedulerWorker(client, publisher, defaultKeyPrefix, 2*time.Hour, nil)
	worker.SetPollInterval(50 * time.Millisecond)

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	go func() {
		_ = worker.Run(workerCtx)
	}()

	messages, err := subscriber.Subscribe(ctx, NotificationTopic)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	var received []NotificationJob
	seenTimes := make(map[string]bool)
	for len(received) < schedule.TotalDoses {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for notifications, got %d", len(received))
		case msg := <-messages:
			var job NotificationJob
			if err := json.Unmarshal(msg.Payload, &job); err != nil {
				t.Fatalf("unmarshal notification job: %v", err)
			}
			if job.UserID != schedule.UserID || job.PrescriptionID != schedule.PrescriptionID {
				msg.Ack()
				continue
			}
			jobTime := job.ScheduledAt.Format("15:04:05")
			if !seenTimes[jobTime] {
				received = append(received, job)
				seenTimes[jobTime] = true
			}
			msg.Ack()
		}
	}

	sort.Slice(received, func(i, j int) bool {
		return received[i].ScheduledAt.Before(received[j].ScheduledAt)
	})

	for i := 0; i < schedule.TotalDoses; i++ {
		expected := start.Add(time.Duration(i) * time.Second)
		actual := received[i].ScheduledAt
		if expected.Format("15:04:05") != actual.Format("15:04:05") {
			t.Fatalf("expected scheduled time %s, got %s", expected.Format("15:04:05"), actual.Format("15:04:05"))
		}
	}
}