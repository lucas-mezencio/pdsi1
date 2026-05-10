package commands

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

func TestDoctorAuthCommandHandler_Register(t *testing.T) {
	repo := &mockDoctorRepo{}
	var saved *doctor.Doctor
	repo.saveFn = func(ctx context.Context, entity *doctor.Doctor) error {
		saved = entity
		return nil
	}

	authProvider := &mockAuthProvider{
		createUserFn: func(ctx context.Context, email, password string) (string, error) {
			return "doctor-firebase-uid-1", nil
		},
	}

	handler := NewDoctorAuthCommandHandler(repo, authProvider)
	entity, err := handler.Register(context.Background(), RegisterDoctorCommand{
		Name:          "Dr. House",
		Email:         "house@example.com",
		Phone:         "+100000000",
		Password:      "Password123!",
		Specialty:     "Clinico",
		LicenseNumber: "CRM-12345",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity == nil {
		t.Fatal("expected doctor")
	}
	if entity.FirebaseID != "doctor-firebase-uid-1" {
		t.Fatalf("expected firebase id doctor-firebase-uid-1, got %s", entity.FirebaseID)
	}
	if saved == nil || saved.FirebaseID != "doctor-firebase-uid-1" {
		t.Fatal("expected saved doctor with firebase id")
	}
}

func TestDoctorAuthCommandHandler_Register_LicenseAlreadyInUse(t *testing.T) {
	repo := &mockDoctorRepo{
		findByEmailFn: func(ctx context.Context, email string) (*doctor.Doctor, error) {
			return nil, doctor.ErrDoctorNotFound
		},
		findByLicenseFn: func(ctx context.Context, license string) (*doctor.Doctor, error) {
			return &doctor.Doctor{ID: "doc-1", LicenseNumber: license}, nil
		},
	}
	handler := NewDoctorAuthCommandHandler(repo, &mockAuthProvider{})

	_, err := handler.Register(context.Background(), RegisterDoctorCommand{
		Name:          "Dr. House",
		Email:         "house@example.com",
		Phone:         "+100000000",
		Password:      "Password123!",
		Specialty:     "Clinico",
		LicenseNumber: "CRM-12345",
	})
	if !errors.Is(err, application.ErrLicenseAlreadyInUse) {
		t.Fatalf("expected license already in use, got %v", err)
	}
}

func TestDoctorAuthCommandHandler_LoginByFirebaseID(t *testing.T) {
	authProvider := &mockAuthProvider{
		signInFn: func(ctx context.Context, email, password string) (string, error) {
			return "doctor-firebase-uid-1", nil
		},
	}
	repo := &mockDoctorRepo{
		findByFirebaseIDFn: func(ctx context.Context, firebaseID string) (*doctor.Doctor, error) {
			return &doctor.Doctor{ID: "doc-1", FirebaseID: firebaseID}, nil
		},
	}
	handler := NewDoctorAuthCommandHandler(repo, authProvider)

	entity, err := handler.Login(context.Background(), LoginDoctorCommand{
		Email:    "house@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity.ID != "doc-1" {
		t.Fatalf("expected doc-1, got %s", entity.ID)
	}
}

func TestDoctorAuthCommandHandler_Login_BackfillsFirebaseID(t *testing.T) {
	authProvider := &mockAuthProvider{
		signInFn: func(ctx context.Context, email, password string) (string, error) {
			return "doctor-firebase-uid-1", nil
		},
	}

	var saved *doctor.Doctor
	legacyDoctor := &doctor.Doctor{ID: "doc-legacy", Email: "legacy.doc@example.com", LicenseNumber: "CRM-67890"}
	repo := &mockDoctorRepo{
		findByFirebaseIDFn: func(ctx context.Context, firebaseID string) (*doctor.Doctor, error) {
			return nil, doctor.ErrDoctorNotFound
		},
		findByEmailFn: func(ctx context.Context, email string) (*doctor.Doctor, error) {
			return legacyDoctor, nil
		},
		saveFn: func(ctx context.Context, entity *doctor.Doctor) error {
			saved = entity
			return nil
		},
	}
	handler := NewDoctorAuthCommandHandler(repo, authProvider)

	entity, err := handler.Login(context.Background(), LoginDoctorCommand{
		Email:    "legacy.doc@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity.FirebaseID != "doctor-firebase-uid-1" {
		t.Fatalf("expected firebase id backfilled, got %s", entity.FirebaseID)
	}
	if saved == nil || saved.FirebaseID != "doctor-firebase-uid-1" {
		t.Fatal("expected saved doctor with backfilled firebase id")
	}
}
