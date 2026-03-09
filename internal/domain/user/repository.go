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

	// FindByFirebaseID retrieves a user by firebase auth UID
	FindByFirebaseID(ctx context.Context, firebaseID string) (*User, error)

	// FindAll retrieves all users
	FindAll(ctx context.Context) ([]*User, error)

	// Delete removes a user by ID
	Delete(ctx context.Context, id string) error

	// Exists checks if a user exists by ID
	Exists(ctx context.Context, id string) (bool, error)

	// FindCaregivers retrieves all caregivers linked to an elderly user
	FindCaregivers(ctx context.Context, elderlyID string) ([]*User, error)

	// FindCharges retrieves all elderly users linked to a caregiver
	FindCharges(ctx context.Context, caregiverID string) ([]*User, error)

	// IsLinked checks if a caregiver is linked to an elderly user
	IsLinked(ctx context.Context, caregiverID, elderlyID string) (bool, error)

	// LinkUsers creates a caregiver-elderly link
	LinkUsers(ctx context.Context, caregiverID, elderlyID string) error

	// UnlinkUsers removes a caregiver-elderly link
	UnlinkUsers(ctx context.Context, caregiverID, elderlyID string) error
}
