package httpapi

import (
	"net/http"

	gen "github.com.br/lucas-mezencio/pdsi1/internal/api/gen"
	"github.com/go-chi/chi/v5"
)

// NewRouter builds the chi router for the API.
func NewRouter(server gen.ServerInterface) http.Handler {
	router := chi.NewRouter()
	return gen.HandlerFromMuxWithBaseURL(server, router, "/api/v1")
}
