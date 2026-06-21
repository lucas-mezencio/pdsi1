# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.1] - 2026-06-21

### Fixed
- `POST /auth/login` now returns the Firebase ID token (JWT) as the bearer
  token, matching the `AuthResponse{ token, user }` shape defined in the
  OpenAPI contract. Previously the handler returned the user object
  directly and discarded the `idToken` returned by the Firebase
  `signInWithPassword` API. This change:
  - Extends `AuthenticationProvider.SignIn` to also return the `idToken`.
  - Updates `AuthCommandHandler.Login` and `DoctorAuthCommandHandler.Login`
    to return both the local entity and the ID token.
  - Returns `gen.AuthResponse` from the `/auth/login` HTTP handler.
  - Clarifies the OpenAPI descriptions so the bearer token is not confused
    with the user's `firebase_token` (FCM device token for push
    notifications).
- HTTP-level and command-level regression tests assert the token is
  surfaced from login.

## [0.3.0] - 2026-06-21

### Added
- Brazilian CPF (Cadastro de Pessoa Física) field on user registration with
  full checksum validation. The field is optional to support legacy users
  but, when provided, is verified server-side via `user.ValidateCPF`.
- LGPD / GDPR right-of-access data-export endpoint that returns the
  authenticated user's personal data, including the prescribed dose
  records and any caregiver relationships.
- New `RegisterDoctorRequest`/`Doctor` schema fields surfaced through the
  generated OpenAPI types.

### Changed
- `Register` endpoint accepts and persists `cpf`; OpenAPI
  `RegisterRequest` and `User` schemas updated accordingly.

## [0.2.2] - 2026-06-21

### Fixed
- `POST /auth/login` no longer requires authentication — clients can now obtain
  a token. The auth middleware was incorrectly blocking the very request that
  issues credentials. Added regression test
  `TestAuthMiddleware_PublicPaths` to lock in the public-path bypass.

## [0.2.0] - 2026-05-31

### Added
- OpenAPI spec auth routes: `POST /auth/register`, `POST /auth/login`, `POST /auth/logout`
- `RegisterRequest` schema with fields: name, email, cpf, phone, password, role
- `LoginRequest` schema with fields: email, password
- `AuthResponse` schema with user data and timestamps
- `Auth` tag in OpenAPI spec
- Firebase auth integration test (`tests/firebaseauth_integration_test.go`)
- CLI client (`client/cli/main.go`, `client/client.gen.go`)

### Changed
- Auth handlers now generated via `oapi-codegen` (chi-server template)
- Firebase auth is stateless — stores only Firebase UID (not tokens)
- `FirebaseToken` removed from `RegisterCommand`
- Updated `golang.org/x/crypto` and `golang.org/x/net` to fix vulnerabilities
- Compose split for dev profile (traefik + mednotify + postgres + redis)
- Go version upgraded to 1.26.3
- Project renamed to "CareConnect API"

### Removed
- `ExtendedServer` auth methods (Register, Login, Doctor auth routes)
- `DoctorAuthCommandHandler` from main.go
- Doctor auth routes from router (deprecated)

## [0.1.0] - 2026-05-23

### Added

#### Domain & Business Logic
- **User relations & invitation system**: caretakers can be invited and managed
- **Dose record management**: tracking of medication doses taken
- **Medicament dosage on prescription**: structured medication data with dosage

#### Application (CQRS)
- **Commands handlers**: CreatePrescription, UpdatePrescription, CreateUser, etc.
- **Query handlers**: GetUserPrescriptions, GetDoctorPrescriptions, etc.
- **Domain tests with mutation coverage**: gremlins-based testing at 84%+

#### Infrastructure
- **PostgreSQL repositories**: user, doctor, prescription persistence with migrations
- **Redis + Watermill**: message streaming, consumer, job scheduling with lookback window
- **Firebase integration**: push notifications via Firebase Auth
- **Fake Firebase subscriber**: test tool for notification simulation

#### API Layer
- **REST handlers**: chi router with middleware (logging, auth simulation)
- **OpenAPI spec + Swagger UI**: interactive API documentation at `/docs`

#### Development Environment
- **Docker Compose + Traefik v3.7**: containerized dev environment with auto-routing
- **Makefile**: `make install`, `make compose` and other dev commands
- **Dockerfile**: multi-stage build for mednotify service

#### CI/CD
- **GitHub Actions workflow**: test, coverage, build-push with metadata extraction
- **Mutation testing with gremlins**: isolated integration tests, 98%+ coverage on application

### Fixed
- Firebase credential options (deprecated SA1019)
- Traefik HTTP entrypoint routing
- golangci-lint findings across codebase
- kin-openapi yaml compatibility and API file splitting
- Prescription array handling in repository

### Documentation
- Architecture diagrams (class, sequence, state, ER)
- Requirements mapping document
- MEZ-29 sprint scope for API gaps
- CLAUDE.md with project conventions and commands

---

[0.3.1]: https://github.com/lucas-mezencio/CareConnect/releases/tag/v0.3.1
[0.3.0]: https://github.com/lucas-mezencio/CareConnect/releases/tag/v0.3.0
[0.2.2]: https://github.com/lucas-mezencio/CareConnect/releases/tag/v0.2.2
[0.2.0]: https://github.com/lucas-mezencio/CareConnect/releases/tag/v0.2.0
[0.1.0]: https://github.com/lucas-mezencio/CareConnect/releases/tag/v0.1.0
