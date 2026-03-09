package queries

import (
	"context"
	"errors"
	"strings"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// ListDoseRecordsQuery retrieves dose records for a user.
type ListDoseRecordsQuery struct {
	// UserID is the target elderly user.
	UserID string
	// CallerID is the requesting user (for RBAC). Empty = no check.
	CallerID string
}

// ListCaregiversQuery retrieves caregivers for an elderly user.
type ListCaregiversQuery struct {
	ElderlyID string
	CallerID  string
}

// ListChargesQuery retrieves elderly users linked to a caregiver.
type ListChargesQuery struct {
	CaregiverID string
	CallerID    string
}

// ListCaregiverInvitationsQuery retrieves invitations for a caregiver.
type ListCaregiverInvitationsQuery struct {
	CaregiverID string
	CallerID    string
}

// GetInvitationByTokenQuery retrieves an invitation by token.
type GetInvitationByTokenQuery struct {
	Token string
}

// DoseRecordQueryHandler handles dose record read operations.
type DoseRecordQueryHandler struct {
	doseRepo prescription.DoseRecordRepository
	userRepo user.Repository
}

// NewDoseRecordQueryHandler creates a DoseRecordQueryHandler.
func NewDoseRecordQueryHandler(doseRepo prescription.DoseRecordRepository, userRepo user.Repository) *DoseRecordQueryHandler {
	return &DoseRecordQueryHandler{doseRepo: doseRepo, userRepo: userRepo}
}

// ListByUser retrieves dose records for a user (with access control).
func (h *DoseRecordQueryHandler) ListByUser(ctx context.Context, query ListDoseRecordsQuery) ([]*prescription.DoseRecord, error) {
	if query.UserID == "" {
		return nil, application.ErrInvalidInput
	}

	if err := h.checkAccess(ctx, query.CallerID, query.UserID); err != nil {
		return nil, err
	}

	records, err := h.doseRepo.FindByUserID(ctx, query.UserID)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (h *DoseRecordQueryHandler) checkAccess(ctx context.Context, callerID, ownerID string) error {
	if callerID == "" || callerID == ownerID {
		return nil
	}

	linked, err := h.userRepo.IsLinked(ctx, callerID, ownerID)
	if err != nil {
		return err
	}
	if !linked {
		return application.ErrForbidden
	}
	return nil
}

// LinkedUserQueryHandler handles linked-user read operations.
type LinkedUserQueryHandler struct {
	userRepo   user.Repository
	inviteRepo user.InvitationRepository
}

// NewLinkedUserQueryHandler creates a LinkedUserQueryHandler.
func NewLinkedUserQueryHandler(userRepo user.Repository, inviteRepo user.InvitationRepository) *LinkedUserQueryHandler {
	return &LinkedUserQueryHandler{userRepo: userRepo, inviteRepo: inviteRepo}
}

// ListCaregivers returns the caregivers of an elderly user.
func (h *LinkedUserQueryHandler) ListCaregivers(ctx context.Context, query ListCaregiversQuery) ([]*user.User, error) {
	if query.ElderlyID == "" {
		return nil, application.ErrInvalidInput
	}

	if query.CallerID != "" && query.CallerID != query.ElderlyID {
		// A caregiver can list caregivers of their charges.
		linked, err := h.userRepo.IsLinked(ctx, query.CallerID, query.ElderlyID)
		if err != nil {
			return nil, err
		}
		if !linked {
			return nil, application.ErrForbidden
		}
	}

	return h.userRepo.FindCaregivers(ctx, query.ElderlyID)
}

// ListCharges returns the elderly users of a caregiver.
func (h *LinkedUserQueryHandler) ListCharges(ctx context.Context, query ListChargesQuery) ([]*user.User, error) {
	if query.CaregiverID == "" {
		return nil, application.ErrInvalidInput
	}

	if query.CallerID != "" && query.CallerID != query.CaregiverID {
		return nil, application.ErrForbidden
	}

	return h.userRepo.FindCharges(ctx, query.CaregiverID)
}

// ListCaregiverInvitations returns invitations addressed to a caregiver.
func (h *LinkedUserQueryHandler) ListCaregiverInvitations(ctx context.Context, query ListCaregiverInvitationsQuery) ([]*user.CaregiverInvitation, error) {
	if query.CaregiverID == "" {
		return nil, application.ErrInvalidInput
	}

	if query.CallerID != "" && query.CallerID != query.CaregiverID {
		return nil, application.ErrForbidden
	}

	return h.inviteRepo.FindByCaregiverID(ctx, query.CaregiverID)
}

// GetInvitationByToken retrieves an invitation by its token.
func (h *LinkedUserQueryHandler) GetInvitationByToken(ctx context.Context, query GetInvitationByTokenQuery) (*user.CaregiverInvitation, error) {
	token := strings.TrimSpace(query.Token)
	if token == "" {
		return nil, application.ErrInvalidInput
	}

	inv, err := h.inviteRepo.FindByToken(ctx, token)
	if err != nil {
		if errors.Is(err, user.ErrInvitationNotFound) {
			return nil, application.ErrInvitationNotFound
		}
		return nil, err
	}

	return inv, nil
}
