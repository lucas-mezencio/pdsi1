package devicetoken

import (
    "strings"
    "time"

    "github.com/google/uuid"
)

// DeviceToken represents a single FCM device token owned by a user.
// A user may have N DeviceTokens (one per device).
type DeviceToken struct {
    ID         string
    UserID     string
    Token      string
    Enabled    bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
    LastUsedAt *time.Time
}

// New constructs a DeviceToken. Generates ID, sets timestamps, sets Enabled=true.
func New(userID, token string) (*DeviceToken, error) {
    dt := &DeviceToken{
        ID:        uuid.New().String(),
        UserID:    userID,
        Token:     token,
        Enabled:   true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    if err := dt.Validate(); err != nil {
        return nil, err
    }
    return dt, nil
}

// Validate enforces a minimal FCM-token shape: non-empty, no whitespace, >= 4 chars.
func (d *DeviceToken) Validate() error {
    if strings.TrimSpace(d.Token) == "" {
        return ErrInvalidToken
    }
    if strings.ContainsAny(d.Token, " \t\n\r") {
        return ErrInvalidToken
    }
    if len(d.Token) < 4 {
        return ErrInvalidToken
    }
    return nil
}

// Enable marks the token active and bumps UpdatedAt.
func (d *DeviceToken) Enable() {
    d.Enabled = true
    d.UpdatedAt = time.Now()
}

// Disable marks the token inactive and bumps UpdatedAt.
func (d *DeviceToken) Disable() {
    d.Enabled = false
    d.UpdatedAt = time.Now()
}

// TouchLastUsed records the timestamp of the last successful send.
func (d *DeviceToken) TouchLastUsed() {
    now := time.Now()
    d.LastUsedAt = &now
}
