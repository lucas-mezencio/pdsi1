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
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// --- minimal mocks reused for HTTP-level LGPD tests ---
// (mirrors the patterns in lgpd_queries_test.go; small surface, hand-written)

type stubUserRepoForExport struct {
	user *user.User
}

func (s *stubUserRepoForExport) Save(ctx context.Context, u *user.User) error { return nil }
func (s *stubUserRepoForExport) FindByID(ctx context.Context, id string) (*user.User, error) {
	if s.user != nil && id == s.user.ID {
		return s.user, nil
	}
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepoForExport) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepoForExport) FindByFirebaseID(ctx context.Context, firebaseID string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepoForExport) FindAll(ctx context.Context) ([]*user.User, error) {
	return nil, nil
}
func (s *stubUserRepoForExport) Delete(ctx context.Context, id string) error { return nil }
func (s *stubUserRepoForExport) Exists(ctx context.Context, id string) (bool, error) {
	return s.user != nil && id == s.user.ID, nil
}
func (s *stubUserRepoForExport) FindCaregivers(ctx context.Context, elderlyID string) ([]*user.User, error) {
	return nil, nil
}
func (s *stubUserRepoForExport) FindCharges(ctx context.Context, caregiverID string) ([]*user.User, error) {
	return nil, nil
}
func (s *stubUserRepoForExport) IsLinked(ctx context.Context, caregiverID, elderlyID string) (bool, error) {
	return false, nil
}
func (s *stubUserRepoForExport) LinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}
func (s *stubUserRepoForExport) UnlinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}

type stubPrescriptionRepoForExport struct {
	items []*prescription.Prescription
}

func (s *stubPrescriptionRepoForExport) Save(ctx context.Context, p *prescription.Prescription) error {
	return nil
}
func (s *stubPrescriptionRepoForExport) FindByID(ctx context.Context, id string) (*prescription.Prescription, error) {
	return nil, prescription.ErrPrescriptionNotFound
}
func (s *stubPrescriptionRepoForExport) FindByUserID(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
	return s.items, nil
}
func (s *stubPrescriptionRepoForExport) FindByMedicID(ctx context.Context, medicID string) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepoForExport) FindActive(ctx context.Context) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepoForExport) FindActiveByUserID(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepoForExport) FindAll(ctx context.Context) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepoForExport) Delete(ctx context.Context, id string) error { return nil }
func (s *stubPrescriptionRepoForExport) Exists(ctx context.Context, id string) (bool, error) {
	return false, nil
}

type stubDoseRepoForExport struct {
	items []*prescription.DoseRecord
}

func (s *stubDoseRepoForExport) Save(ctx context.Context, r *prescription.DoseRecord) error { return nil }
func (s *stubDoseRepoForExport) FindByID(ctx context.Context, id string) (*prescription.DoseRecord, error) {
	return nil, prescription.ErrDoseRecordNotFound
}
func (s *stubDoseRepoForExport) FindByUserID(ctx context.Context, userID string) ([]*prescription.DoseRecord, error) {
	return s.items, nil
}
func (s *stubDoseRepoForExport) FindByPrescriptionID(ctx context.Context, prescriptionID string) ([]*prescription.DoseRecord, error) {
	return nil, nil
}
func (s *stubDoseRepoForExport) FindPendingBefore(ctx context.Context, before time.Time) ([]*prescription.DoseRecord, error) {
	return nil, nil
}

type stubInvitationRepoForExport struct{}

func (s *stubInvitationRepoForExport) Save(ctx context.Context, inv *user.CaregiverInvitation) error {
	return nil
}
func (s *stubInvitationRepoForExport) FindByToken(ctx context.Context, token string) (*user.CaregiverInvitation, error) {
	return nil, user.ErrInvitationNotFound
}
func (s *stubInvitationRepoForExport) FindByElderlyID(ctx context.Context, elderlyID string) ([]*user.CaregiverInvitation, error) {
	return nil, nil
}
func (s *stubInvitationRepoForExport) FindByCaregiverID(ctx context.Context, caregiverID string) ([]*user.CaregiverInvitation, error) {
	return nil, nil
}

// newExtendedServerWithStubExport builds an ExtendedServer with stubs that
// return the given user / prescriptions / dose records from the export.
func newExtendedServerWithStubExport(
	stubUser *user.User,
	stubPrescriptions []*prescription.Prescription,
	stubDoses []*prescription.DoseRecord,
) *ExtendedServer {
	return &ExtendedServer{
		userRepo:          &stubUserRepoForExport{user: stubUser},
		authCommands:      commands.NewAuthCommandHandler(&stubUserRepoForExport{user: stubUser}, nil),
		lgpdQueries: queries.NewLGPDQueryHandler(
			&stubUserRepoForExport{user: stubUser},
			&stubPrescriptionRepoForExport{items: stubPrescriptions},
			&stubDoseRepoForExport{items: stubDoses},
			&stubInvitationRepoForExport{},
		),
	}
}

// callWithCaller invokes the handler with an authenticated caller ID set in
// the context (simulating AuthMiddleware behavior).
func callWithCaller(t *testing.T, ext *ExtendedServer, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/data-export", nil)
	ctx := context.WithValue(req.Context(), contextKeyUserID, callerID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	ext.LGPDDataExport(rr, req)
	return rr
}

func TestExtendedServer_LGPDDataExport_Returns200WithAllSections(t *testing.T) {
	now := time.Now()
	usr := &user.User{
		ID: "user-1", Name: "Maria", Email: "maria@example.com",
		CPF: "52998224725", Phone: "+55119...",
		Role: user.RoleElderly,
	}
	rxs := []*prescription.Prescription{{
		ID:     "rx-1",
		UserID: "user-1",
		Medicaments: []prescription.Medicament{{
			Name: "AAS", Dosage: "100mg", Frequency: "24:00",
			Times: []string{"08:00"}, Doses: 30,
		}},
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}}
	doses := []*prescription.DoseRecord{{
		ID: "dose-1", PrescriptionID: "rx-1", UserID: "user-1",
		MedicamentName: "AAS", Dosage: "100mg",
		ScheduledAt: now, Status: prescription.DoseStatusPending,
	}}

	ext := newExtendedServerWithStubExport(usr, rxs, doses)
	rr := callWithCaller(t, ext, "user-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. body: %s", rr.Code, rr.Body.String())
	}

	var export queries.UserDataExport
	if err := json.Unmarshal(rr.Body.Bytes(), &export); err != nil {
		t.Fatalf("decode response: %v. body=%s", err, rr.Body.String())
	}

	if export.User == nil || export.User.ID != "user-1" {
		t.Fatalf("expected user.id=user-1, got %+v", export.User)
	}
	if export.User.CPF != "52998224725" {
		t.Errorf("expected user.cpf=52998224725, got %q", export.User.CPF)
	}

	// === the LGPD-critical assertion: sensitive health data is included ===
	if len(export.Prescriptions) != 1 || export.Prescriptions[0].ID != "rx-1" {
		t.Fatalf("expected 1 prescription rx-1 (sensitive health data), got %+v", export.Prescriptions)
	}
	if len(export.Prescriptions[0].Medicaments) != 1 || export.Prescriptions[0].Medicaments[0].Name != "AAS" {
		t.Fatalf("expected medicaments to be included, got %+v", export.Prescriptions[0].Medicaments)
	}
	if len(export.DoseRecords) != 1 || export.DoseRecords[0].ID != "dose-1" {
		t.Fatalf("expected 1 dose record dose-1 (sensitive health data), got %+v", export.DoseRecords)
	}
}

func TestExtendedServer_LGPDDataExport_RejectsAnonymous(t *testing.T) {
	usr := &user.User{ID: "user-1"}
	ext := newExtendedServerWithStubExport(usr, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/data-export", nil)
	rr := httptest.NewRecorder()
	ext.LGPDDataExport(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestExtendedServer_LGPDDataExport_OnlyReturnsCallersOwnData(t *testing.T) {
	// If a different user ID is somehow passed (e.g. via header), the LGPD
	// handler must still query by the caller's ID — not the URL or query.
	stored := &user.User{ID: "real-user-1", Name: "Real", CPF: "52998224725"}
	rxs := []*prescription.Prescription{{ID: "rx-real", UserID: "real-user-1", Active: true}}
	doses := []*prescription.DoseRecord{{ID: "dose-real", UserID: "real-user-1", Status: prescription.DoseStatusTaken}}

	ext := newExtendedServerWithStubExport(stored, rxs, doses)
	rr := callWithCaller(t, ext, "real-user-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var export queries.UserDataExport
	if err := json.Unmarshal(rr.Body.Bytes(), &export); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if export.User == nil || export.User.ID != "real-user-1" {
		t.Fatalf("expected caller-only data export, got user %+v", export.User)
	}
}

// Push-delivery tokens live in user_device_tokens and never appear on the
// User shape returned by the LGPD data-export endpoint. The handler must
// not surface any field named like an FCM token on User-shaped objects
// nested in the export body.
func TestExtendedServer_LGPDDataExport_NeverReturnsDeviceToken(t *testing.T) {
	now := time.Now()
	usr := &user.User{
		ID:    "user-1",
		Name:  "Maria",
		Email: "maria@example.com",
	}
	rxs := []*prescription.Prescription{{
		ID: "rx-1", UserID: "user-1", Active: true,
		Medicaments: []prescription.Medicament{{
			Name: "AAS", Dosage: "100mg", Frequency: "24:00",
			Times: []string{"08:00"}, Doses: 30,
		}},
		CreatedAt: now, UpdatedAt: now,
	}}
	doses := []*prescription.DoseRecord{{
		ID: "dose-1", PrescriptionID: "rx-1", UserID: "user-1",
		ScheduledAt: now, Status: prescription.DoseStatusPending,
	}}
	ext := newExtendedServerWithStubExport(usr, rxs, doses)

	rr := callWithCaller(t, ext, "user-1")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	for _, key := range []string{"firebase_token", "fcm_token"} {
		if strings.Contains(body, key) {
			t.Fatalf("%q must NOT appear in LGPD data-export body, got: %s", key, body)
		}
	}
}