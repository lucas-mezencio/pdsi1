package testcontainers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	pgmod "github.com/testcontainers/testcontainers-go/modules/postgres"
	_ "github.com/lib/pq"
)

func StartPostgresContainer(ctx context.Context) (*sql.DB, string, func()) {
	pgContainer, err := pgmod.Run(ctx,
		"postgres:18-alpine",
		pgmod.WithDatabase("mednotify"),
		pgmod.WithUsername("mednotify"),
		pgmod.WithPassword("mednotify"),
	)
	if err != nil {
		fmt.Printf("WARNING: failed to start postgres container: %v\n", err)
		return nil, "", func() {}
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable", "connect_timeout=10")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		fmt.Printf("WARNING: failed to get postgres dsn: %v\n", err)
		return nil, "", func() {}
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		fmt.Printf("WARNING: failed to open postgres connection: %v\n", err)
		return nil, "", func() {}
	}

	for i := 0; i < 30; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, time.Second)
		err := db.PingContext(pingCtx)
		cancel()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
		fmt.Printf("WARNING: failed to ping postgres container after retries: %v\n", err)
		return nil, "", func() {}
	}

	cleanup := func() {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return db, dsn, cleanup
}

func PostgresContainerName() string {
	return fmt.Sprintf("tc-pg-%s", uuid.New().String()[:8])
}