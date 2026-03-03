package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// User represents an elderly user or caretaker who receives medication notifications
type User struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Email                string    `json:"email"`
	Phone                string    `json:"phone"`
	FirebaseToken        string    `json:"firebase_token"`
	NotificationsEnabled bool      `json:"notifications_enabled"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// NewUser creates a new User with generated ID and timestamps
func NewUser(name, email, phone, firebaseToken string) (*User, error) {
	if err := validateUser(name, email, phone); err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:                   uuid.New().String(),
		Name:                 name,
		Email:                email,
		Phone:                phone,
		FirebaseToken:        firebaseToken,
		NotificationsEnabled: true,
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
)
