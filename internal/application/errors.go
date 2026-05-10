package application

import "errors"

// Application-level errors
var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrUserNotFound         = errors.New("user not found")
	ErrDoctorNotFound       = errors.New("doctor not found")
	ErrPrescriptionNotFound = errors.New("prescription not found")
	ErrDoseRecordNotFound   = errors.New("dose record not found")
	ErrInvitationNotFound   = errors.New("invitation not found")
	ErrInvitationNotPending = errors.New("invitation is not pending")
	ErrAlreadyLinked        = errors.New("users are already linked")
	ErrWrongRole            = errors.New("user does not have the required role")
	ErrForbidden            = errors.New("access denied")
	ErrAuthenticationFailed = errors.New("invalid email or password")
	ErrEmailAlreadyInUse    = errors.New("email already in use")
	ErrLicenseAlreadyInUse  = errors.New("license number already in use")
	ErrAuthNotConfigured    = errors.New("firebase authentication is not configured")
)
