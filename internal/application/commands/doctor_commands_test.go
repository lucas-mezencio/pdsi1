package commands

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

type mockDoctorRepo struct {
	saveFn             func(ctx context.Context, doc *doctor.Doctor) error
	findByIDFn         func(ctx context.Context, id string) (*doctor.Doctor, error)
	findByEmailFn      func(ctx context.Context, email string) (*doctor.Doctor, error)
	findByFirebaseIDFn func(ctx context.Context, firebaseID string) (*doctor.Doctor, error)
	findByLicenseFn    func(ctx context.Context, license string) (*doctor.Doctor, error)
	findAllFn          func(ctx context.Context) ([]*doctor.Doctor, error)
	deleteFn           func(ctx context.Context, id string) error
	existsFn           func(ctx context.Context, id string) (bool, error)
}

func (m *mockDoctorRepo) Save(ctx context.Context, entity *doctor.Doctor) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, entity)
	}
	return nil
}

func (m *mockDoctorRepo) FindByID(ctx context.Context, id string) (*doctor.Doctor, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, doctor.ErrDoctorNotFound
}

func (m *mockDoctorRepo) FindByEmail(ctx context.Context, email string) (*doctor.Doctor, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, doctor.ErrDoctorNotFound
}

func (m *mockDoctorRepo) FindByFirebaseID(ctx context.Context, firebaseID string) (*doctor.Doctor, error) {
	if m.findByFirebaseIDFn != nil {
		return m.findByFirebaseIDFn(ctx, firebaseID)
	}
	return nil, doctor.ErrDoctorNotFound
}

func (m *mockDoctorRepo) FindByLicenseNumber(ctx context.Context, licenseNumber string) (*doctor.Doctor, error) {
	if m.findByLicenseFn != nil {
		return m.findByLicenseFn(ctx, licenseNumber)
	}
	return nil, doctor.ErrDoctorNotFound
}

func (m *mockDoctorRepo) FindAll(ctx context.Context) ([]*doctor.Doctor, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx)
	}
	return []*doctor.Doctor{}, nil
}

func (m *mockDoctorRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockDoctorRepo) Exists(ctx context.Context, id string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, id)
	}
	return false, nil
}

func TestDoctorCommandHandler_Create(t *testing.T) {
	repo := &mockDoctorRepo{}
	var saved *doctor.Doctor
	repo.saveFn = func(ctx context.Context, entity *doctor.Doctor) error {
		saved = entity
		return nil
	}

	handler := NewDoctorCommandHandler(repo)
	created, err := handler.Create(context.Background(), CreateDoctorCommand{
		Name:          "Dr. Who",
		Email:         "who@example.com",
		Phone:         "999",
		Specialty:     "Time",
		LicenseNumber: "LIC-1",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created == nil {
		t.Fatal("expected doctor to be created")
	}
	if saved == nil {
		t.Fatal("expected doctor to be saved")
	}
}

func TestDoctorCommandHandler_Update_InvalidInput(t *testing.T) {
	handler := NewDoctorCommandHandler(&mockDoctorRepo{})
	_, err := handler.Update(context.Background(), UpdateDoctorCommand{})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestDoctorCommandHandler_Update_NotFound(t *testing.T) {
	repo := &mockDoctorRepo{
		findByIDFn: func(ctx context.Context, id string) (*doctor.Doctor, error) {
			return nil, doctor.ErrDoctorNotFound
		},
	}

	handler := NewDoctorCommandHandler(repo)
	_, err := handler.Update(context.Background(), UpdateDoctorCommand{ID: "missing"})
	if !errors.Is(err, application.ErrDoctorNotFound) {
		t.Fatalf("expected doctor not found error, got %v", err)
	}
}

func TestDoctorCommandHandler_Update_Success(t *testing.T) {
	entity := &doctor.Doctor{ID: "doc-1", Name: "Old", Email: "old@example.com", Phone: "123", LicenseNumber: "LIC"}
	repo := &mockDoctorRepo{
		findByIDFn: func(ctx context.Context, id string) (*doctor.Doctor, error) {
			return entity, nil
		},
	}
	var saved *doctor.Doctor
	repo.saveFn = func(ctx context.Context, d *doctor.Doctor) error {
		saved = d
		return nil
	}

	handler := NewDoctorCommandHandler(repo)
	updated, err := handler.Update(context.Background(), UpdateDoctorCommand{
		ID:        "doc-1",
		Name:      "New",
		Email:     "new@example.com",
		Phone:     "999",
		Specialty: "General",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "New" {
		t.Fatalf("expected name updated, got %s", updated.Name)
	}
	if saved == nil {
		t.Fatal("expected doctor to be saved")
	}
}

func TestDoctorCommandHandler_Delete(t *testing.T) {
	deleted := false
	repo := &mockDoctorRepo{
		existsFn: func(ctx context.Context, id string) (bool, error) {
			return true, nil
		},
		deleteFn: func(ctx context.Context, id string) error {
			deleted = true
			return nil
		},
	}

	handler := NewDoctorCommandHandler(repo)
	if err := handler.Delete(context.Background(), DeleteDoctorCommand{ID: "doc-1"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleted {
		t.Fatal("expected delete to be called")
	}
}
