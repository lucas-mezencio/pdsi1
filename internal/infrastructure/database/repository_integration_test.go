package database

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

var migrateOnce sync.Once

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL or DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	db, err := NewPostgresDB(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect db: %v", err)
	}

	migrateOnce.Do(func() {
		if err := Migrate(ctx, db); err != nil {
			t.Fatalf("failed to run migrations: %v", err)
		}
	})

	cleanupDB(t, db)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func cleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()
	query := `TRUNCATE medicaments, prescriptions, doctors, users RESTART IDENTITY CASCADE`
	if _, err := db.ExecContext(context.Background(), query); err != nil {
		t.Fatalf("failed to cleanup db: %v", err)
	}
}

func TestUserRepository_CRUD(t *testing.T) {
	db := openTestDB(t)
	repo := NewUserRepository(db)

	ctx := context.Background()
	entity, err := user.NewUser("Alice", "alice@example.com", "+100000000", "token")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if err := repo.Save(ctx, entity); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	found, err := repo.FindByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("failed to find by id: %v", err)
	}
	if found.Email != entity.Email {
		t.Fatalf("expected email %s, got %s", entity.Email, found.Email)
	}

	foundByEmail, err := repo.FindByEmail(ctx, entity.Email)
	if err != nil {
		t.Fatalf("failed to find by email: %v", err)
	}
	if foundByEmail.ID != entity.ID {
		t.Fatalf("expected id %s, got %s", entity.ID, foundByEmail.ID)
	}

	entity.Update("Alice Updated", "alice.updated@example.com", "+200000000")
	if err := repo.Save(ctx, entity); err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	list, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("failed to find all: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 user, got %d", len(list))
	}

	exists, err := repo.Exists(ctx, entity.ID)
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if !exists {
		t.Fatal("expected user to exist")
	}

	if err := repo.Delete(ctx, entity.ID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	_, err = repo.FindByID(ctx, entity.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDoctorRepository_CRUD(t *testing.T) {
	db := openTestDB(t)
	repo := NewDoctorRepository(db)

	ctx := context.Background()
	entity, err := doctor.NewDoctor("Dr. Who", "who@example.com", "999", "Time", "LIC-1")
	if err != nil {
		t.Fatalf("failed to create doctor: %v", err)
	}

	if err := repo.Save(ctx, entity); err != nil {
		t.Fatalf("failed to save doctor: %v", err)
	}

	found, err := repo.FindByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("failed to find by id: %v", err)
	}
	if found.Email != entity.Email {
		t.Fatalf("expected email %s, got %s", entity.Email, found.Email)
	}

	byLicense, err := repo.FindByLicenseNumber(ctx, entity.LicenseNumber)
	if err != nil {
		t.Fatalf("failed to find by license: %v", err)
	}
	if byLicense.ID != entity.ID {
		t.Fatalf("expected id %s, got %s", entity.ID, byLicense.ID)
	}

	entity.Update("Dr. Updated", "updated@example.com", "111", "General")
	if err := repo.Save(ctx, entity); err != nil {
		t.Fatalf("failed to update doctor: %v", err)
	}

	list, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("failed to find all: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 doctor, got %d", len(list))
	}

	exists, err := repo.Exists(ctx, entity.ID)
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if !exists {
		t.Fatal("expected doctor to exist")
	}

	if err := repo.Delete(ctx, entity.ID); err != nil {
		t.Fatalf("failed to delete doctor: %v", err)
	}

	_, err = repo.FindByID(ctx, entity.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestPrescriptionRepository_CRUD(t *testing.T) {
	db := openTestDB(t)
	userRepo := NewUserRepository(db)
	doctorRepo := NewDoctorRepository(db)
	repo := NewPrescriptionRepository(db)

	ctx := context.Background()
	usr, err := user.NewUser("Alice", "alice@example.com", "+100000000", "token")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := userRepo.Save(ctx, usr); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	doc, err := doctor.NewDoctor("Dr. Who", "who@example.com", "999", "Time", "LIC-1")
	if err != nil {
		t.Fatalf("failed to create doctor: %v", err)
	}
	if err := doctorRepo.Save(ctx, doc); err != nil {
		t.Fatalf("failed to save doctor: %v", err)
	}

	entity, err := prescription.NewPrescription(usr.ID, doc.ID, []prescription.Medicament{{
		Name:      "Med",
		Dosage:    "1",
		Frequency: "12:00",
		Times:     []string{"08:00", "20:00"},
		Doses:     2,
	}})
	if err != nil {
		t.Fatalf("failed to create prescription: %v", err)
	}

	if err := repo.Save(ctx, entity); err != nil {
		t.Fatalf("failed to save prescription: %v", err)
	}

	found, err := repo.FindByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("failed to find by id: %v", err)
	}
	if found.UserID != usr.ID {
		t.Fatalf("expected user id %s, got %s", usr.ID, found.UserID)
	}
	if len(found.Medicaments) != 1 {
		t.Fatalf("expected 1 medicament, got %d", len(found.Medicaments))
	}

	activeList, err := repo.FindActive(ctx)
	if err != nil {
		t.Fatalf("failed to find active: %v", err)
	}
	if len(activeList) != 1 {
		t.Fatalf("expected 1 active, got %d", len(activeList))
	}

	byUser, err := repo.FindByUserID(ctx, usr.ID)
	if err != nil {
		t.Fatalf("failed to find by user: %v", err)
	}
	if len(byUser) != 1 {
		t.Fatalf("expected 1 by user, got %d", len(byUser))
	}

	byUserActive, err := repo.FindActiveByUserID(ctx, usr.ID)
	if err != nil {
		t.Fatalf("failed to find active by user: %v", err)
	}
	if len(byUserActive) != 1 {
		t.Fatalf("expected 1 active by user, got %d", len(byUserActive))
	}

	byMedic, err := repo.FindByMedicID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("failed to find by medic: %v", err)
	}
	if len(byMedic) != 1 {
		t.Fatalf("expected 1 by medic, got %d", len(byMedic))
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("failed to find all: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 total, got %d", len(all))
	}

	entity.Medicaments = []prescription.Medicament{{
		Name:      "Med-2",
		Dosage:    "2",
		Frequency: "24:00",
		Times:     []string{"09:00"},
		Doses:     1,
	}}
	if err := repo.Save(ctx, entity); err != nil {
		t.Fatalf("failed to update medicaments: %v", err)
	}

	updated, err := repo.FindByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("failed to load updated: %v", err)
	}
	if len(updated.Medicaments) != 1 || updated.Medicaments[0].Name != "Med-2" {
		t.Fatal("expected medicaments to be replaced")
	}

	exists, err := repo.Exists(ctx, entity.ID)
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if !exists {
		t.Fatal("expected prescription to exist")
	}

	if err := repo.Delete(ctx, entity.ID); err != nil {
		t.Fatalf("failed to delete prescription: %v", err)
	}

	_, err = repo.FindByID(ctx, entity.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
