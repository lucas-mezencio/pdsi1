package queries

import (
	"context"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

// GetDoctorByIDQuery retrieves a doctor by ID.
type GetDoctorByIDQuery struct {
	ID string
}

// GetDoctorByEmailQuery retrieves a doctor by email.
type GetDoctorByEmailQuery struct {
	Email string
}

// GetDoctorByLicenseQuery retrieves a doctor by license number.
type GetDoctorByLicenseQuery struct {
	LicenseNumber string
}

// ListDoctorsQuery retrieves all doctors.
type ListDoctorsQuery struct{}

// DoctorQueryHandler handles doctor read operations.
type DoctorQueryHandler struct {
	repo doctor.Repository
}

// NewDoctorQueryHandler creates a DoctorQueryHandler.
func NewDoctorQueryHandler(repo doctor.Repository) *DoctorQueryHandler {
	return &DoctorQueryHandler{repo: repo}
}

// GetByID retrieves a doctor by ID.
func (h *DoctorQueryHandler) GetByID(ctx context.Context, query GetDoctorByIDQuery) (*doctor.Doctor, error) {
	if query.ID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, query.ID)
	if err != nil {
		if errors.Is(err, doctor.ErrDoctorNotFound) {
			return nil, application.ErrDoctorNotFound
		}
		return nil, err
	}

	return entity, nil
}

// GetByEmail retrieves a doctor by email.
func (h *DoctorQueryHandler) GetByEmail(ctx context.Context, query GetDoctorByEmailQuery) (*doctor.Doctor, error) {
	if query.Email == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByEmail(ctx, query.Email)
	if err != nil {
		if errors.Is(err, doctor.ErrDoctorNotFound) {
			return nil, application.ErrDoctorNotFound
		}
		return nil, err
	}

	return entity, nil
}

// GetByLicense retrieves a doctor by license number.
func (h *DoctorQueryHandler) GetByLicense(ctx context.Context, query GetDoctorByLicenseQuery) (*doctor.Doctor, error) {
	if query.LicenseNumber == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByLicenseNumber(ctx, query.LicenseNumber)
	if err != nil {
		if errors.Is(err, doctor.ErrDoctorNotFound) {
			return nil, application.ErrDoctorNotFound
		}
		return nil, err
	}

	return entity, nil
}

// List retrieves all doctors.
func (h *DoctorQueryHandler) List(ctx context.Context, _ ListDoctorsQuery) ([]*doctor.Doctor, error) {
	return h.repo.FindAll(ctx)
}
