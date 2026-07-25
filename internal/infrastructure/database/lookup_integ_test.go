//go:build integration

package database

import (
	"context"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/notification"
)

func TestPostgresLookup_ActiveTokens(t *testing.T) {
	db := openTestDB(t)
	userRepo := NewUserRepository(db)
	tokenRepo := NewDeviceTokenRepository(db)
	lookup := notification.NewPostgresLookup(tokenRepo)

	ctx := context.Background()
	userID := seedUserForDeviceTokenTest(t, userRepo, 1)

	enabled, err := devicetoken.New(userID, "fcm-token-enabled-1")
	if err != nil {
		t.Fatalf("new enabled token: %v", err)
	}
	disabled, err := devicetoken.New(userID, "fcm-token-disabled-1")
	if err != nil {
		t.Fatalf("new disabled token: %v", err)
	}

	for _, dt := range []*devicetoken.DeviceToken{enabled, disabled} {
		if _, err := tokenRepo.Save(ctx, dt); err != nil {
			t.Fatalf("save %s: %v", dt.Token, err)
		}
	}
	if _, err := tokenRepo.SetEnabled(ctx, disabled.ID, false); err != nil {
		t.Fatalf("disable: %v", err)
	}

	tokens, err := lookup.ActiveTokens(ctx, userID)
	if err != nil {
		t.Fatalf("ActiveTokens: %v", err)
	}

	if len(tokens) != 1 {
		t.Fatalf("expected 1 active token, got %d", len(tokens))
	}
	if tokens[0].FCMToken != "fcm-token-enabled-1" {
		t.Fatalf("unexpected token: %s", tokens[0].FCMToken)
	}
	if tokens[0].DeviceTokenID != enabled.ID {
		t.Fatalf("unexpected id: %s", tokens[0].DeviceTokenID)
	}
}
