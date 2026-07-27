package queries

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// mockUserRepoForLinkedUser implements only what LinkedUserQueryHandler uses.
type mockUserRepoForLinkedUser struct {
	findCaregiversFn func(ctx context.Context, elderlyID string) ([]*user.User, error)
	findChargesFn    func(ctx context.Context, caregiverID string) ([]*user.User, error)
	isLinkedFn       func(ctx context.Context, caregiverID, elderlyID string) (bool, error)
}

func (m *mockUserRepoForLinkedUser) FindCaregivers(ctx context.Context, elderlyID string) ([]*user.User, error) {
	if m.findCaregiversFn != nil {
		return m.findCaregiversFn(ctx, elderlyID)
	}
	return nil, nil
}
func (m *mockUserRepoForLinkedUser) FindCharges(ctx context.Context, caregiverID string) ([]*user.User, error) {
	if m.findChargesFn != nil {
		return m.findChargesFn(ctx, caregiverID)
	}
	return nil, nil
}
func (m *mockUserRepoForLinkedUser) IsLinked(ctx context.Context, caregiverID, elderlyID string) (bool, error) {
	if m.isLinkedFn != nil {
		return m.isLinkedFn(ctx, caregiverID, elderlyID)
	}
	return false, nil
}

// Unused methods — panic to surface accidental wiring.
func (m *mockUserRepoForLinkedUser) Save(context.Context, *user.User) error { panic("not used") }
func (m *mockUserRepoForLinkedUser) FindByID(context.Context, string) (*user.User, error) {
	panic("not used")
}
func (m *mockUserRepoForLinkedUser) FindByEmail(context.Context, string) (*user.User, error) {
	panic("not used")
}
func (m *mockUserRepoForLinkedUser) FindByFirebaseID(context.Context, string) (*user.User, error) {
	panic("not used")
}
func (m *mockUserRepoForLinkedUser) FindAll(context.Context) ([]*user.User, error) { panic("not used") }
func (m *mockUserRepoForLinkedUser) Delete(context.Context, string) error          { panic("not used") }
func (m *mockUserRepoForLinkedUser) Exists(context.Context, string) (bool, error)  { panic("not used") }
func (m *mockUserRepoForLinkedUser) LinkUsers(context.Context, string, string) error {
	panic("not used")
}
func (m *mockUserRepoForLinkedUser) UnlinkUsers(context.Context, string, string) error {
	panic("not used")
}

// mockInviteRepoForLinkedUser implements only FindByToken and FindByCaregiverID.
type mockInviteRepoForLinkedUser struct {
	findByTokenFn      func(ctx context.Context, token string) (*user.CaregiverInvitation, error)
	findByCaregiverFn  func(ctx context.Context, caregiverID string) ([]*user.CaregiverInvitation, error)
}

func (m *mockInviteRepoForLinkedUser) FindByToken(ctx context.Context, token string) (*user.CaregiverInvitation, error) {
	if m.findByTokenFn != nil {
		return m.findByTokenFn(ctx, token)
	}
	return nil, user.ErrInvitationNotFound
}
func (m *mockInviteRepoForLinkedUser) FindByCaregiverID(ctx context.Context, caregiverID string) ([]*user.CaregiverInvitation, error) {
	if m.findByCaregiverFn != nil {
		return m.findByCaregiverFn(ctx, caregiverID)
	}
	return nil, nil
}
func (m *mockInviteRepoForLinkedUser) Save(context.Context, *user.CaregiverInvitation) error {
	panic("not used")
}
func (m *mockInviteRepoForLinkedUser) FindByElderlyID(context.Context, string) ([]*user.CaregiverInvitation, error) {
	panic("not used")
}

func newLinkedUserHandler(u *mockUserRepoForLinkedUser, i *mockInviteRepoForLinkedUser) *LinkedUserQueryHandler {
	return NewLinkedUserQueryHandler(u, i)
}

// TestListCaregivers_ElderlySelfAllowed covers the simplest happy path: the
// elderly user asks for their own caregivers.
func TestListCaregivers_ElderlySelfAllowed(t *testing.T) {
	called := false
	u := &mockUserRepoForLinkedUser{
		findCaregiversFn: func(_ context.Context, elderlyID string) ([]*user.User, error) {
			called = true
			if elderlyID != "elderly-uuid" {
				t.Errorf("expected elderlyID=elderly-uuid, got %q", elderlyID)
			}
			return []*user.User{{ID: "c1", Name: "Maria"}}, nil
		},
	}
	h := newLinkedUserHandler(u, &mockInviteRepoForLinkedUser{})
	out, err := h.ListCaregivers(context.Background(), ListCaregiversQuery{
		ElderlyID: "elderly-uuid",
		CallerID:  "elderly-uuid",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected FindCaregivers to be called")
	}
	if len(out) != 1 || out[0].ID != "c1" {
		t.Errorf("unexpected output: %+v", out)
	}
}

// TestListCaregivers_NonLinkedCallerForbidden covers the access check: a
// different user (not the elderly themselves and not a linked caregiver) is
// rejected with ErrForbidden.
func TestListCaregivers_NonLinkedCallerForbidden(t *testing.T) {
	u := &mockUserRepoForLinkedUser{
		isLinkedFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	h := newLinkedUserHandler(u, &mockInviteRepoForLinkedUser{})
	_, err := h.ListCaregivers(context.Background(), ListCaregiversQuery{
		ElderlyID: "elderly-uuid",
		CallerID:  "random-uuid",
	})
	if !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// TestListCaregivers_LinkedCaregiverAllowed covers the cross-link access
// path: a caregiver of the elderly user can list that user's other caregivers.
func TestListCaregivers_LinkedCaregiverAllowed(t *testing.T) {
	u := &mockUserRepoForLinkedUser{
		isLinkedFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		findCaregiversFn: func(_ context.Context, _ string) ([]*user.User, error) {
			return []*user.User{{ID: "c2"}}, nil
		},
	}
	h := newLinkedUserHandler(u, &mockInviteRepoForLinkedUser{})
	out, err := h.ListCaregivers(context.Background(), ListCaregiversQuery{
		ElderlyID: "elderly-uuid",
		CallerID:  "linked-caregiver-uuid",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 caregiver, got %d", len(out))
	}
}

// TestListCaregivers_EmptyElderlyID covers input validation.
func TestListCaregivers_EmptyElderlyID(t *testing.T) {
	h := newLinkedUserHandler(&mockUserRepoForLinkedUser{}, &mockInviteRepoForLinkedUser{})
	_, err := h.ListCaregivers(context.Background(), ListCaregiversQuery{})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// TestListCharges_OnlySelfAllowed covers the access check: only the caregiver
// themselves can list their charges.
func TestListCharges_OnlySelfAllowed(t *testing.T) {
	h := newLinkedUserHandler(&mockUserRepoForLinkedUser{}, &mockInviteRepoForLinkedUser{})
	_, err := h.ListCharges(context.Background(), ListChargesQuery{
		CaregiverID: "c1",
		CallerID:    "someone-else",
	})
	if !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// TestListCharges_Success covers the happy path.
func TestListCharges_Success(t *testing.T) {
	u := &mockUserRepoForLinkedUser{
		findChargesFn: func(_ context.Context, caregiverID string) ([]*user.User, error) {
			if caregiverID != "c1" {
				t.Errorf("expected caregiverID=c1, got %q", caregiverID)
			}
			return []*user.User{{ID: "e1"}}, nil
		},
	}
	h := newLinkedUserHandler(u, &mockInviteRepoForLinkedUser{})
	out, err := h.ListCharges(context.Background(), ListChargesQuery{
		CaregiverID: "c1",
		CallerID:    "c1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || out[0].ID != "e1" {
		t.Errorf("unexpected output: %+v", out)
	}
}

// TestListCaregiverInvitations_OnlySelfAllowed mirrors the access check for
// the caregiver-invitations listing.
func TestListCaregiverInvitations_OnlySelfAllowed(t *testing.T) {
	h := newLinkedUserHandler(&mockUserRepoForLinkedUser{}, &mockInviteRepoForLinkedUser{})
	_, err := h.ListCaregiverInvitations(context.Background(), ListCaregiverInvitationsQuery{
		CaregiverID: "c1",
		CallerID:    "someone-else",
	})
	if !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// TestListCaregiverInvitations_Success covers the happy path.
func TestListCaregiverInvitations_Success(t *testing.T) {
	i := &mockInviteRepoForLinkedUser{
		findByCaregiverFn: func(_ context.Context, caregiverID string) ([]*user.CaregiverInvitation, error) {
			if caregiverID != "c1" {
				t.Errorf("expected caregiverID=c1, got %q", caregiverID)
			}
			return []*user.CaregiverInvitation{{ID: "i1", CaregiverID: "c1"}}, nil
		},
	}
	h := newLinkedUserHandler(&mockUserRepoForLinkedUser{}, i)
	out, err := h.ListCaregiverInvitations(context.Background(), ListCaregiverInvitationsQuery{
		CaregiverID: "c1",
		CallerID:    "c1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(out))
	}
}

// TestGetInvitationByToken_Success covers the lookup happy path.
func TestGetInvitationByToken_Success(t *testing.T) {
	inv := &user.CaregiverInvitation{ID: "i1", Token: "tok", Status: user.InvitationStatusPending}
	i := &mockInviteRepoForLinkedUser{
		findByTokenFn: func(_ context.Context, token string) (*user.CaregiverInvitation, error) {
			if token != "tok" {
				t.Errorf("expected token=tok, got %q", token)
			}
			return inv, nil
		},
	}
	h := newLinkedUserHandler(&mockUserRepoForLinkedUser{}, i)
	out, err := h.GetInvitationByToken(context.Background(), GetInvitationByTokenQuery{Token: "tok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != "i1" {
		t.Errorf("expected ID=i1, got %q", out.ID)
	}
}

// TestGetInvitationByToken_NotFound covers the not-found mapping.
func TestGetInvitationByToken_NotFound(t *testing.T) {
	h := newLinkedUserHandler(&mockUserRepoForLinkedUser{}, &mockInviteRepoForLinkedUser{})
	_, err := h.GetInvitationByToken(context.Background(), GetInvitationByTokenQuery{Token: "missing"})
	if !errors.Is(err, application.ErrInvitationNotFound) {
		t.Fatalf("expected ErrInvitationNotFound, got %v", err)
	}
}

// TestGetInvitationByToken_EmptyToken covers input validation.
func TestGetInvitationByToken_EmptyToken(t *testing.T) {
	h := newLinkedUserHandler(&mockUserRepoForLinkedUser{}, &mockInviteRepoForLinkedUser{})
	_, err := h.GetInvitationByToken(context.Background(), GetInvitationByTokenQuery{})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
