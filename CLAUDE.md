# CLAUDE.md — CareConnect

## Context

This is a medication notification system designed to help elderly users and
their caretakers manage medication schedules. The system receives prescription
data via REST API and sends timely notifications through Firebase using
Redis-backed message queuing and scheduling.

**Tech Stack:**

- Go 1.26.3
- Watermill (message streaming)
- Redis (message queue & scheduling)
- Firebase (push notifications)
- PostgreSQL (user/doctor/prescription data)

**Architecture:** Clean architecture with CQRS for event handling, simple CRUD
for user/doctor management.

**Go module:** `github.com.br/lucas-mezencio/pdsi1`

---

## Build, Lint, and Test Commands

### Go Commands

```bash
# Build the application
go build -o bin/mednotify ./cmd/api

# Build with version info
go build -ldflags="-X main.Version=$(git describe --tags)" -o bin/mednotify ./cmd/api

# Build all binaries
go build ./...

# Run the API server
go run ./cmd/api

# Run with environment variables
ENV=development go run ./cmd/api

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output and coverage report
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run a single test
go test -v -run TestFunctionName ./path/to/package

# Run tests matching a pattern
go test -v -run "^TestPrescription" ./...

# Run tests with race detection
go test -race ./...
```

### Lint Commands

```bash
# Run golangci-lint (install: https://golangci-lint.run/usage/install/)
golangci-lint run

# Run with auto-fix
golangci-lint run --fix

# Format code
go fmt ./...

# Run go vet
go vet ./...

# Check imports (install: go install golang.org/x/tools/cmd/goimports@latest)
goimports -w .
```

### Dependency Management

```bash
# Add a dependency
go get github.com/ThreeDotsLabs/watermill

# Update dependencies
go get -u ./...

# Tidy dependencies
go mod tidy

# Vendor dependencies (if needed)
go mod vendor
```

---

## Development Workflow

Use `task` for all development commands:

| Task                      | Description                                              |
| ------------------------- | -------------------------------------------------------- |
| `task` or `task validate` | Run all validations (lint + test + mutation + vulncheck) |
| `task lint`               | Run golangci-lint                                        |
| `task test`               | Run tests with race detection                            |
| `task mutation`           | Run mutation tests with gremlins                         |
| `task vulncheck`          | Check for vulnerabilities                                |
| `task setup`              | Install quality gate tools                               |

### Test Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output and coverage report
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run a single test
go test -v -run TestFunctionName ./path/to/package

# Run tests matching a pattern
go test -v -run "^TestPrescription" ./...

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

### TDD Enforcement

**For every feature/bug fix:**

1. Write a failing test first (`_test.go` file)
2. Run tests to verify it fails
3. Implement the minimum code to pass
4. Refactor with confidence

Use `Skill: test-driven-development` for TDD guidance.

### Mutation Testing

**Gotcha: gremlins cache corruption** — Running gremlins twice on the same
package causes TIMED OUT mutations. Always clean test cache first:

```bash
go clean -testcache && gremlins unleash ./internal/domain/
```

Target: maintain 80%+ mutation coverage. Current status: `domain/` 84.62% ✅,
`application/` 98.67% ✅

See `.agents/memory/mutation-tests.md` for details.

### Development Loop

For iterative work (bug fixes, incremental features), use
`Skill: development-loop`:

- Make a small change
- Run tests to verify
- Commit if passing
- Repeat

This prevents scope creep and keeps PRs small.

---

## Project Structure

**Context:** This repository uses git worktrees. Each worktree folder (e.g.,
`requirements/`, `docs/`) is a separate working directory — the project root is
the worktree root, not a fixed directory name.

```
<worktree_dir_name>/          # ← This worktree dir IS the project root
├── cmd/
│   ├── api/              # Main application entry point
│   │   └── main.go
│   └── fakefirebasesub/  # Firebase notification simulator (testing)
├── internal/
│   ├── domain/           # Domain models and business logic
│   │   ├── user/
│   │   ├── doctor/
│   │   └── prescription/
│   ├── application/      # Application services (CQRS)
│   │   ├── commands/     # Write operations
│   │   └── queries/      # Read operations
│   ├── infrastructure/   # External dependencies
│   │   ├── database/     # PostgreSQL repositories
│   │   ├── messaging/    # Watermill/Redis setup
│   │   ├── notification/ # Firebase integration
│   │   └── scheduler/    # Redis-based scheduling
│   ├── config/           # Configuration loading
│   └── api/              # HTTP handlers and routing
│       ├── handlers/
│       ├── middleware/
│       └── routes/
├── migrations/           # Database migrations
├── tests/                # Integration and E2E tests
└── docs/
    ├── diagrams/         # PlantUML architecture docs
    └── pdsi2/            # Sprint documentation
```

---

## Superpowers & Development Flow

**Mandatory skill usage for all non-trivial tasks:**

| Skill                            | When to Use                        | Purpose                                |
| -------------------------------- | ---------------------------------- | -------------------------------------- |
| `brainstorming`                  | Before any implementation task     | Design & requirements clarification    |
| `writing-plans`                  | After brainstorming, before coding | Implementation plan with steps         |
| `test-driven-development`        | During implementation              | Red-green-refactor loop                |
| `development-loop`               | Iterative work, bug fixes          | Incremental progress with verification |
| `verification-before-completion` | Before marking done                | Final quality gate                     |

**The core workflow for new features:**

```
User request → [brainstorming] → [writing-plans] → [TDD implementation] → [verification] → PR
```

1. **Don't implement immediately.** Use `Skill: brainstorming` first to clarify
   requirements and design
2. **After brainstorming, use `Skill: writing-plans`** to create an
   implementation plan
3. **Break the plan into the smallest possible steps** — each PR should be
   ~20-50 lines
4. **Use TDD**: write a failing test first, then implement to make it pass
5. **Before marking done**: use `Skill: verification-before-completion`

### Task Sizing Rules

- A PR should do ONE thing well
- If a task takes more than 2-3 hours of work, it's too large — break it down
- Each commit should be independently meaningful
- "Add feature X" is too large. "Add repository for X", "Add service for X",
  "Add handler for X" are separate tasks

**How to invoke skills:**

```
Skill: <skill-name>
```

e.g., `Skill: brainstorming` activates the brainstorming skill.

Full skill list at: `~/.claude/skills/` or ask `/find-skills` to discover.

---

## Code Style Guidelines

### General Principles

- Follow standard Go conventions (Effective Go, Go Code Review Comments)
- Keep functions small and focused (single responsibility)
- Prefer composition over inheritance
- Use interfaces for abstraction, especially at boundaries
- Write self-documenting code; comments explain "why", not "what"

### Imports

**Order:** Standard library → Third-party → Internal packages

```go
import (
    // Standard library
    "context"
    "fmt"
    "time"

    // Third-party
    "github.com/ThreeDotsLabs/watermill"
    "github.com/go-chi/chi/v5"

    // Internal
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
    "github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/messaging"
)
```

Use `goimports` to automatically organize imports.

### Naming Conventions

- **Packages:** Short, lowercase, single word (e.g., `prescription`, `user`,
  `messaging`)
- **Files:** Lowercase with underscores (e.g., `prescription_service.go`,
  `user_repository.go`)
- **Types:** PascalCase (e.g., `PrescriptionService`, `UserRepository`)
- **Interfaces:** PascalCase, often ending in `-er` (e.g., `Notifier`,
  `Scheduler`, `Repository`)
- **Functions/Methods:** PascalCase for exported, camelCase for unexported
- **Variables:** camelCase (e.g., `userID`, `prescriptionRepo`)
- **Constants:** PascalCase or SCREAMING_SNAKE_CASE for package-level

### Types and Interfaces

```go
// Prefer explicit types for domain models
type Prescription struct {
    ID          string       `json:"id"`
    UserID      string       `json:"user_id"`
    MedicID     string       `json:"medic_id"`
    Medicaments []Medicament `json:"medicaments"`
    CreatedAt   time.Time   `json:"created_at"`
}

// Use interfaces for dependencies
type NotificationSender interface {
    Send(ctx context.Context, userID string, message string) error
}

// Accept interfaces, return structs
func NewPrescriptionService(repo Repository, notifier NotificationSender) *PrescriptionService {
    return &PrescriptionService{repo: repo, notifier: notifier}
}
```

### Error Handling

```go
// Always check errors
result, err := someOperation()
if err != nil {
    return fmt.Errorf("failed to perform operation: %w", err)
}

// Use %w for error wrapping (Go 1.13+)
if err := validatePrescription(p); err != nil {
    return fmt.Errorf("prescription validation failed: %w", err)
}

// Define custom errors for domain logic
var (
    ErrPrescriptionNotFound = errors.New("prescription not found")
    ErrInvalidMedicament    = errors.New("invalid medicament data")
)

// Use errors.Is and errors.As for error checking
if errors.Is(err, ErrPrescriptionNotFound) {
    // Handle not found
}
```

### Context Usage

- Always pass `context.Context` as the first parameter
- Use context for cancellation, timeouts, and request-scoped values
- Don't store contexts in structs

```go
func (s *PrescriptionService) Create(ctx context.Context, p *Prescription) error {
    // Use context for database operations
    return s.repo.Save(ctx, p)
}
```

### Testing

```go
// Test file naming: *_test.go
// Test function naming: TestFunctionName or TestType_Method

func TestPrescriptionService_Create(t *testing.T) {
    // Arrange
    repo := &mockRepository{}
    service := NewPrescriptionService(repo, nil)

    // Act
    err := service.Create(context.Background(), &Prescription{})

    // Assert
    if err != nil {
        t.Errorf("expected no error, got %v", err)
    }
}

// Use table-driven tests for multiple scenarios
func TestValidateMedicament(t *testing.T) {
    tests := []struct {
        name    string
        input   Medicament
        wantErr bool
    }{
        {"valid medicament", Medicament{Name: "Aspirin", Dosage: "100mg"}, false},
        {"missing name", Medicament{Dosage: "100mg"}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateMedicament(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## Architecture Patterns

### CQRS for Events

- **Commands:** Write operations that change state (e.g.,
  `CreatePrescriptionCommand`)
- **Queries:** Read operations (e.g., `GetPrescriptionQuery`)
- Keep command and query models separate

### Dependency Injection

- Use constructor injection for dependencies
- Avoid global state and singletons
- Wire dependencies in `main.go`

### Repository Pattern

```go
type PrescriptionRepository interface {
    Save(ctx context.Context, p *Prescription) error
    FindByID(ctx context.Context, id string) (*Prescription, error)
    FindByUserID(ctx context.Context, userID string) ([]*Prescription, error)
}
```

---

## Message Queue & Scheduling

### Watermill Usage

- Use Watermill for publishing prescription events
- Publish events when prescriptions are created/updated
- Consume events to schedule notifications

### Redis Scheduling

- Use Redis sorted sets for scheduling notifications
- Store notification jobs with timestamp as score
- Worker polls Redis for due notifications

---

## Common Pitfalls to Avoid

- Don't ignore errors (use `golangci-lint` with `errcheck`)
- Don't use `panic` in library code (only in `main` for unrecoverable errors)
- Avoid naked returns in functions longer than a few lines
- Don't use `init()` functions unless absolutely necessary
- Avoid goroutine leaks (always ensure goroutines can exit)
- Don't pass pointers to slices or maps (they're already references)

---

## Additional Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Watermill Documentation](https://watermill.io/)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)

## Gotchas

- **gremlins cache corruption**: Running gremlins twice on the same package
  causes TIMED OUT mutations. Always clean test cache first:
  `go clean -testcache && gremlins unleash ./internal/domain/`
- **Firebase SDK deprecation**: See `README.md` for credential handling
  workaround. Key files:
  - `internal/infrastructure/firebaseauth/service.go:40`
  - `internal/infrastructure/notification/firebase_sender.go:28`

---

## PR Workflow

1. Commit with descriptive messages
2. Push branch: `git push -u origin HEAD`
3. Create PR: `gh pr create --fill --base main`
4. Move issue to 'in_review' only after PR is up

---

## Skill Discovery

If you need a skill for a task (e.g., "debug this", "design this", "review
this"):

- Use `Skill: find-skills` to discover available skills
- Or check `~/.claude/skills/` for full list

**Never guess if a skill exists.** Use find-skills first.

---

## Knowledge Graph

This project has a persistent knowledge graph at `graphify-out/graph.json`.

**MCP Server:** The graph is available as an MCP server via the `graphify` tool
in `.claude/settings.json`. Claude Code automatically has graph query tools
available — no manual commands needed.

**When planning changes:**
- Query affected components: `graphify query "what depends on User struct?"`
- Trace impact: `graphify path "UserRepository" "NotificationSender"`

**Do NOT** grep/read files to answer dependency questions — use the graph.

---

## Memory

Check [.agents/memory/](.agents/memory/) folder for project memory, TODOs, and
investigation notes.

**Important:** `.agents/memory/` is local-only. Do not commit it — add it to
`.gitignore`.

