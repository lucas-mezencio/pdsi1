package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// stubExtendedServer wires up only the device-token dependencies on an
// ExtendedServer so we can call the handlers directly.
func stubExtendedServer(
	t *testing.T,
	dtRepo devicetoken.Repository,
	uRepo user.Repository,
) *ExtendedServer {
	t.Helper()
	dtCmd := commands.NewDeviceTokenCommandHandler(dtRepo, uRepo)
	dtQuery := queries.NewDeviceTokenQueryHandler(dtRepo, uRepo)
	return NewExtendedServer(
		uRepo,
		nil, nil, nil, nil, nil, nil, // existing handlers unused here
		dtCmd, dtQuery,
	)
}

// withCallerID injects a Firebase UID into the request context the way the
// auth middleware would.
func withCallerID(req *http.Request, uid string) *http.Request {
	ctx := context.WithValue(req.Context(), contextKeyUserID, uid)
	return req.WithContext(ctx)
}

func TestPostDeviceToken_Success(t *testing.T) {
	localID := "11111111-1111-1111-1111-111111111111"
	uRepo := &fakeUserRepoByFirebase{localID: localID}
	dtRepo := &fakeDeviceTokenRepo{
		saveFn: func(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error) {
			t.ID = "dt-1"
			return t, nil
		},
	}
	s := stubExtendedServer(t, dtRepo, uRepo)

	body := strings.NewReader(`{"token":"fcm-abc-12345"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/device-tokens", body)
	req.Header.Set("Content-Type", "application/json")
	req = withCallerID(req, "fb-uid")
	rr := httptest.NewRecorder()

	s.RegisterDeviceToken(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["enabled"] != true {
		t.Fatalf("expected enabled=true, got %v", resp["enabled"])
	}
	if _, hasToken := resp["token"]; hasToken {
		t.Fatalf("response leaked raw token field")
	}
}

func TestPostDeviceToken_InvalidBody(t *testing.T) {
	s := stubExtendedServer(t, &fakeDeviceTokenRepo{}, &fakeUserRepoByFirebase{})

	body := strings.NewReader(`{"token":"ab"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/device-tokens", body)
	req.Header.Set("Content-Type", "application/json")
	req = withCallerID(req, "fb-uid")
	rr := httptest.NewRecorder()

	s.RegisterDeviceToken(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestListDeviceTokens(t *testing.T) {
	localID := "11111111-1111-1111-1111-111111111111"
	uRepo := &fakeUserRepoByFirebase{localID: localID}
	dtRepo := &fakeDeviceTokenRepo{
		findByUserIDFn: func(ctx context.Context, uid string) ([]*devicetoken.DeviceToken, error) {
			return []*devicetoken.DeviceToken{
				{ID: "dt-1", UserID: localID, Enabled: true},
			}, nil
		},
	}
	s := stubExtendedServer(t, dtRepo, uRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/device-tokens", nil)
	req = withCallerID(req, "fb-uid")
	rr := httptest.NewRecorder()

	s.ListDeviceTokens(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var out []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 token, got %d", len(out))
	}
	if _, hasToken := out[0]["token"]; hasToken {
		t.Fatalf("response leaked raw token field")
	}
}

func TestDeleteDeviceToken_NotOwner(t *testing.T) {
	uRepo := &fakeUserRepoByFirebase{localID: "self"}
	dtRepo := &fakeDeviceTokenRepo{
		findByIDFn: func(ctx context.Context, id string) (*devicetoken.DeviceToken, error) {
			return &devicetoken.DeviceToken{ID: id, UserID: "other"}, nil
		},
		deleteFn: func(ctx context.Context, id string) error {
			t.Fatalf("delete must not be called for forbidden token")
			return nil
		},
	}
	s := stubExtendedServer(t, dtRepo, uRepo)

	req := deleteRequest("/api/v1/users/me/device-tokens/dt-1")
	req = withCallerID(req, "fb-uid")
	rr := httptest.NewRecorder()

	s.DeleteDeviceToken(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSetDeviceTokenEnabled(t *testing.T) {
	localID := "11111111-1111-1111-1111-111111111111"
	uRepo := &fakeUserRepoByFirebase{localID: localID}
	dtRepo := &fakeDeviceTokenRepo{
		findByIDFn: func(ctx context.Context, id string) (*devicetoken.DeviceToken, error) {
			return &devicetoken.DeviceToken{ID: id, UserID: localID, Enabled: true}, nil
		},
		setEnabledFn: func(ctx context.Context, id string, enabled bool) (*devicetoken.DeviceToken, error) {
			return &devicetoken.DeviceToken{ID: id, UserID: localID, Enabled: enabled}, nil
		},
	}
	s := stubExtendedServer(t, dtRepo, uRepo)

	body := bytes.NewReader([]byte(`{"enabled":false}`))
	req := patchRequest("/api/v1/users/me/device-tokens/dt-1/enabled", body)
	req = withCallerID(req, "fb-uid")
	rr := httptest.NewRecorder()

	s.SetDeviceTokenEnabled(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// --- helpers ---

func deleteRequest(target string) *http.Request {
	r := httptest.NewRequest(http.MethodDelete, target, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tokenId", "dt-1")
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func patchRequest(target string, body *bytes.Reader) *http.Request {
	r := httptest.NewRequest(http.MethodPatch, target, body)
	r.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tokenId", "dt-1")
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// fakeUserRepoByFirebase resolves any caller to a fixed local user ID.
type fakeUserRepoByFirebase struct{ localID string }

func (f *fakeUserRepoByFirebase) FindByFirebaseID(context.Context, string) (*user.User, error) {
	return &user.User{ID: f.localID}, nil
}
func (f *fakeUserRepoByFirebase) Save(context.Context, *user.User) error { panic("unused") }
func (f *fakeUserRepoByFirebase) FindByID(context.Context, string) (*user.User, error) {
	panic("unused")
}
func (f *fakeUserRepoByFirebase) FindByEmail(context.Context, string) (*user.User, error) {
	panic("unused")
}
func (f *fakeUserRepoByFirebase) FindAll(context.Context) ([]*user.User, error) { panic("unused") }
func (f *fakeUserRepoByFirebase) Delete(context.Context, string) error          { panic("unused") }
func (f *fakeUserRepoByFirebase) Exists(context.Context, string) (bool, error)  { panic("unused") }
func (f *fakeUserRepoByFirebase) FindCaregivers(context.Context, string) ([]*user.User, error) {
	panic("unused")
}
func (f *fakeUserRepoByFirebase) FindCharges(context.Context, string) ([]*user.User, error) {
	panic("unused")
}
func (f *fakeUserRepoByFirebase) IsLinked(context.Context, string, string) (bool, error) {
	panic("unused")
}
func (f *fakeUserRepoByFirebase) LinkUsers(context.Context, string, string) error   { panic("unused") }
func (f *fakeUserRepoByFirebase) UnlinkUsers(context.Context, string, string) error { panic("unused") }

// fakeDeviceTokenRepo implements devicetoken.Repository with function fields.
type fakeDeviceTokenRepo struct {
	saveFn          func(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error)
	findByIDFn      func(ctx context.Context, id string) (*devicetoken.DeviceToken, error)
	findByUserIDFn  func(ctx context.Context, userID string) ([]*devicetoken.DeviceToken, error)
	deleteFn        func(ctx context.Context, id string) error
	setEnabledFn    func(ctx context.Context, id string, enabled bool) (*devicetoken.DeviceToken, error)
	touchLastUsedFn func(ctx context.Context, id string) error
}

func (f *fakeDeviceTokenRepo) Save(ctx context.Context, t *devicetoken.DeviceToken) (*devicetoken.DeviceToken, error) {
	if f.saveFn == nil {
		return t, nil
	}
	return f.saveFn(ctx, t)
}
func (f *fakeDeviceTokenRepo) FindByID(ctx context.Context, id string) (*devicetoken.DeviceToken, error) {
	if f.findByIDFn == nil {
		return &devicetoken.DeviceToken{ID: id}, nil
	}
	return f.findByIDFn(ctx, id)
}
func (f *fakeDeviceTokenRepo) FindByUserID(ctx context.Context, userID string) ([]*devicetoken.DeviceToken, error) {
	if f.findByUserIDFn == nil {
		return nil, nil
	}
	return f.findByUserIDFn(ctx, userID)
}
func (f *fakeDeviceTokenRepo) Delete(ctx context.Context, id string) error {
	if f.deleteFn == nil {
		return nil
	}
	return f.deleteFn(ctx, id)
}
func (f *fakeDeviceTokenRepo) SetEnabled(ctx context.Context, id string, enabled bool) (*devicetoken.DeviceToken, error) {
	if f.setEnabledFn == nil {
		return &devicetoken.DeviceToken{ID: id, Enabled: enabled}, nil
	}
	return f.setEnabledFn(ctx, id, enabled)
}
func (f *fakeDeviceTokenRepo) TouchLastUsed(ctx context.Context, id string) error {
	if f.touchLastUsedFn == nil {
		return nil
	}
	return f.touchLastUsedFn(ctx, id)
}
