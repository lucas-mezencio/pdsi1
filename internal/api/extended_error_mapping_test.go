package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
)

func TestWriteExtendedError_MapsErrNotFoundTo404(t *testing.T) {
	rr := httptest.NewRecorder()
	writeExtendedError(rr, application.ErrNotFound)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error == "" {
		t.Fatalf("expected non-empty error message, got empty body %q", rr.Body.String())
	}
}

func TestWriteExtendedError_WrappedErrNotFoundStillMapsTo404(t *testing.T) {
	wrapped := fmt.Errorf("loading device token: %w", application.ErrNotFound)
	rr := httptest.NewRecorder()
	writeExtendedError(rr, wrapped)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for wrapped ErrNotFound, got %d", rr.Code)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error == "" {
		t.Fatalf("expected non-empty error message, got empty body %q", rr.Body.String())
	}
}
