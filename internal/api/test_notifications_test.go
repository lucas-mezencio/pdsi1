package httpapi

import (
	"strings"
	"testing"
)

// Regression guard for the bug where FCM messages reached the browser but
// no notification was displayed because the service worker had no `push`
// event handler and the test page had no `onMessage` callback.
//
// Both behaviors are tested by asserting the served assets contain the
// required registration calls. The browser-side runtime behavior (actually
// displaying the notification) is verified manually end-to-end against
// the running stack; these tests exist to prevent the JS handlers from
// being silently dropped in a future change.
func TestFirebaseMessagingSW_HasPushHandler(t *testing.T) {
	src := string(firebaseMessagingSW)
	if !strings.Contains(src, "addEventListener('push'") &&
		!strings.Contains(src, `addEventListener("push"`) {
		t.Errorf("firebase-messaging-sw.js must register a 'push' event listener "+
			"so background notifications are displayed; current SW source:\n%s", src)
	}
}

func TestFirebaseMessagingSW_HasNotificationClickHandler(t *testing.T) {
	src := string(firebaseMessagingSW)
	if !strings.Contains(src, "addEventListener('notificationclick'") &&
		!strings.Contains(src, `addEventListener("notificationclick"`) {
		t.Errorf("firebase-messaging-sw.js must register a 'notificationclick' "+
			"event listener so clicks dismiss the notification; current SW source:\n%s", src)
	}
}

func TestFirebaseMessagingSW_ShowNotificationCall(t *testing.T) {
	src := string(firebaseMessagingSW)
	if !strings.Contains(src, "showNotification(") {
		t.Errorf("firebase-messaging-sw.js 'push' handler must call "+
			"self.registration.showNotification(...) to display the system "+
			"notification; current SW source:\n%s", src)
	}
}

func TestTestNotificationsHTML_HasOnMessageHandler(t *testing.T) {
	src := string(testNotificationsHTML)
	if !strings.Contains(src, "onMessage(") {
		t.Errorf("test_notifications.html must register an onMessage(...) "+
			"callback on the Firebase Messaging instance so foreground "+
			"notifications are displayed; current HTML source:\n%s", src)
	}
}

func TestTestNotificationsHTML_OnMessageCallsNotificationAPI(t *testing.T) {
	src := string(testNotificationsHTML)
	onIdx := strings.Index(src, "onMessage(")
	if onIdx < 0 {
		t.Fatalf("test_notifications.html is missing onMessage(...) registration")
	}
	tail := src[onIdx:]
	if !strings.Contains(tail, "new Notification(") {
		t.Errorf("test_notifications.html onMessage callback must call "+
			"new Notification(...) to display the system notification; "+
			"current HTML tail after onMessage:\n%s", tail)
	}
}