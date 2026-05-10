package commands

import (
	"context"
	"errors"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

type mockAuthProvider struct {
	createUserFn func(ctx context.Context, email, password string) (string, error)
	deleteUserFn func(ctx context.Context, firebaseID string) error
	signInFn     func(ctx context.Context, email, password string) (string, error)
}

func (m *mockAuthProvider) CreateUser(ctx context.Context, email, password string) (string, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, email, password)
	}
	return "", nil
}

func (m *mockAuthProvider) DeleteUser(ctx context.Context, firebaseID string) error {
	if m.deleteUserFn != nil {
		return m.deleteUserFn(ctx, firebaseID)
	}
	return nil
}

func (m *mockAuthProvider) SignIn(ctx context.Context, email, password string) (string, error) {
	if m.signInFn != nil {
		return m.signInFn(ctx, email, password)
	}
	return "", nil
}

func TestAuthCommandHandler_Register(t *testing.T) {
	repo := &mockUserRepo{}
	var saved *user.User
	repo.saveFn = func(ctx context.Context, entity *user.User) error {
		saved = entity
		return nil
	}

	authProvider := &mockAuthProvider{
		createUserFn: func(ctx context.Context, email, password string) (string, error) {
			return "firebase-uid-1", nil
		},
	}

	handler := NewAuthCommandHandler(repo, authProvider)
	entity, err := handler.Register(context.Background(), RegisterCommand{
		Name:     "Alice",
		Email:    "alice@example.com",
		Phone:    "+100000000",
		Password: "Password123!",
		Role:     string(user.RoleElderly),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity == nil {
		t.Fatal("expected user")
	}
	if entity.FirebaseID != "firebase-uid-1" {
		t.Fatalf("expected firebase id firebase-uid-1, got %s", entity.FirebaseID)
	}
	if saved == nil || saved.FirebaseID != "firebase-uid-1" {
		t.Fatal("expected saved user with firebase id")
	}
}

func TestAuthCommandHandler_Register_EmailAlreadyInUse(t *testing.T) {
	repo := &mockUserRepo{}
	authProvider := &mockAuthProvider{
		createUserFn: func(ctx context.Context, email, password string) (string, error) {
			return "", application.ErrEmailAlreadyInUse
		},
	}
	handler := NewAuthCommandHandler(repo, authProvider)

	_, err := handler.Register(context.Background(), RegisterCommand{
		Name:     "Alice",
		Email:    "alice@example.com",
		Phone:    "+100000000",
		Password: "Password123!",
	})
	if !errors.Is(err, application.ErrEmailAlreadyInUse) {
		t.Fatalf("expected email already in use, got %v", err)
	}
}

func TestAuthCommandHandler_Register_EmailAlreadyInUseInLocalDB(t *testing.T) {
	createCalled := false
	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return &user.User{ID: "local-user-1", Email: email}, nil
		},
	}
	authProvider := &mockAuthProvider{
		createUserFn: func(ctx context.Context, email, password string) (string, error) {
			createCalled = true
			return "firebase-uid-1", nil
		},
	}
	handler := NewAuthCommandHandler(repo, authProvider)

	_, err := handler.Register(context.Background(), RegisterCommand{
		Name:     "Alice",
		Email:    "alice@example.com",
		Phone:    "+100000000",
		Password: "Password123!",
	})
	if !errors.Is(err, application.ErrEmailAlreadyInUse) {
		t.Fatalf("expected email already in use, got %v", err)
	}
	if createCalled {
		t.Fatal("expected firebase create user not to be called")
	}
}

func TestAuthCommandHandler_LoginByFirebaseID(t *testing.T) {
	authProvider := &mockAuthProvider{
		signInFn: func(ctx context.Context, email, password string) (string, error) {
			return "firebase-uid-1", nil
		},
	}
	repo := &mockUserRepo{
		findByFirebaseIDFn: func(ctx context.Context, firebaseID string) (*user.User, error) {
			return &user.User{ID: "user-1", FirebaseID: firebaseID}, nil
		},
	}
	handler := NewAuthCommandHandler(repo, authProvider)

	entity, err := handler.Login(context.Background(), LoginCommand{
		Email:    "alice@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity.ID != "user-1" {
		t.Fatalf("expected user-1, got %s", entity.ID)
	}
}

func TestAuthCommandHandler_Login_BackfillsFirebaseID(t *testing.T) {
	authProvider := &mockAuthProvider{
		signInFn: func(ctx context.Context, email, password string) (string, error) {
			return "firebase-uid-1", nil
		},
	}

	var saved *user.User
	legacyUser := &user.User{ID: "user-legacy", Email: "legacy@example.com"}
	repo := &mockUserRepo{
		findByFirebaseIDFn: func(ctx context.Context, firebaseID string) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
		findByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return legacyUser, nil
		},
		saveFn: func(ctx context.Context, entity *user.User) error {
			saved = entity
			return nil
		},
	}
	handler := NewAuthCommandHandler(repo, authProvider)

	entity, err := handler.Login(context.Background(), LoginCommand{
		Email:    "legacy@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entity.FirebaseID != "firebase-uid-1" {
		t.Fatalf("expected firebase id to be backfilled, got %s", entity.FirebaseID)
	}
	if saved == nil || saved.FirebaseID != "firebase-uid-1" {
		t.Fatal("expected saved user with backfilled firebase id")
	}
}
