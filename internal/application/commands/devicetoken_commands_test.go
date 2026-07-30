package commands

import (
    "context"
    "errors"
    "testing"

    "github.com/google/uuid"

    "github.com.br/lucas-mezencio/pdsi1/internal/application"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
)

type mockDeviceTokenRepo struct {
    saveFn       func(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error)
    findByIDFn   func(ctx context.Context, id string) (*devicetoken.DeviceToken, error)
    findByUserIDFn func(ctx context.Context, userID string) ([]*devicetoken.DeviceToken, error)
    deleteFn     func(ctx context.Context, id string) error
    setEnabledFn func(ctx context.Context, id string, enabled bool) (*devicetoken.DeviceToken, error)
    touchLastUsedFn func(ctx context.Context, id string) error
}

func (m *mockDeviceTokenRepo) Save(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error) {
    return m.saveFn(ctx, t)
}
func (m *mockDeviceTokenRepo) FindByID(ctx context.Context, id string) (*devicetoken.DeviceToken, error) {
    return m.findByIDFn(ctx, id)
}
func (m *mockDeviceTokenRepo) FindByUserID(ctx context.Context, userID string) ([]*devicetoken.DeviceToken, error) {
    return m.findByUserIDFn(ctx, userID)
}
func (m *mockDeviceTokenRepo) Delete(ctx context.Context, id string) error {
    return m.deleteFn(ctx, id)
}
func (m *mockDeviceTokenRepo) SetEnabled(ctx context.Context, id string, enabled bool) (*devicetoken.DeviceToken, error) {
    return m.setEnabledFn(ctx, id, enabled)
}
func (m *mockDeviceTokenRepo) TouchLastUsed(ctx context.Context, id string) error {
    return m.touchLastUsedFn(ctx, id)
}

func TestRegisterDeviceToken_Success(t *testing.T) {
    userID := uuid.New().String()
    dtRepo := &mockDeviceTokenRepo{
        saveFn: func(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error) {
            t.ID = "dt-1"
            return t, nil
        },
    }

    h := NewDeviceTokenCommandHandler(dtRepo, nil)
    got, err := h.RegisterDeviceToken(context.Background(),
        RegisterDeviceTokenCommand{CallerID: userID, Token: "fcm-abc-12345"})

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got.UserID != userID {
        t.Fatalf("expected userID %s, got %s", userID, got.UserID)
    }
    if !got.Enabled {
        t.Fatalf("expected Enabled=true")
    }
}

func TestRegisterDeviceToken_InvalidToken(t *testing.T) {
    h := NewDeviceTokenCommandHandler(&mockDeviceTokenRepo{}, nil)

    _, err := h.RegisterDeviceToken(context.Background(),
        RegisterDeviceTokenCommand{CallerID: uuid.New().String(), Token: "ab"})
    if !errors.Is(err, application.ErrInvalidInput) {
        t.Fatalf("expected ErrInvalidInput, got %v", err)
    }
}

func TestRegisterDeviceToken_Conflict(t *testing.T) {
    dtRepo := &mockDeviceTokenRepo{
        saveFn: func(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error) {
            return nil, devicetoken.ErrConflict
        },
    }

    h := NewDeviceTokenCommandHandler(dtRepo, nil)
    _, err := h.RegisterDeviceToken(context.Background(),
        RegisterDeviceTokenCommand{CallerID: uuid.New().String(), Token: "fcm-abc-12345"})

    if !errors.Is(err, application.ErrConflict) {
        t.Fatalf("expected ErrConflict, got %v", err)
    }
}

func TestDeleteDeviceToken_Forbidden(t *testing.T) {
    ownerID := uuid.New().String()
    otherID := uuid.New().String()

    dtRepo := &mockDeviceTokenRepo{
        findByIDFn: func(ctx context.Context, id string) (*devicetoken.DeviceToken, error) {
            return &devicetoken.DeviceToken{ID: id, UserID: ownerID}, nil
        },
        deleteFn: func(ctx context.Context, id string) error {
            t.Fatalf("delete must not be called for forbidden token")
            return nil
        },
    }

    h := NewDeviceTokenCommandHandler(dtRepo, nil)
    err := h.DeleteDeviceToken(context.Background(),
        DeleteDeviceTokenCommand{CallerID: otherID, TokenID: "dt-1"})

    if !errors.Is(err, application.ErrForbidden) {
        t.Fatalf("expected ErrForbidden, got %v", err)
    }
}

func TestSetDeviceTokenEnabled_Success(t *testing.T) {
    userID := uuid.New().String()
    dtRepo := &mockDeviceTokenRepo{
        findByIDFn: func(ctx context.Context, id string) (*devicetoken.DeviceToken, error) {
            return &devicetoken.DeviceToken{ID: id, UserID: userID, Enabled: true}, nil
        },
        setEnabledFn: func(ctx context.Context, id string, enabled bool) (*devicetoken.DeviceToken, error) {
            if enabled {
                t.Fatalf("expected enabled=false")
            }
            return &devicetoken.DeviceToken{ID: id, UserID: userID, Enabled: false}, nil
        },
    }

    h := NewDeviceTokenCommandHandler(dtRepo, nil)
    got, err := h.SetDeviceTokenEnabled(context.Background(),
        SetDeviceTokenEnabledCommand{CallerID: userID, TokenID: "dt-1", Enabled: false})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got.Enabled {
        t.Fatalf("expected Enabled=false")
    }
}
