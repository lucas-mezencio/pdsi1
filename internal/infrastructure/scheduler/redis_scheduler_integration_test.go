package scheduler

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com.br/lucas-mezencio/pdsi1/internal/config"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

var redisOnce sync.Once

func openTestRedis(t *testing.T) redis.UniversalClient {
	t.Helper()

	appConfig, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = appConfig.RedisAddr
	}
	if addr == "" {
		t.Skip("REDIS_ADDR is not set")
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unavailable: %v", err)
	}

	redisOnce.Do(func() {
		if err := client.FlushDB(ctx).Err(); err != nil {
			t.Fatalf("redis flush failed: %v", err)
		}
	})

	t.Cleanup(func() {
		_ = client.FlushDB(context.Background()).Err()
		_ = client.Close()
	})

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

	start := time.Now().Add(3 * time.Second).Truncate(time.Second)
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

	worker := NewSchedulerWorker(client, publisher, defaultKeyPrefix)
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
	for len(received) < schedule.TotalDoses {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for notifications, got %d", len(received))
		case msg := <-messages:
			var job NotificationJob
			if err := json.Unmarshal(msg.Payload, &job); err != nil {
				t.Fatalf("unmarshal notification job: %v", err)
			}
			received = append(received, job)
			msg.Ack()
		}
	}

	for i := 0; i < schedule.TotalDoses; i++ {
		expected := start.Add(time.Duration(i) * time.Second)
		actual := received[i].ScheduledAt
		if expected.Hour() != actual.Hour() || expected.Minute() != actual.Minute() || expected.Second() != actual.Second() {
			t.Fatalf("expected scheduled time %s, got %s", expected.Format(time.RFC3339), actual.Format(time.RFC3339))
		}
	}
}
