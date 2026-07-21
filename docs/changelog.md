# Changelog

All notable changes to CareConnect are documented here. Versions follow
[Semantic Versioning](https://semver.org/).

## [v0.4.0] — 2026-07-20

### Added

- **Multi-device push tokens.** A user can now register any number of device
  tokens (one per browser, phone, or tablet) and receive medication reminders
  on all of them at once.
- New `user_device_tokens` table plus a Postgres repository
  (`internal/infrastructure/database/device_tokens.go`) and migration
  `0008_user_device_tokens.sql`.
- Domain entity `internal/domain/devicetoken` with `Enabled`,
  `LastSeenAt`, `DeactivatedAt` lifecycle fields.
- Application commands:
  - `RegisterDeviceToken` — upserts a token (auto re-enables if previously
    disabled).
  - `DeleteDeviceToken` — removes a token from the caller's account.
  - `SetDeviceTokenEnabled` — toggles delivery per device.
- Application query `ListUserDeviceTokens` — returns the caller's active
  tokens.
- `notification.Lookup` port + `PostgresLookup` implementation used by the
  scheduler worker so each scheduled reminder is delivered to every enabled
  device for the target user.
- HTTP endpoints on `/api/v1/users/me/device-tokens`:
  - `POST` — register / refresh a token
  - `GET` — list the caller's tokens
  - `PATCH /{tokenID}` — enable / disable a token
  - `DELETE /{tokenID}` — delete a token
- Browser-facing test page at `/api/v1/test-notifications` that lets a logged-in
  caretaker register the current browser, fire a test push, and see which
  devices the API knows about.
- CLI binary `cmd/testnotify` that sends a push to every enabled device for a
  given user (`go run ./cmd/testnotify -user-id <uuid>`).
- Config keys `FIREBASE_WEB_CONFIG` and `FIREBASE_WEB_VAPID_KEY` so the test
  page can initialise the Firebase Web SDK without a hardcoded config.

### Changed

- Notifications are now fanned out per-device by the worker using
  `notification.Lookup`. The notification consumer (`StartNotificationConsumer`)
  receives a `Lookup` instead of relying on a single user-level token.
- Migration `0009_drop_users_firebase_token_and_notifications_enabled.sql`
  backfills existing `users.firebase_token` rows into `user_device_tokens`,
  backfills `notifications_enabled` onto the corresponding token, and drops
  the legacy columns.

### Removed

- `users.firebase_token` and `users.notifications_enabled` columns (migration
  `0009`).
- Domain fields `User.FirebaseToken` / `User.NotificationsEnabled`.
- Legacy app handlers that read or wrote those fields.
- Worker "user fallback" that sent to `user.FirebaseToken` when no device
  tokens existed; delivery now always goes through `user_device_tokens`.
- HTTP endpoints that toggled `notifications_enabled` on the user.

### Security

- Device token endpoints require a valid Firebase ID token and act only on
  the **caller's** own account (no cross-user access).

## [v0.3.0] — earlier

- Doctor CRUD, prescriptions, dose records, invitation flow, scheduler
  worker, LGPD export / delete.
