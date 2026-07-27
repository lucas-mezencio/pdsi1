package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	firebaseauth "firebase.google.com/go/v4/auth"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// stubFirebaseVerifier satisfies the firebaseTokenVerifier interface so the
// middleware can be exercised without contacting Firebase.
type stubFirebaseVerifier struct {
	token *firebaseauth.Token
	err   error
}

func (s *stubFirebaseVerifier) VerifyIDToken(_ context.Context, _ string) (*firebaseauth.Token, error) {
	return s.token, s.err
}

// stubUserRepoForAuth implements user.Repository with just enough behavior to
// drive the middleware's resolution path.
type stubUserRepoForAuth struct {
	local *user.User
	err   error
}

func (s *stubUserRepoForAuth) Save(_ context.Context, _ *user.User) error { return nil }
func (s *stubUserRepoForAuth) FindByID(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepoForAuth) FindByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepoForAuth) FindByFirebaseID(_ context.Context, firebaseID string) (*user.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.local != nil && s.local.FirebaseID == firebaseID {
		return s.local, nil
	}
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepoForAuth) FindAll(_ context.Context) ([]*user.User, error)  { return nil, nil }
func (s *stubUserRepoForAuth) Delete(_ context.Context, _ string) error          { return nil }
func (s *stubUserRepoForAuth) Exists(_ context.Context, _ string) (bool, error)  { return false, nil }
func (s *stubUserRepoForAuth) FindCaregivers(_ context.Context, _ string) ([]*user.User, error) {
	return nil, nil
}
func (s *stubUserRepoForAuth) FindCharges(_ context.Context, _ string) ([]*user.User, error) {
	return nil, nil
}
func (s *stubUserRepoForAuth) IsLinked(_ context.Context, _, _ string) (bool, error) { return false, nil }
func (s *stubUserRepoForAuth) LinkUsers(_ context.Context, _, _ string) error        { return nil }
func (s *stubUserRepoForAuth) UnlinkUsers(_ context.Context, _, _ string) error      { return nil }

const sampleJWT = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1aWQiLCJpYXQiOjE3MDAwMDAwMDB9.signature"

// TestAuthMiddleware_ResolvesFirebaseUIDToLocalUserID ensures that after a
// valid Firebase JWT is verified, the caller's context holds the local user
// UUID (not the Firebase UID). This is the contract every downstream handler
// relies on — leaking the Firebase UID into UUID-bound SQL queries produced
// the production 500 on /users/{uuid}/dose-records.
func TestAuthMiddleware_ResolvesFirebaseUIDToLocalUserID(t *testing.T) {
	const (
		firebaseUID = "uq1OEy7P0UPOvJIiFwCDNQxMJAW2"
		localUUID   = "6b1fb275-2efa-4309-b34a-2f8b8abf6e6c"
	)
	verifier := &stubFirebaseVerifier{token: &firebaseauth.Token{UID: firebaseUID}}
	repo := &stubUserRepoForAuth{local: &user.User{ID: localUUID, FirebaseID: firebaseUID}}

	var observedCaller string
	handler := AuthMiddleware(verifier, "demo-secret", repo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedCaller = callerUserID(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users/"+localUUID+"/dose-records", nil)
	req.Header.Set("Authorization", "Bearer "+sampleJWT)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if observedCaller != localUUID {
		t.Fatalf("expected caller=%q (local UUID), got %q (Firebase UID leaked into context)", localUUID, observedCaller)
	}
	if observedCaller == firebaseUID {
		t.Fatal("Firebase UID must never be stored in caller context — it would crash UUID-bound SQL")
	}
}

// TestAuthMiddleware_RejectsUnknownFirebaseUID ensures that a valid Firebase
// token whose UID has no matching local user returns 401 with a clear error,
// rather than silently storing an unresolvable Firebase UID in the context.
func TestAuthMiddleware_RejectsUnknownFirebaseUID(t *testing.T) {
	verifier := &stubFirebaseVerifier{token: &firebaseauth.Token{UID: "fb-orphan"}}
	repo := &stubUserRepoForAuth{err: user.ErrUserNotFound}

	handler := AuthMiddleware(verifier, "demo-secret", repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("downstream handler must not be called when caller has no local user record")
	}))

	req := httptest.NewRequest("GET", "/api/v1/users/some-uuid/dose-records", nil)
	req.Header.Set("Authorization", "Bearer "+sampleJWT)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// TestAuthMiddleware_RepoLookupErrorReturns500 ensures transient lookup
// failures are surfaced as 500 with the underlying error in details.
func TestAuthMiddleware_RepoLookupErrorReturns500(t *testing.T) {
	verifier := &stubFirebaseVerifier{token: &firebaseauth.Token{UID: "fb-1"}}
	repoErr := errors.New("db unavailable")
	repo := &stubUserRepoForAuth{err: repoErr}

	handler := AuthMiddleware(verifier, "demo-secret", repo)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("downstream handler must not be called on repo failure")
	}))

	req := httptest.NewRequest("GET", "/api/v1/users/x/dose-records", nil)
	req.Header.Set("Authorization", "Bearer "+sampleJWT)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// TestAuthMiddleware_PublicPathsAndDemoSecretStillWork locks in the bypass
// behavior introduced by the new userRepo parameter — public routes and the
// POST /prescriptions demo-secret branch must not touch the repo.
func TestAuthMiddleware_PublicPathsAndDemoSecretStillWork(t *testing.T) {
	t.Run("public path bypasses repo lookup", func(t *testing.T) {
		verifier := &stubFirebaseVerifier{token: &firebaseauth.Token{UID: "fb-1"}}
		repo := &stubUserRepoForAuth{err: errors.New("must not be called")}

		called := false
		handler := AuthMiddleware(verifier, "demo-secret", repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !called {
			t.Error("public path must reach downstream handler without touching repo")
		}
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("POST /prescriptions with demo secret bypasses repo lookup", func(t *testing.T) {
		verifier := &stubFirebaseVerifier{token: &firebaseauth.Token{UID: "fb-1"}}
		repo := &stubUserRepoForAuth{err: errors.New("must not be called")}

		called := false
		handler := AuthMiddleware(verifier, "demo-secret", repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusCreated)
		}))

		req := httptest.NewRequest("POST", "/api/v1/prescriptions", nil)
		req.Header.Set("Authorization", "demo-secret")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !called {
			t.Error("demo-secret POST /prescriptions must reach downstream handler without touching repo")
		}
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rr.Code)
		}
	})
}
