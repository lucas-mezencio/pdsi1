package devicetoken

import (
    "testing"
    "time"

    "github.com/google/uuid"
)

func newDeviceToken(t *testing.T) *DeviceToken {
    t.Helper()
    return &DeviceToken{
        ID:        uuid.New().String(),
        UserID:    uuid.New().String(),
        Token:     "fcm-token-abc",
        Enabled:   true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}

func TestDeviceToken_Enable(t *testing.T) {
    dt := newDeviceToken(t)
    dt.Enabled = false

    before := dt.UpdatedAt
    time.Sleep(time.Millisecond)
    dt.Enable()

    if !dt.Enabled {
        t.Fatalf("expected Enabled=true, got false")
    }
    if !dt.UpdatedAt.After(before) {
        t.Fatalf("expected UpdatedAt to advance, got %v", dt.UpdatedAt)
    }
}

func TestDeviceToken_Disable(t *testing.T) {
    dt := newDeviceToken(t)

    before := dt.UpdatedAt
    time.Sleep(time.Millisecond)
    dt.Disable()

    if dt.Enabled {
        t.Fatalf("expected Enabled=false, got true")
    }
    if !dt.UpdatedAt.After(before) {
        t.Fatalf("expected UpdatedAt to advance, got %v", dt.UpdatedAt)
    }
}

func TestDeviceToken_TouchLastUsed(t *testing.T) {
    dt := newDeviceToken(t)

    if dt.LastUsedAt != nil {
        t.Fatalf("expected LastUsedAt=nil, got %v", *dt.LastUsedAt)
    }

    dt.TouchLastUsed()

    if dt.LastUsedAt == nil {
        t.Fatalf("expected LastUsedAt to be set")
    }
    if time.Since(*dt.LastUsedAt) > time.Second {
        t.Fatalf("expected LastUsedAt close to now, got %v", *dt.LastUsedAt)
    }
}

func TestDeviceToken_Validate(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        wantErr bool
    }{
        {"valid", "abc123", false},
        {"empty", "", true},
        {"whitespace only", "   ", true},
        {"too short", "ab", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dt := newDeviceToken(t)
            dt.Token = tt.token
            err := dt.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() err = %v, wantErr = %v", err, tt.wantErr)
            }
        })
    }
}
