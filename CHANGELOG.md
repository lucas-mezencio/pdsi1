# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-05-31

### Added
- OpenAPI spec auth routes: `POST /auth/register`, `POST /auth/login`, `POST /auth/logout`
- `RegisterRequest` schema with fields: name, email, cpf, phone, password, role
- `LoginRequest` schema with fields: email, password
- `AuthResponse` schema with user data and timestamps
- `Auth` tag in OpenAPI spec
- Firebase auth integration test (`tests/firebaseauth_integration_test.go`)

### Changed
- Auth handlers now generated via `oapi-codegen` (chi-server template)
- Firebase auth is stateless — stores only Firebase UID (not tokens)
- `FirebaseToken` removed from `RegisterCommand`
- Updated `golang.org/x/crypto` and `golang.org/x/net` to fix vulnerabilities

### Removed
- `ExtendedServer` auth methods (Register, Login, Doctor auth routes)
- `DoctorAuthCommandHandler` from main.go
- Doctor auth routes from router (deprecated)