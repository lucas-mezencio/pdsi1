package httpapi

import (
	"net/http"

	gen "github.com.br/lucas-mezencio/pdsi1/internal/api/gen"
	"github.com/go-chi/chi/v5"
)

// NewRouter builds the chi router for the API.
func NewRouter(server gen.ServerInterface, ext *ExtendedServer) http.Handler {
	router := chi.NewRouter()

	// RBAC middleware: enriches context with caller identity from X-User-ID header.
	router.Use(RBACMiddleware(ext.userRepo))

	// Register routes from the generated OpenAPI spec.
	gen.HandlerFromMuxWithBaseURL(server, router, "/api/v1")

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

		// Dose records
		r.Get("/users/{userId}/dose-records", ext.ListDoseRecords)
		r.Post("/dose-records/{doseRecordId}/confirm", ext.ConfirmDose)
		r.Post("/dose-records/{doseRecordId}/miss", ext.MarkDoseMissed)
	})

	return router
}
