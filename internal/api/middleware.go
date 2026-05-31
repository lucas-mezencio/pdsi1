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

// AuthMiddleware validates Firebase JWT Bearer tokens and sets the caller's
// Firebase UID in the request context. Returns 401 Unauthorized if the token
// is missing, malformed, or invalid.
// For POST /prescriptions, it validates the demo secret instead.
func AuthMiddleware(firebaseAuth *auth.Client, demoSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, http.StatusUnauthorized, "invalid authorization format", "")
				return
			}

			token := parts[1]
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
