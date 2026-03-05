package commands

import (
	"context"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// CreateUserCommand holds data to create a new user.
type CreateUserCommand struct {
	Name          string
	Email         string
	Phone         string
	FirebaseToken string
	Role          string // "ELDERLY" or "CAREGIVER" (defaults to "ELDERLY")
}

// UpdateUserCommand holds data to update a user.
type UpdateUserCommand struct {
	ID    string
	Name  string
	Email string
	Phone string
}

// UpdateUserFirebaseTokenCommand updates a user's firebase token.
type UpdateUserFirebaseTokenCommand struct {
	ID            string
	FirebaseToken string
}

// ToggleUserNotificationsCommand enables or disables notifications for a user.
type ToggleUserNotificationsCommand struct {
	ID      string
	Enabled bool
}

// DeleteUserCommand removes a user.
type DeleteUserCommand struct {
	ID string
}

// UserCommandHandler handles user write operations.
type UserCommandHandler struct {
	repo user.Repository
}

// NewUserCommandHandler creates a UserCommandHandler.
func NewUserCommandHandler(repo user.Repository) *UserCommandHandler {
	return &UserCommandHandler{repo: repo}
}

// Create creates a new user.
func (h *UserCommandHandler) Create(ctx context.Context, cmd CreateUserCommand) (*user.User, error) {
	role := user.Role(cmd.Role)
	newUser, err := user.NewUser(cmd.Name, cmd.Email, cmd.Phone, cmd.FirebaseToken, role)
	if err != nil {
		return nil, err
	}

	if err := h.repo.Save(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

// Update updates an existing user.
func (h *UserCommandHandler) Update(ctx context.Context, cmd UpdateUserCommand) (*user.User, error) {
	if cmd.ID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	if err := entity.Update(cmd.Name, cmd.Email, cmd.Phone); err != nil {
		return nil, err
	}

	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// UpdateFirebaseToken updates the firebase token for a user.
func (h *UserCommandHandler) UpdateFirebaseToken(ctx context.Context, cmd UpdateUserFirebaseTokenCommand) (*user.User, error) {
	if cmd.ID == "" || cmd.FirebaseToken == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	entity.UpdateFirebaseToken(cmd.FirebaseToken)
	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// ToggleNotifications enables or disables notifications.
func (h *UserCommandHandler) ToggleNotifications(ctx context.Context, cmd ToggleUserNotificationsCommand) (*user.User, error) {
	if cmd.ID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	if cmd.Enabled {
		entity.EnableNotifications()
	} else {
		entity.DisableNotifications()
	}

	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// Delete removes a user.
func (h *UserCommandHandler) Delete(ctx context.Context, cmd DeleteUserCommand) error {
	if cmd.ID == "" {
		return application.ErrInvalidInput
	}

	exists, err := h.repo.Exists(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if !exists {
		return application.ErrUserNotFound
	}

	return h.repo.Delete(ctx, cmd.ID)
}
