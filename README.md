# CareConnect API

## Quick Start

Prerequisites:

- Go 1.25+
- Docker + Docker Compose

Start infrastructure:

```bash
make compose/infra
```

Run the API:

```bash
go run ./cmd/api
```

The API listens on `http://localhost:8080/api/v1` by default. You can change it with `HTTP_ADDR`.

## Development Tools

This project uses [task](github.com/go-task/task) for task automation.

### Install Tools

```bash
make install/tools
```

This installs:
- [golangci-lint](https://golangci-lint.run/) — multi-purpose linter
- [gremlins](https://github.com/go-gremlins/gremlins) — mutation testing
- [govulncheck](https://golang.org/x/vuln/cmd/govulncheck) — vulnerability scanning

### Available Tasks

```bash
task -l
```

| Task       | Description                          |
|------------|--------------------------------------|
| `task`     | Runs `task validate` (default)       |
| `task setup` | Install quality gate tools          |
| `task lint` | Run golangci-lint                   |
| `task test` | Run all tests with race detection  |
| `task mutation` | Run mutation tests            |
| `task vulncheck` | Run vulnerability scan       |
| `task validate` | Run all validations (lint + test + mutation + vulncheck) |

### Docker Compose

All Docker Compose operations go through the Makefile:

```bash
make compose/build   # Build and start dev profile
make compose/up      # Start dev profile
make compose/down    # Stop containers
make compose/logs    # Follow logs
make compose/infra   # Start postgres and redis only
```

## Notifications

The service supports two notifier modes:

- `dev`: uses a dummy sender that prints notifications to stdout.
- `ready` (or empty): sends notifications via Firebase Cloud Messaging.

Set in `.env`:

```bash
NOTIFIER_MODE=dev
FIREBASE_CREDENTIALS_FILE=/path/to/firebase-service-account.json
```

### Real FCM Test Example

Prereqs:

- A valid FCM device token
- A Firebase service account JSON file

Run the API with Firebase enabled:

```bash
NOTIFIER_MODE=ready FIREBASE_CREDENTIALS_FILE=/path/to/firebase-service-account.json go run ./cmd/api
```

Then create a user/doctor/prescription that fires in seconds (replace `<FCM_TOKEN>`):

```bash
USER_ID=$(curl -s -X POST http://localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"FCM User","email":"fcm-user@example.com","phone":"+1000000000","firebase_token":"<FCM_TOKEN>"}' | jq -r '.id')

DOCTOR_ID=$(curl -s -X POST http://localhost:8080/api/v1/doctors \
  -H 'Content-Type: application/json' \
  -d '{"name":"FCM Doctor","email":"fcm-doc@example.com","phone":"+1000000001","license_number":"LIC-9999"}' | jq -r '.id')

START_TIME=$(date -u -d '+10 seconds' +%H:%M:%S)
curl -s -X POST http://localhost:8080/api/v1/prescriptions \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"'"$USER_ID"'","medic_id":"'"$DOCTOR_ID"'","medicaments":[{"name":"FCM Med","dosage":"1","frequency":"00:00:05","time":["'"$START_TIME"'"],"doses":2}]}'
```

If everything is wired correctly, your device should receive a notification within ~10 seconds.

## Fake Firebase Subscriber

The `fakefirebasesub` executable simulates a mobile device receiving notifications via Redis + Watermill.
It:

1. Creates a user, doctor, and two prescriptions via HTTP.
2. Prints the created entities and expected notification times.
3. Subscribes to the notification topic and prints each notification it receives.

Run it in another terminal while the API is running:

```bash
go run ./cmd/fakefirebasesub
```

It seeds two schedules:

- Starts 3 seconds from now, 10 doses, every 1 second.
- Starts 10 seconds from now, 12 doses, every 5 seconds.

You can override the API URL with:

```bash
API_BASE_URL=http://localhost:8080/api/v1 go run ./cmd/fakefirebasesub
```

## Integration Test (build tag)

The integration test that wires API + Postgres + Redis lives under `tests/` and only runs with a build tag:

```bash
go test -tags=integration ./tests -run TestFakeFirebaseIntegration
```

## Environment

Defaults are in `.env`:

- `DATABASE_URL`
- `REDIS_ADDR`
- `HTTP_ADDR`
- `FIREBASE_CREDENTIALS_FILE`
- `FIREBASE_WEB_API_KEY`

## Firebase Auth (email/password)

Auth endpoints:

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`

`register` creates the account in Firebase Authentication and stores the user in Postgres with `firebase_id`.
`login` validates credentials in Firebase and returns the linked local user.

Examples:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Maria","email":"maria@example.com","phone":"+5534999999999","password":"Password123!","role":"ELDERLY"}'

curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"maria@example.com","password":"Password123!"}'
```

## ⚠️ Firebase Credential Handling — Pending Review

**Status:** Workaround applied. Requires analysis of actual `firebase.google.com/go` SDK behavior.

### What happened

The `google.golang.org/api/option` package deprecated:
- `option.WithCredentialsFile()` — SA1019 warning
- `option.WithCredentialsJSON()` — SA1019 warning

Reason cited: "potential security risk" (no credential type validation).

### Current workaround

Using `option.WithAuthCredentialsJSON(option.ServiceAccount, credentialsJSON)` which validates the JSON is a service account before loading. This was applied in:

- `internal/infrastructure/firebaseauth/service.go:40`
- `internal/infrastructure/notification/firebase_sender.go:28`

### What needs to be reviewed

The user should analyze the current `firebase.google.com/go/v4` SDK to determine the **canonical, non-deprecated way** to provide credentials. Options to investigate:

1. **Application Default Credentials (ADC)** — Let the SDK auto-discover credentials via `GOOGLE_APPLICATION_CREDENTIALS` env var or metadata server (best for GCP deployments)
2. **Check for newer non-deprecated `option` functions** — The SDK may have introduced alternative APIs since this code was written
3. **Firebase-specific credential loading** — There may be a `firebase.NewApp` configuration that doesn't rely on the generic `google.golang.org/api/option` package

### Impact if not fixed

The current workaround suppresses the linter warning but may not be the intended long-term approach. The Firebase SDK's migration path may involve ADC-only initialization or Firebase-specific credential helpers.
