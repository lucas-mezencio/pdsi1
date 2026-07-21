package queries

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
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
	uRepo := minimalUserRepo(userID)

	h := NewDeviceTokenQueryHandler(dtRepo, uRepo)
	out, err := h.ListDeviceTokens(context.Background(),
		ListDeviceTokensQuery{CallerFirebaseID: "fb-uid"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(out))
	}
}

// minimalUserRepo returns a user.Repository stub that resolves a single uid.
func minimalUserRepo(localID string) user.Repository {
	return &minimalUserRepoImpl{localID: localID}
}

type minimalUserRepoImpl struct{ localID string }

func (m *minimalUserRepoImpl) FindByFirebaseID(ctx context.Context, fid string) (*user.User, error) {
	return &user.User{ID: m.localID}, nil
}

// Other methods panic — they must not be called.
func (m *minimalUserRepoImpl) Save(context.Context, *user.User) error { panic("not used") }
func (m *minimalUserRepoImpl) FindByID(context.Context, string) (*user.User, error) { panic("not used") }
func (m *minimalUserRepoImpl) FindByEmail(context.Context, string) (*user.User, error) { panic("not used") }
func (m *minimalUserRepoImpl) FindAll(context.Context) ([]*user.User, error) { panic("not used") }
func (m *minimalUserRepoImpl) Delete(context.Context, string) error { panic("not used") }
func (m *minimalUserRepoImpl) Exists(context.Context, string) (bool, error) { panic("not used") }
func (m *minimalUserRepoImpl) FindCaregivers(context.Context, string) ([]*user.User, error) { panic("not used") }
func (m *minimalUserRepoImpl) FindCharges(context.Context, string) ([]*user.User, error) { panic("not used") }
func (m *minimalUserRepoImpl) IsLinked(context.Context, string, string) (bool, error) { panic("not used") }
func (m *minimalUserRepoImpl) LinkUsers(context.Context, string, string) error { panic("not used") }
func (m *minimalUserRepoImpl) UnlinkUsers(context.Context, string, string) error { panic("not used") }
