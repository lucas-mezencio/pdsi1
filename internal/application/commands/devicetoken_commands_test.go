package commands

import (
    "context"
    "errors"
    "testing"

    "github.com/google/uuid"

    "github.com.br/lucas-mezencio/pdsi1/internal/application"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

type mockDeviceTokenRepo struct {
    saveFn          func(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error)
    findByIDFn      func(ctx context.Context, id string) (*devicetoken.DeviceToken, error)
    findByUserIDFn  func(ctx context.Context, userID string) ([]*devicetoken.DeviceToken, error)
    deleteFn        func(ctx context.Context, id string) error
    setEnabledFn    func(ctx context.Context, id string, enabled bool) (*devicetoken.DeviceToken, error)
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

// mockDeviceTokenUserRepo is a minimal user.Repository mock for device-token tests.
// Renamed to avoid collision with the existing mockUserRepo in user_commands_test.go.
type mockDeviceTokenUserRepo struct {
    findByFirebaseIDFn func(ctx context.Context, fid string) (*user.User, error)
}

func (m *mockDeviceTokenUserRepo) FindByFirebaseID(ctx context.Context, fid string) (*user.User, error) {
    return m.findByFirebaseIDFn(ctx, fid)
}

// Other user.Repository methods are stubs that fail loudly:
func (m *mockDeviceTokenUserRepo) Save(context.Context, *user.User) error { panic("not used") }
func (m *mockDeviceTokenUserRepo) FindByID(context.Context, string) (*user.User, error) { panic("not used") }
func (m *mockDeviceTokenUserRepo) FindByEmail(context.Context, string) (*user.User, error) { panic("not used") }
func (m *mockDeviceTokenUserRepo) FindAll(context.Context) ([]*user.User, error) { panic("not used") }
func (m *mockDeviceTokenUserRepo) Delete(context.Context, string) error { panic("not used") }
func (m *mockDeviceTokenUserRepo) Exists(context.Context, string) (bool, error) { panic("not used") }
func (m *mockDeviceTokenUserRepo) FindCaregivers(context.Context, string) ([]*user.User, error) { panic("not used") }
func (m *mockDeviceTokenUserRepo) FindCharges(context.Context, string) ([]*user.User, error) { panic("not used") }
func (m *mockDeviceTokenUserRepo) IsLinked(context.Context, string, string) (bool, error) { panic("not used") }
func (m *mockDeviceTokenUserRepo) LinkUsers(context.Context, string, string) error { panic("not used") }
func (m *mockDeviceTokenUserRepo) UnlinkUsers(context.Context, string, string) error { panic("not used") }

func newDeviceTokenHandler(
    dtRepo devicetoken.Repository,
    uRepo user.Repository,
) *DeviceTokenCommandHandler {
    return NewDeviceTokenCommandHandler(dtRepo, uRepo)
}

func TestRegisterDeviceToken_Success(t *testing.T) {
    userID := uuid.New().String()
    dtRepo := &mockDeviceTokenRepo{
        saveFn: func(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error) {
            t.ID = "dt-1"
            return t, nil
        },
    }
    uRepo := &mockDeviceTokenUserRepo{
        findByFirebaseIDFn: func(ctx context.Context, fid string) (*user.User, error) {
            return &user.User{ID: userID}, nil
        },
    }

    h := newDeviceTokenHandler(dtRepo, uRepo)
    got, err := h.RegisterDeviceToken(context.Background(),
        RegisterDeviceTokenCommand{CallerFirebaseID: "fb-uid", Token: "fcm-abc-12345"})

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
    uRepo := &mockDeviceTokenUserRepo{
        findByFirebaseIDFn: func(ctx context.Context, fid string) (*user.User, error) {
            return &user.User{ID: uuid.New().String()}, nil
        },
    }
    h := newDeviceTokenHandler(&mockDeviceTokenRepo{}, uRepo)

    _, err := h.RegisterDeviceToken(context.Background(),
        RegisterDeviceTokenCommand{CallerFirebaseID: "fb-uid", Token: "ab"})
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
    uRepo := &mockDeviceTokenUserRepo{
        findByFirebaseIDFn: func(ctx context.Context, fid string) (*user.User, error) {
            return &user.User{ID: uuid.New().String()}, nil
        },
    }

    h := newDeviceTokenHandler(dtRepo, uRepo)
    _, err := h.RegisterDeviceToken(context.Background(),
        RegisterDeviceTokenCommand{CallerFirebaseID: "fb-uid", Token: "fcm-abc-12345"})

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
    uRepo := &mockDeviceTokenUserRepo{
        findByFirebaseIDFn: func(ctx context.Context, fid string) (*user.User, error) {
            return &user.User{ID: otherID}, nil
        },
    }

    h := newDeviceTokenHandler(dtRepo, uRepo)
    err := h.DeleteDeviceToken(context.Background(),
        DeleteDeviceTokenCommand{CallerFirebaseID: "fb-uid", TokenID: "dt-1"})

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
    uRepo := &mockDeviceTokenUserRepo{
        findByFirebaseIDFn: func(ctx context.Context, fid string) (*user.User, error) {
            return &user.User{ID: userID}, nil
        },
    }

    h := newDeviceTokenHandler(dtRepo, uRepo)
    got, err := h.SetDeviceTokenEnabled(context.Background(),
        SetDeviceTokenEnabledCommand{CallerFirebaseID: "fb-uid", TokenID: "dt-1", Enabled: false})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got.Enabled {
        t.Fatalf("expected Enabled=false")
    }
}