package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

type stubPrescriptionRepo struct {
	active []*prescription.Prescription
}

func (s *stubPrescriptionRepo) Save(_ context.Context, _ *prescription.Prescription) error {
	return nil
}
func (s *stubPrescriptionRepo) FindAll(_ context.Context) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) FindByID(_ context.Context, _ string) (*prescription.Prescription, error) {
	return nil, prescription.ErrPrescriptionNotFound
}
func (s *stubPrescriptionRepo) FindByUserID(_ context.Context, _ string) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) FindByMedicID(_ context.Context, _ string) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) FindActive(_ context.Context) ([]*prescription.Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) FindActiveByUserID(_ context.Context, userID string) ([]*prescription.Prescription, error) {
	if userID == "user-1" {
		return s.active, nil
	}
	return nil, nil
}
func (s *stubPrescriptionRepo) Delete(_ context.Context, _ string) error { return nil }
func (s *stubPrescriptionRepo) Exists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

type stubDoseRepo struct {
	records []*prescription.DoseRecord
}

func (s *stubDoseRepo) Save(_ context.Context, _ *prescription.DoseRecord) error {
	return nil
}
func (s *stubDoseRepo) FindByID(_ context.Context, _ string) (*prescription.DoseRecord, error) {
	return nil, prescription.ErrDoseRecordNotFound
}
func (s *stubDoseRepo) FindByUserID(_ context.Context, userID string) ([]*prescription.DoseRecord, error) {
	if userID == "user-1" {
		return s.records, nil
	}
	return nil, nil
}
func (s *stubDoseRepo) FindByPrescriptionID(_ context.Context, _ string) ([]*prescription.DoseRecord, error) {
	return nil, nil
}
func (s *stubDoseRepo) FindPendingBefore(_ context.Context, _ time.Time) ([]*prescription.DoseRecord, error) {
	return nil, nil
}

type allowAllUserRepo struct{}

func (allowAllUserRepo) FindByID(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (allowAllUserRepo) FindByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (allowAllUserRepo) Save(_ context.Context, _ *user.User) error { return nil }
func (allowAllUserRepo) Delete(_ context.Context, _ string) error  { return nil }
func (allowAllUserRepo) FindByFirebaseID(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (allowAllUserRepo) FindAll(_ context.Context) ([]*user.User, error) { return nil, nil }
func (allowAllUserRepo) Exists(_ context.Context, _ string) (bool, error) { return false, nil }
func (allowAllUserRepo) IsLinked(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
func (allowAllUserRepo) FindCaregivers(_ context.Context, _ string) ([]*user.User, error) {
	return nil, nil
}
func (allowAllUserRepo) FindCharges(_ context.Context, _ string) ([]*user.User, error) {
	return nil, nil
}
func (allowAllUserRepo) LinkUsers(_ context.Context, _, _ string) error    { return nil }
func (allowAllUserRepo) UnlinkUsers(_ context.Context, _, _ string) error  { return nil }

func TestListScheduleForUser_ReconstructsPendingSlots(t *testing.T) {
	futureTime := time.Now().In(prescription.BrazilLocation).Add(2 * time.Hour)
	scheduleStr := futureTime.Format("15:04")
	p := &prescription.Prescription{
		ID:        "pres-1",
		UserID:    "user-1",
		MedicID:   "doc-1",
		Active:    true,
		CreatedAt: futureTime.Add(-1 * time.Hour),
		Medicaments: []prescription.Medicament{
			{Name: "AAS", Dosage: "100mg", Frequency: "24:00", Times: []string{scheduleStr}, Doses: 2},
		},
	}

	h := NewDoseRecordQueryHandler(&stubDoseRepo{}, allowAllUserRepo{}, &stubPrescriptionRepo{active: []*prescription.Prescription{p}})
	got, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "user-1", CallerID: "user-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(got))
	}
	for i, slot := range got {
		if slot.Status != prescription.DoseStatusPending {
			t.Errorf("slot %d status = %q, want PENDING", i, slot.Status)
		}
		if slot.DoseRecordID != nil {
			t.Errorf("slot %d dose_record_id = %v, want nil", i, *slot.DoseRecordID)
		}
	}
}

func TestListScheduleForUser_OverlaysTakenRecord(t *testing.T) {
	futureTime := time.Now().In(prescription.BrazilLocation).Add(2 * time.Hour)
	scheduleStr := futureTime.Format("15:04")
	createdAt := futureTime.Add(-1 * time.Hour)
	p := &prescription.Prescription{
		ID:        "pres-1",
		UserID:    "user-1",
		MedicID:   "doc-1",
		Active:    true,
		CreatedAt: createdAt,
		Medicaments: []prescription.Medicament{
			{Name: "AAS", Dosage: "100mg", Frequency: "24:00", Times: []string{scheduleStr}, Doses: 1},
		},
	}
	confirmed := futureTime.Add(5 * time.Minute)
	rec := &prescription.DoseRecord{
		ID:             "rec-1",
		PrescriptionID: "pres-1",
		UserID:         "user-1",
		MedicamentName: "AAS",
		Dosage:         "100mg",
		ScheduledAt:    futureTime,
		Status:         prescription.DoseStatusTaken,
		ConfirmedAt:    &confirmed,
	}
	h := NewDoseRecordQueryHandler(&stubDoseRepo{records: []*prescription.DoseRecord{rec}}, allowAllUserRepo{}, &stubPrescriptionRepo{active: []*prescription.Prescription{p}})

	got, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "user-1", CallerID: "user-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(got))
	}
	if got[0].Status != prescription.DoseStatusTaken {
		t.Errorf("status = %q, want TAKEN", got[0].Status)
	}
	if got[0].DoseRecordID == nil || *got[0].DoseRecordID != "rec-1" {
		t.Errorf("dose_record_id = %v, want rec-1", got[0].DoseRecordID)
	}
	if got[0].ConfirmedAt == nil {
		t.Errorf("confirmed_at should be set for TAKEN slot")
	}
}

func TestListScheduleForUser_PreservesOrphanRecord(t *testing.T) {
	takenAt := time.Now().Add(-30 * time.Minute)
	rec := &prescription.DoseRecord{
		ID: "rec-orphan", PrescriptionID: "pres-gone", UserID: "user-1",
		MedicamentName: "OldDrug", Dosage: "5mg",
		ScheduledAt: takenAt, Status: prescription.DoseStatusTaken,
	}
	h := NewDoseRecordQueryHandler(&stubDoseRepo{records: []*prescription.DoseRecord{rec}}, allowAllUserRepo{}, &stubPrescriptionRepo{})

	got, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "user-1", CallerID: "user-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 orphan slot, got %d", len(got))
	}
	if got[0].DoseRecordID == nil || *got[0].DoseRecordID != "rec-orphan" {
		t.Errorf("dose_record_id = %v, want rec-orphan", got[0].DoseRecordID)
	}
	if got[0].MedicamentName != "OldDrug" {
		t.Errorf("medicament_name = %q, want OldDrug", got[0].MedicamentName)
	}
}

func TestListScheduleForUser_RejectsEmptyUserID(t *testing.T) {
	h := NewDoseRecordQueryHandler(&stubDoseRepo{}, allowAllUserRepo{}, &stubPrescriptionRepo{})
	_, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "", CallerID: ""})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}
}

type restrictiveUserRepo struct{ allowAllUserRepo }

func (restrictiveUserRepo) IsLinked(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func TestListScheduleForUser_ForbiddenForUnlinkedCaller(t *testing.T) {
	h := NewDoseRecordQueryHandler(&stubDoseRepo{}, &restrictiveUserRepo{}, &stubPrescriptionRepo{})
	_, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "user-1", CallerID: "user-2"})
	if !errors.Is(err, application.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}