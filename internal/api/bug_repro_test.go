package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// stubAuthProviderForBugRepro reproduces the production behaviour:
// CreateUser returns a deterministic UID; DeleteUser is a no-op.
type stubAuthProviderForBugRepro struct{}

func (stubAuthProviderForBugRepro) CreateUser(ctx context.Context, email, password string) (string, error) {
	return "firebase-uid-from-admin-sdk", nil
}
func (stubAuthProviderForBugRepro) DeleteUser(ctx context.Context, firebaseID string) error { return nil }
func (stubAuthProviderForBugRepro) SignIn(ctx context.Context, email, password string) (string, string, error) {
	return "firebase-uid-from-admin-sdk", "id-token", nil
}

type stubUserRepoForBugRepro struct {
	items []*user.User
}

func (s *stubUserRepoForBugRepro) Save(ctx context.Context, u *user.User) error {
	s.items = append(s.items, u)
	return nil
}
func (s *stubUserRepoForBugRepro) FindByID(ctx context.Context, id string) (*user.User, error) {
	for _, u := range s.items {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepoForBugRepro) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	for _, u := range s.items {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepoForBugRepro) FindByFirebaseID(ctx context.Context, firebaseID string) (*user.User, error) {
	for _, u := range s.items {
		if u.FirebaseID == firebaseID {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepoForBugRepro) FindAll(ctx context.Context) ([]*user.User, error) {
	return s.items, nil
}
func (s *stubUserRepoForBugRepro) Delete(ctx context.Context, id string) error { return nil }
func (s *stubUserRepoForBugRepro) Exists(ctx context.Context, id string) (bool, error) {
	for _, u := range s.items {
		if u.ID == id {
			return true, nil
		}
	}
	return false, nil
}
func (s *stubUserRepoForBugRepro) FindCaregivers(ctx context.Context, elderlyID string) ([]*user.User, error) {
	return nil, nil
}
func (s *stubUserRepoForBugRepro) FindCharges(ctx context.Context, caregiverID string) ([]*user.User, error) {
	return nil, nil
}
func (s *stubUserRepoForBugRepro) IsLinked(ctx context.Context, caregiverID, elderlyID string) (bool, error) {
	return false, nil
}
func (s *stubUserRepoForBugRepro) LinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}
func (s *stubUserRepoForBugRepro) UnlinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}

type stubDoctorRepoForBugRepro struct {
	items []*doctor.Doctor
}

func (s *stubDoctorRepoForBugRepro) Save(ctx context.Context, d *doctor.Doctor) error {
	s.items = append(s.items, d)
	return nil
}
func (s *stubDoctorRepoForBugRepro) FindByID(ctx context.Context, id string) (*doctor.Doctor, error) {
	for _, d := range s.items {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, doctor.ErrDoctorNotFound
}
func (s *stubDoctorRepoForBugRepro) FindByEmail(ctx context.Context, email string) (*doctor.Doctor, error) {
	for _, d := range s.items {
		if d.Email == email {
			return d, nil
		}
	}
	return nil, doctor.ErrDoctorNotFound
}
func (s *stubDoctorRepoForBugRepro) FindByLicenseNumber(ctx context.Context, license string) (*doctor.Doctor, error) {
	for _, d := range s.items {
		if d.LicenseNumber == license {
			return d, nil
		}
	}
	return nil, doctor.ErrDoctorNotFound
}
func (s *stubDoctorRepoForBugRepro) FindAll(ctx context.Context) ([]*doctor.Doctor, error) {
	return s.items, nil
}
func (s *stubDoctorRepoForBugRepro) Delete(ctx context.Context, id string) error { return nil }
func (s *stubDoctorRepoForBugRepro) Exists(ctx context.Context, id string) (bool, error) {
	for _, d := range s.items {
		if d.ID == id {
			return true, nil
		}
	}
	return false, nil
}

func newBugReproServer() (*Server, *stubUserRepoForBugRepro, *stubDoctorRepoForBugRepro) {
	userRepo := &stubUserRepoForBugRepro{}
	doctorRepo := &stubDoctorRepoForBugRepro{}
	auth := stubAuthProviderForBugRepro{}

	server := &Server{
		userCommands:   commands.NewUserCommandHandler(userRepo, auth),
		userQueries:    queries.NewUserQueryHandler(userRepo),
		doctorCommands: commands.NewDoctorCommandHandler(doctorRepo),
		doctorQueries:  queries.NewDoctorQueryHandler(doctorRepo),
		authCommands:   commands.NewAuthCommandHandler(userRepo, auth),
	}
	return server, userRepo, doctorRepo
}

func TestCreateUser_RejectsFirebaseIDField(t *testing.T) {
	server, _, _ := newBugReproServer()

	body := strings.NewReader(`{
		"name":"Teste Auth 1",
		"email":"auth.doe@example.com",
		"phone":"+123456712890",
		"password":"S3cretP@ss",
		"firebase_id":"firebase-uid-supplied-by-client"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.CreateUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown firebase_id field, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateUser_AutoCreatesFirebaseUser(t *testing.T) {
	server, userRepo, _ := newBugReproServer()

	body := strings.NewReader(`{
		"name":"Teste Auth 1",
		"email":"auth.doe@example.com",
		"phone":"+123456712890",
		"password":"S3cretP@ss"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.CreateUser(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(userRepo.items) != 1 {
		t.Fatalf("expected 1 user saved, got %d", len(userRepo.items))
	}
	if userRepo.items[0].FirebaseID != "firebase-uid-from-admin-sdk" {
		t.Fatalf("expected firebase_id to come from Firebase Admin SDK, got %q", userRepo.items[0].FirebaseID)
	}

	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := raw["firebase_token"]; ok {
		t.Fatalf("firebase_token must NOT appear in create response, got: %s", rr.Body.String())
	}
	if raw["firebase_id"] != "firebase-uid-from-admin-sdk" {
		t.Fatalf("expected firebase_id in response, got: %s", rr.Body.String())
	}
}

func TestCreateUser_TwoCreatesDoNotCollideOnEmptyFirebaseID(t *testing.T) {
	// Reproduces the bug from Image 3: two consecutive creates previously
	// violated idx_users_firebase_id_unique because empty string was written
	// instead of NULL.
	server, userRepo, _ := newBugReproServer()

	for i, email := range []string{"a@example.com", "b@example.com"} {
		body := strings.NewReader(`{
			"name":"User ` + intToStr(i) + `",
			"email":"` + email + `",
			"phone":"+10000000` + intToStr(i) + `",
			"password":"S3cretP@ss"
		}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.CreateUser(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create #%d: expected 201, got %d: %s", i, rr.Code, rr.Body.String())
		}
	}
	if len(userRepo.items) != 2 {
		t.Fatalf("expected 2 users, got %d", len(userRepo.items))
	}
}

func TestCreateDoctor_RejectsFirebaseIDField(t *testing.T) {
	server, _, _ := newBugReproServer()

	body := strings.NewReader(`{
		"name":"Dr. X",
		"email":"x@example.com",
		"phone":"999",
		"password":"S3cretP@ss",
		"specialty":"Traumatologist",
		"license_number":"MED-999",
		"firebase_id":"unique-firebase-uid-10012005"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/doctors", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.CreateDoctor(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown firebase_id, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateDoctor_SucceedsWithoutPassword(t *testing.T) {
	server, _, doctorRepo := newBugReproServer()

	body := strings.NewReader(`{
		"name":"Dr. Jane Smith",
		"email":"jane.smith@hospital.com",
		"phone":"+1234567890",
		"specialty":"Cardiology",
		"license_number":"MED-12345"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/doctors", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.CreateDoctor(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(doctorRepo.items) != 1 {
		t.Fatalf("expected 1 doctor saved, got %d", len(doctorRepo.items))
	}
	if doctorRepo.items[0].FirebaseID != "" {
		t.Fatalf("expected no firebase_id (doctor is local-only), got %q", doctorRepo.items[0].FirebaseID)
	}
}

func TestListUsers_NeverReturnsFirebaseToken(t *testing.T) {
	server, userRepo, _ := newBugReproServer()
	userRepo.items = append(userRepo.items, &user.User{
		ID: "u1", Name: "Alice", Email: "a@example.com", Phone: "+1",
		FirebaseID: "firebase-uid-1",
		Role:       user.RoleElderly,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	server.ListUsers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var raw []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("expected 1 user, got %d", len(raw))
	}
	if _, ok := raw[0]["firebase_token"]; ok {
		t.Fatalf("firebase_token must NOT appear in GET /users, got: %s", rr.Body.String())
	}
	if raw[0]["firebase_id"] != "firebase-uid-1" {
		t.Fatalf("firebase_id must remain, got: %s", rr.Body.String())
	}
}

func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}