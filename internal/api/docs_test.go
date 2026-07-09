package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsHandler(t *testing.T) {
	handler := DocsHandler()

	t.Run("GET /api/v1/docs serves HTML", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/docs", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); ct != "text/html" {
			t.Errorf("expected Content-Type text/html, got %s", ct)
		}
	})

	t.Run("HTML contains Swagger UI bundle", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/docs", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		body := rr.Body.String()
		if !strings.Contains(body, "swagger-ui") {
			t.Error("HTML does not contain swagger-ui reference")
		}
		if !strings.Contains(body, "/api/v1/docs/openapi.yaml") {
			t.Error("HTML does not reference openapi.yaml")
		}
	})

	t.Run("GET /api/v1/docs/openapi.yaml serves YAML", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/docs/openapi.yaml", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/x-yaml" {
			t.Errorf("expected Content-Type application/x-yaml, got %s", ct)
		}
	})

	t.Run("YAML contains OpenAPI spec", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/docs/openapi.yaml", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		body := rr.Body.String()
		if !strings.Contains(body, "openapi:") {
			t.Error("YAML does not contain openapi version")
		}
		if !strings.Contains(body, "CareConnect API") {
			t.Error("YAML does not contain API title")
		}
	})
}