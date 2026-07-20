package httpapi

import (
	_ "embed"
	"encoding/json"
	"net/http"
)

//go:embed test_notifications.html
var testNotificationsHTML []byte

// TestNotificationsConfig holds the runtime config injected into the HTML.
type TestNotificationsConfig struct {
	FirebaseWebConfig   string
	FirebaseWebVAPIDKey string
}

// TestNotificationsPage serves the browser test page.
func TestNotificationsPage(cfg TestNotificationsConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(testNotificationsHTML)
	}
}

// TestNotificationsConfigJSON returns the JSON config the HTML page fetches.
func TestNotificationsConfigJSON(cfg TestNotificationsConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// If FirebaseWebConfig is empty, return 204 so the JS treats it as "no config".
		if cfg.FirebaseWebConfig == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"firebase_config": cfg.FirebaseWebConfig,
			"vapid_key":       cfg.FirebaseWebVAPIDKey,
		})
	}
}
