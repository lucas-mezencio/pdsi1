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

## Fake Firebase Subscriber

The `fakefirebasesub` executable simulates a mobile device receiving notifications via Redis + Watermill.
It:
1) Creates a user, doctor, and two prescriptions via HTTP.
2) Prints the created entities and expected notification times.
3) Subscribes to the notification topic and prints each notification it receives.

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

## Environment

Defaults are in `.env`:
- `DATABASE_URL`
- `REDIS_ADDR`
- `HTTP_ADDR`
