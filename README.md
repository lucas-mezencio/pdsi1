# MedNotify API

## Quick Start

Prerequisites:

- Go 1.25+
- Docker + Docker Compose

Start infrastructure:

```bash
docker compose up -d postgres redis
```

Run the API:

```bash
go run ./cmd/api
```

The API listens on `http://localhost:8080/api/v1` by default. You can change it with `HTTP_ADDR`.

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
docker compose up -d postgres redis
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
