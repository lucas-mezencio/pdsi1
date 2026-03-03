package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/database"
)

const (
	defaultAddr = ":8080"
	defaultDSN  = "postgres://mednotify:mednotify@localhost:5432/mednotify?sslmode=disable"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := envString("HTTP_ADDR", defaultAddr)
	dsn := envString("DATABASE_URL", defaultDSN)

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

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("http listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown failed: %v", err)
	}
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
