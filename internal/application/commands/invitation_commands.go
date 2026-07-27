package commands

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// CreateInvitationCommand creates a caregiver invitation by looking up the
// caregiver via email rather than expecting the caller to know the user's
// UUID. ElderlyID is the local UUID of the patient inviting the caregiver,
// derived from the authenticated caller's context at the API layer.
type CreateInvitationCommand struct {
	// ElderlyID is the local UUID of the patient doing the inviting.
	ElderlyID string
	// CaregiverEmail is the email of the caregiver being invited.
	CaregiverEmail string
}

// AcceptInvitationCommand accepts a pending invitation by token.
type AcceptInvitationCommand struct {
	Token string
}

// RejectInvitationCommand rejects a pending invitation by token.
type RejectInvitationCommand struct {
	Token string
}

// UnlinkUsersCommand removes the link between a caregiver and an elderly user.
type UnlinkUsersCommand struct {
	CaregiverID string
	ElderlyID   string
}

// InvitationCommandHandler handles invitation write operations.
type InvitationCommandHandler struct {
	userRepo   user.Repository
	inviteRepo user.InvitationRepository
}

// NewInvitationCommandHandler creates an InvitationCommandHandler.
func NewInvitationCommandHandler(userRepo user.Repository, inviteRepo user.InvitationRepository) *InvitationCommandHandler {
	return &InvitationCommandHandler{userRepo: userRepo, inviteRepo: inviteRepo}
}

// Create creates a new caregiver invitation by looking up the caregiver via
// email. Returns application errors mapped to HTTP status codes by the API
// layer:
//
//	ErrInvalidInput      → 400 (empty IDs, malformed email)
//	ErrUserNotFound      → 404 (caregiver email not in DB)
//	ErrWrongRole         → 400 (caregiver user has the wrong role, or self-invite)
//	ErrAlreadyLinked     → 409 (caregiver already linked to this elderly user)
func (h *InvitationCommandHandler) Create(ctx context.Context, cmd CreateInvitationCommand) (*user.CaregiverInvitation, error) {
	cmd.CaregiverEmail = strings.TrimSpace(cmd.CaregiverEmail)
	if cmd.ElderlyID == "" || cmd.CaregiverEmail == "" {
		return nil, application.ErrInvalidInput
	}
	if _, err := mail.ParseAddress(cmd.CaregiverEmail); err != nil {
		return nil, application.ErrInvalidInput
	}

	elderly, err := h.userRepo.FindByID(ctx, cmd.ElderlyID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}
	if !elderly.IsElderly() {
		return nil, application.ErrWrongRole
	}

	caregiver, err := h.userRepo.FindByEmail(ctx, cmd.CaregiverEmail)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}
	if !caregiver.IsCaregiver() {
		return nil, application.ErrWrongRole
	}
	if caregiver.ID == elderly.ID {
		return nil, application.ErrConflict
	}

	linked, err := h.userRepo.IsLinked(ctx, caregiver.ID, cmd.ElderlyID)
	if err != nil {
		return nil, err
	}
	if linked {
		return nil, application.ErrAlreadyLinked
	}

	inv, err := user.NewCaregiverInvitation(caregiver.ID, cmd.ElderlyID)
	if err != nil {
		return nil, err
	}

	if err := h.inviteRepo.Save(ctx, inv); err != nil {
		return nil, err
	}

	return inv, nil
}

// Accept accepts a pending invitation and creates the user link.
func (h *InvitationCommandHandler) Accept(ctx context.Context, cmd AcceptInvitationCommand) (*user.CaregiverInvitation, error) {
	token := strings.TrimSpace(cmd.Token)
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

	if err := inv.Accept(); err != nil {
		if errors.Is(err, user.ErrInvitationNotPending) {
			return nil, application.ErrInvitationNotPending
		}
		return nil, err
	}

	if err := h.inviteRepo.Save(ctx, inv); err != nil {
		return nil, err
	}

	if err := h.userRepo.LinkUsers(ctx, inv.CaregiverID, inv.ElderlyID); err != nil {
		return nil, err
	}

	return inv, nil
}

// Reject rejects a pending invitation.
func (h *InvitationCommandHandler) Reject(ctx context.Context, cmd RejectInvitationCommand) (*user.CaregiverInvitation, error) {
	token := strings.TrimSpace(cmd.Token)
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

	if err := inv.Reject(); err != nil {
		if errors.Is(err, user.ErrInvitationNotPending) {
			return nil, application.ErrInvitationNotPending
		}
		return nil, err
	}

	if err := h.inviteRepo.Save(ctx, inv); err != nil {
		return nil, err
	}

	return inv, nil
}

// Unlink removes the caregiver-elderly link.
func (h *InvitationCommandHandler) Unlink(ctx context.Context, cmd UnlinkUsersCommand) error {
	if cmd.CaregiverID == "" || cmd.ElderlyID == "" {
		return application.ErrInvalidInput
	}
	return h.userRepo.UnlinkUsers(ctx, cmd.CaregiverID, cmd.ElderlyID)
}
