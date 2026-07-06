package commands

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// stubAuthProvider captures call sequences for assertions.
type stubAuthProvider struct {
	createCalls  int
	createEmail  string
	createPasswd string
	createUID    string
	createErr    error

	deleteCalls int
	deleteUID   string
	deleteErr   error
}

func (s *stubAuthProvider) CreateUser(ctx context.Context, email, password string) (string, error) {
	s.createCalls++
	s.createEmail = email
	s.createPasswd = password
	if s.createErr != nil {
		return "", s.createErr
	}
	if s.createUID == "" {
		s.createUID = "firebase-uid-auto"
	}
	return s.createUID, nil
}

func (s *stubAuthProvider) DeleteUser(ctx context.Context, firebaseID string) error {
	s.deleteCalls++
	s.deleteUID = firebaseID
	return s.deleteErr
}

func (s *stubAuthProvider) SignIn(ctx context.Context, email, password string) (string, string, error) {
	return "", "", errors.New("not implemented")
}

func TestUserCommandHandler_Create_AutoCreatesFirebaseUserAndLinks(t *testing.T) {
	repo := &mockUserRepo{}
	var saved *user.User
	repo.saveFn = func(ctx context.Context, entity *user.User) error {
		saved = entity
		return nil
	}

	auth := &stubAuthProvider{createUID: "firebase-uid-abc123"}
	handler := NewUserCommandHandler(repo, auth)

	created, err := handler.Create(context.Background(), CreateUserCommand{
		Name:     "Alice",
		Email:    "alice@example.com",
		Phone:    "+100000000",
		Password: "S3cretP@ss",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created == nil {
		t.Fatal("expected user to be created")
	}
	if created.FirebaseID != "firebase-uid-abc123" {
		t.Fatalf("expected firebase_id to be linked, got %q", created.FirebaseID)
	}
	if saved == nil {
		t.Fatal("expected user to be saved")
	}
	if saved.FirebaseID != "firebase-uid-abc123" {
		t.Fatalf("expected saved entity to carry firebase_id, got %q", saved.FirebaseID)
	}
	if auth.createCalls != 1 {
		t.Fatalf("expected CreateUser to be called once, got %d", auth.createCalls)
	}
	if auth.createEmail != "alice@example.com" {
		t.Fatalf("expected email forwarded to firebase, got %q", auth.createEmail)
	}
	if auth.createPasswd != "S3cretP@ss" {
		t.Fatalf("expected password forwarded to firebase, got %q", auth.createPasswd)
	}
}

func TestUserCommandHandler_Create_NoFirebaseFieldsOnCommandSurface(t *testing.T) {
	// CreateUserCommand must not carry firebase_id / firebase_token. We assert
	// this by reflecting on the struct type, so reintroduction via a new field
	// would be caught at compile time too.
	cmdType := reflect.TypeOf(CreateUserCommand{})
	for _, name := range []string{"FirebaseID", "FirebaseToken"} {
		if _, ok := cmdType.FieldByName(name); ok {
			t.Fatalf("CreateUserCommand must not have field %s", name)
		}
	}
}

func TestUserCommandHandler_Create_FirebaseFailureSkipsRepoSave(t *testing.T) {
	repo := &mockUserRepo{}
	saveCalls := 0
	repo.saveFn = func(ctx context.Context, entity *user.User) error {
		saveCalls++
		return nil
	}

	auth := &stubAuthProvider{createErr: application.ErrAuthNotConfigured}
	handler := NewUserCommandHandler(repo, auth)

	_, err := handler.Create(context.Background(), CreateUserCommand{
		Name:     "Bob",
		Email:    "bob@example.com",
		Phone:    "+100000001",
		Password: "S3cretP@ss",
	})

	if !errors.Is(err, application.ErrAuthNotConfigured) {
		t.Fatalf("expected auth-not-configured error, got %v", err)
	}
	if saveCalls != 0 {
		t.Fatalf("expected zero saves when firebase create fails, got %d", saveCalls)
	}
}

func TestUserCommandHandler_Create_RepoFailureRollsBackFirebase(t *testing.T) {
	repo := &mockUserRepo{}
	repoErr := errors.New("db boom")
	repo.saveFn = func(ctx context.Context, entity *user.User) error {
		return repoErr
	}

	auth := &stubAuthProvider{createUID: "firebase-uid-xyz"}
	handler := NewUserCommandHandler(repo, auth)

	_, err := handler.Create(context.Background(), CreateUserCommand{
		Name:     "Carol",
		Email:    "carol@example.com",
		Phone:    "+100000002",
		Password: "S3cretP@ss",
	})

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected db error to propagate, got %v", err)
	}
	if auth.deleteCalls != 1 {
		t.Fatalf("expected firebase user to be rolled back, got %d delete calls", auth.deleteCalls)
	}
	if auth.deleteUID != "firebase-uid-xyz" {
		t.Fatalf("expected rollback to delete the just-created uid, got %q", auth.deleteUID)
	}
}

func TestUserCommandHandler_Create_DuplicateEmailIsRejected(t *testing.T) {
	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return &user.User{Email: email}, nil
		},
	}
	auth := &stubAuthProvider{}
	handler := NewUserCommandHandler(repo, auth)

	_, err := handler.Create(context.Background(), CreateUserCommand{
		Name:     "Dave",
		Email:    "dup@example.com",
		Phone:    "+100000003",
		Password: "S3cretP@ss",
	})

	if !errors.Is(err, application.ErrEmailAlreadyInUse) {
		t.Fatalf("expected ErrEmailAlreadyInUse, got %v", err)
	}
	if auth.createCalls != 0 {
		t.Fatalf("expected firebase NOT to be called when email is taken, got %d", auth.createCalls)
	}
}

func TestUserCommandHandler_Create_MissingPasswordRejected(t *testing.T) {
	repo := &mockUserRepo{}
	auth := &stubAuthProvider{}
	handler := NewUserCommandHandler(repo, auth)

	_, err := handler.Create(context.Background(), CreateUserCommand{
		Name:  "Eve",
		Email: "eve@example.com",
		Phone: "+100000004",
	})

	if !errors.Is(err, application.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if auth.createCalls != 0 {
		t.Fatalf("expected firebase NOT to be called without password, got %d", auth.createCalls)
	}
}