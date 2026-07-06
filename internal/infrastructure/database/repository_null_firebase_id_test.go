package database

import (
	"context"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

func TestUserRepository_Save_EmptyFirebaseIDDoesNotCollide(t *testing.T) {
	db := openTestDB(t)
	repo := NewUserRepository(db, testEncryptionKey)
	ctx := context.Background()

	first, err := user.NewUser("Alice", "alice@example.com", "+100000000", "52998224725", "token", user.RoleElderly)
	if err != nil {
		t.Fatalf("failed to create first user: %v", err)
	}

	second, err := user.NewUser("Bob", "bob@example.com", "+100000001", "39053344705", "token", user.RoleElderly)
	if err != nil {
		t.Fatalf("failed to create second user: %v", err)
	}

	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("failed to save first user: %v", err)
	}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatalf("expected second user with empty firebase_id to save without conflict, got: %v", err)
	}

	loaded, err := repo.FindByID(ctx, second.ID)
	if err != nil {
		t.Fatalf("failed to load second user: %v", err)
	}
	if loaded.FirebaseID != "" {
		t.Fatalf("expected empty firebase_id, got %q", loaded.FirebaseID)
	}
}

func TestDoctorRepository_Save_EmptyFirebaseIDDoesNotCollide(t *testing.T) {
	db := openTestDB(t)
	repo := NewDoctorRepository(db, testEncryptionKey)
	ctx := context.Background()

	first, err := doctor.NewDoctor("Dr. Who", "who@example.com", "999", "Time", "LIC-1")
	if err != nil {
		t.Fatalf("failed to create first doctor: %v", err)
	}

	second, err := doctor.NewDoctor("Dr. Strange", "strange@example.com", "998", "Magic", "LIC-2")
	if err != nil {
		t.Fatalf("failed to create second doctor: %v", err)
	}

	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("failed to save first doctor: %v", err)
	}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatalf("expected second doctor with empty firebase_id to save without conflict, got: %v", err)
	}

	loaded, err := repo.FindByID(ctx, second.ID)
	if err != nil {
		t.Fatalf("failed to load second doctor: %v", err)
	}
	if loaded.FirebaseID != "" {
		t.Fatalf("expected empty firebase_id, got %q", loaded.FirebaseID)
	}
}