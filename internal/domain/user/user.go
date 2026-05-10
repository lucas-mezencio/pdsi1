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
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Email                string    `json:"email"`
	Phone                string    `json:"phone"`
	FirebaseID           string    `json:"firebase_id,omitempty"`
	FirebaseToken        string    `json:"firebase_token"`
	NotificationsEnabled bool      `json:"notifications_enabled"`
	Role                 Role      `json:"role"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// NewUser creates a new User with generated ID and timestamps
func NewUser(name, email, phone, firebaseToken string, role Role) (*User, error) {
	if err := validateUser(name, email, phone); err != nil {
		return nil, err
	}

	if role != RoleElderly && role != RoleCaregiver {
		role = RoleElderly
	}

	now := time.Now()
	return &User{
		ID:                   uuid.New().String(),
		Name:                 name,
		Email:                email,
		Phone:                phone,
		FirebaseToken:        firebaseToken,
		NotificationsEnabled: true,
		Role:                 role,
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

// Update updates user information
func (u *User) Update(name, email, phone string) error {
	if err := validateUser(name, email, phone); err != nil {
		return err
	}

	u.Name = name
	u.Email = email
	u.Phone = phone
	u.UpdatedAt = time.Now()
	return nil
}

// UpdateFirebaseToken updates the user's Firebase token for notifications
func (u *User) UpdateFirebaseToken(token string) {
	u.FirebaseToken = token
	u.UpdatedAt = time.Now()
}

// LinkFirebaseAccount links the local user to a Firebase Auth UID.
func (u *User) LinkFirebaseAccount(firebaseID string) {
	u.FirebaseID = firebaseID
	u.UpdatedAt = time.Now()
}

// EnableNotifications enables notifications for the user
func (u *User) EnableNotifications() {
	u.NotificationsEnabled = true
	u.UpdatedAt = time.Now()
}

// DisableNotifications disables notifications for the user
func (u *User) DisableNotifications() {
	u.NotificationsEnabled = false
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

// Domain errors
var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidName  = errors.New("invalid user name")
	ErrInvalidEmail = errors.New("invalid user email")
	ErrInvalidPhone = errors.New("invalid user phone")
	ErrInvalidRole  = errors.New("invalid user role")
)
