package httpapi

import (
	"net/http"

	"firebase.google.com/go/v4/auth"
	gen "github.com.br/lucas-mezencio/pdsi1/internal/api/gen"
	"github.com.br/lucas-mezencio/pdsi1/internal/config"
	"github.com/go-chi/chi/v5"
)

// NewRouter builds the chi router for the API.
func NewRouter(server gen.ServerInterface, ext *ExtendedServer, firebaseAuth *auth.Client, demoSecret string, cfg config.Config) http.Handler {
	router := chi.NewRouter()

	// Logging middleware: emits one slog entry per request (must run first so
	// unauthorized requests are still observed).
	router.Use(LoggingMiddleware)

	// Auth middleware: validates Firebase JWT Bearer tokens (or demo secret for POST /prescriptions)
	if firebaseAuth != nil {
		router.Use(AuthMiddleware(firebaseAuth, demoSecret))
	}

	// RBAC middleware: enriches context with caller identity from X-User-ID header.
	// This is a fallback for routes that don't use Firebase JWT but still need user context.
	router.Use(RBACMiddleware(ext.userRepo))

	// Register routes from the generated OpenAPI spec.
	gen.HandlerFromMuxWithBaseURL(server, router, "/api/v1")

	// Serve Swagger UI at /api/v1/docs
	router.Mount("/api/v1/docs", DocsHandler())

	// Register additional routes not covered by the generated spec.
	router.Route("/api/v1", func(r chi.Router) {
		// Invitations
		r.Post("/invitations", ext.CreateInvitation)
		r.Get("/invitations/{token}", ext.GetInvitationByToken)
		r.Post("/invitations/{token}/accept", ext.AcceptInvitation)
		r.Post("/invitations/{token}/reject", ext.RejectInvitation)

		// User link management
		r.Get("/users/{userId}/caregivers", ext.ListCaregivers)
		r.Delete("/users/{userId}/caregivers/{caregiverId}", ext.UnlinkUsers)
		r.Get("/users/{userId}/charges", ext.ListCharges)
		r.Get("/users/{userId}/invitations", ext.ListCaregiverInvitations)

		// Dose records
		r.Get("/users/{userId}/dose-records", ext.ListDoseRecords)
		r.Post("/dose-records/{doseRecordId}/confirm", ext.ConfirmDose)
		r.Post("/dose-records/{doseRecordId}/miss", ext.MarkDoseMissed)

		// LGPD data-export (Brazilian LGPD / GDPR right of access)
		r.Get("/users/me/data-export", ext.LGPDDataExport)

		// Device tokens (Firebase Cloud Messaging registration)
		r.Route("/users/me/device-tokens", func(r chi.Router) {
			r.Post("/", ext.RegisterDeviceToken)
			r.Get("/", ext.ListDeviceTokens)
			r.Route("/{tokenId}", func(r chi.Router) {
				r.Delete("/", ext.DeleteDeviceToken)
				r.Patch("/enabled", ext.SetDeviceTokenEnabled)
			})
		})

		// Browser test page for device tokens. Only registered when
		// ENABLE_TEST_PAGE=true so the dev tooling is unreachable in production.
		if cfg.EnableTestPage {
			testCfg := TestNotificationsConfig{
				FirebaseWebConfig:   cfg.FirebaseWebConfig,
				FirebaseWebVAPIDKey: cfg.FirebaseWebVAPIDKey,
			}
			r.Get("/test-notifications", TestNotificationsPage(testCfg))
			r.Get("/test-notifications/config", TestNotificationsConfigJSON(testCfg))
		}
	})

	return router
}
