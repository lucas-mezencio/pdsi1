package scheduler

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisCleanupStore struct {
	client    redis.UniversalClient
	keyPrefix string
}

func NewRedisCleanupStore(client redis.UniversalClient, keyPrefix string) *RedisCleanupStore {
	if keyPrefix == "" {
		keyPrefix = defaultKeyPrefix
	}
	return &RedisCleanupStore{client: client, keyPrefix: keyPrefix}
}

func (r *RedisCleanupStore) Delete(ctx context.Context, jobID string) error {
	pipeline := r.client.TxPipeline()
	pipeline.ZRem(ctx, r.scheduleKey(), jobID)
	pipeline.HDel(ctx, r.jobsKey(), jobID)
	if _, err := pipeline.Exec(ctx); err != nil {
		return fmt.Errorf("delete redis job: %w", err)
	}
	return nil
}

func (r *RedisCleanupStore) scheduleKey() string {
	return fmt.Sprintf("%s:notification_schedule", r.keyPrefix)
}

func (r *RedisCleanupStore) jobsKey() string {
	return fmt.Sprintf("%s:notification_jobs", r.keyPrefix)
}
