package commands

import (
	"context"
	"errors"
	"strings"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// CreateInvitationCommand creates a caregiver invitation.
type CreateInvitationCommand struct {
	// ElderlyID is the user being cared for.
	ElderlyID string
	// CaregiverID is the user who will care.
	CaregiverID string
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

// Create creates a new caregiver invitation.
func (h *InvitationCommandHandler) Create(ctx context.Context, cmd CreateInvitationCommand) (*user.CaregiverInvitation, error) {
	if cmd.ElderlyID == "" || cmd.CaregiverID == "" {
		return nil, application.ErrInvalidInput
	}

	// Ensure both users exist.
	elderly, err := h.userRepo.FindByID(ctx, cmd.ElderlyID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	caregiver, err := h.userRepo.FindByID(ctx, cmd.CaregiverID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	if !elderly.IsElderly() {
		return nil, application.ErrWrongRole
	}
	if !caregiver.IsCaregiver() {
		return nil, application.ErrWrongRole
	}

	// Check if already linked.
	linked, err := h.userRepo.IsLinked(ctx, cmd.CaregiverID, cmd.ElderlyID)
	if err != nil {
		return nil, err
	}
	if linked {
		return nil, application.ErrAlreadyLinked
	}

	inv, err := user.NewCaregiverInvitation(cmd.CaregiverID, cmd.ElderlyID)
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
