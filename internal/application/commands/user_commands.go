package commands

import (
	"context"
	"errors"
	"strings"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// CreateUserCommand holds data to create a new user.
//
// Firebase account creation is server-side: the caller supplies a Password,
// the handler asks AuthenticationProvider to create the Firebase user, then
// stores the returned UID locally. firebase_id and firebase_token are NOT
// accepted as input — they are server-internal concerns.
type CreateUserCommand struct {
	Name     string
	Email    string
	Phone    string
	CPF      string
	Password string
	Role     string // "ELDERLY" or "CAREGIVER" (defaults to "ELDERLY")
}

// UpdateUserCommand holds data to update a user.
type UpdateUserCommand struct {
	ID    string
	Name  string
	Email string
	Phone string
	CPF   string
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
	repo         user.Repository
	authProvider AuthenticationProvider
}

// NewUserCommandHandler creates a UserCommandHandler.
func NewUserCommandHandler(repo user.Repository, authProvider AuthenticationProvider) *UserCommandHandler {
	return &UserCommandHandler{repo: repo, authProvider: authProvider}
}

// Create creates a new user.
//
// The flow auto-provisions the corresponding Firebase Auth user via the
// configured AuthenticationProvider. firebase_id is server-generated from
// Firebase; firebase_token is empty at creation time (the mobile app refreshes
// it after first sign-in via UpdateFirebaseToken).
func (h *UserCommandHandler) Create(ctx context.Context, cmd CreateUserCommand) (*user.User, error) {
	if h.authProvider == nil {
		return nil, application.ErrAuthNotConfigured
	}
	if strings.TrimSpace(cmd.Name) == "" ||
		strings.TrimSpace(cmd.Email) == "" ||
		strings.TrimSpace(cmd.Phone) == "" ||
		strings.TrimSpace(cmd.Password) == "" {
		return nil, application.ErrInvalidInput
	}
	email := strings.TrimSpace(cmd.Email)

	if _, err := h.repo.FindByEmail(ctx, email); err == nil {
		return nil, application.ErrEmailAlreadyInUse
	} else if !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	firebaseID, err := h.authProvider.CreateUser(ctx, email, cmd.Password)
	if err != nil {
		return nil, err
	}

	role := user.Role(cmd.Role)
	newUser, err := user.NewUser(
		strings.TrimSpace(cmd.Name),
		email,
		strings.TrimSpace(cmd.Phone),
		strings.TrimSpace(cmd.CPF),
		"", // firebaseToken: not set at creation; mobile updates later
		role,
	)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCPF) {
			_ = h.authProvider.DeleteUser(ctx, firebaseID)
			return nil, application.ErrInvalidInput
		}
		_ = h.authProvider.DeleteUser(ctx, firebaseID)
		return nil, err
	}
	newUser.LinkFirebaseAccount(firebaseID)

	if err := h.repo.Save(ctx, newUser); err != nil {
		_ = h.authProvider.DeleteUser(ctx, firebaseID)
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

	if err := entity.Update(cmd.Name, cmd.Email, cmd.Phone, cmd.CPF); err != nil {
		if errors.Is(err, user.ErrInvalidCPF) {
			return nil, application.ErrInvalidInput
		}
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
