package httpapi

import (
	"embed"
	"net/http"

	"github.com.br/lucas-mezencio/pdsi1/docs"
)

//go:embed swagger.html
var swaggerFS embed.FS

// DocsHandler returns an http.Handler that serves Swagger UI.
func DocsHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/docs", func(w http.ResponseWriter, r *http.Request) {
		html, err := swaggerFS.ReadFile("swagger.html")
		if err != nil {
			http.Error(w, "swagger ui not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(html)
	})

	mux.HandleFunc("GET /api/v1/docs/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(docs.OpenAPI)
	})

	return mux
}