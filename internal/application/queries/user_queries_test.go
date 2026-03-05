package queries

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

type mockUserRepo struct {
	findByIDFn    func(ctx context.Context, id string) (*user.User, error)
	findByEmailFn func(ctx context.Context, email string) (*user.User, error)
	findAllFn     func(ctx context.Context) ([]*user.User, error)
}

func (m *mockUserRepo) Save(ctx context.Context, entity *user.User) error { return nil }
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*user.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepo) FindAll(ctx context.Context) ([]*user.User, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx)
	}
	return []*user.User{}, nil
}
func (m *mockUserRepo) Delete(ctx context.Context, id string) error         { return nil }
func (m *mockUserRepo) Exists(ctx context.Context, id string) (bool, error) { return false, nil }
func (m *mockUserRepo) FindCaregivers(ctx context.Context, elderlyID string) ([]*user.User, error) {
	return nil, nil
}
func (m *mockUserRepo) FindCharges(ctx context.Context, caregiverID string) ([]*user.User, error) {
	return nil, nil
}
func (m *mockUserRepo) IsLinked(ctx context.Context, caregiverID, elderlyID string) (bool, error) {
	return false, nil
}
func (m *mockUserRepo) LinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}
func (m *mockUserRepo) UnlinkUsers(ctx context.Context, caregiverID, elderlyID string) error {
	return nil
}

func TestUserQueryHandler_GetByID(t *testing.T) {
	repo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id string) (*user.User, error) {
			return &user.User{ID: id, Name: "Alice"}, nil
		},
	}

	handler := NewUserQueryHandler(repo)
	entity, err := handler.GetByID(context.Background(), GetUserByIDQuery{ID: "user-1"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity.ID != "user-1" {
		t.Fatalf("expected user ID user-1, got %s", entity.ID)
	}
}

func TestUserQueryHandler_GetByID_InvalidInput(t *testing.T) {
	handler := NewUserQueryHandler(&mockUserRepo{})
	_, err := handler.GetByID(context.Background(), GetUserByIDQuery{})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestUserQueryHandler_GetByEmail_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}
	handler := NewUserQueryHandler(repo)
	_, err := handler.GetByEmail(context.Background(), GetUserByEmailQuery{Email: "missing@example.com"})
	if !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("expected user not found, got %v", err)
	}
}

func TestUserQueryHandler_List(t *testing.T) {
	repo := &mockUserRepo{
		findAllFn: func(ctx context.Context) ([]*user.User, error) {
			return []*user.User{{ID: "1"}, {ID: "2"}}, nil
		},
	}
	handler := NewUserQueryHandler(repo)
	list, err := handler.List(context.Background(), ListUsersQuery{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list))
	}
}
