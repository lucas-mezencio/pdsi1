# Design: Prescription Demo CLI (Bubbletea + GitHub Action)

**Date:** 2026-07-30
**Status:** Approved (design)
**Scope:** Standalone bubbletea CLI that drives the prescription create flow
end-to-end (login → form → confirm → submit), plus a GitHub Action that
cross-compiles the binary for easy download.

## 1. Goals

Provide a friendly, terminal-only demo runner for the medication notification
system. The CLI logs in as a real user, prompts for a single medicament
(nicely styled with bubbletea), and POSTs a `prescription` to the public demo
endpoint using the static `DEMO_PRESCRIPTION_SECRET`. A CI workflow
cross-compiles the binary on every push to `main` and on manual dispatch,
publishing Linux + Windows artifacts (and a GitHub Release on demand) so the
team can grab a binary without installing Go.

## 2. Decisions recap

| #  | Decision                              | Choice                                                                          |
|----|---------------------------------------|---------------------------------------------------------------------------------|
| 1  | Binary location                       | `cmd/prescriptioncli/` (new, sibling of `cmd/api/`, `cmd/fakefirebasesub/`)     |
| 2  | Reuse existing `client/cli`?          | No — separate binary, no shared token storage                                    |
| 3  | UI library                            | `bubbletea` + `bubbles` + `lipgloss` (no `huh` — keep vanilla bubbletea)        |
| 4  | Stage model                           | Single `tea.Model` with stages: login → form → confirm → submitting → done      |
| 5  | HTTP layer                            | `net/http` directly (login is unauthenticated; create needs only the raw secret)|
| 6  | Hardcoded medic fallback              | Constant UUID with `TODO` comment; overridable via `-medic-id` flag             |
| 7  | Auth header for `POST /prescriptions` | Raw `DEMO_PRESCRIPTION_SECRET` (NOT `Bearer …`) — matches middleware            |
| 8  | Time offset                           | `+3h` applied to user-entered start time before submit                          |
| 9  | Doses prompt                          | Always prompt (per user)                                                        |
| 10 | Tests                                 | HTTP layer only with `httptest`; UI is hand-verified                            |
| 11 | Branch                                | `feat/prescription-cli`                                                         |
| 12 | CI trigger                            | Push to `main` (build only) + `workflow_dispatch` with `release` input          |
| 13 | CI matrix                             | `ubuntu-latest` + `windows-latest`, Go 1.26.3                                   |
| 14 | CI release behavior                   | Manual dispatch with `release=true` uploads both binaries to a GH Release      |

## 3. CLI surface

```
prescriptioncli \
  -secret   <DEMO_PRESCRIPTION_SECRET>          # required, raw value (no Bearer)
  -api      http://localhost:8080/api/v1        # default matches client/cli
  -medic-id <UUID>                              # optional, falls back to const
```

- `-secret` is required, no env fallback (explicit per user requirement).
- `-api` defaults to `http://localhost:8080/api/v1` (matches `client/cli/main.go:62`).
- `-medic-id` optional; falls back to package-level `defaultMedicID` constant.

```go
// TODO: replace with the UUID you seeded for Dr. Test Silva
// (the doctor row must already exist in postgres before this CLI can submit).
const defaultMedicID = "11111111-1111-1111-1111-111111111111"
```

## 4. File layout

```
cmd/prescriptioncli/
├── main.go                  # parse flags, build deps, run tea program
├── model.go                 # root tea.Model + stage enum + Init/Update/View
├── screens.go               # login/form/confirm/submitting/done models
├── api.go                   # Login, CreatePrescription (net/http + JSON)
├── api_test.go              # httptest-driven tests for api.go
└── styles.go                # lipgloss styles + helpers

.github/workflows/
└── build-prescription-cli.yml
```

## 5. Bubbletea flow

Stages as a typed enum:

```go
type stage int
const (
    stageLogin stage = iota
    stageForm
    stageConfirm
    stageSubmitting
    stageDone
)
```

Each stage owns its own `update(msg) (model, tea.Cmd)` and `view() string`. The
root `Model` holds a `current stage` plus per-stage state. Reuses
`bubbles/textinput` (with `EchoPassword` for the password field) and
`bubbles/spinner`.

| Stage        | Behavior                                                                          |
|--------------|-----------------------------------------------------------------------------------|
| `login`      | email + password textinputs; submit → `api.Login`; 401 → red inline error          |
| `form`       | 5 inputs: name, dose, frequency (HH:MM), start time (HH:MM), doses (int)          |
| `confirm`    | `lipgloss`-styled summary box; `y` → submit, `n`/esc → back to form               |
| `submitting` | spinner + "Sending prescription…"; awaits `api.CreatePrescription` result         |
| `done`       | green: prescription id + scheduled times; red: error w/ retry hint; `q` to quit   |

Inline validation stays on the offending field:
- `frequency` and `start time` must match `^([01]\d|2[0-3]):[0-5]\d$`.
- `doses` must be `> 0`.

## 6. HTTP layer

`api.go` exposes:

```go
type API struct {
    BaseURL string
    Secret  string
    HTTP    *http.Client
}

func (a *API) Login(ctx context.Context, email, pw string) (userID string, err error)
func (a *API) CreatePrescription(ctx context.Context, p prescription) (*prescriptionResp, error)
```

- `Login` → `POST /auth/login` with JSON `{email, password}`; decodes `id` field.
- `CreatePrescription` → `POST /prescriptions` with raw header
  `Authorization: <a.Secret>` (literal equality check in
  `internal/api/middleware.go:88`). JSON body:

```json
{
  "user_id": "<loginResp.id>",
  "medic_id": "<flag or defaultMedicID>",
  "medicaments": [{
    "name":      "<form>",
    "dosage":    "<form>",
    "frequency": "<form>",
    "time":      ["<form startTime + 3h, formatted HH:MM>"],
    "doses":     <form>
  }]
}
```

## 7. +3h time offset

Pure string-level operation: parse `HH:MM` into a `time.Date` on a fixed
reference date, add `3 * time.Hour`, format back as `HH:MM`. Wraps midnight
correctly (`23:30 + 3h = 02:30`).

```go
func shiftHHMM(s string, delta time.Duration) (string, error) {
    t, err := time.Parse("15:04", s)
    if err != nil { return "", err }
    return t.Add(delta).Format("15:04"), nil
}
```

## 8. Testing

`api_test.go` uses `httptest.NewServer`:
- `Login`: 200 returns user id; 401 surfaces wrapped error.
- `CreatePrescription`: 201 returns id; verifies request body matches expected
  JSON; verifies raw `Authorization` header equals secret (NOT `Bearer secret`).
- 400 propagates server `error` + `details`.

UI tests skipped — view is hand-verified in the manual smoke test below.

## 9. GitHub Action

`.github/workflows/build-prescription-cli.yml`:

```yaml
name: build-prescription-cli
on:
  push:
    branches: [main]
    paths:
      - "cmd/prescriptioncli/**"
      - "go.mod"
      - "go.sum"
      - ".github/workflows/build-prescription-cli.yml"
  workflow_dispatch:
    inputs:
      release:
        description: "Publish a GitHub Release with the binaries"
        type: boolean
        default: false
```

Jobs:
1. `build` matrix `ubuntu-latest` + `windows-latest`, Go 1.26.3,
   `actions/setup-go` with module cache.
2. `go build -trimpath -ldflags="-s -w" -o prescriptioncli-${{ matrix.os }} ./cmd/prescriptioncli`
   (`.exe` suffix on Windows).
3. `actions/upload-artifact` always.
4. On `workflow_dispatch` with `release=true`: `softprops/action-gh-release`
   (or equivalent) creating/updating tag `prescription-cli-latest` and
   attaching both binaries.

## 10. Error handling

- Network / DNS error: red banner with `r` to retry, `q` to quit.
- Login 401: "Invalid credentials" inline; focus returns to password field.
- Create 401: "Wrong demo secret — check `-secret` matches `DEMO_PRESCRIPTION_SECRET`".
- Create 400: render server `error` and `details` fields.

## 11. Out of scope

- Persisting login token across runs.
- Editing the doctor row.
- Multiple medicaments in one prescription.
- Bubbletea view snapshot tests.

## 12. Manual smoke test

```bash
go build -o bin/prescriptioncli ./cmd/prescriptioncli
DEMO_PRESCRIPTION_SECRET=$(grep ^DEMO_PRESCRIPTION_SECRET .env | cut -d= -f2) \
  ./bin/prescriptioncli \
  -secret "$DEMO_PRESCRIPTION_SECRET" \
  -medic-id "<seeded doctor uuid>"
```

Then walk through the bubbletea prompts and verify the prescription lands in
the DB (and the scheduled notification fires after the start time).
