//go:build integration

package database

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

func seedUserForDeviceTokenTest(t *testing.T, repo *UserRepository, idx int) string {
	t.Helper()
	cpfs := []string{"52998224725", "39053344705"}
	cpf := cpfs[idx%len(cpfs)]
	u, err := user.NewUser(
		fmt.Sprintf("Alice-%d", idx),
		fmt.Sprintf("alice-%d@example.com", idx),
		fmt.Sprintf("+1000000000%d", idx),
		cpf,
		"token",
		user.RoleElderly,
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := repo.Save(context.Background(), u); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}
	return u.ID
}

func TestDeviceTokenRepository_CRUD(t *testing.T) {
	db := openTestDB(t)
	userRepo := NewUserRepository(db)
	repo := NewDeviceTokenRepository(db)
	ctx := context.Background()

	userA := seedUserForDeviceTokenTest(t, userRepo, 1)
	userB := seedUserForDeviceTokenTest(t, userRepo, 2)

	dt, err := devicetoken.New(userA, "token-aaaa-1111")
	if err != nil {
		t.Fatalf("new token: %v", err)
	}

	saved, err := repo.Save(ctx, dt)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.ID != dt.ID {
		t.Fatalf("expected id %s, got %s", dt.ID, saved.ID)
	}

	found, err := repo.FindByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.UserID != userA {
		t.Fatalf("expected user_id %s, got %s", userA, found.UserID)
	}
	if found.Token != dt.Token {
		t.Fatalf("expected token %s, got %s", dt.Token, found.Token)
	}
	if !found.Enabled {
		t.Fatal("expected enabled=true on new token")
	}

	dt2, err := devicetoken.New(userA, "token-bbbb-2222")
	if err != nil {
		t.Fatalf("new token2: %v", err)
	}
	if _, err := repo.Save(ctx, dt2); err != nil {
		t.Fatalf("save2: %v", err)
	}

	list, err := repo.FindByUserID(ctx, userA)
	if err != nil {
		t.Fatalf("find by user: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 tokens for userA, got %d", len(list))
	}

	conflict, err := devicetoken.New(userB, dt.Token)
	if err != nil {
		t.Fatalf("new conflict: %v", err)
	}
	_, err = repo.Save(ctx, conflict)
	if !errors.Is(err, devicetoken.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	updated, err := repo.SetEnabled(ctx, saved.ID, false)
	if err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	if updated.Enabled {
		t.Fatal("expected enabled=false after SetEnabled(false)")
	}

	foundAgain, err := repo.FindByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("find after set enabled: %v", err)
	}
	if foundAgain.Enabled {
		t.Fatal("expected enabled=false persisted")
	}

	if err := repo.Delete(ctx, saved.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = repo.FindByID(ctx, saved.ID)
	if !errors.Is(err, devicetoken.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
