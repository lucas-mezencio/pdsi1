package queries

import (
	"context"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// GetUserByIDQuery retrieves a user by ID.
type GetUserByIDQuery struct {
	ID string
}

// ListUsersQuery retrieves all users.
type ListUsersQuery struct{}

// GetUserByEmailQuery retrieves a user by email.
type GetUserByEmailQuery struct {
	Email string
}

// UserQueryHandler handles user read operations.
type UserQueryHandler struct {
	repo user.Repository
}

// NewUserQueryHandler creates a UserQueryHandler.
func NewUserQueryHandler(repo user.Repository) *UserQueryHandler {
	return &UserQueryHandler{repo: repo}
}

// GetByID retrieves a user by ID.
func (h *UserQueryHandler) GetByID(ctx context.Context, query GetUserByIDQuery) (*user.User, error) {
	if query.ID == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByID(ctx, query.ID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	return entity, nil
}

// GetByEmail retrieves a user by email.
func (h *UserQueryHandler) GetByEmail(ctx context.Context, query GetUserByEmailQuery) (*user.User, error) {
	if query.Email == "" {
		return nil, application.ErrInvalidInput
	}

	entity, err := h.repo.FindByEmail(ctx, query.Email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, err
	}

	return entity, nil
}

// List retrieves all users.
func (h *UserQueryHandler) List(ctx context.Context, _ ListUsersQuery) ([]*user.User, error) {
	return h.repo.FindAll(ctx)
}
