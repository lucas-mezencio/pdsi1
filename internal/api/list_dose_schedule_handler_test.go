package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

// capturingSchedulePrescriptionRepo captures the UserID argument passed
// to FindActiveByUserID so the smoke test can assert the handler forwards
// the local UUID (post-AuthMiddleware), not the Firebase UID.
type capturingSchedulePrescriptionRepo struct {
	noopPrescriptionRepo
	gotUserID string
}

func (s *capturingSchedulePrescriptionRepo) FindActiveByUserID(_ context.Context, userID string) ([]*prescription.Prescription, error) {
	s.gotUserID = userID
	return nil, nil
}

// capturingScheduleDoseRepo captures the UserID argument passed to
// FindByUserID for the same UUID-invariant reason.
type capturingScheduleDoseRepo struct {
	gotUserID string
}

func (s *capturingScheduleDoseRepo) Save(_ context.Context, _ *prescription.DoseRecord) error {
	return nil
}
func (s *capturingScheduleDoseRepo) FindByID(_ context.Context, _ string) (*prescription.DoseRecord, error) {
	return nil, prescription.ErrDoseRecordNotFound
}
func (s *capturingScheduleDoseRepo) FindByUserID(_ context.Context, userID string) ([]*prescription.DoseRecord, error) {
	s.gotUserID = userID
	return nil, nil
}
func (s *capturingScheduleDoseRepo) FindByPrescriptionID(_ context.Context, _ string) ([]*prescription.DoseRecord, error) {
	return nil, nil
}
func (s *capturingScheduleDoseRepo) FindPendingBefore(_ context.Context, _ time.Time) ([]*prescription.DoseRecord, error) {
	return nil, nil
}

// TestListDoseSchedule_DoesNotPassFirebaseUIDToIsLinked locks in the
// same local-UUID invariant the existing list-dose-records smoke test
// enforces (internal/api/list_dose_records_handler_test.go) for the new
// /users/{userId}/doses endpoint.
func TestListDoseSchedule_DoesNotPassFirebaseUIDToIsLinked(t *testing.T) {
	const (
		localUserUUID = "6b1fb275-2efa-4309-b34a-2f8b8abf6e6c"
		firebaseUID   = "uq1OEy7P0UPOvJIiFwCDNQxMJAW2"
	)

	doseRepo := &capturingScheduleDoseRepo{}
	userRepo := &recordingUserRepo{}
	prescriptionRepo := &capturingSchedulePrescriptionRepo{}

	ext := &ExtendedServer{
		userRepo:    userRepo,
		doseQueries: queries.NewDoseRecordQueryHandler(doseRepo, userRepo, prescriptionRepo),
	}

	req := newChiRequestWithParam(http.MethodGet, "/api/v1/users/"+localUserUUID+"/doses", "userId", localUserUUID, localUserUUID)
	rr := httptest.NewRecorder()
	ext.ListDoseSchedule(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if doseRepo.gotUserID != localUserUUID {
		t.Fatalf("doseRepo.FindByUserID called with %q, want %q", doseRepo.gotUserID, localUserUUID)
	}
	if prescriptionRepo.gotUserID != localUserUUID {
		t.Fatalf("prescriptionRepo.FindActiveByUserID called with %q, want %q", prescriptionRepo.gotUserID, localUserUUID)
	}
	for _, call := range userRepo.linkedCalls {
		if call.CaregiverID == firebaseUID {
			t.Fatalf("IsLinked received the Firebase UID %q — bug regressed. args=%+v", firebaseUID, call)
		}
		if call.CaregiverID != localUserUUID || call.ElderlyID != localUserUUID {
			t.Fatalf("IsLinked called with non-UUID args: %+v", call)
		}
	}

	var decoded []*prescription.ScheduledDose
	if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response should be a JSON array of ScheduledDose: %v body=%s", err, rr.Body.String())
	}
}
