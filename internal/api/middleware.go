package httpapi

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

type contextKey string

const (
	contextKeyUserID   contextKey = "caller_user_id"
	contextKeyUserRole contextKey = "caller_user_role"
)

// RBACMiddleware reads the X-User-ID header and enriches the request context
// with the caller's ID and role. If the header is absent or the user is not
// found the request proceeds without caller information (unauthenticated mode).
func RBACMiddleware(userRepo user.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			if userID != "" {
				entity, err := userRepo.FindByID(r.Context(), userID)
				if err == nil {
					ctx := context.WithValue(r.Context(), contextKeyUserID, entity.ID)
					ctx = context.WithValue(ctx, contextKeyUserRole, entity.Role)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// publicPaths are routes that do not require authentication.
var publicPaths = map[string]bool{
	"/api/v1/auth/login":                    true, // login (public, clients need a token before they can authenticate)
	"/api/v1/auth/register":                 true, // registration (public, users don't have accounts yet)
	"/api/v1/docs":                          true, // swagger UI
	"/api/v1/docs/openapi.yaml":             true, // openapi spec
	"/api/v1/health":                        true, // health check
	"/api/v1/test-notifications":            true, // dev-only browser test page (only registered when ENABLE_TEST_PAGE=true; see router.go)
	"/api/v1/test-notifications/config":     true, // dev-only browser test page config endpoint
	"/firebase-messaging-sw.js":             true, // FCM service worker for the test page (must be auth-free so Firebase's default registration can fetch it)
}

func isPublicPath(path string) bool {
	return publicPaths[path]
}

// AuthMiddleware validates Firebase JWT Bearer tokens and sets the caller's
// Firebase UID in the request context. Returns 401 Unauthorized if the token
// is missing, malformed, or invalid.
// For POST /prescriptions, it validates the demo secret instead.
func AuthMiddleware(firebaseAuth *auth.Client, demoSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Public routes bypass auth
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// POST /prescriptions uses demo secret auth instead of Firebase JWT
			if r.URL.Path == "/api/v1/prescriptions" && r.Method == http.MethodPost {
				authHeader := r.Header.Get("Authorization")
				if authHeader != demoSecret {
					writeError(w, http.StatusUnauthorized, "invalid secret", "")
					return
				}
				// Demo secret validated, proceed without Firebase UID
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header", "")
				return
			}

			token, ok := extractBearerToken(authHeader)
			if !ok {
				writeError(w, http.StatusUnauthorized, "invalid authorization format", "")
				return
			}

			firebaseToken, err := firebaseAuth.VerifyIDToken(r.Context(), token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token", err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, firebaseToken.UID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// callerUserID extracts the caller's user ID from the request context.
func callerUserID(r *http.Request) string {
	v, _ := r.Context().Value(contextKeyUserID).(string)
	return v
}

// callerRole extracts the caller's role from the request context.
func callerRole(r *http.Request) user.Role {
	v, _ := r.Context().Value(contextKeyUserRole).(user.Role)
	return v
}

// extractBearerToken pulls the Firebase ID token out of an Authorization
// header value. It accepts:
//
//   - `Bearer <jwt>` (RFC 6750, case-insensitive scheme, whitespace tolerated)
//   - `<jwt>` (raw token, exactly what /auth/login returns)
//
// Anything else — non-Bearer schemes (`Basic xyz`, `Token xyz`), malformed
// JWTs, or empty values — is rejected so we never pass nonsense to Firebase
// VerifyIDToken.
func extractBearerToken(header string) (string, bool) {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return "", false
	}

	if parts := strings.SplitN(trimmed, " ", 2); len(parts) == 2 {
		if !strings.EqualFold(parts[0], "Bearer") {
			return "", false
		}
		token := strings.TrimSpace(parts[1])
		if !looksLikeJWT(token) {
			return "", false
		}
		return token, true
	}

	if !looksLikeJWT(trimmed) {
		return "", false
	}
	return trimmed, true
}

// looksLikeJWT reports whether value has the three-segment shape of a JWT
// (header.payload.signature), where each segment is non-empty and contains
// no whitespace.
func looksLikeJWT(value string) bool {
	segments := strings.Split(value, ".")
	if len(segments) != 3 {
		return false
	}
	for _, segment := range segments {
		if segment == "" || strings.ContainsAny(segment, " \t\r\n") {
			return false
		}
	}
	return true
}

// DemoSecretMiddleware validates a static secret in Authorization header
// Used for demo/testing purposes (e.g., POST /prescriptions on public VPS)
func DemoSecretMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != secret {
				writeError(w, http.StatusUnauthorized, "invalid secret", "")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
