package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Role represents the role of a user in the system.
type Role string

const (
	RoleElderly   Role = "ELDERLY"
	RoleCaregiver Role = "CAREGIVER"
)

// User represents an elderly user or caretaker who receives medication notifications
type User struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	CPF        string    `json:"cpf,omitempty"`
	FirebaseID string    `json:"firebase_id,omitempty"`
	Role       Role      `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewUser creates a new User with generated ID and timestamps.
// CPF is optional; when non-empty it must be a valid Brazilian CPF (full checksum
// verification, see ValidateCPF).
func NewUser(name, email, phone, cpf string, role Role) (*User, error) {
	normalizedCPF := ""
	if trimmed := trimSpace(cpf); trimmed != "" {
		validated, err := ValidateCPF(trimmed)
		if err != nil {
			return nil, ErrInvalidCPF
		}
		normalizedCPF = validated
	}

	if err := validateUser(name, email, phone); err != nil {
		return nil, err
	}

	if role != RoleElderly && role != RoleCaregiver {
		role = RoleElderly
	}

	now := time.Now()
	return &User{
		ID:         uuid.New().String(),
		Name:       name,
		Email:      email,
		Phone:      phone,
		CPF:        normalizedCPF,
		Role:       role,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Update updates user information, including CPF (optional, validated when non-empty).
func (u *User) Update(name, email, phone, cpf string) error {
	normalizedCPF := u.CPF
	if trimmed := trimSpace(cpf); trimmed != "" {
		validated, err := ValidateCPF(trimmed)
		if err != nil {
			return ErrInvalidCPF
		}
		normalizedCPF = validated
	}

	if err := validateUser(name, email, phone); err != nil {
		return err
	}

	u.Name = name
	u.Email = email
	u.Phone = phone
	u.CPF = normalizedCPF
	u.UpdatedAt = time.Now()
	return nil
}

// LinkFirebaseAccount links the local user to a Firebase Auth UID.
func (u *User) LinkFirebaseAccount(firebaseID string) {
	u.FirebaseID = firebaseID
	u.UpdatedAt = time.Now()
}

// IsElderly returns true if the user is an elderly user
func (u *User) IsElderly() bool {
	return u.Role == RoleElderly
}

// IsCaregiver returns true if the user is a caregiver
func (u *User) IsCaregiver() bool {
	return u.Role == RoleCaregiver
}

// validateUser validates user fields
func validateUser(name, email, phone string) error {
	if name == "" {
		return ErrInvalidName
	}
	if email == "" {
		return ErrInvalidEmail
	}
	if phone == "" {
		return ErrInvalidPhone
	}
	return nil
}

// trimSpace trims leading/trailing whitespace without importing strings at the
// package level (kept local to avoid a larger import surface).
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end {
		if s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r' {
			start++
			continue
		}
		break
	}
	for end > start {
		if s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r' {
			end--
			continue
		}
		break
	}
	return s[start:end]
}

// Domain errors
var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidName  = errors.New("invalid user name")
	ErrInvalidEmail = errors.New("invalid user email")
	ErrInvalidPhone = errors.New("invalid user phone")
	ErrInvalidRole  = errors.New("invalid user role")
)