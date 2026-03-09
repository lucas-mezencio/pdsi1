package queries

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/doctor"
)

type mockDoctorRepo struct {
	findByIDFn         func(ctx context.Context, id string) (*doctor.Doctor, error)
	findByEmailFn      func(ctx context.Context, email string) (*doctor.Doctor, error)
	findByFirebaseIDFn func(ctx context.Context, firebaseID string) (*doctor.Doctor, error)
	findByLicenseFn    func(ctx context.Context, license string) (*doctor.Doctor, error)
	findAllFn          func(ctx context.Context) ([]*doctor.Doctor, error)
}

func (m *mockDoctorRepo) Save(ctx context.Context, entity *doctor.Doctor) error { return nil }
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
func (m *mockDoctorRepo) Delete(ctx context.Context, id string) error         { return nil }
func (m *mockDoctorRepo) Exists(ctx context.Context, id string) (bool, error) { return false, nil }

func TestDoctorQueryHandler_GetByID(t *testing.T) {
	repo := &mockDoctorRepo{
		findByIDFn: func(ctx context.Context, id string) (*doctor.Doctor, error) {
			return &doctor.Doctor{ID: id, Name: "Dr"}, nil
		},
	}

	handler := NewDoctorQueryHandler(repo)
	entity, err := handler.GetByID(context.Background(), GetDoctorByIDQuery{ID: "doc-1"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity.ID != "doc-1" {
		t.Fatalf("expected doctor ID doc-1, got %s", entity.ID)
	}
}

func TestDoctorQueryHandler_GetByID_InvalidInput(t *testing.T) {
	handler := NewDoctorQueryHandler(&mockDoctorRepo{})
	_, err := handler.GetByID(context.Background(), GetDoctorByIDQuery{})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestDoctorQueryHandler_GetByEmail_NotFound(t *testing.T) {
	repo := &mockDoctorRepo{
		findByEmailFn: func(ctx context.Context, email string) (*doctor.Doctor, error) {
			return nil, doctor.ErrDoctorNotFound
		},
	}
	handler := NewDoctorQueryHandler(repo)
	_, err := handler.GetByEmail(context.Background(), GetDoctorByEmailQuery{Email: "missing"})
	if !errors.Is(err, application.ErrDoctorNotFound) {
		t.Fatalf("expected doctor not found, got %v", err)
	}
}

func TestDoctorQueryHandler_GetByLicense(t *testing.T) {
	repo := &mockDoctorRepo{
		findByLicenseFn: func(ctx context.Context, license string) (*doctor.Doctor, error) {
			return &doctor.Doctor{ID: "doc-1", LicenseNumber: license}, nil
		},
	}
	handler := NewDoctorQueryHandler(repo)
	entity, err := handler.GetByLicense(context.Background(), GetDoctorByLicenseQuery{LicenseNumber: "LIC"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity.LicenseNumber != "LIC" {
		t.Fatalf("expected license LIC, got %s", entity.LicenseNumber)
	}
}

func TestDoctorQueryHandler_List(t *testing.T) {
	repo := &mockDoctorRepo{
		findAllFn: func(ctx context.Context) ([]*doctor.Doctor, error) {
			return []*doctor.Doctor{{ID: "1"}, {ID: "2"}}, nil
		},
	}
	handler := NewDoctorQueryHandler(repo)
	list, err := handler.List(context.Background(), ListDoctorsQuery{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 doctors, got %d", len(list))
	}
}
