package queries

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// ListDoseScheduleQuery retrieves the full planned dose schedule for a user.
type ListDoseScheduleQuery struct {
	UserID   string
	CallerID string
}

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
	doseRepo         prescription.DoseRecordRepository
	userRepo         user.Repository
	prescriptionRepo prescription.Repository
}

// NewDoseRecordQueryHandler creates a DoseRecordQueryHandler.
func NewDoseRecordQueryHandler(
	doseRepo prescription.DoseRecordRepository,
	userRepo user.Repository,
	prescriptionRepo prescription.Repository,
) *DoseRecordQueryHandler {
	return &DoseRecordQueryHandler{
		doseRepo:         doseRepo,
		userRepo:         userRepo,
		prescriptionRepo: prescriptionRepo,
	}
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

// ListScheduleForUser returns the user's full medication schedule:
// every planned dose across active prescriptions, overlaid with existing
// dose_records (TAKEN / MISSED win). Records with no matching
// prescription slot are preserved as orphans so callers can still see
// history that outlived a deactivated prescription.
func (h *DoseRecordQueryHandler) ListScheduleForUser(ctx context.Context, q ListDoseScheduleQuery) ([]*prescription.ScheduledDose, error) {
	if q.UserID == "" {
		return nil, application.ErrInvalidInput
	}
	if err := h.checkAccess(ctx, q.CallerID, q.UserID); err != nil {
		return nil, err
	}

	prescriptions, err := h.prescriptionRepo.FindActiveByUserID(ctx, q.UserID)
	if err != nil {
		return nil, err
	}

	records, err := h.doseRepo.FindByUserID(ctx, q.UserID)
	if err != nil {
		return nil, err
	}

	overlay := make(map[string]*prescription.DoseRecord, len(records))
	for _, r := range records {
		overlay[overlayKey(r.PrescriptionID, r.ScheduledAt, r.MedicamentName)] = r
	}

	out := make([]*prescription.ScheduledDose, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, p := range prescriptions {
		for _, slot := range p.ExpandSchedule(prescription.BrazilLocation) {
			k := overlayKey(slot.PrescriptionID, slot.ScheduledAt, slot.MedicamentName)
			seen[k] = struct{}{}
			if rec, ok := overlay[k]; ok {
				id := rec.ID
				confirmed := rec.ConfirmedAt
				out = append(out, &prescription.ScheduledDose{
					PrescriptionID: rec.PrescriptionID,
					MedicamentName: rec.MedicamentName,
					Dosage:         rec.Dosage,
					ScheduledAt:    rec.ScheduledAt,
					Status:         rec.Status,
					DoseRecordID:   &id,
					ConfirmedAt:    confirmed,
				})
				continue
			}
			slotCopy := slot
			out = append(out, &slotCopy)
		}
	}

	for _, r := range records {
		k := overlayKey(r.PrescriptionID, r.ScheduledAt, r.MedicamentName)
		if _, ok := seen[k]; ok {
			continue
		}
		id := r.ID
		confirmed := r.ConfirmedAt
		out = append(out, &prescription.ScheduledDose{
			PrescriptionID: r.PrescriptionID,
			MedicamentName: r.MedicamentName,
			Dosage:         r.Dosage,
			ScheduledAt:    r.ScheduledAt,
			Status:         r.Status,
			DoseRecordID:   &id,
			ConfirmedAt:    confirmed,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ScheduledAt.Before(out[j].ScheduledAt)
	})

	return out, nil
}

// overlayKey builds the canonical key used to match a planned slot to
// its materialized dose_record. Times are truncated to the minute in
// UTC so that sub-minute drift between reconstruction (nanos=0) and
// scheduler-created records (also nanos=0 in production, but tests
// may use time.Now()-derived values) does not break the match.
func overlayKey(prescriptionID string, scheduledAt time.Time, medicamentName string) string {
	truncated := scheduledAt.UTC().Truncate(time.Minute)
	return prescriptionID + "|" + truncated.Format(time.RFC3339) + "|" + medicamentName
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
