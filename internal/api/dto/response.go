package dto

import (
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// UserResponse is the JSON-safe representation of a user returned by the API.
// It deliberately omits FirebaseToken, which is server-internal and must never
// leak through HTTP responses.
type UserResponse struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Email                string    `json:"email"`
	Phone                string    `json:"phone"`
	CPF                  string    `json:"cpf,omitempty"`
	FirebaseID           string    `json:"firebase_id,omitempty"`
	NotificationsEnabled bool      `json:"notifications_enabled"`
	Role                 user.Role `json:"role"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// UserResponseFromDomain converts a domain user into the API response shape.
func UserResponseFromDomain(u *user.User) UserResponse {
	if u == nil {
		return UserResponse{}
	}
	return UserResponse{
		ID:                   u.ID,
		Name:                 u.Name,
		Email:                u.Email,
		Phone:                u.Phone,
		CPF:                  u.CPF,
		FirebaseID:           u.FirebaseID,
		NotificationsEnabled: u.NotificationsEnabled,
		Role:                 u.Role,
		CreatedAt:            u.CreatedAt,
		UpdatedAt:            u.UpdatedAt,
	}
}

// UserResponsesFromDomain converts a slice of domain users into the API
// response shape.
func UserResponsesFromDomain(users []*user.User) []UserResponse {
	out := make([]UserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, UserResponseFromDomain(u))
	}
	return out
}

// DoctorResponse is the JSON-safe representation of a doctor returned by the
// API. firebase_id is included only when populated so the field stays opaque
// to clients that shouldn't depend on it.
type DoctorResponse struct {
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

// DoctorResponseFromDomain converts a domain doctor into the API response shape.
func DoctorResponseFromDomain(d *doctor.Doctor) DoctorResponse {
	if d == nil {
		return DoctorResponse{}
	}
	return DoctorResponse{
		ID:            d.ID,
		Name:          d.Name,
		Email:         d.Email,
		Phone:         d.Phone,
		FirebaseID:    d.FirebaseID,
		Specialty:     d.Specialty,
		LicenseNumber: d.LicenseNumber,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// DoctorResponsesFromDomain converts a slice of domain doctors into the API
// response shape.
func DoctorResponsesFromDomain(doctors []*doctor.Doctor) []DoctorResponse {
	out := make([]DoctorResponse, 0, len(doctors))
	for _, d := range doctors {
		out = append(out, DoctorResponseFromDomain(d))
	}
	return out
}