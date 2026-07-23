package queries

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

type mockPrescriptionRepo struct {
	findByIDFn           func(ctx context.Context, id string) (*prescription.Prescription, error)
	findByUserIDFn       func(ctx context.Context, userID string) ([]*prescription.Prescription, error)
	findByMedicIDFn      func(ctx context.Context, medicID string) ([]*prescription.Prescription, error)
	findActiveFn         func(ctx context.Context) ([]*prescription.Prescription, error)
	findActiveByUserIDFn func(ctx context.Context, userID string) ([]*prescription.Prescription, error)
	findAllFn            func(ctx context.Context) ([]*prescription.Prescription, error)
}

func (m *mockPrescriptionRepo) Save(ctx context.Context, entity *prescription.Prescription) error {
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
func (m *mockPrescriptionRepo) Delete(ctx context.Context, id string) error { return nil }
func (m *mockPrescriptionRepo) Exists(ctx context.Context, id string) (bool, error) {
	return false, nil
}

func TestPrescriptionQueryHandler_GetByID(t *testing.T) {
	repo := &mockPrescriptionRepo{
		findByIDFn: func(ctx context.Context, id string) (*prescription.Prescription, error) {
			return &prescription.Prescription{ID: id}, nil
		},
	}

	handler := NewPrescriptionQueryHandler(repo)
	entity, err := handler.GetByID(context.Background(), GetPrescriptionByIDQuery{ID: "rx-1"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity.ID != "rx-1" {
		t.Fatalf("expected prescription ID rx-1, got %s", entity.ID)
	}
}

func TestPrescriptionQueryHandler_GetByID_InvalidInput(t *testing.T) {
	handler := NewPrescriptionQueryHandler(&mockPrescriptionRepo{})
	_, err := handler.GetByID(context.Background(), GetPrescriptionByIDQuery{})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestPrescriptionQueryHandler_List_ByUserActive(t *testing.T) {
	called := false
	repo := &mockPrescriptionRepo{
		findActiveByUserIDFn: func(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
			called = true
			return []*prescription.Prescription{{ID: "rx-1"}}, nil
		},
	}

	active := true
	handler := NewPrescriptionQueryHandler(repo)
	list, err := handler.List(context.Background(), ListPrescriptionsQuery{UserID: "user-1", Active: &active})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected FindActiveByUserID to be called")
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 prescription, got %d", len(list))
	}
}

func TestPrescriptionQueryHandler_List_ByUser(t *testing.T) {
	called := false
	repo := &mockPrescriptionRepo{
		findByUserIDFn: func(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
			called = true
			return []*prescription.Prescription{{ID: "rx-1"}}, nil
		},
	}

	handler := NewPrescriptionQueryHandler(repo)
	list, err := handler.List(context.Background(), ListPrescriptionsQuery{UserID: "user-1"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected FindByUserID to be called")
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 prescription, got %d", len(list))
	}
}

func TestPrescriptionQueryHandler_List_ByMedic(t *testing.T) {
	called := false
	repo := &mockPrescriptionRepo{
		findByMedicIDFn: func(ctx context.Context, medicID string) ([]*prescription.Prescription, error) {
			called = true
			return []*prescription.Prescription{{ID: "rx-2"}}, nil
		},
	}

	handler := NewPrescriptionQueryHandler(repo)
	list, err := handler.List(context.Background(), ListPrescriptionsQuery{MedicID: "doc-1"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected FindByMedicID to be called")
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 prescription, got %d", len(list))
	}
}

func TestPrescriptionQueryHandler_List_Active(t *testing.T) {
	called := false
	repo := &mockPrescriptionRepo{
		findActiveFn: func(ctx context.Context) ([]*prescription.Prescription, error) {
			called = true
			return []*prescription.Prescription{{ID: "rx-3"}}, nil
		},
	}

	active := true
	handler := NewPrescriptionQueryHandler(repo)
	list, err := handler.List(context.Background(), ListPrescriptionsQuery{Active: &active})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected FindActive to be called")
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 prescription, got %d", len(list))
	}
}

func TestPrescriptionQueryHandler_List_All(t *testing.T) {
	repo := &mockPrescriptionRepo{
		findAllFn: func(ctx context.Context) ([]*prescription.Prescription, error) {
			return []*prescription.Prescription{{ID: "rx-4"}}, nil
		},
	}

	handler := NewPrescriptionQueryHandler(repo)
	list, err := handler.List(context.Background(), ListPrescriptionsQuery{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 prescription, got %d", len(list))
	}
}
