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

// --- mocks for LGPD query handler ---

type mockUserRepoForLGPD struct {
	findByIDFn      func(ctx context.Context, id string) (*user.User, error)
	findCaregiversFn func(ctx context.Context, elderlyID string) ([]*user.User, error)
	findChargesFn    func(ctx context.Context, caregiverID string) ([]*user.User, error)
}

func (m *mockUserRepoForLGPD) Save(ctx context.Context, u *user.User) error { return nil }
func (m *mockUserRepoForLGPD) FindByID(ctx context.Context, id string) (*user.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepoForLGPD) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepoForLGPD) FindByFirebaseID(ctx context.Context, firebaseID string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepoForLGPD) FindAll(ctx context.Context) ([]*user.User, error) {
	return nil, nil
}
func (m *mockUserRepoForLGPD) Delete(ctx context.Context, id string) error { return nil }
func (m *mockUserRepoForLGPD) Exists(ctx context.Context, id string) (bool, error) {
	return true, nil
}
func (m *mockUserRepoForLGPD) FindCaregivers(ctx context.Context, elderlyID string) ([]*user.User, error) {
	if m.findCaregiversFn != nil {
		return m.findCaregiversFn(ctx, elderlyID)
	}
	return nil, nil
}
func (m *mockUserRepoForLGPD) FindCharges(ctx context.Context, caregiverID string) ([]*user.User, error) {
	if m.findChargesFn != nil {
		return m.findChargesFn(ctx, caregiverID)
	}
	return nil, nil
}
func (m *mockUserRepoForLGPD) IsLinked(ctx context.Context, caregiverID, elderlyID string) (bool, error) {
	return false, nil
}
func (m *mockUserRepoForLGPD) LinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}
func (m *mockUserRepoForLGPD) UnlinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}

type mockDoseRepoForLGPD struct {
	findByUserIDFn func(ctx context.Context, userID string) ([]*prescription.DoseRecord, error)
}

func (m *mockDoseRepoForLGPD) Save(ctx context.Context, r *prescription.DoseRecord) error { return nil }
func (m *mockDoseRepoForLGPD) FindByID(ctx context.Context, id string) (*prescription.DoseRecord, error) {
	return nil, prescription.ErrDoseRecordNotFound
}
func (m *mockDoseRepoForLGPD) FindByUserID(ctx context.Context, userID string) ([]*prescription.DoseRecord, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockDoseRepoForLGPD) FindByPrescriptionID(ctx context.Context, prescriptionID string) ([]*prescription.DoseRecord, error) {
	return nil, nil
}
func (m *mockDoseRepoForLGPD) FindPendingBefore(ctx context.Context, before time.Time) ([]*prescription.DoseRecord, error) {
	return nil, nil
}

type mockInvitationRepoForLGPD struct {
	findByElderlyIDFn   func(ctx context.Context, elderlyID string) ([]*user.CaregiverInvitation, error)
	findByCaregiverIDFn func(ctx context.Context, caregiverID string) ([]*user.CaregiverInvitation, error)
}

func (m *mockInvitationRepoForLGPD) Save(ctx context.Context, inv *user.CaregiverInvitation) error {
	return nil
}
func (m *mockInvitationRepoForLGPD) FindByToken(ctx context.Context, token string) (*user.CaregiverInvitation, error) {
	return nil, user.ErrInvitationNotFound
}
func (m *mockInvitationRepoForLGPD) FindByElderlyID(ctx context.Context, elderlyID string) ([]*user.CaregiverInvitation, error) {
	if m.findByElderlyIDFn != nil {
		return m.findByElderlyIDFn(ctx, elderlyID)
	}
	return nil, nil
}
func (m *mockInvitationRepoForLGPD) FindByCaregiverID(ctx context.Context, caregiverID string) ([]*user.CaregiverInvitation, error) {
	if m.findByCaregiverIDFn != nil {
		return m.findByCaregiverIDFn(ctx, caregiverID)
	}
	return nil, nil
}

// --- tests ---

func TestLGPDQueryHandler_ExportForUser_ReturnsAllData(t *testing.T) {
	now := time.Now()
	userRepo := &mockUserRepoForLGPD{
		findByIDFn: func(ctx context.Context, id string) (*user.User, error) {
			return &user.User{ID: id, Name: "Maria", CPF: "52998224725", Email: "maria@example.com"}, nil
		},
		findCaregiversFn: func(ctx context.Context, elderlyID string) ([]*user.User, error) {
			return []*user.User{{ID: "caregiver-1", Name: "João"}}, nil
		},
		findChargesFn: func(ctx context.Context, caregiverID string) ([]*user.User, error) {
			return nil, nil
		},
	}
	prescriptionRepo := &mockPrescriptionRepo{
		findByUserIDFn: func(ctx context.Context, userID string) ([]*prescription.Prescription, error) {
			return []*prescription.Prescription{{
				ID:     "rx-1",
				UserID: userID,
				Medicaments: []prescription.Medicament{{
					Name: "AAS", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 30,
				}},
				Active: true,
			}}, nil
		},
	}
	doseRepo := &mockDoseRepoForLGPD{
		findByUserIDFn: func(ctx context.Context, userID string) ([]*prescription.DoseRecord, error) {
			return []*prescription.DoseRecord{{
				ID: "dose-1", UserID: userID, PrescriptionID: "rx-1",
				MedicamentName: "AAS", Dosage: "100mg",
				ScheduledAt: now, Status: prescription.DoseStatusTaken,
			}}, nil
		},
	}
	invitationRepo := &mockInvitationRepoForLGPD{
		findByElderlyIDFn: func(ctx context.Context, elderlyID string) ([]*user.CaregiverInvitation, error) {
			return []*user.CaregiverInvitation{{ID: "inv-1", ElderlyID: elderlyID, Status: user.InvitationStatusPending}}, nil
		},
		findByCaregiverIDFn: func(ctx context.Context, caregiverID string) ([]*user.CaregiverInvitation, error) {
			return nil, nil
		},
	}

	handler := NewLGPDQueryHandler(userRepo, prescriptionRepo, doseRepo, invitationRepo)
	export, err := handler.ExportForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if export.User == nil || export.User.ID != "user-1" {
		t.Fatalf("expected user.user-1, got %+v", export.User)
	}
	if export.User.CPF != "52998224725" {
		t.Errorf("expected user.cpf 52998224725, got %q", export.User.CPF)
	}
	if len(export.Prescriptions) != 1 || export.Prescriptions[0].ID != "rx-1" {
		t.Fatalf("expected 1 prescription rx-1, got %+v", export.Prescriptions)
	}
	if len(export.Prescriptions[0].Medicaments) != 1 {
		t.Fatalf("expected 1 medicament on prescription, got %d", len(export.Prescriptions[0].Medicaments))
	}
	if len(export.DoseRecords) != 1 || export.DoseRecords[0].ID != "dose-1" {
		t.Fatalf("expected 1 dose record dose-1, got %+v", export.DoseRecords)
	}
	if len(export.Caregivers) != 1 || export.Caregivers[0].ID != "caregiver-1" {
		t.Fatalf("expected 1 caregiver, got %+v", export.Caregivers)
	}
	if len(export.Charges) != 0 {
		t.Fatalf("expected 0 charges, got %d", len(export.Charges))
	}
	if len(export.Invitations) != 1 || export.Invitations[0].ID != "inv-1" {
		t.Fatalf("expected 1 invitation, got %+v", export.Invitations)
	}
}

func TestLGPDQueryHandler_ExportForUser_EmptyUserID(t *testing.T) {
	handler := NewLGPDQueryHandler(&mockUserRepoForLGPD{}, &mockPrescriptionRepo{}, &mockDoseRepoForLGPD{}, &mockInvitationRepoForLGPD{})
	_, err := handler.ExportForUser(context.Background(), "")
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestLGPDQueryHandler_ExportForUser_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepoForLGPD{
		findByIDFn: func(ctx context.Context, id string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}
	handler := NewLGPDQueryHandler(userRepo, &mockPrescriptionRepo{}, &mockDoseRepoForLGPD{}, &mockInvitationRepoForLGPD{})
	_, err := handler.ExportForUser(context.Background(), "missing")
	if !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}