package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestLogin_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["email"] != "a@b.com" || body["password"] != "pw" {
			t.Errorf("unexpected body: %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		// Mirror the real AuthResponse shape returned by internal/api/server.go:348:
		// { "token": "...", "user": { "id": "<uuid>", ... } }
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token": "jwt-token",
			"user": map[string]any{
				"id":    "u-1",
				"email": "a@b.com",
				"name":  "Test",
				"phone": "+1",
			},
		})
	}))
	defer srv.Close()

	api := New(srv.URL, "secret")
	id, err := api.Login(context.Background(), "a@b.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "u-1" {
		t.Errorf("got id %q, want u-1", id)
	}
}

func TestLogin_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
	}))
	defer srv.Close()

	api := New(srv.URL, "secret")
	_, err := api.Login(context.Background(), "a@b.com", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("expected error to contain 'invalid credentials', got %v", err)
	}
}

func TestCreatePrescription_SendsRawSecretHeader(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": uuid.New().String()})
	}))
	defer srv.Close()

	api := New(srv.URL, "the-demo-secret")
	uid := uuid.New()
	resp, err := api.CreatePrescription(context.Background(), Prescription{
		UserID:     uid.String(),
		MedicID:    "11111111-1111-1111-1111-111111111111",
		Name:       "Aspirin",
		Dosage:     "100mg",
		Frequency:  "24:00",
		StartTime:  "02:30",
		Doses:      1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.ID.String() == "" {
		t.Fatal("expected non-empty response id")
	}
	if gotAuth != "the-demo-secret" {
		t.Errorf("expected raw Authorization header %q, got %q (must NOT include 'Bearer')", "the-demo-secret", gotAuth)
	}
	if gotPath != "/prescriptions" {
		t.Errorf("expected path /prescriptions, got %s", gotPath)
	}

	meds, ok := gotBody["medicaments"].([]any)
	if !ok || len(meds) != 1 {
		t.Fatalf("expected one medicament, got %v", gotBody["medicaments"])
	}
	m := meds[0].(map[string]any)
	if m["name"] != "Aspirin" || m["dosage"] != "100mg" || m["frequency"] != "24:00" {
		t.Errorf("unexpected medicament fields: %v", m)
	}
	if m["doses"].(float64) != 1 {
		t.Errorf("expected doses=1, got %v", m["doses"])
	}
	times, _ := m["time"].([]any)
	if len(times) != 1 || times[0] != "05:30" {
		t.Errorf("expected time=[05:30] (start 02:30 + 3h), got %v", m["time"])
	}
}

func TestCreatePrescription_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid secret"})
	}))
	defer srv.Close()

	api := New(srv.URL, "wrong-secret")
	_, err := api.CreatePrescription(context.Background(), Prescription{
		UserID: uuid.New().String(), MedicID: uuid.New().String(),
		Name: "A", Dosage: "1", Frequency: "24:00", StartTime: "08:00", Doses: 1,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid secret") {
		t.Errorf("expected error to contain 'invalid secret', got %v", err)
	}
}

func TestShiftStartTime(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"08:00", "11:00"},
		{"00:30", "03:30"},
		{"23:30", "02:30"},
		{"12:00", "15:00"},
	}
	for _, c := range cases {
		got, err := shiftStartTime(c.in)
		if err != nil {
			t.Errorf("shiftStartTime(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("shiftStartTime(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestShiftStartTime_InvalidInput(t *testing.T) {
	if _, err := shiftStartTime("not-a-time"); err == nil {
		t.Error("expected error for invalid input, got nil")
	}
}
