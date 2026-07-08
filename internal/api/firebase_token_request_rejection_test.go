package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mirrors the existing TestCreateUser_RejectsFirebaseIDField but
// specifically for the firebase_token field that production clients send
// (see Image 1 in the bug report). Locks in the symptom: handler must
// reject unknown fields with 400, not silently accept them and 500 later.
func TestCreateUser_RejectsFirebaseTokenField(t *testing.T) {
	server, _, _ := newBugReproServer()

	body := strings.NewReader(`{
		"name":"John Doe",
		"email":"john.doe@example.com",
		"phone":"+1234567890",
		"password":"S3cretP@ss",
		"firebase_token":"fCM_token_authdoe"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.CreateUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown firebase_token field, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateDoctor_RejectsFirebaseTokenField(t *testing.T) {
	server, _, _ := newBugReproServer()

	body := strings.NewReader(`{
		"name":"Dr. Y",
		"email":"y@example.com",
		"phone":"999",
		"password":"S3cretP@ss",
		"specialty":"Traumatologist",
		"license_number":"MED-111",
		"firebase_token":"fCM_token_supplied"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/doctors", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.CreateDoctor(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown firebase_token, got %d: %s", rr.Code, rr.Body.String())
	}
}
