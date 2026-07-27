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
		handler := AuthMiddleware(nil, demoSecret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		handler := AuthMiddleware(nil, demoSecret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		handler := AuthMiddleware(nil, demoSecret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		handler := AuthMiddleware(nil, demoSecret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// TestAuthMiddleware_PublicPaths locks in the bypass behavior for every entry
// in the publicPaths map. Without an auth header the downstream handler must
// be reached for each public path; non-public paths must still be rejected.
func TestAuthMiddleware_PublicPaths(t *testing.T) {
	demoSecret := "demo-secret"

	// expectedPublicPaths is the authoritative list of routes that must bypass
	// authentication. The middleware's publicPaths map must be a superset of
	// this list, so missing entries cause the test to fail loudly.
	expectedPublicPaths := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/docs",
		"/api/v1/docs/openapi.yaml",
		"/api/v1/health",
	}

	for _, path := range expectedPublicPaths {
		t.Run("public path "+path, func(t *testing.T) {
			called := false
			handler := AuthMiddleware(nil, demoSecret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("POST", path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if !called {
				t.Error("handler should have been called for public path")
			}
			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
		})
	}

	t.Run("protected path without auth still rejected", func(t *testing.T) {
		handler := AuthMiddleware(nil, demoSecret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})
}

// TestExtractBearerToken exercises the helper that decides what counts as a
// valid Authorization header value. It must accept the standard
// `Bearer <jwt>` form, case-insensitive variants, and the raw JWT returned by
// the login endpoint. Anything that doesn't look like a JWT (no `Bearer`
// scheme and not three dot-separated segments) must be rejected so we never
// pass nonsense like `Basic xyz` to Firebase VerifyIDToken.
func TestExtractBearerToken(t *testing.T) {
	const rawJWT = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1aWQiLCJpYXQiOjE3MDAwMDAwMDB9.signature"

	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{
			name:      "Bearer prefix with raw JWT",
			header:    "Bearer " + rawJWT,
			wantToken: rawJWT,
			wantOK:    true,
		},
		{
			name:      "lowercase bearer prefix",
			header:    "bearer " + rawJWT,
			wantToken: rawJWT,
			wantOK:    true,
		},
		{
			name:      "uppercase BEARER prefix",
			header:    "BEARER " + rawJWT,
			wantToken: rawJWT,
			wantOK:    true,
		},
		{
			name:      "mixed-case BeArEr prefix",
			header:    "BeArEr " + rawJWT,
			wantToken: rawJWT,
			wantOK:    true,
		},
		{
			name:      "Bearer prefix with surrounding whitespace",
			header:    "  Bearer " + rawJWT + "  ",
			wantToken: rawJWT,
			wantOK:    true,
		},
		{
			name:      "raw JWT without prefix",
			header:    rawJWT,
			wantToken: rawJWT,
			wantOK:    true,
		},
		{
			name:      "raw JWT with surrounding whitespace",
			header:    "  " + rawJWT + "  ",
			wantToken: rawJWT,
			wantOK:    true,
		},
		{
			name:      "Bearer with empty token",
			header:    "Bearer ",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "Bearer prefix and single-segment token rejected",
			header:    "Bearer notajwt",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "non-JWT garbage rejected",
			header:    "this is not a token",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "Basic scheme rejected",
			header:    "Basic dXNlcjpwYXNz",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "Token scheme rejected",
			header:    "Token " + rawJWT,
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "empty string rejected",
			header:    "",
			wantToken: "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, gotOK := extractBearerToken(tt.header)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v (token=%q)", gotOK, tt.wantOK, gotToken)
			}
			if gotToken != tt.wantToken {
				t.Errorf("token = %q, want %q", gotToken, tt.wantToken)
			}
		})
	}
}