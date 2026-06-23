package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

// responseRecorder wraps http.ResponseWriter to capture the status code and
// number of bytes written, so the logging middleware can report them.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.statusCode == 0 {
		// Mirrors net/http's implicit 200 when the handler never calls WriteHeader.
		r.statusCode = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// LoggingMiddleware emits one slog entry per HTTP request at INFO for 2xx/3xx
// responses, WARN for 4xx, and ERROR for 5xx. It logs method, path, status,
// duration in milliseconds, and response size — but never request or response
// bodies (sensitive data, e.g. credentials or PHI, must not land in logs).
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}

		defer func() {
			duration := time.Since(start)
			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.statusCode),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.Int("bytes", rec.bytes),
			}
			ctx := r.Context()
			switch {
			case rec.statusCode >= 500:
				slog.ErrorContext(ctx, "request completed", attrs...)
			case rec.statusCode >= 400:
				slog.WarnContext(ctx, "request completed", attrs...)
			default:
				slog.InfoContext(ctx, "request completed", attrs...)
			}
		}()

		next.ServeHTTP(rec, r)
	})
}
