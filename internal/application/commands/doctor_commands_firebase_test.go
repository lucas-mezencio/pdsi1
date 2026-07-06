package commands

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

func TestDoctorCommandHandler_Create_AutoCreatesFirebaseUserAndLinks(t *testing.T) {
	repo := &mockDoctorRepo{}
	var saved *doctor.Doctor
	repo.saveFn = func(ctx context.Context, entity *doctor.Doctor) error {
		saved = entity
		return nil
	}

	auth := &stubAuthProvider{createUID: "firebase-uid-doc-abc"}
	handler := NewDoctorCommandHandler(repo, auth)

	created, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. Who",
		Email:         "who@example.com",
		Phone:         "+55119...",
		Password:      "S3cretP@ss",
		Specialty:     "Time",
		LicenseNumber: "LIC-1",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created.FirebaseID != "firebase-uid-doc-abc" {
		t.Fatalf("expected firebase_id linked, got %q", created.FirebaseID)
	}
	if saved == nil || saved.FirebaseID != "firebase-uid-doc-abc" {
		t.Fatalf("expected saved doctor to carry firebase_id, got %+v", saved)
	}
	if auth.createCalls != 1 {
		t.Fatalf("expected CreateUser called once, got %d", auth.createCalls)
	}
}

func TestDoctorCommandHandler_Create_DuplicateEmailRejected(t *testing.T) {
	repo := &mockDoctorRepo{
		findByEmailFn: func(ctx context.Context, email string) (*doctor.Doctor, error) {
			return &doctor.Doctor{Email: email}, nil
		},
	}
	auth := &stubAuthProvider{}
	handler := NewDoctorCommandHandler(repo, auth)

	_, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. Dup",
		Email:         "dup@example.com",
		Phone:         "+55119...",
		Password:      "S3cretP@ss",
		LicenseNumber: "LIC-DUP",
	})

	if !errors.Is(err, application.ErrEmailAlreadyInUse) {
		t.Fatalf("expected ErrEmailAlreadyInUse, got %v", err)
	}
	if auth.createCalls != 0 {
		t.Fatalf("expected firebase NOT to be called, got %d", auth.createCalls)
	}
}

func TestDoctorCommandHandler_Create_DuplicateLicenseRejected(t *testing.T) {
	repo := &mockDoctorRepo{
		findByLicenseNumberFn: func(ctx context.Context, license string) (*doctor.Doctor, error) {
			return &doctor.Doctor{LicenseNumber: license}, nil
		},
	}
	auth := &stubAuthProvider{}
	handler := NewDoctorCommandHandler(repo, auth)

	_, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. Dup",
		Email:         "dup@example.com",
		Phone:         "+55119...",
		Password:      "S3cretP@ss",
		LicenseNumber: "LIC-DUP",
	})

	if !errors.Is(err, application.ErrLicenseAlreadyInUse) {
		t.Fatalf("expected ErrLicenseAlreadyInUse, got %v", err)
	}
	if auth.createCalls != 0 {
		t.Fatalf("expected firebase NOT to be called, got %d", auth.createCalls)
	}
}

func TestDoctorCommandHandler_Create_RepoFailureRollsBackFirebase(t *testing.T) {
	repoErr := errors.New("db boom")
	repo := &mockDoctorRepo{
		saveFn: func(ctx context.Context, entity *doctor.Doctor) error {
			return repoErr
		},
	}
	auth := &stubAuthProvider{createUID: "firebase-uid-doc-rollback"}
	handler := NewDoctorCommandHandler(repo, auth)

	_, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. X",
		Email:         "x@example.com",
		Phone:         "+55119...",
		Password:      "S3cretP@ss",
		LicenseNumber: "LIC-X",
	})

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected db error to propagate, got %v", err)
	}
	if auth.deleteCalls != 1 || auth.deleteUID != "firebase-uid-doc-rollback" {
		t.Fatalf("expected firebase user rolled back, got deleteCalls=%d deleteUID=%q", auth.deleteCalls, auth.deleteUID)
	}
}

func TestDoctorCommandHandler_Create_MissingPasswordRejected(t *testing.T) {
	repo := &mockDoctorRepo{}
	auth := &stubAuthProvider{}
	handler := NewDoctorCommandHandler(repo, auth)

	_, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. Y",
		Email:         "y@example.com",
		Phone:         "+55119...",
		LicenseNumber: "LIC-Y",
	})

	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if auth.createCalls != 0 {
		t.Fatalf("expected firebase NOT to be called without password, got %d", auth.createCalls)
	}
}