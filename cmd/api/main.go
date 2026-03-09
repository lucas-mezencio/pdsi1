package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"

	httpapi "github.com.br/lucas-mezencio/pdsi1/internal/api"
	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/config"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/database"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/firebaseauth"
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

	userRepo := database.NewUserRepository(db)
	doctorRepo := database.NewDoctorRepository(db)
	prescriptionRepo := database.NewPrescriptionRepository(db)
	doseRecordRepo := database.NewDoseRecordRepository(db)
	invitationRepo := database.NewInvitationRepository(db)
	eventStore := database.NewNotificationEventStore(db)

	var authProvider commands.AuthenticationProvider
	firebaseAuthService, err := firebaseauth.NewService(ctx, appConfig.FirebaseCredentialsFile, appConfig.FirebaseWebAPIKey)
	if err != nil {
		if errors.Is(err, application.ErrAuthNotConfigured) {
			log.Printf("firebase auth disabled: set FIREBASE_CREDENTIALS_FILE and FIREBASE_WEB_API_KEY")
		} else {
			log.Printf("firebase auth init failed: %v", err)
		}
	} else {
		authProvider = firebaseAuthService
	}

	schedulerAdapter, err := scheduler.NewRedisScheduler(scheduler.RedisSchedulerConfig{Client: redisClient})
	if err != nil {
		log.Fatalf("scheduler init failed: %v", err)
	}

	worker := scheduler.NewSchedulerWorker(redisClient, publisher, "", appConfig.NotificationLookback, eventStore)
	worker.WithDoseRecordStore(doseRecordRepo)
	go func() {
		if err := worker.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("scheduler worker stopped: %v", err)
		}
	}()

	go func() {
		var sender notification.Sender
		switch appConfig.NotifierMode {
		case "dev":
			sender = &notification.DummySender{}
		case "ready", "":
			firebaseSender, err := notification.NewFirebaseSender(ctx, appConfig.FirebaseCredentialsFile)
			if err != nil {
				log.Printf("firebase sender init failed: %v", err)
				return
			}
			sender = firebaseSender
		default:
			log.Printf("unknown notifier mode: %s", appConfig.NotifierMode)
			return
		}

		cleanup := scheduler.NewRedisCleanupStore(redisClient, "")
		if err := scheduler.StartNotificationConsumer(ctx, subscriber, sender, userRepo, cleanup); err != nil && err != context.Canceled {
			log.Printf("notification consumer stopped: %v", err)
		}
	}()

	userCommands := commands.NewUserCommandHandler(userRepo)
	authCommands := commands.NewAuthCommandHandler(userRepo, authProvider)
	doctorAuthCommands := commands.NewDoctorAuthCommandHandler(doctorRepo, authProvider)
	userQueries := queries.NewUserQueryHandler(userRepo)
	doctorCommands := commands.NewDoctorCommandHandler(doctorRepo)
	doctorQueries := queries.NewDoctorQueryHandler(doctorRepo)
	prescriptionCommands := commands.NewPrescriptionCommandHandler(prescriptionRepo, userRepo, doctorRepo, schedulerAdapter)
	prescriptionQueries := queries.NewPrescriptionQueryHandler(prescriptionRepo)
	inviteCommands := commands.NewInvitationCommandHandler(userRepo, invitationRepo)
	doseCommands := commands.NewDoseRecordCommandHandler(doseRecordRepo, userRepo)
	doseQueries := queries.NewDoseRecordQueryHandler(doseRecordRepo, userRepo)
	linkedUserQueries := queries.NewLinkedUserQueryHandler(userRepo, invitationRepo)

	apiServer := httpapi.NewServer(
		userCommands,
		userQueries,
		doctorCommands,
		doctorQueries,
		prescriptionCommands,
		prescriptionQueries,
	)

	extServer := httpapi.NewExtendedServer(
		userRepo,
		authCommands,
		doctorAuthCommands,
		inviteCommands,
		doseCommands,
		doseQueries,
		linkedUserQueries,
	)

	handler := httpapi.NewRouter(apiServer, extServer)

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
