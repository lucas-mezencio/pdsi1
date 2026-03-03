package queries

import (
	"context"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

// GetPrescriptionByIDQuery retrieves a prescription by ID.
type GetPrescriptionByIDQuery struct {
	ID string
}

// ListPrescriptionsQuery retrieves prescriptions with optional filters.
type ListPrescriptionsQuery struct {
	UserID  string
	MedicID string
	Active  *bool
}

// PrescriptionQueryHandler handles prescription read operations.
type PrescriptionQueryHandler struct {
	repo prescription.Repository
}

// NewPrescriptionQueryHandler creates a PrescriptionQueryHandler.
func NewPrescriptionQueryHandler(repo prescription.Repository) *PrescriptionQueryHandler {
	return &PrescriptionQueryHandler{repo: repo}
}

// GetByID retrieves a prescription by ID.
func (h *PrescriptionQueryHandler) GetByID(ctx context.Context, query GetPrescriptionByIDQuery) (*prescription.Prescription, error) {
	if query.ID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, query.ID)
	if err != nil {
		if errors.Is(err, prescription.ErrPrescriptionNotFound) {
			return nil, application.ErrPrescriptionNotFound
		}
		return nil, err
	}

	return entity, nil
}

// List retrieves prescriptions based on filters.
func (h *PrescriptionQueryHandler) List(ctx context.Context, query ListPrescriptionsQuery) ([]*prescription.Prescription, error) {
	if query.UserID != "" {
		if query.Active != nil && *query.Active {
			return h.repo.FindActiveByUserID(ctx, query.UserID)
		}
		return h.repo.FindByUserID(ctx, query.UserID)
	}

	if query.MedicID != "" {
		return h.repo.FindByMedicID(ctx, query.MedicID)
	}

	if query.Active != nil && *query.Active {
		return h.repo.FindActive(ctx)
	}

	return h.repo.FindAll(ctx)
}
