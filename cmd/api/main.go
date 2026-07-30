package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

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

// setupLogger builds the global slog logger from config and installs it as the
// default. After this returns every call to slog.Info / slog.Error / etc. goes
// through the configured handler at the configured level.
func setupLogger(cfg config.Config) {
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.ToLower(cfg.LogFormat) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

// newNotificationSender builds the Firebase Admin SDK notification.Sender.
// Returns an error if the credentials file is missing or Firebase init fails,
// so main can fail fast with the same os.Exit(1) pattern used by every other
// init above.
func newNotificationSender(ctx context.Context, cfg config.Config) (notification.Sender, error) {
	return notification.NewFirebaseSender(ctx, cfg.FirebaseCredentialsFile)
}

func main() {
	// ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	// defer stop()
	//
	ctx := context.Background()

	appConfig, err := config.Load(".env")
	if err != nil {
		// Logger not configured yet — fall back to stdlib for this single message.
		log.Fatalf("config load failed: %v", err)
	}

	setupLogger(*appConfig)
	slog.Info("logger initialized", "format", appConfig.LogFormat, "level", appConfig.LogLevel)

	addr := appConfig.HTTPAddr
	dsn := appConfig.DatabaseURL
	redisAddr := appConfig.RedisAddr

	db, err := database.NewPostgresDB(ctx, dsn)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("db close failed", "err", err)
		}
	}()

	if err := database.Migrate(ctx, db); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("redis connect failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error("redis close failed", "err", err)
		}
	}()

	logger := watermill.NewSlogLogger(slog.Default())
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{Client: redisClient}, logger)
	if err != nil {
		slog.Error("publisher init failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			slog.Error("publisher close failed", "err", err)
		}
	}()

	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        redisClient,
		ConsumerGroup: "mednotify",
		Consumer:      "api",
		BlockTime:     100 * time.Millisecond,
	}, logger)
	if err != nil {
		slog.Error("subscriber init failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := subscriber.Close(); err != nil {
			slog.Error("subscriber close failed", "err", err)
		}
	}()

	userRepo := database.NewUserRepository(db)
	doctorRepo := database.NewDoctorRepository(db)
	prescriptionRepo := database.NewPrescriptionRepository(db)
	doseRecordRepo := database.NewDoseRecordRepository(db)
	invitationRepo := database.NewInvitationRepository(db)
	deviceTokenRepo := database.NewDeviceTokenRepository(db)
	eventStore := database.NewNotificationEventStore(db)

	lookup := notification.NewPostgresLookup(deviceTokenRepo)

	var authProvider commands.AuthenticationProvider
	firebaseAuthService, err := firebaseauth.NewService(ctx, appConfig.FirebaseCredentialsFile, appConfig.FirebaseCredentialsJSON, appConfig.FirebaseWebAPIKey)
	if err != nil {
		if errors.Is(err, application.ErrAuthNotConfigured) {
			slog.Warn("firebase auth disabled: set FIREBASE_CREDENTIALS_FILE and FIREBASE_WEB_API_KEY")
		} else {
			slog.Error("firebase auth init failed", "err", err)
		}
	} else {
		authProvider = firebaseAuthService
	}

	schedulerAdapter, err := scheduler.NewRedisScheduler(scheduler.RedisSchedulerConfig{Client: redisClient})
	if err != nil {
		slog.Error("scheduler init failed", "err", err)
		os.Exit(1)
	}

	worker := scheduler.NewSchedulerWorker(redisClient, publisher, "", appConfig.NotificationLookback, eventStore)
	worker.WithDoseRecordStore(doseRecordRepo)

	sender, err := newNotificationSender(ctx, *appConfig)
	if err != nil {
		slog.Error("notification sender init failed", "err", err)
		os.Exit(1)
	}

	userCommands := commands.NewUserCommandHandler(userRepo, authProvider)
	authCommands := commands.NewAuthCommandHandler(userRepo, authProvider)
	userQueries := queries.NewUserQueryHandler(userRepo)
	doctorCommands := commands.NewDoctorCommandHandler(doctorRepo)
	doctorQueries := queries.NewDoctorQueryHandler(doctorRepo)
	prescriptionCommands := commands.NewPrescriptionCommandHandler(prescriptionRepo, userRepo, doctorRepo, schedulerAdapter)
	prescriptionQueries := queries.NewPrescriptionQueryHandler(prescriptionRepo)
	inviteCommands := commands.NewInvitationCommandHandler(userRepo, invitationRepo)
	doseCommands := commands.NewDoseRecordCommandHandler(doseRecordRepo, userRepo)
	doseQueries := queries.NewDoseRecordQueryHandler(doseRecordRepo, userRepo, prescriptionRepo)
	linkedUserQueries := queries.NewLinkedUserQueryHandler(userRepo, invitationRepo)
	lgpdQueries := queries.NewLGPDQueryHandler(userRepo, prescriptionRepo, doseRecordRepo, invitationRepo)
	deviceTokenCommands := commands.NewDeviceTokenCommandHandler(deviceTokenRepo, userRepo)
	deviceTokenQueries := queries.NewDeviceTokenQueryHandler(deviceTokenRepo)

	apiServer := httpapi.NewServer(
		userCommands,
		userQueries,
		doctorCommands,
		doctorQueries,
		prescriptionCommands,
		prescriptionQueries,
		authCommands,
	)

	extServer := httpapi.NewExtendedServer(
		userRepo,
		authCommands,
		inviteCommands,
		doseCommands,
		doseQueries,
		linkedUserQueries,
		lgpdQueries,
		deviceTokenCommands,
		deviceTokenQueries,
	).WithInvitationBaseURL(appConfig.InvitationBaseURL)

	firebaseAuth := firebaseauth.GetAuthClient(firebaseAuthService)

	demoSecret := appConfig.DemoPrescriptionSecret

	handler := httpapi.NewRouter(apiServer, extServer, firebaseAuth, demoSecret, *appConfig)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := worker.Run(gCtx); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("scheduler worker stopped: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		cleanup := scheduler.NewRedisCleanupStore(redisClient, "")
		if err := scheduler.StartNotificationConsumer(gCtx, subscriber, sender, userRepo, lookup, cleanup, eventStore); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("notification consumer stopped: %w", err)
		}
		return nil
	})

	go func() {
		if err := g.Wait(); err != nil {
			slog.Error("background workers exited with error", "err", err)
		}
	}()

	slog.Info("http listening", "addr", addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http server failed", "err", err)
		os.Exit(1)
	}
}
