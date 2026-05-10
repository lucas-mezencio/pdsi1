package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

const (
	defaultKeyPrefix  = "mednotify"
	NotificationTopic = "notifications"
)

type RedisSchedulerConfig struct {
	Client    redis.UniversalClient
	KeyPrefix string
}

type RedisScheduler struct {
	client    redis.UniversalClient
	keyPrefix string
}

type NotificationJob struct {
	ID             string    `json:"id"`
	PrescriptionID string    `json:"prescription_id"`
	UserID         string    `json:"user_id"`
	MedicamentName string    `json:"medicament_name"`
	Dosage         string    `json:"dosage"`
	ScheduledAt    time.Time `json:"scheduled_at"`
}

func NewRedisScheduler(config RedisSchedulerConfig) (*RedisScheduler, error) {
	if config.Client == nil {
		return nil, errors.New("redis client is required")
	}

	keyPrefix := config.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = defaultKeyPrefix
	}

	return &RedisScheduler{
		client:    config.Client,
		keyPrefix: keyPrefix,
	}, nil
}

func (s *RedisScheduler) Schedule(ctx context.Context, schedule prescription.NotificationSchedule, startDate time.Time) error {
	if schedule.TotalDoses <= 0 {
		return nil
	}

	interval := intervalForSchedule(schedule)
	firstTime, err := nextScheduleTime(startDate, schedule.Time, interval)
	if err != nil {
		return err
	}

	pipeline := s.client.TxPipeline()
	for i := 0; i < schedule.TotalDoses; i++ {
		scheduledAt := firstTime.Add(time.Duration(i) * interval)
		job := NotificationJob{
			ID:             uuid.New().String(),
			PrescriptionID: schedule.PrescriptionID,
			UserID:         schedule.UserID,
			MedicamentName: schedule.MedicamentName,
			Dosage:         schedule.Dosage,
			ScheduledAt:    scheduledAt,
		}

		payload, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("marshal notification job: %w", err)
		}

		pipeline.HSet(ctx, s.jobsKey(), job.ID, payload)
		pipeline.ZAdd(ctx, s.scheduleKey(), redis.Z{
			Score:  float64(scheduledAt.UnixNano()),
			Member: job.ID,
		})
		pipeline.SAdd(ctx, s.prescriptionKey(schedule.PrescriptionID), job.ID)
	}

	if _, err := pipeline.Exec(ctx); err != nil {
		return fmt.Errorf("store notification jobs: %w", err)
	}

	return nil
}

func (s *RedisScheduler) CancelByPrescriptionID(ctx context.Context, prescriptionID string) error {
	if prescriptionID == "" {
		return nil
	}

	jobIDs, err := s.client.SMembers(ctx, s.prescriptionKey(prescriptionID)).Result()
	if err != nil {
		return fmt.Errorf("load prescription jobs: %w", err)
	}
	if len(jobIDs) == 0 {
		return nil
	}

	pipeline := s.client.TxPipeline()
	for _, jobID := range jobIDs {
		pipeline.ZRem(ctx, s.scheduleKey(), jobID)
		pipeline.HDel(ctx, s.jobsKey(), jobID)
	}
	pipeline.Del(ctx, s.prescriptionKey(prescriptionID))

	if _, err := pipeline.Exec(ctx); err != nil {
		return fmt.Errorf("cancel notification jobs: %w", err)
	}

	return nil
}

func (s *RedisScheduler) scheduleKey() string {
	return fmt.Sprintf("%s:notification_schedule", s.keyPrefix)
}

func (s *RedisScheduler) jobsKey() string {
	return fmt.Sprintf("%s:notification_jobs", s.keyPrefix)
}

func (s *RedisScheduler) prescriptionKey(prescriptionID string) string {
	return fmt.Sprintf("%s:prescription_jobs:%s", s.keyPrefix, prescriptionID)
}

func intervalForSchedule(schedule prescription.NotificationSchedule) time.Duration {
	if schedule.Frequency != "" {
		duration, err := parseClockDuration(schedule.Frequency)
		if err == nil && duration > 0 {
			return duration
		}
	}

	return 24 * time.Hour
}

func nextScheduleTime(startDate time.Time, timeStr string, interval time.Duration) (time.Time, error) {
	hours, minutes, seconds, err := parseClockTime(timeStr)
	if err != nil {
		return time.Time{}, err
	}

	if interval <= 0 {
		interval = 24 * time.Hour
	}

	scheduled := time.Date(
		startDate.Year(), startDate.Month(), startDate.Day(),
		hours, minutes, seconds, 0,
		startDate.Location(),
	)

	for scheduled.Before(startDate) {
		scheduled = scheduled.Add(interval)
	}

	return scheduled, nil
}

func parseClockTime(value string) (int, int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, 0, 0, prescription.ErrInvalidTimeFormat
	}

	if len(parts[0]) != 2 || len(parts[1]) != 2 {
		return 0, 0, 0, prescription.ErrInvalidTimeFormat
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil || hours < 0 || hours > 23 {
		return 0, 0, 0, prescription.ErrInvalidTimeFormat
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil || minutes < 0 || minutes > 59 {
		return 0, 0, 0, prescription.ErrInvalidTimeFormat
	}

	seconds := 0
	if len(parts) == 3 {
		if len(parts[2]) != 2 {
			return 0, 0, 0, prescription.ErrInvalidTimeFormat
		}
		seconds, err = strconv.Atoi(parts[2])
		if err != nil || seconds < 0 || seconds > 59 {
			return 0, 0, 0, prescription.ErrInvalidTimeFormat
		}
	}

	return hours, minutes, seconds, nil
}

func parseClockDuration(value string) (time.Duration, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, prescription.ErrInvalidTimeFormat
	}

	if len(parts[0]) != 2 || len(parts[1]) != 2 {
		return 0, prescription.ErrInvalidTimeFormat
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil || hours < 0 || hours > 24 {
		return 0, prescription.ErrInvalidTimeFormat
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil || minutes < 0 || minutes > 59 {
		return 0, prescription.ErrInvalidTimeFormat
	}

	seconds := 0
	if len(parts) == 3 {
		if len(parts[2]) != 2 {
			return 0, prescription.ErrInvalidTimeFormat
		}
		seconds, err = strconv.Atoi(parts[2])
		if err != nil || seconds < 0 || seconds > 59 {
			return 0, prescription.ErrInvalidTimeFormat
		}
	}

	if hours == 24 && (minutes != 0 || seconds != 0) {
		return 0, prescription.ErrInvalidTimeFormat
	}

	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second, nil
}
