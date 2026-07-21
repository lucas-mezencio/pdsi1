package queries

import (
	"context"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// UserDataExport is the full export of a user's data, assembled for LGPD/GDPR
// data-subject access requests. It contains every category of personal data
// CareConnect holds about the user, including sensitive health data
// (prescriptions and dose records).
//
// Push-delivery tokens live in user_device_tokens and are never exposed in
// this export.
type UserDataExport struct {
	User          *user.User                  `json:"user"`
	Prescriptions []*prescription.Prescription `json:"prescriptions"`
	DoseRecords   []*prescription.DoseRecord  `json:"dose_records"`
	Caregivers    []*user.User                `json:"caregivers"` // users linked to caller as caregiver (when caller is elderly)
	Charges       []*user.User                `json:"charges"`    // users caller is caregiver for
	Invitations   []*user.CaregiverInvitation `json:"invitations"`
}

// LGPDQueryHandler assembles the full user data export (LGPD right of access).
type LGPDQueryHandler struct {
	userRepo         user.Repository
	prescriptionRepo prescription.Repository
	doseRepo         prescription.DoseRecordRepository
	invitationRepo   user.InvitationRepository
}

// NewLGPDQueryHandler creates a new LGPDQueryHandler.
func NewLGPDQueryHandler(
	userRepo user.Repository,
	prescriptionRepo prescription.Repository,
	doseRepo prescription.DoseRecordRepository,
	invitationRepo user.InvitationRepository,
) *LGPDQueryHandler {
	return &LGPDQueryHandler{
		userRepo:         userRepo,
		prescriptionRepo: prescriptionRepo,
		doseRepo:         doseRepo,
		invitationRepo:   invitationRepo,
	}
}

// ExportForUser assembles every piece of personal data CareConnect stores
// about the given user. The userID MUST be the authenticated caller's own ID
// (the HTTP handler enforces this).
func (h *LGPDQueryHandler) ExportForUser(ctx context.Context, userID string) (*UserDataExport, error) {
	if userID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	prescriptions, err := h.prescriptionRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	doseRecords, err := h.doseRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	caregivers, err := h.userRepo.FindCaregivers(ctx, userID)
	if err != nil {
		return nil, err
	}

	charges, err := h.userRepo.FindCharges(ctx, userID)
	if err != nil {
		return nil, err
	}

	// An invitation can be associated with the user either as elderly or caregiver.
	byElderly, err := h.invitationRepo.FindByElderlyID(ctx, userID)
	if err != nil {
		return nil, err
	}
	byCaregiver, err := h.invitationRepo.FindByCaregiverID(ctx, userID)
	if err != nil {
		return nil, err
	}
	invitations := append(byElderly, byCaregiver...)

	return &UserDataExport{
		User:          entity,
		Prescriptions: prescriptions,
		DoseRecords:   doseRecords,
		Caregivers:    caregivers,
		Charges:       charges,
		Invitations:   invitations,
	}, nil
}