package commands

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

type mockUserRepo struct {
	saveFn        func(ctx context.Context, user *user.User) error
	findByIDFn    func(ctx context.Context, id string) (*user.User, error)
	findByEmailFn func(ctx context.Context, email string) (*user.User, error)
	findAllFn     func(ctx context.Context) ([]*user.User, error)
	deleteFn      func(ctx context.Context, id string) error
	existsFn      func(ctx context.Context, id string) (bool, error)
}

func (m *mockUserRepo) Save(ctx context.Context, entity *user.User) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, entity)
	}
	return nil
}

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

func (m *mockUserRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserRepo) Exists(ctx context.Context, id string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, id)
	}
	return false, nil
}

func TestUserCommandHandler_Create(t *testing.T) {
	repo := &mockUserRepo{}
	var saved *user.User
	repo.saveFn = func(ctx context.Context, entity *user.User) error {
		saved = entity
		return nil
	}

	handler := NewUserCommandHandler(repo)

	created, err := handler.Create(context.Background(), CreateUserCommand{
		Name:          "Alice",
		Email:         "alice@example.com",
		Phone:         "+100000000",
		FirebaseToken: "token",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created == nil {
		t.Fatal("expected user to be created")
	}
	if created.Name != "Alice" {
		t.Fatalf("expected name to be Alice, got %s", created.Name)
	}
	if saved == nil {
		t.Fatal("expected user to be saved")
	}
	if saved != created {
		t.Fatal("expected saved user to match created user")
	}
}

func TestUserCommandHandler_Update_InvalidInput(t *testing.T) {
	handler := NewUserCommandHandler(&mockUserRepo{})

	_, err := handler.Update(context.Background(), UpdateUserCommand{})
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestUserCommandHandler_Update_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}

	handler := NewUserCommandHandler(repo)
	_, err := handler.Update(context.Background(), UpdateUserCommand{ID: "missing"})
	if !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("expected user not found error, got %v", err)
	}
}

func TestUserCommandHandler_Update_Success(t *testing.T) {
	entity := &user.User{ID: "user-1", Name: "Old", Email: "old@example.com", Phone: "123"}

	repo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id string) (*user.User, error) {
			return entity, nil
		},
	}
	var saved *user.User
	repo.saveFn = func(ctx context.Context, u *user.User) error {
		saved = u
		return nil
	}

	handler := NewUserCommandHandler(repo)
	updated, err := handler.Update(context.Background(), UpdateUserCommand{
		ID:    "user-1",
		Name:  "New",
		Email: "new@example.com",
		Phone: "999",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "New" {
		t.Fatalf("expected name to update, got %s", updated.Name)
	}
	if saved == nil {
		t.Fatal("expected user to be saved")
	}
}

func TestUserCommandHandler_UpdateFirebaseToken(t *testing.T) {
	entity := &user.User{ID: "user-1", FirebaseToken: "old"}
	repo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id string) (*user.User, error) {
			return entity, nil
		},
	}
	var saved *user.User
	repo.saveFn = func(ctx context.Context, u *user.User) error {
		saved = u
		return nil
	}

	handler := NewUserCommandHandler(repo)
	updated, err := handler.UpdateFirebaseToken(context.Background(), UpdateUserFirebaseTokenCommand{
		ID:            "user-1",
		FirebaseToken: "new-token",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.FirebaseToken != "new-token" {
		t.Fatalf("expected token updated, got %s", updated.FirebaseToken)
	}
	if saved == nil {
		t.Fatal("expected user to be saved")
	}
}

func TestUserCommandHandler_ToggleNotifications(t *testing.T) {
	entity := &user.User{ID: "user-1", NotificationsEnabled: true}
	repo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id string) (*user.User, error) {
			return entity, nil
		},
	}

	handler := NewUserCommandHandler(repo)
	updated, err := handler.ToggleNotifications(context.Background(), ToggleUserNotificationsCommand{
		ID:      "user-1",
		Enabled: false,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.NotificationsEnabled {
		t.Fatal("expected notifications disabled")
	}
}

func TestUserCommandHandler_Delete(t *testing.T) {
	deleted := false
	repo := &mockUserRepo{
		existsFn: func(ctx context.Context, id string) (bool, error) {
			return true, nil
		},
		deleteFn: func(ctx context.Context, id string) error {
			deleted = true
			return nil
		},
	}

	handler := NewUserCommandHandler(repo)
	if err := handler.Delete(context.Background(), DeleteUserCommand{ID: "user-1"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleted {
		t.Fatal("expected delete to be called")
	}
}
