package prescription

import "context"

// Repository defines the interface for prescription data persistence
type Repository interface {
	// Save creates or updates a prescription
	Save(ctx context.Context, prescription *Prescription) error

	// FindAll retrieves all prescriptions
	FindAll(ctx context.Context) ([]*Prescription, error)

	// FindByID retrieves a prescription by ID
	FindByID(ctx context.Context, id string) (*Prescription, error)

	// FindByUserID retrieves all prescriptions for a user
	FindByUserID(ctx context.Context, userID string) ([]*Prescription, error)

	// FindByMedicID retrieves all prescriptions created by a doctor
	FindByMedicID(ctx context.Context, medicID string) ([]*Prescription, error)

	// FindActive retrieves all active prescriptions
	FindActive(ctx context.Context) ([]*Prescription, error)

	// FindActiveByUserID retrieves all active prescriptions for a user
	FindActiveByUserID(ctx context.Context, userID string) ([]*Prescription, error)

	// Delete removes a prescription by ID
	Delete(ctx context.Context, id string) error

	// Exists checks if a prescription exists by ID
	Exists(ctx context.Context, id string) (bool, error)
}
