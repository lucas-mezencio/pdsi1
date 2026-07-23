package testcontainers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
)

func StartRedisContainer(ctx context.Context) (redis.UniversalClient, string, func()) {
	info, err := newRedisContainer(ctx)
	if err != nil {
		fmt.Printf("WARNING: failed to start redis container: %v\n", err)
		return nil, "", func() {}
	}

	client := redis.NewClient(&redis.Options{Addr: info.Addr})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		info.Terminate(ctx)
		fmt.Printf("WARNING: failed to ping redis container: %v\n", err)
		return nil, "", func() {}
	}

	cleanup := func() {
		_ = client.Close()
		info.Terminate(ctx)
	}

	return client, info.Addr, cleanup
}

func RedisContainerName() string {
	return fmt.Sprintf("tc-redis-%s", uuid.New().String()[:8])
}

type redisContainerInfo struct {
	Host     string
	Port     int
	Addr     string
	terminate func()
}

func (r *redisContainerInfo) Terminate(ctx context.Context) {
	if r.terminate != nil {
		r.terminate()
	}
}

func newRedisContainer(ctx context.Context) (*redisContainerInfo, error) {
	req := testcontainers.ContainerRequest{
		Image: "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get port: %w", err)
	}

	info := &redisContainerInfo{
		Host: host,
		Port: int(port.Num()),
		Addr: fmt.Sprintf("%s:%d", host, port.Num()),
		terminate: func() {
			_ = container.Terminate(context.Background())
		},
	}

	return info, nil
}