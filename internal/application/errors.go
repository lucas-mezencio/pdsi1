package application

import "errors"

// Application-level errors
var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrUserNotFound         = errors.New("user not found")
	ErrDoctorNotFound       = errors.New("doctor not found")
	ErrPrescriptionNotFound = errors.New("prescription not found")
)
