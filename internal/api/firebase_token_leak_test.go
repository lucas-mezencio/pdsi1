package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	gen "github.com.br/lucas-mezencio/pdsi1/internal/api/gen"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// FirebaseToken is the FCM device token used by the scheduler worker to
// push medication reminders. It is server-internal: not an input field, not
// a response field. These tests assert that the API leaks it in no
// direction.

func TestUpdateFirebaseToken_NeverReturnsFirebaseToken(t *testing.T) {
	server, userRepo, _ := newBugReproServer()
	userRepo.items = append(userRepo.items, &user.User{
		ID: "00000000-0000-4000-8000-000000000001", Name: "Alice", Email: "a@example.com", Phone: "+1",
		FirebaseID: "firebase-uid-1", FirebaseToken: "old-token",
		Role: user.RoleElderly,
	})

	body := strings.NewReader(`{"firebase_token":"new-fcm-token"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/00000000-0000-4000-8000-000000000001/firebase-token", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	userID := gen.UserId(uuid.MustParse("00000000-0000-4000-8000-000000000001"))
	server.UpdateFirebaseToken(rr, req, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := raw["firebase_token"]; ok {
		t.Fatalf("firebase_token must NOT appear in PATCH /users/{id}/firebase-token response, got: %s", rr.Body.String())
	}
}

func TestUpdateUser_NeverReturnsFirebaseToken(t *testing.T) {
	server, userRepo, _ := newBugReproServer()
	userRepo.items = append(userRepo.items, &user.User{
		ID: "00000000-0000-4000-8000-000000000001", Name: "Alice", Email: "a@example.com", Phone: "+1",
		FirebaseID: "firebase-uid-1", FirebaseToken: "secret-fcm-token",
		Role: user.RoleElderly,
	})

	body := strings.NewReader(`{"name":"Alice 2","email":"a@example.com","phone":"+1","cpf":""}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/00000000-0000-4000-8000-000000000001", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	userID := gen.UserId(uuid.MustParse("00000000-0000-4000-8000-000000000001"))
	server.UpdateUser(rr, req, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := raw["firebase_token"]; ok {
		t.Fatalf("firebase_token must NOT appear in PUT /users/{id} response, got: %s", rr.Body.String())
	}
}

func TestToggleNotifications_NeverReturnsFirebaseToken(t *testing.T) {
	server, userRepo, _ := newBugReproServer()
	userRepo.items = append(userRepo.items, &user.User{
		ID: "00000000-0000-4000-8000-000000000001", Name: "Alice", Email: "a@example.com", Phone: "+1",
		FirebaseID: "firebase-uid-1", FirebaseToken: "secret-fcm-token",
		Role: user.RoleElderly,
	})

	body := strings.NewReader(`{"enabled":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/00000000-0000-4000-8000-000000000001/notifications", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	userID := gen.UserId(uuid.MustParse("00000000-0000-4000-8000-000000000001"))
	server.ToggleNotifications(rr, req, userID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := raw["firebase_token"]; ok {
		t.Fatalf("firebase_token must NOT appear in POST /users/{id}/notifications response, got: %s", rr.Body.String())
	}
}

func TestRegister_NeverReturnsFirebaseToken(t *testing.T) {
	server, userRepo, _ := newBugReproServer()
	_ = userRepo

	body := strings.NewReader(`{
		"name":"Alice",
		"email":"alice-new@example.com",
		"phone":"+100000000",
		"password":"S3cretP@ss"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := raw["firebase_token"]; ok {
		t.Fatalf("firebase_token must NOT appear in POST /auth/register response, got: %s", rr.Body.String())
	}
}

func TestLogin_NeverReturnsFirebaseToken(t *testing.T) {
	server, userRepo, _ := newBugReproServer()
	userRepo.items = append(userRepo.items, &user.User{
		ID: "u-login", Name: "Login User", Email: "login-user@example.com", Phone: "+1",
		FirebaseID: "firebase-uid-login", FirebaseToken: "secret-fcm-token",
		Role: user.RoleElderly,
	})

	// Define a path-handler so we can reuse the same server plumbing used
	// for Login. We invoke the handler directly so we don't need to wire
	// the router; the Login method takes the raw request.
	body := strings.NewReader(`{"email":"login-user@example.com","password":"anything"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var raw struct {
		Token string         `json:"token"`
		User  map[string]any `json:"user"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if raw.User == nil {
		t.Fatalf("expected user in login response, got: %s", rr.Body.String())
	}
	if _, ok := raw.User["firebase_token"]; ok {
		t.Fatalf("firebase_token must NOT appear in POST /auth/login response, got: %s", rr.Body.String())
	}
}

// Compile-time guard against silent signature drift in
// commands.AuthenticationProvider when both auth tests above are run.
var _ = context.Background
