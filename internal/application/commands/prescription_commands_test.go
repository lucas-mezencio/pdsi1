package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

type mockPrescriptionRepo struct {
	saveFn               func(ctx context.Context, entity *prescription.Prescription) error
	findByIDFn           func(ctx context.Context, id string) (*prescription.Prescription, error)
	findByUserIDFn       func(ctx context.Context, userID string) ([]*prescription.Prescription, error)
	findByMedicIDFn      func(ctx context.Context, medicID string) ([]*prescription.Prescription, error)
	findActiveFn         func(ctx context.Context) ([]*prescription.Prescription, error)
	findActiveByUserIDFn func(ctx context.Context, userID string) ([]*prescription.Prescription, error)
	findAllFn            func(ctx context.Context) ([]*prescription.Prescription, error)
	deleteFn             func(ctx context.Context, id string) error
	existsFn             func(ctx context.Context, id string) (bool, error)
}

func (m *mockPrescriptionRepo) Save(ctx context.Context, entity *prescription.Prescription) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, entity)
	}
	return nil
}

func (m *mockPrescriptionRepo) FindByID(ctx context.Context, id string) (*prescription.Prescription, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, prescription.ErrPrescriptionNotFound
}

func (m *mockPrescriptionRepo) FindByUserID(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID)
	}
	return []*prescription.Prescription{}, nil
}

func (m *mockPrescriptionRepo) FindByMedicID(ctx context.Context, medicID string) ([]*prescription.Prescription, error) {
	if m.findByMedicIDFn != nil {
		return m.findByMedicIDFn(ctx, medicID)
	}
	return []*prescription.Prescription{}, nil
}

func (m *mockPrescriptionRepo) FindActive(ctx context.Context) ([]*prescription.Prescription, error) {
	if m.findActiveFn != nil {
		return m.findActiveFn(ctx)
	}
	return []*prescription.Prescription{}, nil
}

func (m *mockPrescriptionRepo) FindActiveByUserID(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
	if m.findActiveByUserIDFn != nil {
		return m.findActiveByUserIDFn(ctx, userID)
	}
	return []*prescription.Prescription{}, nil
}

func (m *mockPrescriptionRepo) FindAll(ctx context.Context) ([]*prescription.Prescription, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx)
	}
	return []*prescription.Prescription{}, nil
}

func (m *mockPrescriptionRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockPrescriptionRepo) Exists(ctx context.Context, id string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, id)
	}
	return false, nil
}

type mockScheduler struct {
	scheduled []prescription.NotificationSchedule
	canceled  []string
	fail      bool
}

func (m *mockScheduler) Schedule(ctx context.Context, schedule prescription.NotificationSchedule, startDate time.Time) error {
	if m.fail {
		return errors.New("schedule failed")
	}
	m.scheduled = append(m.scheduled, schedule)
	return nil
}

func (m *mockScheduler) CancelByPrescriptionID(ctx context.Context, prescriptionID string) error {
	if m.fail {
		return errors.New("cancel failed")
	}
	m.canceled = append(m.canceled, prescriptionID)
	return nil
}

type mockExistsRepo struct {
	exists bool
	err    error
}

func (m *mockExistsRepo) Exists(ctx context.Context, id string) (bool, error) {
	return m.exists, m.err
}

func (m *mockExistsRepo) Save(ctx context.Context, entity *user.User) error { return nil }
func (m *mockExistsRepo) FindByID(ctx context.Context, id string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (m *mockExistsRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (m *mockExistsRepo) FindAll(ctx context.Context) ([]*user.User, error) { return nil, nil }
func (m *mockExistsRepo) Delete(ctx context.Context, id string) error       { return nil }

type mockDoctorExistsRepo struct {
	exists bool
	err    error
}

func (m *mockDoctorExistsRepo) Exists(ctx context.Context, id string) (bool, error) {
	return m.exists, m.err
}

func (m *mockDoctorExistsRepo) Save(ctx context.Context, entity *doctor.Doctor) error { return nil }
func (m *mockDoctorExistsRepo) FindByID(ctx context.Context, id string) (*doctor.Doctor, error) {
	return nil, doctor.ErrDoctorNotFound
}
func (m *mockDoctorExistsRepo) FindByEmail(ctx context.Context, email string) (*doctor.Doctor, error) {
	return nil, doctor.ErrDoctorNotFound
}
func (m *mockDoctorExistsRepo) FindByLicenseNumber(ctx context.Context, licenseNumber string) (*doctor.Doctor, error) {
	return nil, doctor.ErrDoctorNotFound
}
func (m *mockDoctorExistsRepo) FindAll(ctx context.Context) ([]*doctor.Doctor, error) {
	return nil, nil
}
func (m *mockDoctorExistsRepo) Delete(ctx context.Context, id string) error { return nil }

func TestPrescriptionCommandHandler_Create_InvalidInput(t *testing.T) {
	handler := NewPrescriptionCommandHandler(&mockPrescriptionRepo{}, &mockExistsRepo{}, &mockDoctorExistsRepo{}, &mockScheduler{})
	_, err := handler.Create(context.Background(), CreatePrescriptionCommand{})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestPrescriptionCommandHandler_Create_UserNotFound(t *testing.T) {
	userRepo := &mockExistsRepo{exists: false}
	doctorRepo := &mockDoctorExistsRepo{exists: true}
	handler := NewPrescriptionCommandHandler(&mockPrescriptionRepo{}, userRepo, doctorRepo, &mockScheduler{})

	_, err := handler.Create(context.Background(), CreatePrescriptionCommand{
		UserID:  "user-1",
		MedicID: "doc-1",
		Medicaments: []prescription.Medicament{{
			Name:      "Med",
			Dosage:    "1",
			Frequency: "24:00",
			Times:     []string{"08:00"},
			Doses:     1,
		}},
	})

	if !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("expected user not found error, got %v", err)
	}
}

func TestPrescriptionCommandHandler_Create_DoctorNotFound(t *testing.T) {
	userRepo := &mockExistsRepo{exists: true}
	doctorRepo := &mockDoctorExistsRepo{exists: false}
	handler := NewPrescriptionCommandHandler(&mockPrescriptionRepo{}, userRepo, doctorRepo, &mockScheduler{})

	_, err := handler.Create(context.Background(), CreatePrescriptionCommand{
		UserID:  "user-1",
		MedicID: "doc-1",
		Medicaments: []prescription.Medicament{{
			Name:      "Med",
			Dosage:    "1",
			Frequency: "24:00",
			Times:     []string{"08:00"},
			Doses:     1,
		}},
	})

	if !errors.Is(err, application.ErrDoctorNotFound) {
		t.Fatalf("expected doctor not found error, got %v", err)
	}
}

func TestPrescriptionCommandHandler_Create_SchedulesNotifications(t *testing.T) {
	userRepo := &mockExistsRepo{exists: true}
	doctorRepo := &mockDoctorExistsRepo{exists: true}
	scheduler := &mockScheduler{}

	var saved *prescription.Prescription
	repo := &mockPrescriptionRepo{
		saveFn: func(ctx context.Context, entity *prescription.Prescription) error {
			saved = entity
			return nil
		},
	}

	handler := NewPrescriptionCommandHandler(repo, userRepo, doctorRepo, scheduler)
	created, err := handler.Create(context.Background(), CreatePrescriptionCommand{
		UserID:  "user-1",
		MedicID: "doc-1",
		Medicaments: []prescription.Medicament{{
			Name:      "Med",
			Dosage:    "1",
			Frequency: "12:00",
			Times:     []string{"08:00", "20:00"},
			Doses:     2,
		}},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if saved == nil || saved != created {
		t.Fatal("expected prescription saved")
	}
	if len(scheduler.scheduled) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(scheduler.scheduled))
	}
}

func TestPrescriptionCommandHandler_UpdateMedicaments(t *testing.T) {
	entity := &prescription.Prescription{ID: "rx-1", UserID: "user-1", MedicID: "doc-1", Active: true}
	entity.Medicaments = []prescription.Medicament{{
		Name:      "Old",
		Dosage:    "1",
		Frequency: "24:00",
		Times:     []string{"08:00"},
		Doses:     1,
	}}

	scheduler := &mockScheduler{}
	repo := &mockPrescriptionRepo{
		findByIDFn: func(ctx context.Context, id string) (*prescription.Prescription, error) {
			return entity, nil
		},
	}

	handler := NewPrescriptionCommandHandler(repo, &mockExistsRepo{exists: true}, &mockDoctorExistsRepo{exists: true}, scheduler)
	updated, err := handler.UpdateMedicaments(context.Background(), UpdatePrescriptionCommand{
		ID: "rx-1",
		Medicaments: []prescription.Medicament{{
			Name:      "New",
			Dosage:    "1",
			Frequency: "12:00",
			Times:     []string{"08:00", "20:00"},
			Doses:     2,
		}},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Medicaments[0].Name != "New" {
		t.Fatalf("expected medicament updated, got %s", updated.Medicaments[0].Name)
	}
	if len(scheduler.canceled) != 1 {
		t.Fatal("expected schedules canceled")
	}
	if len(scheduler.scheduled) != 2 {
		t.Fatalf("expected schedules recreated, got %d", len(scheduler.scheduled))
	}
}

func TestPrescriptionCommandHandler_Deactivate(t *testing.T) {
	entity := &prescription.Prescription{ID: "rx-1", Active: true}
	repo := &mockPrescriptionRepo{
		findByIDFn: func(ctx context.Context, id string) (*prescription.Prescription, error) {
			return entity, nil
		},
	}
	scheduler := &mockScheduler{}

	handler := NewPrescriptionCommandHandler(repo, &mockExistsRepo{exists: true}, &mockDoctorExistsRepo{exists: true}, scheduler)
	updated, err := handler.Deactivate(context.Background(), DeactivatePrescriptionCommand{ID: "rx-1"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Active {
		t.Fatal("expected prescription deactivated")
	}
	if len(scheduler.canceled) != 1 {
		t.Fatal("expected notifications canceled")
	}
}

func TestPrescriptionCommandHandler_Activate(t *testing.T) {
	entity := &prescription.Prescription{ID: "rx-1", Active: false}
	entity.Medicaments = []prescription.Medicament{{
		Name:      "Med",
		Dosage:    "1",
		Frequency: "24:00",
		Times:     []string{"08:00"},
		Doses:     1,
	}}
	repo := &mockPrescriptionRepo{
		findByIDFn: func(ctx context.Context, id string) (*prescription.Prescription, error) {
			return entity, nil
		},
	}
	scheduler := &mockScheduler{}

	handler := NewPrescriptionCommandHandler(repo, &mockExistsRepo{exists: true}, &mockDoctorExistsRepo{exists: true}, scheduler)
	updated, err := handler.Activate(context.Background(), ActivatePrescriptionCommand{ID: "rx-1"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !updated.Active {
		t.Fatal("expected prescription active")
	}
	if len(scheduler.scheduled) == 0 {
		t.Fatal("expected notifications scheduled")
	}
}

func TestPrescriptionCommandHandler_Delete(t *testing.T) {
	deleted := false
	repo := &mockPrescriptionRepo{
		existsFn: func(ctx context.Context, id string) (bool, error) {
			return true, nil
		},
		deleteFn: func(ctx context.Context, id string) error {
			deleted = true
			return nil
		},
	}
	scheduler := &mockScheduler{}

	handler := NewPrescriptionCommandHandler(repo, &mockExistsRepo{exists: true}, &mockDoctorExistsRepo{exists: true}, scheduler)
	if err := handler.Delete(context.Background(), DeletePrescriptionCommand{ID: "rx-1"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleted {
		t.Fatal("expected delete to be called")
	}
	if len(scheduler.canceled) != 1 {
		t.Fatal("expected notifications canceled")
	}
}
