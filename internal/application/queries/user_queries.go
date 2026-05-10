package queries

import (
	"context"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// GetUserByIDQuery retrieves a user by ID.
type GetUserByIDQuery struct {
	ID string
}

// ListUsersQuery retrieves all users.
type ListUsersQuery struct{}

// GetUserByEmailQuery retrieves a user by email.
type GetUserByEmailQuery struct {
	Email string
}

// ExportUserDataQuery retrieves all data for a user for LGPD export.
type ExportUserDataQuery struct {
	ID string
}

// ExportUserDataResult holds all user data for LGPD compliance export.
type ExportUserDataResult struct {
	User           *user.User
	Prescriptions  []*prescription.Prescription
	DoseRecords    []*prescription.DoseRecord
	Caregivers     []*user.User
	Charges        []*user.User
	Invitations    []*user.Invitation
}

// UserQueryHandler handles user read operations.
type UserQueryHandler struct {
	repo      user.Repository
	prescRepo prescription.Repository
	doseRepo  prescription.DoseRecordRepository
}

// NewUserQueryHandler creates a UserQueryHandler.
func NewUserQueryHandler(repo user.Repository) *UserQueryHandler {
	return &UserQueryHandler{repo: repo}
}

// NewUserQueryHandlerWithExport creates a UserQueryHandler with export support.
func NewUserQueryHandlerWithExport(repo user.Repository, prescRepo prescription.Repository, doseRepo prescription.DoseRecordRepository) *UserQueryHandler {
	return &UserQueryHandler{repo: repo, prescRepo: prescRepo, doseRepo: doseRepo}
}

// GetByID retrieves a user by ID.
func (h *UserQueryHandler) GetByID(ctx context.Context, query GetUserByIDQuery) (*user.User, error) {
	if query.ID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, query.ID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	return entity, nil
}

// GetByEmail retrieves a user by email.
func (h *UserQueryHandler) GetByEmail(ctx context.Context, query GetUserByEmailQuery) (*user.User, error) {
	if query.Email == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByEmail(ctx, query.Email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	return entity, nil
}

// List retrieves all users.
func (h *UserQueryHandler) List(ctx context.Context, _ ListUsersQuery) ([]*user.User, error) {
	return h.repo.FindAll(ctx)
}

// ExportUserData exports all data for a user (LGPD data portability).
func (h *UserQueryHandler) ExportUserData(ctx context.Context, query ExportUserDataQuery) (*ExportUserDataResult, error) {
	if query.ID == "" {
		return nil, application.ErrInvalidInput
	}

	u, err := h.repo.FindByID(ctx, query.ID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	prescs, err := h.prescRepo.FindByUserID(ctx, query.ID)
	if err != nil {
		return nil, err
	}

	doses, err := h.doseRepo.FindByUserID(ctx, query.ID)
	if err != nil {
		return nil, err
	}

	caregivers, err := h.repo.FindCaregivers(ctx, query.ID)
	if err != nil {
		return nil, err
	}

	charges, err := h.repo.FindCharges(ctx, query.ID)
	if err != nil {
		return nil, err
	}

	return &ExportUserDataResult{
		User:          u,
		Prescriptions: prescs,
		DoseRecords:   doses,
		Caregivers:    caregivers,
		Charges:       charges,
	}, nil
}
