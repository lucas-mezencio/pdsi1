package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDemoSecretMiddleware(t *testing.T) {
	secret := "my-demo-secret"
	handler := DemoSecretMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "valid secret",
			authHeader:     "my-demo-secret",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid secret",
			authHeader:     "wrong-secret",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/prescriptions", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestAuthMiddleware_FirebaseJWT(t *testing.T) {
	// Skip if no Firebase credentials available
	if testing.Short() {
		t.Skip("skipping Firebase auth test in short mode")
	}

	demoSecret := "demo-secret"

	// Create a mock Firebase auth client that always fails
	// Real tests would use a proper mock
	t.Run("missing authorization header", func(t *testing.T) {
		handler := AuthMiddleware(nil, demoSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})

	t.Run("invalid authorization format", func(t *testing.T) {
		handler := AuthMiddleware(nil, demoSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})
}

func TestAuthMiddleware_PostPrescriptions_DemoSecret(t *testing.T) {
	demoSecret := "my-secret-123"

	t.Run("POST /prescriptions with valid demo secret", func(t *testing.T) {
		called := false
		handler := AuthMiddleware(nil, demoSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusCreated)
		}))

		req := httptest.NewRequest("POST", "/api/v1/prescriptions", nil)
		req.Header.Set("Authorization", "my-secret-123")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !called {
			t.Error("handler should have been called")
		}
		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rr.Code)
		}
	})

	t.Run("POST /prescriptions with invalid demo secret", func(t *testing.T) {
		handler := AuthMiddleware(nil, demoSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("POST", "/api/v1/prescriptions", nil)
		req.Header.Set("Authorization", "wrong-secret")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})
}