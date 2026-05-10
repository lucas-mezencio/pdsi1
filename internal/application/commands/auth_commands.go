package commands

import (
	"context"
	"errors"
	"strings"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// AuthenticationProvider handles firebase auth operations required by register/login.
type AuthenticationProvider interface {
	CreateUser(ctx context.Context, email, password string) (string, error)
	DeleteUser(ctx context.Context, firebaseID string) error
	SignIn(ctx context.Context, email, password string) (string, error)
}

// RegisterCommand holds data to create account in Firebase and local DB.
type RegisterCommand struct {
	Name          string
	Email         string
	Phone         string
	Password      string
	Role          string
	FirebaseToken string
}

// LoginCommand holds credentials for authentication.
type LoginCommand struct {
	Email    string
	Password string
}

// AuthCommandHandler handles register and login operations.
type AuthCommandHandler struct {
	repo         user.Repository
	authProvider AuthenticationProvider
}

// NewAuthCommandHandler creates an AuthCommandHandler.
func NewAuthCommandHandler(repo user.Repository, authProvider AuthenticationProvider) *AuthCommandHandler {
	return &AuthCommandHandler{
		repo:         repo,
		authProvider: authProvider,
	}
}

// Register creates user in Firebase Auth and links it to local user.firebase_id.
func (h *AuthCommandHandler) Register(ctx context.Context, cmd RegisterCommand) (*user.User, error) {
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

	_, err := h.repo.FindByEmail(ctx, email)
	if err == nil {
		return nil, application.ErrEmailAlreadyInUse
	}
	if !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	firebaseID, err := h.authProvider.CreateUser(ctx, email, cmd.Password)
	if err != nil {
		return nil, err
	}

	entity, err := user.NewUser(
		strings.TrimSpace(cmd.Name),
		email,
		strings.TrimSpace(cmd.Phone),
		cmd.FirebaseToken,
		user.Role(cmd.Role),
	)
	if err != nil {
		_ = h.authProvider.DeleteUser(ctx, firebaseID)
		return nil, err
	}

	entity.LinkFirebaseAccount(firebaseID)

	if err := h.repo.Save(ctx, entity); err != nil {
		_ = h.authProvider.DeleteUser(ctx, firebaseID)
		return nil, err
	}

	return entity, nil
}

// Login validates credentials at Firebase and returns linked local user.
func (h *AuthCommandHandler) Login(ctx context.Context, cmd LoginCommand) (*user.User, error) {
	if h.authProvider == nil {
		return nil, application.ErrAuthNotConfigured
	}
	email := strings.TrimSpace(cmd.Email)
	password := strings.TrimSpace(cmd.Password)
	if email == "" || password == "" {
		return nil, application.ErrInvalidInput
	}

	firebaseID, err := h.authProvider.SignIn(ctx, email, password)
	if err != nil {
		return nil, err
	}

	entity, err := h.repo.FindByFirebaseID(ctx, firebaseID)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	// Backfill legacy users that were created before firebase_id existed.
	entity, err = h.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	entity.LinkFirebaseAccount(firebaseID)
	if err := h.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}
