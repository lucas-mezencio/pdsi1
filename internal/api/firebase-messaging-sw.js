// Minimal FCM service worker for the test page. Background push
// notifications are not yet handled (deferred per design spec); this
// file exists so Firebase's default service worker registration in
// getToken() succeeds and a foreground token can be returned.
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()));