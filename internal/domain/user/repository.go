package user

import "context"

// Repository defines the interface for user data persistence
type Repository interface {
	// Save creates or updates a user
	Save(ctx context.Context, user *User) error

	// FindByID retrieves a user by ID
	FindByID(ctx context.Context, id string) (*User, error)

	// FindByEmail retrieves a user by email
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindAll retrieves all users
	FindAll(ctx context.Context) ([]*User, error)

	// Delete removes a user by ID
	Delete(ctx context.Context, id string) error

	// Exists checks if a user exists by ID
	Exists(ctx context.Context, id string) (bool, error)
}
