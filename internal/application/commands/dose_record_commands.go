package commands

import (
	"context"
	"errors"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// ConfirmDoseCommand marks a dose record as TAKEN.
type ConfirmDoseCommand struct {
	// DoseRecordID is the ID of the dose record to confirm.
	DoseRecordID string
	// CallerID is the ID of the user making the request (for RBAC).
	CallerID string
}

// MissDoseCommand marks a dose record as MISSED.
type MissDoseCommand struct {
	DoseRecordID string
	CallerID     string
}

// DoseRecordCommandHandler handles dose record write operations.
type DoseRecordCommandHandler struct {
	doseRepo prescription.DoseRecordRepository
	userRepo  user.Repository
}

// NewDoseRecordCommandHandler creates a DoseRecordCommandHandler.
func NewDoseRecordCommandHandler(doseRepo prescription.DoseRecordRepository, userRepo user.Repository) *DoseRecordCommandHandler {
	return &DoseRecordCommandHandler{doseRepo: doseRepo, userRepo: userRepo}
}

// Confirm marks a dose as taken. Only the dose owner or a linked caregiver can confirm.
func (h *DoseRecordCommandHandler) Confirm(ctx context.Context, cmd ConfirmDoseCommand) (*prescription.DoseRecord, error) {
	if cmd.DoseRecordID == "" {
		return nil, application.ErrInvalidInput
	}

	record, err := h.doseRepo.FindByID(ctx, cmd.DoseRecordID)
	if err != nil {
		if errors.Is(err, prescription.ErrDoseRecordNotFound) {
			return nil, application.ErrDoseRecordNotFound
		}
		return nil, err
	}

	if err := h.checkAccess(ctx, cmd.CallerID, record.UserID); err != nil {
		return nil, err
	}

	record.MarkTaken(time.Now())
	if err := h.doseRepo.Save(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

// Miss marks a dose as missed. Only the dose owner or a linked caregiver can mark.
func (h *DoseRecordCommandHandler) Miss(ctx context.Context, cmd MissDoseCommand) (*prescription.DoseRecord, error) {
	if cmd.DoseRecordID == "" {
		return nil, application.ErrInvalidInput
	}

	record, err := h.doseRepo.FindByID(ctx, cmd.DoseRecordID)
	if err != nil {
		if errors.Is(err, prescription.ErrDoseRecordNotFound) {
			return nil, application.ErrDoseRecordNotFound
		}
		return nil, err
	}

	if err := h.checkAccess(ctx, cmd.CallerID, record.UserID); err != nil {
		return nil, err
	}

	record.MarkMissed()
	if err := h.doseRepo.Save(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

// checkAccess verifies that callerID is either the owner or a linked caregiver.
// If callerID is empty, access is allowed (unauthenticated mode).
func (h *DoseRecordCommandHandler) checkAccess(ctx context.Context, callerID, ownerID string) error {
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
