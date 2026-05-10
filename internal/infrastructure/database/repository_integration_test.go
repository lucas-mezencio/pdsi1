package database

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/config"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

var migrateOnce sync.Once

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	appConfig, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = appConfig.DatabaseURL
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
	entity, err := user.NewUser("Alice", "alice@example.com", "+100000000", "token", user.RoleElderly)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	entity.LinkFirebaseAccount("firebase-uid-1")

	if err := repo.Save(ctx, entity); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	found, err := repo.FindByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("failed to find by id: %v", err)
	}
	assertUserEqual(t, entity, found)

	foundByEmail, err := repo.FindByEmail(ctx, entity.Email)
	if err != nil {
		t.Fatalf("failed to find by email: %v", err)
	}
	assertUserEqual(t, entity, foundByEmail)

	foundByFirebaseID, err := repo.FindByFirebaseID(ctx, entity.FirebaseID)
	if err != nil {
		t.Fatalf("failed to find by firebase id: %v", err)
	}
	assertUserEqual(t, entity, foundByFirebaseID)

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
	assertUserEqual(t, entity, list[0])

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
	entity.LinkFirebaseAccount("doctor-firebase-uid-1")

	if err := repo.Save(ctx, entity); err != nil {
		t.Fatalf("failed to save doctor: %v", err)
	}

	found, err := repo.FindByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("failed to find by id: %v", err)
	}
	assertDoctorEqual(t, entity, found)

	byFirebaseID, err := repo.FindByFirebaseID(ctx, entity.FirebaseID)
	if err != nil {
		t.Fatalf("failed to find by firebase id: %v", err)
	}
	assertDoctorEqual(t, entity, byFirebaseID)

	byLicense, err := repo.FindByLicenseNumber(ctx, entity.LicenseNumber)
	if err != nil {
		t.Fatalf("failed to find by license: %v", err)
	}
	assertDoctorEqual(t, entity, byLicense)

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
	assertDoctorEqual(t, entity, list[0])

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
	usr, err := user.NewUser("Alice", "alice@example.com", "+100000000", "token", user.RoleElderly)
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
	assertPrescriptionEqual(t, entity, found)

	activeList, err := repo.FindActive(ctx)
	if err != nil {
		t.Fatalf("failed to find active: %v", err)
	}
	if len(activeList) != 1 {
		t.Fatalf("expected 1 active, got %d", len(activeList))
	}
	assertPrescriptionEqual(t, entity, activeList[0])

	byUser, err := repo.FindByUserID(ctx, usr.ID)
	if err != nil {
		t.Fatalf("failed to find by user: %v", err)
	}
	if len(byUser) != 1 {
		t.Fatalf("expected 1 by user, got %d", len(byUser))
	}
	assertPrescriptionEqual(t, entity, byUser[0])

	byUserActive, err := repo.FindActiveByUserID(ctx, usr.ID)
	if err != nil {
		t.Fatalf("failed to find active by user: %v", err)
	}
	if len(byUserActive) != 1 {
		t.Fatalf("expected 1 active by user, got %d", len(byUserActive))
	}
	assertPrescriptionEqual(t, entity, byUserActive[0])

	byMedic, err := repo.FindByMedicID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("failed to find by medic: %v", err)
	}
	if len(byMedic) != 1 {
		t.Fatalf("expected 1 by medic, got %d", len(byMedic))
	}
	assertPrescriptionEqual(t, entity, byMedic[0])

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("failed to find all: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 total, got %d", len(all))
	}
	assertPrescriptionEqual(t, entity, all[0])

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
	assertPrescriptionEqual(t, entity, updated)

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

func assertUserEqual(t *testing.T, expected *user.User, actual *user.User) {
	t.Helper()
	if expected.ID != actual.ID {
		t.Fatalf("expected id %s, got %s", expected.ID, actual.ID)
	}
	if expected.Name != actual.Name {
		t.Fatalf("expected name %s, got %s", expected.Name, actual.Name)
	}
	if expected.Email != actual.Email {
		t.Fatalf("expected email %s, got %s", expected.Email, actual.Email)
	}
	if expected.Phone != actual.Phone {
		t.Fatalf("expected phone %s, got %s", expected.Phone, actual.Phone)
	}
	if expected.FirebaseToken != actual.FirebaseToken {
		t.Fatalf("expected firebase token %s, got %s", expected.FirebaseToken, actual.FirebaseToken)
	}
	if expected.FirebaseID != actual.FirebaseID {
		t.Fatalf("expected firebase id %s, got %s", expected.FirebaseID, actual.FirebaseID)
	}
	if expected.NotificationsEnabled != actual.NotificationsEnabled {
		t.Fatalf("expected notifications enabled %v, got %v", expected.NotificationsEnabled, actual.NotificationsEnabled)
	}
}

func assertDoctorEqual(t *testing.T, expected *doctor.Doctor, actual *doctor.Doctor) {
	t.Helper()
	if expected.ID != actual.ID {
		t.Fatalf("expected id %s, got %s", expected.ID, actual.ID)
	}
	if expected.Name != actual.Name {
		t.Fatalf("expected name %s, got %s", expected.Name, actual.Name)
	}
	if expected.Email != actual.Email {
		t.Fatalf("expected email %s, got %s", expected.Email, actual.Email)
	}
	if expected.Phone != actual.Phone {
		t.Fatalf("expected phone %s, got %s", expected.Phone, actual.Phone)
	}
	if expected.FirebaseID != actual.FirebaseID {
		t.Fatalf("expected firebase id %s, got %s", expected.FirebaseID, actual.FirebaseID)
	}
	if expected.Specialty != actual.Specialty {
		t.Fatalf("expected specialty %s, got %s", expected.Specialty, actual.Specialty)
	}
	if expected.LicenseNumber != actual.LicenseNumber {
		t.Fatalf("expected license %s, got %s", expected.LicenseNumber, actual.LicenseNumber)
	}
}

func assertPrescriptionEqual(t *testing.T, expected *prescription.Prescription, actual *prescription.Prescription) {
	t.Helper()
	if expected.ID != actual.ID {
		t.Fatalf("expected id %s, got %s", expected.ID, actual.ID)
	}
	if expected.UserID != actual.UserID {
		t.Fatalf("expected user id %s, got %s", expected.UserID, actual.UserID)
	}
	if expected.MedicID != actual.MedicID {
		t.Fatalf("expected medic id %s, got %s", expected.MedicID, actual.MedicID)
	}
	if expected.Active != actual.Active {
		t.Fatalf("expected active %v, got %v", expected.Active, actual.Active)
	}
	if len(expected.Medicaments) != len(actual.Medicaments) {
		t.Fatalf("expected %d medicaments, got %d", len(expected.Medicaments), len(actual.Medicaments))
	}
	for i := range expected.Medicaments {
		assertMedicamentEqual(t, expected.Medicaments[i], actual.Medicaments[i])
	}
}

func assertMedicamentEqual(t *testing.T, expected prescription.Medicament, actual prescription.Medicament) {
	t.Helper()
	if expected.Name != actual.Name {
		t.Fatalf("expected medicament name %s, got %s", expected.Name, actual.Name)
	}
	if expected.Dosage != actual.Dosage {
		t.Fatalf("expected dosage %s, got %s", expected.Dosage, actual.Dosage)
	}
	if expected.Frequency != actual.Frequency {
		t.Fatalf("expected frequency %s, got %s", expected.Frequency, actual.Frequency)
	}
	if expected.Doses != actual.Doses {
		t.Fatalf("expected doses %d, got %d", expected.Doses, actual.Doses)
	}
	if len(expected.Times) != len(actual.Times) {
		t.Fatalf("expected %d times, got %d", len(expected.Times), len(actual.Times))
	}
	for i := range expected.Times {
		if expected.Times[i] != actual.Times[i] {
			t.Fatalf("expected time %s, got %s", expected.Times[i], actual.Times[i])
		}
	}
}
