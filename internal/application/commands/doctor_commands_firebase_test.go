package commands

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

// Doctor is purely local under the post-fix flow: no AuthenticationProvider
// is consulted and no firebase_id is linked. These tests lock that contract.

func TestDoctorCommandHandler_Create_SucceedsWithoutPasswordOrAuth(t *testing.T) {
	repo := &mockDoctorRepo{}
	var saved *doctor.Doctor
	repo.saveFn = func(ctx context.Context, entity *doctor.Doctor) error {
		saved = entity
		return nil
	}

	handler := NewDoctorCommandHandler(repo)

	created, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. Who",
		Email:         "who@example.com",
		Phone:         "+55119...",
		Specialty:     "Time",
		LicenseNumber: "LIC-1",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created == nil {
		t.Fatal("expected doctor to be created")
	}
	if created.FirebaseID != "" {
		t.Fatalf("expected no firebase_id (doctor is local), got %q", created.FirebaseID)
	}
	if saved == nil {
		t.Fatal("expected doctor to be saved")
	}
	if saved.FirebaseID != "" {
		t.Fatalf("expected saved doctor to have no firebase_id, got %q", saved.FirebaseID)
	}
}

func TestDoctorCommandHandler_Create_DuplicateEmailRejected(t *testing.T) {
	repo := &mockDoctorRepo{
		findByEmailFn: func(ctx context.Context, email string) (*doctor.Doctor, error) {
			return &doctor.Doctor{Email: email}, nil
		},
	}

	handler := NewDoctorCommandHandler(repo)

	_, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. Dup",
		Email:         "dup@example.com",
		Phone:         "+55119...",
		Specialty:     "Trauma",
		LicenseNumber: "LIC-DUP",
	})

	if !errors.Is(err, application.ErrEmailAlreadyInUse) {
		t.Fatalf("expected ErrEmailAlreadyInUse, got %v", err)
	}
}

func TestDoctorCommandHandler_Create_DuplicateLicenseRejected(t *testing.T) {
	repo := &mockDoctorRepo{
		findByLicenseNumberFn: func(ctx context.Context, license string) (*doctor.Doctor, error) {
			return &doctor.Doctor{LicenseNumber: license}, nil
		},
	}

	handler := NewDoctorCommandHandler(repo)

	_, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. Dup",
		Email:         "dup@example.com",
		Phone:         "+55119...",
		Specialty:     "Trauma",
		LicenseNumber: "LIC-DUP",
	})

	if !errors.Is(err, application.ErrLicenseAlreadyInUse) {
		t.Fatalf("expected ErrLicenseAlreadyInUse, got %v", err)
	}
}

func TestDoctorCommandHandler_Create_RepoFailurePropagates(t *testing.T) {
	repoErr := errors.New("db boom")
	repo := &mockDoctorRepo{
		saveFn: func(ctx context.Context, entity *doctor.Doctor) error {
			return repoErr
		},
	}

	handler := NewDoctorCommandHandler(repo)

	_, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. X",
		Email:         "x@example.com",
		Phone:         "+55119...",
		Specialty:     "Trauma",
		LicenseNumber: "LIC-X",
	})

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected db error to propagate, got %v", err)
	}
}

func TestDoctorCommandHandler_Create_MissingFieldsRejected(t *testing.T) {
	repo := &mockDoctorRepo{}
	handler := NewDoctorCommandHandler(repo)

	tests := []struct {
		name    string
		cmd     CreateDoctorCommand
		wantErr error
	}{
		{
			name:    "missing name",
			cmd:     CreateDoctorCommand{Email: "a@b", Phone: "1", LicenseNumber: "L"},
			wantErr: application.ErrInvalidInput,
		},
		{
			name:    "missing email",
			cmd:     CreateDoctorCommand{Name: "Dr", Phone: "1", LicenseNumber: "L"},
			wantErr: application.ErrInvalidInput,
		},
		{
			name:    "missing phone",
			cmd:     CreateDoctorCommand{Name: "Dr", Email: "a@b", LicenseNumber: "L"},
			wantErr: application.ErrInvalidInput,
		},
		{
			name:    "missing license_number",
			cmd:     CreateDoctorCommand{Name: "Dr", Email: "a@b", Phone: "1"},
			wantErr: application.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.Create(context.Background(), tt.cmd)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}