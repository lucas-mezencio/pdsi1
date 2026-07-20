# Design: Multi-device FCM Tokens for CareConnect

**Date:** 2026-07-20
**Status:** Approved (design)
**Scope:** Multi-device FCM tokens, per-device enable/disable, HTML test page, test CLI

## 1. Goals

Allow each user to register N device tokens (phone, tablet, browser tab) for
medication reminders. Per-device enable/disable. Drop the legacy single
`users.firebase_token` column and `users.notifications_enabled` flag. Provide a
small HTML page + CLI to exercise the full loop end-to-end.

## 2. Decisions recap

| #  | Decision                                    | Choice                                                           |
|----|---------------------------------------------|------------------------------------------------------------------|
| 1  | Legacy `users.firebase_token`               | Replace — data-migrated into the new table, then column dropped  |
| 2  | Enable/disable scope                        | Per-device only — drop `users.notifications_enabled`             |
| 3  | Stored metadata                             | Minimal: id, user_id, token, enabled, timestamps, last_used_at   |
| 4  | Scheduler worker                            | Updated in this change to read active tokens via a new port     |
| 5  | HTML page scope                             | Full demo: Firebase Web SDK + login + all CRUD buttons           |
| 6  | GET own path                                | `/users/me/device-tokens` (matches existing `/users/me/data-export`) |
| 7  | Test page auth                              | Email/password via existing `/auth/login`                        |
| 8  | Test trigger                                | Go CLI at `cmd/testnotify/` (no HTTP trigger endpoint)           |

## 3. Data layer

New table `user_device_tokens`:

```text
id            UUID PRIMARY KEY
user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
token         TEXT NOT NULL UNIQUE
enabled       BOOLEAN NOT NULL DEFAULT TRUE
created_at    TIMESTAMPTZ NOT NULL
updated_at    TIMESTAMPTZ NOT NULL
last_used_at  TIMESTAMPTZ NULL
```

Indexes on `user_id` and `token`. The `UNIQUE` on `token` already covers token
lookups; the additional index on `user_id` keeps `FindByUserID` fast.

Migration split:

- `migrations/0008_user_device_tokens.sql` — create table.
- `migrations/0009_drop_users_firebase_token_and_notifications_enabled.sql`
  — backfill existing tokens (one row per non-null `firebase_token`,
  `enabled = true`), then `DROP COLUMN` both legacy columns. Idempotent via
  `IF EXISTS` guards and a `WHERE NOT EXISTS` check on the backfill.

## 4. Domain & application

New package `internal/domain/devicetoken/` with `DeviceToken` struct and
`Repository` interface:

```go
type Repository interface {
    Save(ctx context.Context, t *DeviceToken) error            // upsert by token
    FindByID(ctx context.Context, id string) (*DeviceToken, error)
    FindByUserID(ctx context.Context, userID string) ([]*DeviceToken, error)
    Delete(ctx context.Context, id string) error
    SetEnabled(ctx context.Context, id string, enabled bool) (*DeviceToken, error)
}
```

`Save` upserts by token. Behavior:

- Token does not exist → insert a new row with `enabled = true`.
- Token exists for the same `user_id` → idempotent re-registration: return the
  existing row with `enabled = true`, `updated_at` bumped.
- Token exists for a different `user_id` → return `ErrConflict` (HTTP 409).
  Reassigning silently would kick the previous owner off their device, which
  is surprising. The caller can decide how to handle the conflict.

New `internal/application/commands/devicetoken_commands.go` and
`internal/application/queries/devicetoken_queries.go` with handlers:

- `RegisterDeviceTokenCommand { CallerID, Token }` → upserts; sets `enabled=true`.
- `ListDeviceTokensQuery { CallerID }` → returns tokens where `user_id = CallerID`.
- `DeleteDeviceTokenCommand { CallerID, TokenID }` → 403 if token belongs to
  another user, 404 if missing.
- `SetDeviceTokenEnabledCommand { CallerID, TokenID, Enabled }` → same
  ownership check.

All commands return `application.ErrForbidden` / `ErrNotFound` /
`ErrInvalidInput` so the existing `writeExtendedError` mapper handles status
codes.

## 5. HTTP endpoints

Manually registered in `internal/api/extended_server.go` and
`internal/api/router.go`. All four routes live under the existing
`r.Route("/api/v1", ...)` block alongside `/users/me/data-export`.

| Method   | Path                                                          | Auth        | Body                  | Response                          |
|----------|---------------------------------------------------------------|-------------|-----------------------|-----------------------------------|
| `POST`   | `/api/v1/users/me/device-tokens`                              | required    | `{"token": "..."}`    | `201` + `DeviceTokenResponse`     |
| `GET`    | `/api/v1/users/me/device-tokens`                              | required    | —                     | `200` + `[]DeviceTokenResponse`   |
| `DELETE` | `/api/v1/users/me/device-tokens/{tokenId}`                    | own-only    | —                     | `204`                             |
| `PATCH`  | `/api/v1/users/me/device-tokens/{tokenId}/enabled`            | own-only    | `{"enabled": bool}`   | `200` + `DeviceTokenResponse`     |

`DeviceTokenResponse` deliberately omits the raw `token` field (security —
matches the existing `firebase_token` redaction policy in
`internal/api/dto/response.go`).

Errors flow through `writeExtendedError` (400 / 403 / 404 / 409 / 500).

## 6. Scheduler worker

New port `internal/infrastructure/notification/lookup.go`:

```go
type Lookup interface {
    ActiveTokens(ctx context.Context, userID string) ([]Token, error)
}
type Token struct {
    DeviceTokenID string
    FCMToken      string
}
```

`PostgresLookup` implementation wraps the new repo and returns only rows with
`enabled = true`.

Worker at `internal/infrastructure/scheduler/worker.go` is updated to iterate
active tokens for the primary user, then for each linked caregiver, calling
`sender.Send` per token. Behavior preserved:

- Primary user send failure still nacks the Watermill message.
- Caregiver send failure remains non-fatal (logged only).
- `last_used_at` is bumped on successful send (best-effort; non-blocking
  error log on failure).

The `notification.Notification.FirebaseToken` field stays — the sender still
receives one token per call.

## 7. HTML test page

`GET /api/v1/test-notifications` serves `internal/api/test_notifications.html`
(embedded with `//go:embed`). Single-page JS, no framework. Firebase JS SDK
loaded from CDN with the project web config (new `FIREBASE_WEB_CONFIG` env
var, JSON string) and VAPID key (new `FIREBASE_WEB_VAPID_KEY` env var).

Flow:

1. Login form → `POST /auth/login` → store bearer token.
2. Initialize Firebase Web app with the provided config.
3. Request `Notification.permission`, then `getToken({ vapidKey })`.
4. POST the FCM token to `/users/me/device-tokens`.
5. Render the user's token list with toggle / delete buttons that hit the
   corresponding endpoints.
6. "Send test notification" button copies
   `cmd/testnotify -user-id <me> -medicament "..." -dosage "..."` to the
   clipboard, since the actual push is fired by the CLI, not an HTTP endpoint.

The route is registered unconditionally for this iteration. A code comment
notes that production gating behind a config flag is deferred.

## 8. CLI: `cmd/testnotify/main.go`

Standalone Go binary in the same module. Reads the same `DATABASE_URL`,
`FIREBASE_CREDENTIALS_FILE`, and `NOTIFIER_MODE` as the API.

```
go run ./cmd/testnotify -user-id <uuid> -medicament "Aspirin" -dosage "100mg"
```

Behavior:

1. Open DB, run migrations (reuse `database.Migrate`).
2. Build `deviceTokenRepo`, `lookup`, and a `FirebaseSender` from the same
   config the API uses.
3. Iterate `ActiveTokens(ctx, userID)`, call `sender.Send` for each enabled
   token.
4. Print a per-token result table; non-zero exit on any failure.

Reuses the existing `notification.Sender` and the new `notification.Lookup`
port — no duplicated logic.

## 9. Dependency wiring (`cmd/api/main.go`)

```go
deviceTokenRepo  := database.NewDeviceTokenRepository(db)
lookup           := notification.NewPostgresLookup(deviceTokenRepo)
userCommands     := commands.NewUserCommandHandler(userRepo, authProvider)
deviceTokenCmds  := commands.NewDeviceTokenCommandHandler(deviceTokenRepo, userRepo)
deviceTokenQs    := queries.NewDeviceTokenQueryHandler(deviceTokenRepo)

extServer := httpapi.NewExtendedServer(
    userRepo, authCommands, inviteCommands,
    doseCommands, doseQueries,
    linkedUserQueries, lgpdQueries,
    deviceTokenCmds, deviceTokenQs,
)

StartNotificationConsumer(gCtx, subscriber, sender, userRepo, lookup, cleanup)
```

The `extended_server.go` constructor gains two more arguments.

## 10. Configuration (`internal/config/config.go`)

New env vars:

- `FIREBASE_WEB_CONFIG` — JSON string with the Firebase web app config
  (`apiKey`, `authDomain`, `projectId`, `appId`, `messagingSenderId`).
- `FIREBASE_WEB_VAPID_KEY` — public VAPID key for FCM web push.

Both default to empty. When `FIREBASE_WEB_CONFIG` is empty, the test page
shows an explanatory message instead of trying to initialize Firebase.

## 11. Testing

| Layer             | Coverage                                                                                          |
|-------------------|---------------------------------------------------------------------------------------------------|
| Domain            | enable / disable methods, validation (table-driven)                                                |
| Application       | handler unit tests with handwritten mocks (existing pattern)                                      |
| Infrastructure    | Testcontainers Postgres CRUD + `PostgresLookup`                                                    |
| API               | `httptest` handler tests; own-only enforcement; token-not-leaked-in-response                       |
| Scheduler         | worker integration with fake sender + in-memory lookup                                            |
| Migration         | golden test: empty users table, and one populated row that migrates correctly                      |
| Regression        | extend `firebase_token_leak_test.go` to assert neither legacy column is exposed                    |

## 12. Out of scope

- Auto-cleanup of stale tokens on Firebase `UNREGISTERED` errors (deferred).
- LGPD export of device tokens (legal review pending).
- Production gating of `/test-notifications` (kept unconditional for dev;
  comment added).
- Bulk delete / bulk enable endpoints.
- Per-device platform / browser metadata.