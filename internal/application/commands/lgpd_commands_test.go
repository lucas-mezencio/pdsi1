package commands

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// TestLGPDDeleteUser tests LGPD-compliant full data deletion.
func TestLGPDDeleteUser(t *testing.T) {
	t.Run("deletes user and cascades to prescriptions and dose records", func(t *testing.T) {
		var userDeleted, prescDeleted, doseDeleted bool

		repo := &mockUserRepo{
			existsFn: func(ctx context.Context, id string) (bool, error) {
				return true, nil
			},
			deleteFn: func(ctx context.Context, id string) error {
				userDeleted = true
				return nil
			},
		}
		prescRepo := &mockPrescriptionRepo{
			deleteByUserIDFn: func(ctx context.Context, userID string) error {
				prescDeleted = true
				return nil
			},
		}
		doseRepo := &mockDoseRecordRepo{
			deleteByUserIDFn: func(ctx context.Context, userID string) error {
				doseDeleted = true
				return nil
			},
		}

		handler := NewUserCommandHandlerWithDelete(repo, prescRepo, doseRepo)
		if err := handler.DeleteWithLGPD(context.Background(), DeleteUserCommand{ID: "user-1"}); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !userDeleted {
			t.Fatal("expected user to be deleted")
		}
		if !prescDeleted {
			t.Fatal("expected prescriptions to be cascade deleted")
		}
		if !doseDeleted {
			t.Fatal("expected dose records to be cascade deleted")
		}
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		repo := &mockUserRepo{
			existsFn: func(ctx context.Context, id string) (bool, error) {
				return false, nil
			},
		}
		handler := NewUserCommandHandlerWithDelete(repo, &mockPrescriptionRepo{}, &mockDoseRecordRepo{})

		err := handler.DeleteWithLGPD(context.Background(), DeleteUserCommand{ID: "missing"})
		if !errors.Is(err, application.ErrUserNotFound) {
			t.Fatalf("expected user not found error, got %v", err)
		}
	})

	t.Run("returns error for empty user id", func(t *testing.T) {
		handler := NewUserCommandHandlerWithDelete(&mockUserRepo{}, &mockPrescriptionRepo{}, &mockDoseRecordRepo{})

		err := handler.DeleteWithLGPD(context.Background(), DeleteUserCommand{})
		if !errors.Is(err, application.ErrInvalidInput) {
			t.Fatalf("expected invalid input error, got %v", err)
		}
	})
}

// --- Mock repositories for tests ---

type mockPrescriptionRepo struct {
	findByUserIDFn      func(ctx context.Context, userID string) ([]*prescription.Prescription, error)
	deleteByUserIDFn    func(ctx context.Context, userID string) error
	findByIDFn          func(ctx context.Context, id string) (*prescription.Prescription, error)
	findAllFn           func(ctx context.Context) ([]*prescription.Prescription, error)
	saveFn              func(ctx context.Context, p *prescription.Prescription) error
	deleteFn            func(ctx context.Context, id string) error
	existsFn            func(ctx context.Context, id string) (bool, error)
	findByMedicIDFn     func(ctx context.Context, medicID string) ([]*prescription.Prescription, error)
	findActiveFn        func(ctx context.Context) ([]*prescription.Prescription, error)
	findActiveByUserIDFn func(ctx context.Context, userID string) ([]*prescription.Prescription, error)
}

func (m *mockPrescriptionRepo) Save(ctx context.Context, p *prescription.Prescription) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, p)
	}
	return nil
}
func (m *mockPrescriptionRepo) FindAll(ctx context.Context) ([]*prescription.Prescription, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx)
	}
	return nil, nil
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
	return nil, nil
}
func (m *mockPrescriptionRepo) FindByMedicID(ctx context.Context, medicID string) ([]*prescription.Prescription, error) {
	if m.findByMedicIDFn != nil {
		return m.findByMedicIDFn(ctx, medicID)
	}
	return nil, nil
}
func (m *mockPrescriptionRepo) FindActive(ctx context.Context) ([]*prescription.Prescription, error) {
	if m.findActiveFn != nil {
		return m.findActiveFn(ctx)
	}
	return nil, nil
}
func (m *mockPrescriptionRepo) FindActiveByUserID(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
	if m.findActiveByUserIDFn != nil {
		return m.findActiveByUserIDFn(ctx, userID)
	}
	return nil, nil
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
func (m *mockPrescriptionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}

type mockDoseRecordRepo struct {
	findByUserIDFn  func(ctx context.Context, userID string) ([]*prescription.DoseRecord, error)
	deleteByUserIDFn func(ctx context.Context, userID string) error
}

func (m *mockDoseRecordRepo) FindByUserID(ctx context.Context, userID string) ([]*prescription.DoseRecord, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockDoseRecordRepo) DeleteByUserID(ctx context.Context, userID string) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}