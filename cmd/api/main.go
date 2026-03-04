package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"

	httpapi "github.com.br/lucas-mezencio/pdsi1/internal/api"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/config"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/database"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/notification"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/scheduler"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appConfig, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	addr := appConfig.HTTPAddr
	dsn := appConfig.DatabaseURL
	redisAddr := appConfig.RedisAddr

	db, err := database.NewPostgresDB(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("db close failed: %v", err)
		}
	}()

	if err := database.Migrate(ctx, db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis connect failed: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("redis close failed: %v", err)
		}
	}()

	logger := watermill.NopLogger{}
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{Client: redisClient}, &logger)
	if err != nil {
		log.Fatalf("publisher init failed: %v", err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			log.Printf("publisher close failed: %v", err)
		}
	}()

	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        redisClient,
		ConsumerGroup: "mednotify",
		Consumer:      "api",
		BlockTime:     100 * time.Millisecond,
	}, &logger)
	if err != nil {
		log.Fatalf("subscriber init failed: %v", err)
	}
	defer func() {
		if err := subscriber.Close(); err != nil {
			log.Printf("subscriber close failed: %v", err)
		}
	}()

	schedulerAdapter, err := scheduler.NewRedisScheduler(scheduler.RedisSchedulerConfig{Client: redisClient})
	if err != nil {
		log.Fatalf("scheduler init failed: %v", err)
	}

	worker := scheduler.NewSchedulerWorker(redisClient, publisher, "")
	go func() {
		if err := worker.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("scheduler worker stopped: %v", err)
		}
	}()

	go func() {
		sender := &notification.DummySender{}
		if err := scheduler.StartNotificationConsumer(ctx, subscriber, sender); err != nil && err != context.Canceled {
			log.Printf("notification consumer stopped: %v", err)
		}
	}()

	userRepo := database.NewUserRepository(db)
	doctorRepo := database.NewDoctorRepository(db)
	prescriptionRepo := database.NewPrescriptionRepository(db)

	userCommands := commands.NewUserCommandHandler(userRepo)
	userQueries := queries.NewUserQueryHandler(userRepo)
	doctorCommands := commands.NewDoctorCommandHandler(doctorRepo)
	doctorQueries := queries.NewDoctorQueryHandler(doctorRepo)
	prescriptionCommands := commands.NewPrescriptionCommandHandler(prescriptionRepo, userRepo, doctorRepo, schedulerAdapter)
	prescriptionQueries := queries.NewPrescriptionQueryHandler(prescriptionRepo)

	apiServer := httpapi.NewServer(
		userCommands,
		userQueries,
		doctorCommands,
		doctorQueries,
		prescriptionCommands,
		prescriptionQueries,
	)

	handler := httpapi.NewRouter(apiServer)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("http listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown failed: %v", err)
	}
}
