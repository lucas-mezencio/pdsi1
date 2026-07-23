package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureLogs swaps slog.Default() for a buffer-backed JSON handler at LevelDebug
// for the duration of the test, restoring the original logger afterwards.
// Returns the buffer so the test can assert on what was written.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	buf := &bytes.Buffer{}
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))
	return buf
}

// findEntry decodes the JSON-lines log buffer into a single entry. slog writes
// one JSON object per line, so we look for the first line that decodes cleanly.
func findEntry(t *testing.T, buf *bytes.Buffer, predicate func(map[string]any) bool) map[string]any {
	t.Helper()
	for _, line := range strings.Split(buf.String(), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if predicate(entry) {
			return entry
		}
	}
	t.Fatalf("no matching log entry found in:\n%s", buf.String())
	return nil
}

func TestLoggingMiddleware_LogsAtInfoForSuccess(t *testing.T) {
	buf := captureLogs(t)

	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	entry := findEntry(t, buf, func(e map[string]any) bool {
		return e["msg"] == "request completed"
	})

	if entry["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", entry["level"])
	}
	if entry["method"] != "GET" {
		t.Errorf("expected method GET, got %v", entry["method"])
	}
	if entry["path"] != "/api/v1/health" {
		t.Errorf("expected path /api/v1/health, got %v", entry["path"])
	}
	// JSON numbers decode to float64
	if status, _ := entry["status"].(float64); status != 200 {
		t.Errorf("expected status 200, got %v", entry["status"])
	}
	if bytes, _ := entry["bytes"].(float64); bytes != 2 {
		t.Errorf("expected bytes 2, got %v", entry["bytes"])
	}
}

func TestLoggingMiddleware_LogsAtWarnFor4xx(t *testing.T) {
	buf := captureLogs(t)

	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest("GET", "/api/v1/missing", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	entry := findEntry(t, buf, func(e map[string]any) bool {
		return e["msg"] == "request completed"
	})

	if entry["level"] != "WARN" {
		t.Errorf("expected level WARN, got %v", entry["level"])
	}
	if status, _ := entry["status"].(float64); status != 404 {
		t.Errorf("expected status 404, got %v", entry["status"])
	}
}

func TestLoggingMiddleware_LogsAtErrorFor5xx(t *testing.T) {
	buf := captureLogs(t)

	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))

	req := httptest.NewRequest("POST", "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	entry := findEntry(t, buf, func(e map[string]any) bool {
		return e["msg"] == "request completed"
	})

	if entry["level"] != "ERROR" {
		t.Errorf("expected level ERROR, got %v", entry["level"])
	}
	if status, _ := entry["status"].(float64); status != 500 {
		t.Errorf("expected status 500, got %v", entry["status"])
	}
}

func TestLoggingMiddleware_CapturesDuration(t *testing.T) {
	buf := captureLogs(t)

	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	entry := findEntry(t, buf, func(e map[string]any) bool {
		return e["msg"] == "request completed"
	})

	if _, ok := entry["duration_ms"]; !ok {
		t.Errorf("expected duration_ms attribute, got entry: %v", entry)
	}
}

func TestResponseRecorder_DefaultsStatusTo200(t *testing.T) {
	rr := httptest.NewRecorder()
	rec := &responseRecorder{ResponseWriter: rr}

	// Write without calling WriteHeader — net/http would send 200.
	_, _ = rec.Write([]byte("hello"))

	if rec.statusCode != http.StatusOK {
		t.Errorf("expected status 200 when Write called without WriteHeader, got %d", rec.statusCode)
	}
	if rec.bytes != len("hello") {
		t.Errorf("expected bytes %d, got %d", len("hello"), rec.bytes)
	}
}

func TestResponseRecorder_CapturesExplicitStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	rec := &responseRecorder{ResponseWriter: rr}

	rec.WriteHeader(http.StatusTeapot)
	_, _ = rec.Write([]byte("coffee"))

	if rec.statusCode != http.StatusTeapot {
		t.Errorf("expected status 418, got %d", rec.statusCode)
	}
	if rec.bytes != len("coffee") {
		t.Errorf("expected bytes %d, got %d", len("coffee"), rec.bytes)
	}
}
