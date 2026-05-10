package httpapi

import (
	"context"
	"net/http"

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
