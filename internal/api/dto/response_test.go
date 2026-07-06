package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/api/dto"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

func TestUserResponse_StripsFirebaseToken(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	usr := &user.User{
		ID:            "user-1",
		Name:          "Maria",
		Email:         "maria@example.com",
		Phone:         "+55119...",
		CPF:           "52998224725",
		FirebaseID:    "firebase-uid-1",
		FirebaseToken: "super-secret-firebase-token",
		Role:          user.RoleElderly,
		NotificationsEnabled: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	resp := dto.UserResponseFromDomain(usr)
	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := raw["firebase_token"]; ok {
		t.Fatalf("firebase_token must NOT appear in the response, got: %s", body)
	}
	if _, ok := raw["firebaseToken"]; ok {
		t.Fatalf("firebaseToken must NOT appear in the response, got: %s", body)
	}

	if raw["firebase_id"] != "firebase-uid-1" {
		t.Fatalf("firebase_id must remain in the response, got: %s", body)
	}
	if raw["id"] != "user-1" {
		t.Fatalf("expected id=user-1, got: %s", body)
	}
}

func TestDoctorResponse_StripsFirebaseIDWhenEmptyAndKeepsWhenSet(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)

	withID := &doctor.Doctor{
		ID: "doc-1", Name: "Dr. Who", Email: "who@example.com", Phone: "999",
		FirebaseID: "firebase-uid-doc-1", Specialty: "Time", LicenseNumber: "LIC-1",
		CreatedAt: now, UpdatedAt: now,
	}
	bodyWithID, err := json.Marshal(dto.DoctorResponseFromDomain(withID))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var rawWithID map[string]any
	if err := json.Unmarshal(bodyWithID, &rawWithID); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rawWithID["firebase_id"] != "firebase-uid-doc-1" {
		t.Fatalf("expected firebase_id to remain, got: %s", bodyWithID)
	}

	withoutID := &doctor.Doctor{
		ID: "doc-2", Name: "Dr. Strange", Email: "strange@example.com", Phone: "998",
		FirebaseID: "", Specialty: "Magic", LicenseNumber: "LIC-2",
		CreatedAt: now, UpdatedAt: now,
	}
	bodyWithoutID, err := json.Marshal(dto.DoctorResponseFromDomain(withoutID))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var rawWithoutID map[string]any
	if err := json.Unmarshal(bodyWithoutID, &rawWithoutID); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := rawWithoutID["firebase_id"]; ok {
		t.Fatalf("firebase_id must be omitted when empty, got: %s", bodyWithoutID)
	}
}