package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	httpapi "github.com.br/lucas-mezencio/pdsi1/internal/api"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/config"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/database"
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

	userRepo := database.NewUserRepository(db)
	doctorRepo := database.NewDoctorRepository(db)
	prescriptionRepo := database.NewPrescriptionRepository(db)

	userCommands := commands.NewUserCommandHandler(userRepo)
	userQueries := queries.NewUserQueryHandler(userRepo)
	doctorCommands := commands.NewDoctorCommandHandler(doctorRepo)
	doctorQueries := queries.NewDoctorQueryHandler(doctorRepo)
	prescriptionCommands := commands.NewPrescriptionCommandHandler(prescriptionRepo, userRepo, doctorRepo, nil)
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
