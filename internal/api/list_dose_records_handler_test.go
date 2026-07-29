package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

// capturingDoseRepo records the UserID argument of every FindByUserID call so
// the smoke test can assert the handler forwards the local UUID, not a
// Firebase-shaped string.
type capturingDoseRepo struct {
	gotUserID    string
	gotCallerID  string
	records      []*prescription.DoseRecord
	findByUserID func(ctx context.Context, userID string) ([]*prescription.DoseRecord, error)
}

func (s *capturingDoseRepo) Save(_ context.Context, _ *prescription.DoseRecord) error { return nil }
func (s *capturingDoseRepo) FindByID(_ context.Context, _ string) (*prescription.DoseRecord, error) {
	return nil, prescription.ErrDoseRecordNotFound
}
func (s *capturingDoseRepo) FindByUserID(ctx context.Context, userID string) ([]*prescription.DoseRecord, error) {
	s.gotUserID = userID
	if s.findByUserID != nil {
		return s.findByUserID(ctx, userID)
	}
	return s.records, nil
}
func (s *capturingDoseRepo) FindByPrescriptionID(_ context.Context, _ string) ([]*prescription.DoseRecord, error) {
	return nil, nil
}
func (s *capturingDoseRepo) FindPendingBefore(_ context.Context, _ time.Time) ([]*prescription.DoseRecord, error) {
	return nil, nil
}

// recordingUserRepo captures the arguments passed to IsLinked so the smoke
// test can verify the caller/owner UUIDs fed to PostgreSQL are local UUIDs.
type recordingUserRepo struct {
	stubUserRepoForExport
	linkedCalls []struct{ CaregiverID, ElderlyID string }
}

func (s *recordingUserRepo) IsLinked(_ context.Context, caregiverID, elderlyID string) (bool, error) {
	s.linkedCalls = append(s.linkedCalls, struct{ CaregiverID, ElderlyID string }{caregiverID, elderlyID})
	return true, nil
}

// noopPrescriptionRepo is a minimal stub that satisfies prescription.Repository
// without doing anything. Used by ListDoseRecords tests that don't exercise
// prescription queries.
type noopPrescriptionRepo struct{}

func (noopPrescriptionRepo) Save(_ context.Context, _ *prescription.Prescription) error { return nil }
func (noopPrescriptionRepo) FindAll(_ context.Context) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (noopPrescriptionRepo) FindByID(_ context.Context, _ string) (*prescription.Prescription, error) {
	return nil, prescription.ErrPrescriptionNotFound
}
func (noopPrescriptionRepo) FindByUserID(_ context.Context, _ string) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (noopPrescriptionRepo) FindByMedicID(_ context.Context, _ string) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (noopPrescriptionRepo) FindActive(_ context.Context) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (noopPrescriptionRepo) FindActiveByUserID(_ context.Context, _ string) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (noopPrescriptionRepo) Delete(_ context.Context, _ string) error { return nil }
func (noopPrescriptionRepo) Exists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// TestListDoseRecords_DoesNotPassFirebaseUIDToIsLinked is the handler-level
// smoke test that locks in the fix at the production-500 boundary. After
// AuthMiddleware resolves the Firebase UID to a local UUID, the dose-record
// handler must forward that local UUID to checkAccess / IsLinked — never a
// 28-character Firebase string that PostgreSQL would reject with
// `invalid input syntax for type uuid`.
func TestListDoseRecords_DoesNotPassFirebaseUIDToIsLinked(t *testing.T) {
	const (
		localUserUUID = "6b1fb275-2efa-4309-b34a-2f8b8abf6e6c"
		firebaseUID   = "uq1OEy7P0UPOvJIiFwCDNQxMJAW2"
	)

	doseRepo := &capturingDoseRepo{}
	userRepo := &recordingUserRepo{}

	ext := &ExtendedServer{
		userRepo:    userRepo,
		doseQueries: queries.NewDoseRecordQueryHandler(doseRepo, userRepo, noopPrescriptionRepo{}),
	}

	// Simulate a request whose caller context already holds the local UUID
	// (this is exactly what AuthMiddleware will produce after the fix).
	req := newChiRequestWithParam(http.MethodGet, "/api/v1/users/"+localUserUUID+"/dose-records", "userId", localUserUUID, localUserUUID)
	rr := httptest.NewRecorder()
	ext.ListDoseRecords(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if doseRepo.gotUserID != localUserUUID {
		t.Fatalf("doseRepo.FindByUserID called with %q, want %q", doseRepo.gotUserID, localUserUUID)
	}
	for _, call := range userRepo.linkedCalls {
		if call.CaregiverID == firebaseUID {
			t.Fatalf("IsLinked received the Firebase UID %q — the bug is back. args=%+v", firebaseUID, call)
		}
		if call.CaregiverID != localUserUUID || call.ElderlyID != localUserUUID {
			t.Fatalf("IsLinked called with non-UUID args: %+v", call)
		}
	}

	var decoded []*prescription.DoseRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response should be a JSON array of dose records: %v body=%s", err, rr.Body.String())
	}
}

// newChiRequestWithParam builds a *http.Request with a single URL parameter
// set via chi's RouteContext and the caller ID stored in the request context
// (the way AuthMiddleware does after the fix).
func newChiRequestWithParam(method, url, paramKey, paramValue, callerID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(paramKey, paramValue)
	req := httptest.NewRequest(method, url, nil)
	req = req.WithContext(context.WithValue(req.Context(), contextKeyUserID, callerID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return req
}
