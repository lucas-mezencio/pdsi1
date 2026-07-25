// FCM service worker for the test page. The SW is registered at the
// Firebase-mandated scope /firebase-cloud-messaging-push-scope so
// Firebase's getToken() default registration succeeds and a foreground
// token can be returned. This SW also handles incoming push events so
// the system notification is actually displayed both when the page is
// in the background and when it has been closed entirely.
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()));

self.addEventListener('push', (event) => {
  if (!event.data) return;
  let payload = {};
  try {
    payload = event.data.json();
  } catch (_) {
    payload = { notification: { title: 'Medication Reminder', body: event.data.text() } };
  }
  const n = payload.notification || {};
  const title = n.title || 'Medication Reminder';
  const options = {
    body: n.body || '',
    icon: n.icon || '/icon.png',
    badge: n.badge || '/badge.png',
    data: payload.data || {},
  };
  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const target = (event.notification.data && event.notification.data.url) || '/';
  event.waitUntil(clients.openWindow(target));
});