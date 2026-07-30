package queries

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
)

type mockDeviceTokenRepo struct {
	findByUserIDFn func(ctx context.Context, userID string) ([]*devicetoken.DeviceToken, error)
}

func (m *mockDeviceTokenRepo) Save(context.Context, *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error) {
	panic("not used")
}
func (m *mockDeviceTokenRepo) FindByID(context.Context, string) (*devicetoken.DeviceToken, error) {
	panic("not used")
}
func (m *mockDeviceTokenRepo) FindByUserID(ctx context.Context, userID string) ([]*devicetoken.DeviceToken, error) {
	return m.findByUserIDFn(ctx, userID)
}
func (m *mockDeviceTokenRepo) Delete(context.Context, string) error { panic("not used") }
func (m *mockDeviceTokenRepo) SetEnabled(context.Context, string, bool) (*devicetoken.DeviceToken, error) {
	panic("not used")
}
func (m *mockDeviceTokenRepo) TouchLastUsed(context.Context, string) error { panic("not used") }

func TestListDeviceTokens(t *testing.T) {
	userID := uuid.New().String()
	dtRepo := &mockDeviceTokenRepo{
		findByUserIDFn: func(ctx context.Context, uid string) ([]*devicetoken.DeviceToken, error) {
			if uid != userID {
				t.Fatalf("expected %s, got %s", userID, uid)
			}
			return []*devicetoken.DeviceToken{
				{ID: "dt-1", UserID: userID, Enabled: true},
				{ID: "dt-2", UserID: userID, Enabled: false},
			}, nil
		},
	}

	h := NewDeviceTokenQueryHandler(dtRepo)
	out, err := h.ListDeviceTokens(context.Background(),
		ListDeviceTokensQuery{CallerID: userID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(out))
	}
}
