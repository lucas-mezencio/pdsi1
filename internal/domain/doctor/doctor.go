package doctor

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Doctor represents a medical professional who prescribes medications
type Doctor struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	FirebaseID    string    `json:"firebase_id,omitempty"`
	Specialty     string    `json:"specialty"`
	LicenseNumber string    `json:"license_number"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewDoctor creates a new Doctor with generated ID and timestamps
func NewDoctor(name, email, phone, specialty, licenseNumber string) (*Doctor, error) {
	if err := validateDoctor(name, email, phone, licenseNumber); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Doctor{
		ID:            uuid.New().String(),
		Name:          name,
		Email:         email,
		Phone:         phone,
		Specialty:     specialty,
		LicenseNumber: licenseNumber,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// Update updates doctor information
func (d *Doctor) Update(name, email, phone, specialty string) error {
	if err := validateDoctor(name, email, phone, d.LicenseNumber); err != nil {
		return err
	}

	d.Name = name
	d.Email = email
	d.Phone = phone
	d.Specialty = specialty
	d.UpdatedAt = time.Now()
	return nil
}

// LinkFirebaseAccount links the doctor to a Firebase Auth UID.
func (d *Doctor) LinkFirebaseAccount(firebaseID string) {
	d.FirebaseID = firebaseID
	d.UpdatedAt = time.Now()
}

// validateDoctor validates doctor fields
func validateDoctor(name, email, phone, licenseNumber string) error {
	if name == "" {
		return ErrInvalidName
	}
	if email == "" {
		return ErrInvalidEmail
	}
	if phone == "" {
		return ErrInvalidPhone
	}
	if licenseNumber == "" {
		return ErrInvalidLicenseNumber
	}
	return nil
}

// Domain errors
var (
	ErrDoctorNotFound       = errors.New("doctor not found")
	ErrInvalidName          = errors.New("invalid doctor name")
	ErrInvalidEmail         = errors.New("invalid doctor email")
	ErrInvalidPhone         = errors.New("invalid doctor phone")
	ErrInvalidLicenseNumber = errors.New("invalid license number")
)
