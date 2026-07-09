package commands

import (
	"context"
	"errors"
	"strings"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

// CreateDoctorCommand holds data to create a new doctor.
//
// Doctor is a semantic entity that identifies who issued a prescription;
// it is not an authenticatable principal and has no password. firebase_id
// is therefore always empty for doctors.
type CreateDoctorCommand struct {
	Name          string
	Email         string
	Phone         string
	Specialty     string
	LicenseNumber string
}

// UpdateDoctorCommand holds data to update a doctor.
type UpdateDoctorCommand struct {
	ID        string
	Name      string
	Email     string
	Phone     string
	Specialty string
}

// DeleteDoctorCommand removes a doctor.
type DeleteDoctorCommand struct {
	ID string
}

// DoctorCommandHandler handles doctor write operations.
//
// Doctors are local-only semantic entities. There is intentionally no
// AuthenticationProvider dependency: nothing about the doctor lifecycle
// requires talking to Firebase Auth.
type DoctorCommandHandler struct {
	repo doctor.Repository
}

// NewDoctorCommandHandler creates a DoctorCommandHandler.
func NewDoctorCommandHandler(repo doctor.Repository) *DoctorCommandHandler {
	return &DoctorCommandHandler{repo: repo}
}

// Create creates a new doctor.
//
// Doctor is a semantic entity — it identifies who issued a prescription.
// It is not an authenticatable principal and never gets a Firebase Auth
// account. firebase_id is always empty.
func (h *DoctorCommandHandler) Create(ctx context.Context, cmd CreateDoctorCommand) (*doctor.Doctor, error) {
	if strings.TrimSpace(cmd.Name) == "" ||
		strings.TrimSpace(cmd.Email) == "" ||
		strings.TrimSpace(cmd.Phone) == "" ||
		strings.TrimSpace(cmd.LicenseNumber) == "" {
		return nil, application.ErrInvalidInput
	}
	email := strings.TrimSpace(cmd.Email)
	license := strings.TrimSpace(cmd.LicenseNumber)

	if _, err := h.repo.FindByEmail(ctx, email); err == nil {
		return nil, application.ErrEmailAlreadyInUse
	} else if !errors.Is(err, doctor.ErrDoctorNotFound) {
		return nil, err
	}
	if _, err := h.repo.FindByLicenseNumber(ctx, license); err == nil {
		return nil, application.ErrLicenseAlreadyInUse
	} else if !errors.Is(err, doctor.ErrDoctorNotFound) {
		return nil, err
	}

	newDoctor, err := doctor.NewDoctor(
		strings.TrimSpace(cmd.Name),
		email,
		strings.TrimSpace(cmd.Phone),
		strings.TrimSpace(cmd.Specialty),
		license,
	)
	if err != nil {
		return nil, err
	}

	if err := h.repo.Save(ctx, newDoctor); err != nil {
		return nil, err
	}

	return newDoctor, nil
}

// Update updates an existing doctor.
func (h *DoctorCommandHandler) Update(ctx context.Context, cmd UpdateDoctorCommand) (*doctor.Doctor, error) {
	if cmd.ID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		if errors.Is(err, doctor.ErrDoctorNotFound) {
			return nil, application.ErrDoctorNotFound
		}
		return nil, err
	}

	if err := entity.Update(cmd.Name, cmd.Email, cmd.Phone, cmd.Specialty); err != nil {
		return nil, err
	}

	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// Delete removes a doctor.
func (h *DoctorCommandHandler) Delete(ctx context.Context, cmd DeleteDoctorCommand) error {
	if cmd.ID == "" {
		return application.ErrInvalidInput
	}

	exists, err := h.repo.Exists(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if !exists {
		return application.ErrDoctorNotFound
	}

	return h.repo.Delete(ctx, cmd.ID)
}
