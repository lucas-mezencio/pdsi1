package httpapi

import (
	_ "embed"
	"encoding/json"
	"net/http"
)

//go:embed test_notifications.html
var testNotificationsHTML []byte

//go:embed firebase-messaging-sw.js
var firebaseMessagingSW []byte

// FirebaseMessagingServiceWorker serves the minimal FCM service worker
// so Firebase's default SW registration succeeds. The Service-Worker-Allowed
// header is required because Firebase registers the SW at scope
// /firebase-cloud-messaging-push-scope but the file lives at the root.
func FirebaseMessagingServiceWorker() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Service-Worker-Allowed", "/")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(firebaseMessagingSW)
	}
}

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

// TestNotificationsConfigJSON returns the parsed Firebase config object plus
// the VAPID key as JSON. If FIREBASE_WEB_CONFIG is unset the endpoint returns
// 204 so the page can detect "not configured" without a JSON parse failure.
// If the env value is non-empty but malformed, returns 500 with the parse
// error so misconfiguration is visible immediately.
func TestNotificationsConfigJSON(cfg TestNotificationsConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.FirebaseWebConfig == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(cfg.FirebaseWebConfig), &parsed); err != nil {
			http.Error(w, "invalid FIREBASE_WEB_CONFIG: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"firebase_config": parsed,
			"vapid_key":       cfg.FirebaseWebVAPIDKey,
		})
	}
}
