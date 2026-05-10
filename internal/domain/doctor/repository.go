package doctor

import "context"

// Repository defines the interface for doctor data persistence
type Repository interface {
	// Save creates or updates a doctor
	Save(ctx context.Context, doctor *Doctor) error

	// FindByID retrieves a doctor by ID
	FindByID(ctx context.Context, id string) (*Doctor, error)

	// FindByEmail retrieves a doctor by email
	FindByEmail(ctx context.Context, email string) (*Doctor, error)

	// FindByFirebaseID retrieves a doctor by firebase auth UID
	FindByFirebaseID(ctx context.Context, firebaseID string) (*Doctor, error)

	// FindByLicenseNumber retrieves a doctor by license number
	FindByLicenseNumber(ctx context.Context, licenseNumber string) (*Doctor, error)

	// FindAll retrieves all doctors
	FindAll(ctx context.Context) ([]*Doctor, error)

	// Delete removes a doctor by ID
	Delete(ctx context.Context, id string) error

	// Exists checks if a doctor exists by ID
	Exists(ctx context.Context, id string) (bool, error)
}
