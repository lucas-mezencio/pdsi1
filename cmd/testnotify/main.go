package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com.br/lucas-mezencio/pdsi1/internal/config"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/database"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/notification"
)

func main() {
	var (
		userID     = flag.String("user-id", "", "target user UUID (required)")
		medicament = flag.String("medicament", "Aspirin", "medicament name")
		dosage     = flag.String("dosage", "100mg", "dosage label")
	)
	flag.Parse()

	if *userID == "" {
		log.Fatalf("-user-id is required")
	}

	cfg, err := config.Load("cmd/testnotify/.env")
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx := context.Background()

	db, err := database.NewPostgresDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	tokenRepo := database.NewDeviceTokenRepository(db)
	lookup := notification.NewPostgresLookup(tokenRepo)

	sender, err := notification.NewFirebaseSender(ctx, cfg.FirebaseCredentialsFile)
	if err != nil {
		log.Fatalf("firebase: %v", err)
	}

	tokens, err := lookup.ActiveTokens(ctx, *userID)
	if err != nil {
		log.Fatalf("lookup: %v", err)
	}
	if len(tokens) == 0 {
		log.Fatalf("no active device tokens for user %s", *userID)
	}

	failures := 0
	for _, tok := range tokens {
		err := sender.Send(ctx, notification.Notification{
			UserID:         *userID,
			MedicamentName: *medicament,
			Dosage:         *dosage,
			FirebaseToken:  tok.FCMToken,
		})
		if err != nil {
			failures++
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", tok.DeviceTokenID, err)
			continue
		}
		fmt.Printf("OK   %s\n", tok.DeviceTokenID)
	}
	if failures > 0 {
		os.Exit(1)
	}
}
