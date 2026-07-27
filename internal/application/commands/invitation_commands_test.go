package commands

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// mockUserRepoForInvitation drives the invitation command handler with just
// enough methods exercised; everything else panics so missing wiring fails
// loudly.
type mockUserRepoForInvitation struct {
	findByEmailFn func(ctx context.Context, email string) (*user.User, error)
	findByIDFn    func(ctx context.Context, id string) (*user.User, error)
	isLinkedFn    func(ctx context.Context, caregiverID, elderlyID string) (bool, error)
	linkUsersFn   func(ctx context.Context, caregiverID, elderlyID string) error
	unlinkFn      func(ctx context.Context, caregiverID, elderlyID string) error
}

func (m *mockUserRepoForInvitation) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepoForInvitation) FindByID(ctx context.Context, id string) (*user.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepoForInvitation) IsLinked(ctx context.Context, caregiverID, elderlyID string) (bool, error) {
	if m.isLinkedFn != nil {
		return m.isLinkedFn(ctx, caregiverID, elderlyID)
	}
	return false, nil
}
func (m *mockUserRepoForInvitation) LinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	if m.linkUsersFn != nil {
		return m.linkUsersFn(ctx, caregiverID, elderlyID)
	}
	return nil
}
func (m *mockUserRepoForInvitation) UnlinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	if m.unlinkFn != nil {
		return m.unlinkFn(ctx, caregiverID, elderlyID)
	}
	return nil
}

// Unused methods — panic to surface accidental wiring.
func (m *mockUserRepoForInvitation) Save(context.Context, *user.User) error { panic("not used") }
func (m *mockUserRepoForInvitation) FindByFirebaseID(context.Context, string) (*user.User, error) {
	panic("not used")
}
func (m *mockUserRepoForInvitation) FindAll(context.Context) ([]*user.User, error) { panic("not used") }
func (m *mockUserRepoForInvitation) Delete(context.Context, string) error          { panic("not used") }
func (m *mockUserRepoForInvitation) Exists(context.Context, string) (bool, error)  { panic("not used") }
func (m *mockUserRepoForInvitation) FindCaregivers(context.Context, string) ([]*user.User, error) {
	panic("not used")
}
func (m *mockUserRepoForInvitation) FindCharges(context.Context, string) ([]*user.User, error) {
	panic("not used")
}

type mockInvitationRepo struct {
	saveFn          func(ctx context.Context, inv *user.CaregiverInvitation) error
	findByTokenFn   func(ctx context.Context, token string) (*user.CaregiverInvitation, error)
	findByCaregiver func(ctx context.Context, caregiverID string) ([]*user.CaregiverInvitation, error)
}

func (m *mockInvitationRepo) Save(ctx context.Context, inv *user.CaregiverInvitation) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, inv)
	}
	return nil
}
func (m *mockInvitationRepo) FindByToken(ctx context.Context, token string) (*user.CaregiverInvitation, error) {
	if m.findByTokenFn != nil {
		return m.findByTokenFn(ctx, token)
	}
	return nil, user.ErrInvitationNotFound
}
func (m *mockInvitationRepo) FindByElderlyID(_ context.Context, _ string) ([]*user.CaregiverInvitation, error) {
	return nil, nil
}
func (m *mockInvitationRepo) FindByCaregiverID(ctx context.Context, caregiverID string) ([]*user.CaregiverInvitation, error) {
	if m.findByCaregiver != nil {
		return m.findByCaregiver(ctx, caregiverID)
	}
	return nil, nil
}

func newInvitationHandler(uRepo *mockUserRepoForInvitation, iRepo *mockInvitationRepo) *InvitationCommandHandler {
	return NewInvitationCommandHandler(uRepo, iRepo)
}

// TestCreateInvitation_ByEmail_Success covers the new simplified API: caller
// provides a caregiver email, the handler looks up the caregiver by email,
// and a PENDING invitation is persisted with a non-empty token.
func TestCreateInvitation_ByEmail_Success(t *testing.T) {
	const (
		elderlyID     = "elderly-uuid"
		caregiverID   = "caregiver-uuid"
		caregiverMail = "maria@example.com"
	)
	elderly := &user.User{ID: elderlyID, Role: user.RoleElderly}
	caregiver := &user.User{ID: caregiverID, Role: user.RoleCaregiver, Email: caregiverMail}

	uRepo := &mockUserRepoForInvitation{
		findByIDFn: func(_ context.Context, id string) (*user.User, error) {
			if id == elderlyID {
				return elderly, nil
			}
			return nil, user.ErrUserNotFound
		},
		findByEmailFn: func(_ context.Context, email string) (*user.User, error) {
			if email == caregiverMail {
				return caregiver, nil
			}
			return nil, user.ErrUserNotFound
		},
		isLinkedFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	var savedInv *user.CaregiverInvitation
	iRepo := &mockInvitationRepo{
		saveFn: func(_ context.Context, inv *user.CaregiverInvitation) error {
			savedInv = inv
			return nil
		},
	}

	h := newInvitationHandler(uRepo, iRepo)
	inv, err := h.Create(context.Background(), CreateInvitationCommand{
		ElderlyID:        elderlyID,
		CaregiverEmail:   caregiverMail,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv == nil {
		t.Fatal("expected non-nil invitation")
	}
	if inv.CaregiverID != caregiverID {
		t.Errorf("expected CaregiverID=%q, got %q", caregiverID, inv.CaregiverID)
	}
	if inv.ElderlyID != elderlyID {
		t.Errorf("expected ElderlyID=%q, got %q", elderlyID, inv.ElderlyID)
	}
	if inv.Token == "" {
		t.Error("expected non-empty token")
	}
	if inv.Status != user.InvitationStatusPending {
		t.Errorf("expected PENDING, got %q", inv.Status)
	}
	if savedInv == nil {
		t.Error("expected Save to be called")
	}
}

// TestCreateInvitation_CaregiverNotFound covers the "user typed an email we
// don't know" path. Returns application.ErrUserNotFound so the API layer
// surfaces a 404.
func TestCreateInvitation_CaregiverNotFound(t *testing.T) {
	uRepo := &mockUserRepoForInvitation{
		findByIDFn: func(_ context.Context, _ string) (*user.User, error) {
			return &user.User{ID: "elderly-uuid", Role: user.RoleElderly}, nil
		},
		findByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}
	iRepo := &mockInvitationRepo{}

	h := newInvitationHandler(uRepo, iRepo)
	_, err := h.Create(context.Background(), CreateInvitationCommand{
		ElderlyID:      "elderly-uuid",
		CaregiverEmail: "ghost@example.com",
	})
	if !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

// TestCreateInvitation_UserNotCaregiver covers the wrong-role path. The user
// exists but has role=ELDERLY (or anything other than CAREGIVER).
func TestCreateInvitation_UserNotCaregiver(t *testing.T) {
	uRepo := &mockUserRepoForInvitation{
		findByIDFn: func(_ context.Context, _ string) (*user.User, error) {
			return &user.User{ID: "elderly-uuid", Role: user.RoleElderly}, nil
		},
		findByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return &user.User{ID: "x", Email: "x@example.com", Role: user.RoleElderly}, nil
		},
	}
	iRepo := &mockInvitationRepo{}

	h := newInvitationHandler(uRepo, iRepo)
	_, err := h.Create(context.Background(), CreateInvitationCommand{
		ElderlyID:      "elderly-uuid",
		CaregiverEmail: "x@example.com",
	})
	if !errors.Is(err, application.ErrWrongRole) {
		t.Fatalf("expected ErrWrongRole, got %v", err)
	}
}

// TestCreateInvitation_AlreadyLinked covers the duplicate-link guard.
func TestCreateInvitation_AlreadyLinked(t *testing.T) {
	uRepo := &mockUserRepoForInvitation{
		findByIDFn: func(_ context.Context, _ string) (*user.User, error) {
			return &user.User{ID: "elderly-uuid", Role: user.RoleElderly}, nil
		},
		findByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return &user.User{ID: "caregiver-uuid", Role: user.RoleCaregiver, Email: "c@example.com"}, nil
		},
		isLinkedFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}
	iRepo := &mockInvitationRepo{}

	h := newInvitationHandler(uRepo, iRepo)
	_, err := h.Create(context.Background(), CreateInvitationCommand{
		ElderlyID:      "elderly-uuid",
		CaregiverEmail: "c@example.com",
	})
	if !errors.Is(err, application.ErrAlreadyLinked) {
		t.Fatalf("expected ErrAlreadyLinked, got %v", err)
	}
}

// TestCreateInvitation_EmptyInputs covers input validation.
func TestCreateInvitation_EmptyInputs(t *testing.T) {
	h := newInvitationHandler(&mockUserRepoForInvitation{}, &mockInvitationRepo{})
	_, err := h.Create(context.Background(), CreateInvitationCommand{})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// TestCreateInvitation_SelfInviteRejected covers the case where the elderly
// user types their own email. The link would be a no-op, so we refuse.
func TestCreateInvitation_SelfInviteRejected(t *testing.T) {
	const elderlyMail = "self@example.com"
	uRepo := &mockUserRepoForInvitation{
		findByIDFn: func(_ context.Context, _ string) (*user.User, error) {
			return &user.User{ID: "elderly-uuid", Role: user.RoleElderly, Email: elderlyMail}, nil
		},
		findByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return &user.User{ID: "elderly-uuid", Role: user.RoleElderly, Email: elderlyMail}, nil
		},
	}
	iRepo := &mockInvitationRepo{}

	h := newInvitationHandler(uRepo, iRepo)
	_, err := h.Create(context.Background(), CreateInvitationCommand{
		ElderlyID:      "elderly-uuid",
		CaregiverEmail: elderlyMail,
	})
	if !errors.Is(err, application.ErrConflict) && !errors.Is(err, application.ErrWrongRole) {
		t.Fatalf("expected ErrConflict or ErrWrongRole, got %v", err)
	}
}
