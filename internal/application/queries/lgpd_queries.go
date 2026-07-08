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
// FirebaseToken is intentionally NOT a field of the embedded *user.User
// shapes here: it is the FCM device token the scheduler worker uses to
// push medication reminders and is server-internal. The handler strips
// it on every *user.User before assembling the response (see stripFirebase
// Token and stripFirebaseTokenSlice).
type UserDataExport struct {
	User          *user.User                  `json:"user"`
	Prescriptions []*prescription.Prescription `json:"prescriptions"`
	DoseRecords   []*prescription.DoseRecord  `json:"dose_records"`
	Caregivers    []*user.User                `json:"caregivers"` // users linked to caller as caregiver (when caller is elderly)
	Charges       []*user.User                `json:"charges"`    // users caller is caregiver for
	Invitations   []*user.CaregiverInvitation `json:"invitations"`
}

// stripFirebaseToken returns a shallow copy of u with FirebaseToken cleared.
// firebase_token is the FCM device token used by the scheduler worker for
// push delivery (see internal/infrastructure/scheduler/worker.go and
// internal/infrastructure/notification/firebase_sender.go). It must never
// leave the server; the LGPD self-export endpoint gives the caller a copy
// of their own data, so we strip the token even though the caller owns the
// underlying Firebase user.
func stripFirebaseToken(u *user.User) *user.User {
	if u == nil {
		return nil
	}
	clone := *u
	clone.FirebaseToken = ""
	return &clone
}

func stripFirebaseTokenSlice(users []*user.User) []*user.User {
	if users == nil {
		return nil
	}
	out := make([]*user.User, 0, len(users))
	for _, u := range users {
		out = append(out, stripFirebaseToken(u))
	}
	return out
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
		User:          stripFirebaseToken(entity),
		Prescriptions: prescriptions,
		DoseRecords:   doseRecords,
		Caregivers:    stripFirebaseTokenSlice(caregivers),
		Charges:       stripFirebaseTokenSlice(charges),
		Invitations:   invitations,
	}, nil
}