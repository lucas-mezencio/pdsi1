package user

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// InvitationStatus represents the status of a caregiver invitation.
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "PENDING"
	InvitationStatusAccepted InvitationStatus = "ACCEPTED"
	InvitationStatusRejected InvitationStatus = "REJECTED"
)

// CaregiverInvitation represents an invitation for a caregiver to be linked to an elderly user.
type CaregiverInvitation struct {
	ID          string           `json:"id"`
	CaregiverID string           `json:"caregiver_id"`
	ElderlyID   string           `json:"elderly_id"`
	Token       string           `json:"token"`
	Status      InvitationStatus `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// NewCaregiverInvitation creates a new pending invitation.
func NewCaregiverInvitation(caregiverID, elderlyID string) (*CaregiverInvitation, error) {
	if caregiverID == "" {
		return nil, ErrInvalidCaregiverID
	}
	if elderlyID == "" {
		return nil, ErrInvalidElderlyID
	}

	now := time.Now()
	return &CaregiverInvitation{
		ID:          uuid.New().String(),
		CaregiverID: caregiverID,
		ElderlyID:   elderlyID,
		Token:       uuid.New().String(),
		Status:      InvitationStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Accept accepts the invitation.
func (i *CaregiverInvitation) Accept() error {
	if i.Status != InvitationStatusPending {
		return ErrInvitationNotPending
	}
	i.Status = InvitationStatusAccepted
	i.UpdatedAt = time.Now()
	return nil
}

// Reject rejects the invitation.
func (i *CaregiverInvitation) Reject() error {
	if i.Status != InvitationStatusPending {
		return ErrInvitationNotPending
	}
	i.Status = InvitationStatusRejected
	i.UpdatedAt = time.Now()
	return nil
}

// AcceptURL returns the URL the caregiver can visit to accept this invitation.
// baseURL is the configured invitation base (e.g. "https://app.example.com")
// and token is the invitation's acceptance token. An empty baseURL produces
// a relative path so the link is still useful in environments without a
// configured public URL.
func (i *CaregiverInvitation) AcceptURL(baseURL, token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if baseURL == "" {
		return "/invitations/" + url.PathEscape(token) + "/accept"
	}
	return strings.TrimRight(baseURL, "/") + "/invitations/" + url.PathEscape(token) + "/accept"
}

// InvitationRepository defines persistence for caregiver invitations.
type InvitationRepository interface {
	Save(ctx context.Context, inv *CaregiverInvitation) error
	FindByToken(ctx context.Context, token string) (*CaregiverInvitation, error)
	FindByElderlyID(ctx context.Context, elderlyID string) ([]*CaregiverInvitation, error)
	FindByCaregiverID(ctx context.Context, caregiverID string) ([]*CaregiverInvitation, error)
}

// Domain errors
var (
	ErrInvitationNotFound  = errors.New("invitation not found")
	ErrInvitationNotPending = errors.New("invitation is not pending")
	ErrInvalidCaregiverID  = errors.New("invalid caregiver ID")
	ErrInvalidElderlyID    = errors.New("invalid elderly ID")
	ErrAlreadyLinked       = errors.New("users are already linked")
)
