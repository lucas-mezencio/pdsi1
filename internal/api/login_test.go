package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// stubAuthProvider implements commands.AuthenticationProvider for the login
// HTTP test. It returns a deterministic Firebase UID and ID token so we can
// assert both fields reach the response.
type stubAuthProvider struct{}

func (stubAuthProvider) CreateUser(ctx context.Context, email, password string) (string, error) {
	return "firebase-uid", nil
}

func (stubAuthProvider) DeleteUser(ctx context.Context, firebaseID string) error { return nil }

func (stubAuthProvider) SignIn(ctx context.Context, email, password string) (string, string, error) {
	return "firebase-uid", "id-token-jwt", nil
}

// stubUserRepoForLogin returns a known user when looked up by Firebase ID.
type stubUserRepoForLogin struct{}

func (stubUserRepoForLogin) Save(ctx context.Context, u *user.User) error { return nil }
func (stubUserRepoForLogin) FindByID(ctx context.Context, id string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (stubUserRepoForLogin) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (stubUserRepoForLogin) FindByFirebaseID(ctx context.Context, firebaseID string) (*user.User, error) {
	created, _ := time.Parse(time.RFC3339Nano, "2026-06-21T21:44:59.383261Z")
	updated, _ := time.Parse(time.RFC3339Nano, "2026-06-21T21:44:59.383414Z")
	return &user.User{
		ID:                   "11111111-1111-4111-8111-111111111111",
		Name:                 "Joao Doe",
		Email:                "joao.doe@example.com",
		Phone:                "+123456320",
		FirebaseID:           firebaseID,
		Role:                 user.RoleElderly,
		CreatedAt:            created,
		UpdatedAt:            updated,
		NotificationsEnabled: true,
	}, nil
}
func (stubUserRepoForLogin) FindAll(ctx context.Context) ([]*user.User, error) { return nil, nil }
func (stubUserRepoForLogin) Delete(ctx context.Context, id string) error         { return nil }
func (stubUserRepoForLogin) Exists(ctx context.Context, id string) (bool, error) { return false, nil }
func (stubUserRepoForLogin) FindCaregivers(ctx context.Context, elderlyID string) ([]*user.User, error) {
	return nil, nil
}
func (stubUserRepoForLogin) FindCharges(ctx context.Context, caregiverID string) ([]*user.User, error) {
	return nil, nil
}
func (stubUserRepoForLogin) IsLinked(ctx context.Context, caregiverID, elderlyID string) (bool, error) {
	return false, nil
}
func (stubUserRepoForLogin) LinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}
func (stubUserRepoForLogin) UnlinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}

func TestServerLogin_ReturnsAuthResponseWithToken(t *testing.T) {
	repo := &stubUserRepoForLogin{}
	authProvider := stubAuthProvider{}
	authHandler := commands.NewAuthCommandHandler(repo, authProvider)

	server := &Server{authCommands: authHandler}

	body := strings.NewReader(`{"email":"joao.doe@example.com","password":"password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"user"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Token != "id-token-jwt" {
		t.Fatalf("expected token id-token-jwt, got %q", resp.Token)
	}
	if resp.User.Email != "joao.doe@example.com" {
		t.Fatalf("expected user email joao.doe@example.com, got %q", resp.User.Email)
	}
}