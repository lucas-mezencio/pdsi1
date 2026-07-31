# Prescription Demo CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `cmd/prescriptioncli/` — a bubbletea-driven Go binary that logs in as a real user, prompts for one medicament with a styled form, and POSTs a prescription to the public demo endpoint using `DEMO_PRESCRIPTION_SECRET`. Add `.github/workflows/build-prescription-cli.yml` to cross-compile Linux + Windows on push/main and on manual dispatch.

**Architecture:** One `tea.Model` with a stage enum (`login → form → confirm → submitting → done`). `net/http` directly (no generated client) — login is unauthenticated and the prescription POST only needs the raw secret in `Authorization`. Lipgloss for styling, bubbles for textinput + spinner. HTTP layer is the only thing under test (UI is hand-verified).

**Tech Stack:** Go 1.26.3, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles`, `github.com/charmbracelet/lipgloss`, `github.com/google/uuid`, stdlib `net/http` + `encoding/json`.

**Spec:** `docs/superpowers/specs/2026-07-30-prescription-cli-design.md`

---

## Global Constraints

- Go 1.26, module `github.com.br/lucas-mezencio/pdsi1`.
- New code lives under `cmd/prescriptioncli/` (single `package main`).
- `Authorization` header on `POST /prescriptions` is the **raw** `DEMO_PRESCRIPTION_SECRET` value (no `Bearer` prefix). See `internal/api/middleware.go:88`.
- User-entered `start time` (`HH:MM`) is shifted `+3h` before submission.
- Doctor UUID: hardcoded constant `11111111-1111-1111-1111-111111111111` with a `TODO` comment for the user to replace with their seeded row; overridable via `-medic-id`.
- Doses are always prompted (no default).
- All commits use conventional prefixes: `feat:`, `test:`, `chore:`, `docs:`.
- TDD: failing test first, then implementation, then commit.
- Lint/typecheck: `go vet ./...` before each commit.

---

## File Structure

### New files

```
cmd/prescriptioncli/main.go
cmd/prescriptioncli/model.go
cmd/prescriptioncli/screens.go
cmd/prescriptioncli/styles.go
cmd/prescriptioncli/api.go
cmd/prescriptioncli/api_test.go
.github/workflows/build-prescription-cli.yml
```

---

## Task 1: Bootstrap CLI binary with flag parsing

**Files:**
- Create: `cmd/prescriptioncli/main.go`

**Interfaces:**
- Produces: `package main` binary named `prescriptioncli` that parses `-secret`, `-api`, `-medic-id` and prints help when `-secret` is missing.

- [ ] **Step 1: Create `cmd/prescriptioncli/main.go`**

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

// TODO: replace with the UUID you seeded for Dr. Test Silva
// (the doctor row must already exist in postgres before this CLI can submit).
const defaultMedicID = "11111111-1111-1111-1111-111111111111"

const defaultAPI = "http://localhost:8080/api/v1"

func main() {
	secret := flag.String("secret", "", "DEMO_PRESCRIPTION_SECRET value (raw, sent as Authorization header)")
	apiURL := flag.String("api", defaultAPI, "API base URL")
	medicID := flag.String("medic-id", defaultMedicID, "doctor UUID creating the prescription")
	flag.Parse()

	if *secret == "" {
		fmt.Fprintln(os.Stderr, "error: -secret is required (raw DEMO_PRESCRIPTION_SECRET value)")
		flag.Usage()
		os.Exit(2)
	}

	fmt.Printf("prescriptioncli starting\n")
	fmt.Printf("  api:     %s\n", *apiURL)
	fmt.Printf("  medic:   %s\n", *medicID)
	fmt.Printf("  secret:  <set, %d chars>\n", len(*secret))
}
```

- [ ] **Step 2: Build and verify**

Run: `go build -o bin/prescriptioncli ./cmd/prescriptioncli`
Expected: build succeeds, binary at `bin/prescriptioncli`.

Run: `./bin/prescriptioncli`
Expected: exit code 2, stderr contains `-secret is required`.

Run: `./bin/prescriptioncli -secret x`
Expected: stdout contains `api:` `medic:` `secret: <set, 1 chars>`.

- [ ] **Step 3: Add dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/google/uuid@latest
go mod tidy
```
Expected: `go.mod` updated with the three `charmbracelet/*` modules and `uuid`.

- [ ] **Step 4: Commit**

```bash
git add cmd/prescriptioncli/main.go go.mod go.sum
git commit -m "feat(prescriptioncli): bootstrap binary with flag parsing"
```

---

## Task 2: HTTP API layer (TDD)

**Files:**
- Create: `cmd/prescriptioncli/api.go`
- Create: `cmd/prescriptioncli/api_test.go`

**Interfaces:**
- Produces:
  - `type API struct { BaseURL string; Secret string; HTTP *http.Client }`
  - `func New(baseURL, secret string) *API`
  - `func (a *API) Login(ctx context.Context, email, pw string) (userID string, err error)`
  - `func (a *API) CreatePrescription(ctx context.Context, p Prescription) (*PrescriptionResponse, error)`
  - `func (a *API) ShiftStartTime(start string) (string, error)` — adds +3h.
  - `type Prescription struct { UserID, MedicID, Name, Dosage, Frequency, StartTime string; Doses int }`
  - `type PrescriptionResponse struct { ID uuid.UUID \`json:"id"\` }`
  - `type APIError struct { Status int; Code string; Details string }` — implements `error`.

- [ ] **Step 1: Write the failing tests in `cmd/prescriptioncli/api_test.go`**

```go
package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLogin_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["email"] != "a@b.com" || body["password"] != "pw" {
			t.Errorf("unexpected body: %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "u-1", "email": "a@b.com"})
	}))
	defer srv.Close()

	api := New(srv.URL, "secret")
	id, err := api.Login(context.Background(), "a@b.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "u-1" {
		t.Errorf("got id %q, want u-1", id)
	}
}

func TestLogin_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
	}))
	defer srv.Close()

	api := New(srv.URL, "secret")
	_, err := api.Login(context.Background(), "a@b.com", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("expected error to contain 'invalid credentials', got %v", err)
	}
}

func TestCreatePrescription_SendsRawSecretHeader(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": uuid.New().String()})
	}))
	defer srv.Close()

	api := New(srv.URL, "the-demo-secret")
	uid := uuid.New()
	resp, err := api.CreatePrescription(context.Background(), Prescription{
		UserID:     uid.String(),
		MedicID:    "11111111-1111-1111-1111-111111111111",
		Name:       "Aspirin",
		Dosage:     "100mg",
		Frequency:  "24:00",
		StartTime:  "02:30",
		Doses:      1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.ID.String() == "" {
		t.Fatal("expected non-empty response id")
	}
	if gotAuth != "the-demo-secret" {
		t.Errorf("expected raw Authorization header %q, got %q (must NOT include 'Bearer')", "the-demo-secret", gotAuth)
	}
	if gotPath != "/prescriptions" {
		t.Errorf("expected path /prescriptions, got %s", gotPath)
	}

	meds, ok := gotBody["medicaments"].([]any)
	if !ok || len(meds) != 1 {
		t.Fatalf("expected one medicament, got %v", gotBody["medicaments"])
	}
	m := meds[0].(map[string]any)
	if m["name"] != "Aspirin" || m["dosage"] != "100mg" || m["frequency"] != "24:00" {
		t.Errorf("unexpected medicament fields: %v", m)
	}
	if m["doses"].(float64) != 1 {
		t.Errorf("expected doses=1, got %v", m["doses"])
	}
	times, _ := m["time"].([]any)
	if len(times) != 1 || times[0] != "05:30" {
		t.Errorf("expected time=[05:30] (start 02:30 + 3h), got %v", m["time"])
	}
}

func TestCreatePrescription_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid secret"})
	}))
	defer srv.Close()

	api := New(srv.URL, "wrong-secret")
	_, err := api.CreatePrescription(context.Background(), Prescription{
		UserID: uuid.New().String(), MedicID: uuid.New().String(),
		Name: "A", Dosage: "1", Frequency: "24:00", StartTime: "08:00", Doses: 1,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid secret") {
		t.Errorf("expected error to contain 'invalid secret', got %v", err)
	}
}

func TestShiftStartTime(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"08:00", "11:00"},
		{"00:30", "03:30"},
		{"23:30", "02:30"}, // wraps midnight
		{"12:00", "15:00"},
	}
	for _, c := range cases {
		got, err := shiftStartTime(c.in)
		if err != nil {
			t.Errorf("shiftStartTime(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("shiftStartTime(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestShiftStartTime_InvalidInput(t *testing.T) {
	if _, err := shiftStartTime("not-a-time"); err == nil {
		t.Error("expected error for invalid input, got nil")
	}
	_ = time.Second // keep time import used; remove if unused
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/prescriptioncli/...`
Expected: FAIL (no `api.go` yet, package won't compile).

- [ ] **Step 3: Implement `cmd/prescriptioncli/api.go`**

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// timeOffset is added to user-entered start times. The demo endpoint is
// hosted in a timezone 3 hours ahead of the user's local clock.
const timeOffset = 3 * time.Hour

type Prescription struct {
	UserID    string
	MedicID   string
	Name      string
	Dosage    string
	Frequency string
	StartTime string // HH:MM (24h), will be shifted by timeOffset on submit
	Doses     int
}

type PrescriptionResponse struct {
	ID uuid.UUID `json:"id"`
}

type APIError struct {
	Status  int
	Message string
	Details string
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

type API struct {
	BaseURL string
	Secret  string
	HTTP    *http.Client
}

func New(baseURL, secret string) *API {
	return &API{BaseURL: baseURL, Secret: secret, HTTP: http.DefaultClient}
}

type loginResponse struct {
	ID string `json:"id"`
}

func (a *API) Login(ctx context.Context, email, pw string) (string, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": pw})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", decodeAPIError(resp.StatusCode, data)
	}

	var lr loginResponse
	if err := json.Unmarshal(data, &lr); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}
	if lr.ID == "" {
		return "", fmt.Errorf("login response missing id")
	}
	return lr.ID, nil
}

func (a *API) CreatePrescription(ctx context.Context, p Prescription) (*PrescriptionResponse, error) {
	shifted, err := shiftStartTime(p.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start time %q: %w", p.StartTime, err)
	}

	payload := map[string]any{
		"user_id":  p.UserID,
		"medic_id": p.MedicID,
		"medicaments": []map[string]any{
			{
				"name":      p.Name,
				"dosage":    p.Dosage,
				"frequency": p.Frequency,
				"time":      []string{shifted},
				"doses":     p.Doses,
			},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/prescriptions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build prescription request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", a.Secret) // raw secret, NOT "Bearer ..."

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create prescription failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return nil, decodeAPIError(resp.StatusCode, data)
	}

	var pr PrescriptionResponse
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("decode prescription response: %w", err)
	}
	if pr.ID.String() == "" {
		return nil, fmt.Errorf("prescription response missing id")
	}
	return &pr, nil
}

func decodeAPIError(status int, body []byte) error {
	var env struct {
		Error   string `json:"error"`
		Details string `json:"details"`
	}
	_ = json.Unmarshal(body, &env)
	return &APIError{Status: status, Message: env.Error, Details: env.Details}
}

func shiftStartTime(in string) (string, error) {
	t, err := time.Parse("15:04", in)
	if err != nil {
		return "", err
	}
	return t.Add(timeOffset).Format("15:04"), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v ./cmd/prescriptioncli/...`
Expected: PASS for all 6 tests.

- [ ] **Step 5: Commit**

```bash
git add cmd/prescriptioncli/api.go cmd/prescriptioncli/api_test.go
git commit -m "feat(prescriptioncli): HTTP API layer with auth and +3h time shift"
```

---

## Task 3: Lipgloss styles module

**Files:**
- Create: `cmd/prescriptioncli/styles.go`

**Interfaces:**
- Produces: package-level vars `StyleTitle`, `StyleSubtle`, `StyleSuccess`, `StyleError`, `StyleBox` (a `lipgloss.Style` with border + padding), `StyleFieldLabel`, `StyleSummaryKey`, `StyleSummaryValue`.

- [ ] **Step 1: Write `cmd/prescriptioncli/styles.go`**

```go
package main

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary = lipgloss.Color("#7D56F4")
	colorSuccess = lipgloss.Color("#04B575")
	colorError   = lipgloss.Color("#FF5F56")
	colorMuted   = lipgloss.Color("#888888")

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	StyleSubtle = lipgloss.NewStyle().Foreground(colorMuted)

	StyleSuccess = lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	StyleError   = lipgloss.NewStyle().Bold(true).Foreground(colorError)

	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	StyleFieldLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginTop(1)

	StyleSummaryKey   = lipgloss.NewStyle().Bold(true).Width(14)
	StyleSummaryValue = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
)
```

- [ ] **Step 2: Verify build**

Run: `go build ./cmd/prescriptioncli/...`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add cmd/prescriptioncli/styles.go
git commit -m "feat(prescriptioncli): lipgloss styles module"
```

---

## Task 4: Root tea.Model with stage enum

**Files:**
- Create: `cmd/prescriptioncli/model.go`

**Interfaces:**
- Produces:
  - `type stage int` + constants `stageLogin, stageForm, stageConfirm, stageSubmitting, stageDone`.
  - `type sessionData struct { UserID, Name, Dosage, Frequency, StartTime string; Doses int }`
  - `type Model struct { stage stage; data sessionData; api *API; width int; err error }`
  - `func NewModel(api *API) Model`
  - `func (m Model) Init() tea.Cmd`
  - `func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)`
  - `func (m Model) View() string`

The model just dispatches to per-screen `updateScreen(msg)` and `viewScreen()` methods (implemented in Task 5+).

- [ ] **Step 1: Write `cmd/prescriptioncli/model.go`**

```go
package main

import (
	"github.com/charmbracelet/bubbletea"
)

type stage int

const (
	stageLogin stage = iota
	stageForm
	stageConfirm
	stageSubmitting
	stageDone
)

func (s stage) String() string {
	switch s {
	case stageLogin:
		return "login"
	case stageForm:
		return "form"
	case stageConfirm:
		return "confirm"
	case stageSubmitting:
		return "submitting"
	case stageDone:
		return "done"
	default:
		return "unknown"
	}
}

type sessionData struct {
	UserID    string
	Name      string
	Dosage    string
	Frequency string
	StartTime string
	Doses     int
}

type Model struct {
	stage stage
	data  sessionData
	api   *API
	width int
	err   error

	// screens are populated in later tasks; methods are no-ops until then.
	login    loginScreen
	form     formScreen
	confirm  confirmScreen
	submit   submittingScreen
	done     doneScreen
	quitting bool
}

func NewModel(api *API) Model {
	return Model{
		stage: stageLogin,
		api:   api,
		login: newLoginScreen(),
		form:  newFormScreen(),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		// fall through and let the active screen handle the key
	}

	// Cross-cutting messages are handled at the root so screen-local state
	// does not need to know how to mutate the stage.
	switch msg := msg.(type) {
	case stageChangeMsg:
		m.stage = msg.to
		switch m.stage {
		case stageForm:
			m.form = newFormScreen()
		case stageConfirm:
			m.confirm = newConfirmScreen(m.data)
			m.confirm.userID = m.data.UserID
			m.confirm.medicID = currentMedicID()
		case stageSubmitting:
			m.submit = newSubmittingScreen()
		}
		return m, nil
	case loginSuccessMsg:
		m.data.UserID = msg.userID
		return m, transitionMsg(stageForm)
	case formSubmitMsg:
		m.data.Name = msg.data.Name
		m.data.Dosage = msg.data.Dosage
		m.data.Frequency = msg.data.Frequency
		m.data.StartTime = msg.data.StartTime
		m.data.Doses = msg.data.Doses
		return m, transitionMsg(stageConfirm)
	case prescriptionCreatedMsg:
		shifted, _ := shiftStartTime(m.data.StartTime)
		m.done = doneScreen{success: true, id: msg.resp.ID.String(), shifted: shifted}
		return m, transitionMsg(stageDone)
	case prescriptionFailedMsg:
		m.done = doneScreen{success: false, err: msg.err.Error()}
		return m, transitionMsg(stageDone)
	}

	var cmd tea.Cmd
	switch m.stage {
	case stageLogin:
		m.login, cmd = m.login.update(msg)
	case stageForm:
		m.form, cmd = m.form.update(msg)
	case stageConfirm:
		m.confirm, cmd = m.confirm.update(msg)
	case stageSubmitting:
		m.submit, cmd = m.submit.update(msg)
	case stageDone:
		m.done, cmd = m.done.update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	switch m.stage {
	case stageLogin:
		return m.login.view()
	case stageForm:
		return m.form.view()
	case stageConfirm:
		return m.confirm.view()
	case stageSubmitting:
		return m.submit.view()
	case stageDone:
		return m.done.view()
	default:
		return StyleTitle.Render("prescriptioncli")
	}
}
```

- [ ] **Step 2: Verify build (login/form screens don't exist yet)**

Run: `go build ./cmd/prescriptioncli/...`
Expected: FAIL — `loginScreen`, `formScreen`, `update`, `view` methods don't exist yet.

- [ ] **Step 3: Add empty screen stubs in `cmd/prescriptioncli/screens.go` (placeholder until Task 5)**

```go
package main

import "github.com/charmbracelet/bubbletea"

type loginScreen struct{}

func newLoginScreen() loginScreen { return loginScreen{} }

func (s loginScreen) update(msg tea.Msg) (loginScreen, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
		// placeholder: Task 5 wires this up
	}
	return s, nil
}

func (s loginScreen) view() string {
	return StyleTitle.Render("prescriptioncli") + "\nlogin screen (coming soon)"
}

type formScreen struct{}

func newFormScreen() formScreen { return formScreen{} }

func (s formScreen) update(msg tea.Msg) (formScreen, tea.Cmd) { return s, nil }
func (s formScreen) view() string {
	return StyleTitle.Render("prescriptioncli") + "\nform screen (coming soon)"
}

// confirming / submitting / done stubs are added in Task 6/7.
type confirmScreen struct{}

func (s confirmScreen) update(msg tea.Msg) (confirmScreen, tea.Cmd) { return s, nil }
func (s confirmScreen) view() string                              { return "" }

type submittingScreen struct{}

func (s submittingScreen) update(msg tea.Msg) (submittingScreen, tea.Cmd) {
	return s, nil
}
func (s submittingScreen) view() string { return "" }

type doneScreen struct{}

func (s doneScreen) update(msg tea.Msg) (doneScreen, tea.Cmd) { return s, nil }
func (s doneScreen) view() string                             { return "" }
```

- [ ] **Step 4: Verify build now passes**

Run: `go build ./cmd/prescriptioncli/...`
Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add cmd/prescriptioncli/model.go cmd/prescriptioncli/screens.go
git commit -m "feat(prescriptioncli): root tea model with stage enum"
```

---

## Task 5: Login screen

**Files:**
- Modify: `cmd/prescriptioncli/screens.go`

**Interfaces:**
- `loginScreen` owns two `textinput.Model` (email, password with `EchoPassword`), `focused` index 0/1, and `err string`.
- `update(msg)` handles tab/shift-tab/enter; on enter dispatches `loginCmd` tea.Cmd.
- `view()` shows title + two inputs + helper line; shows red error inline if `err != ""`.

Add async message types in `model.go`:
- `type loginSuccessMsg struct{ userID string }`
- `type loginErrMsg struct{ err error }`

Add `loginCmd(api *API, email, pw string) tea.Cmd` in `screens.go` (uses `tea.Sequence` not needed; plain func-returning-msg).

- [ ] **Step 1: Replace `loginScreen` in `cmd/prescriptioncli/screens.go`**

```go
import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type loginScreen struct {
	email    textinput.Model
	password textinput.Model
	focused  int
	err      string
}

func newLoginScreen() loginScreen {
	email := textinput.New()
	email.Placeholder = "you@example.com"
	email.CharLimit = 120
	email.Focus()

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoPassword = '•'
	password.EchoMode = textinput.EchoPassword
	password.CharLimit = 128

	return loginScreen{email: email, password: password, focused: 0}
}

type loginSuccessMsg struct{ userID string }
type loginErrMsg struct{ err error }

func loginCmd(api *API, email, pw string) tea.Cmd {
	return func() tea.Msg {
		id, err := api.Login(context.Background(), email, pw)
		if err != nil {
			return loginErrMsg{err: err}
		}
		return loginSuccessMsg{userID: id}
	}
}

func (s loginScreen) update(msg tea.Msg) (loginScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case loginErrMsg:
		s.err = msg.err.Error()
		s.password.SetValue("")
		s.password.Focus()
		s.focused = 1
		return s, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			s.focused = (s.focused + 1) % 2
		case "shift+tab", "up":
			s.focused = (s.focused + 1) % 2
		case "enter":
			if s.email.Value() == "" || s.password.Value() == "" {
				s.err = "email and password are required"
				return s, nil
			}
			s.err = ""
			return s, loginCmd(currentAPI(), s.email.Value(), s.password.Value())
		}
	}

	var cmd tea.Cmd
	if s.focused == 0 {
		s.email, cmd = s.email.Update(msg)
	} else {
		s.password, cmd = s.password.Update(msg)
	}
	return s, cmd
}

func (s loginScreen) view() string {
	if s.focused == 0 {
		s.email.Focus()
		s.password.Blur()
	} else {
		s.email.Blur()
		s.password.Focus()
	}

	var body string
	body += StyleFieldLabel.Render("Email") + "\n"
	body += s.email.View() + "\n"
	body += StyleFieldLabel.Render("Password") + "\n"
	body += s.password.View() + "\n"
	if s.err != "" {
		body += "\n" + StyleError.Render(s.err)
	}
	body += "\n" + StyleSubtle.Render("tab to switch · enter to submit · ctrl+c to quit")

	return StyleTitle.Render("prescriptioncli — login") + "\n" + body
}
```

- [ ] **Step 2: Add a `currentAPI()` accessor and `transitionMsg` in `model.go`**

Add to `cmd/prescriptioncli/model.go`:

```go
// currentAPI returns the active API handle. Stored on the package via
// NewModel's caller (see main.go in Task 8). Implemented as a package-level
// pointer to avoid threading it through every screen method.
var currentAPIPtr **API

func currentAPI() *API {
	if currentAPIPtr == nil || *currentAPIPtr == nil {
		panic("prescriptioncli: currentAPI called before NewModel wiring")
	}
	return *currentAPIPtr
}

func SetCurrentAPI(p **API) { currentAPIPtr = p }

type stageChangeMsg struct{ to stage }

func transitionMsg(to stage) tea.Cmd {
	return func() tea.Msg { return stageChangeMsg{to: to} }
}
```

Update `Update` in `model.go` to handle `stageChangeMsg`:

```go
case stageChangeMsg:
	m.stage = msg.to
	// reset screen-local state when entering a screen
	switch m.stage {
	case stageForm:
		m.form = newFormScreen()
	case stageConfirm:
		m.confirm = newConfirmScreen(m.data)
	case stageSubmitting:
		m.submit = newSubmittingScreen()
		m.done = doneScreen{}
	}
	return m, nil
```

- [ ] **Step 3: Verify build**

Run: `go build ./cmd/prescriptioncli/...`
Expected: FAIL — `newConfirmScreen`, `newSubmittingScreen` not defined yet. They are added in Task 6/7; for now stub them in `screens.go`:

```go
type confirmScreen struct{ data sessionData }

func newConfirmScreen(d sessionData) confirmScreen { return confirmScreen{data: d} }

type submittingScreen struct{}

func newSubmittingScreen() submittingScreen { return submittingScreen{} }
```

(The previous Task 4 stubs for `confirmScreen`/`submittingScreen` need their constructors adjusted — see Step 4.)

- [ ] **Step 4: Adjust the Task 4 stubs to match**

In `cmd/prescriptioncli/screens.go`, replace the earlier `confirmScreen`/`submittingScreen` stubs with the definitions from Step 3, and update `Model` initialization in `model.go` to call `newConfirmScreen`/`newSubmittingScreen` lazily (already done in `stageChangeMsg` handler).

- [ ] **Step 5: Verify build now passes**

Run: `go build ./cmd/prescriptioncli/...`
Expected: build succeeds.

- [ ] **Step 6: Commit**

```bash
git add cmd/prescriptioncli/model.go cmd/prescriptioncli/screens.go
git commit -m "feat(prescriptioncli): login screen with masked password"
```

---

## Task 6: Form screen with inline HH:MM and integer validation

**Files:**
- Modify: `cmd/prescriptioncli/screens.go`

**Interfaces:**
- `formScreen` owns five `textinput.Model` (name, dose, frequency, startTime, doses), `focused` 0–4, and `err string`.
- On `enter` at the last field → validates, returns `transitionMsg(stageConfirm)` if ok; otherwise sets `err`.
- `view()` renders labels + inputs + helper + red error if any.

Add `newFormScreen` to construct all five inputs (focused=0). Keep `update`/`view` as already-stubbed.

- [ ] **Step 1: Replace `formScreen` in `cmd/prescriptioncli/screens.go`**

```go
import "regexp"

var hhmmRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

type formScreen struct {
	inputs  [5]textinput.Model
	focused int
	err     string
}

func newFormScreen() formScreen {
	labels := [5]string{"Medication name", "Dose (e.g. 100mg)", "Frequency (HH:MM, e.g. 24:00)", "Start time (HH:MM)", "Number of doses"}
	var inputs [5]textinput.Model
	for i := range inputs {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.CharLimit = 64
		inputs[i] = ti
	}
	inputs[0].Focus()
	return formScreen{inputs: inputs, focused: 0}
}

func (s *formScreen) values() (name, dose, freq, start string, doses int, err error) {
	name = s.inputs[0].Value()
	dose = s.inputs[1].Value()
	freq = s.inputs[2].Value()
	start = s.inputs[3].Value()
	dosesStr := s.inputs[4].Value()

	if name == "" || dose == "" || freq == "" || start == "" || dosesStr == "" {
		return "", "", "", "", 0, errors.New("all fields are required")
	}
	if !hhmmRe.MatchString(freq) {
		return "", "", "", "", 0, fmt.Errorf("frequency must match HH:MM (24h), got %q", freq)
	}
	if !hhmmRe.MatchString(start) {
		return "", "", "", "", 0, fmt.Errorf("start time must match HH:MM (24h), got %q", start)
	}
	var n int
	if _, err := fmt.Sscanf(dosesStr, "%d", &n); err != nil || n <= 0 {
		return "", "", "", "", 0, fmt.Errorf("doses must be a positive integer, got %q", dosesStr)
	}
	return name, dose, freq, start, n, nil
}

func (s formScreen) update(msg tea.Msg) (formScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			s.focused = (s.focused + 1) % 5
		case "shift+tab", "up":
			s.focused = (s.focused + 4) % 5
		case "enter":
			if s.focused < 4 {
				s.focused++
			} else {
				name, dose, freq, start, doses, err := s.values()
				if err != nil {
					s.err = err.Error()
					return s, nil
				}
				s.err = ""
				return s, formSubmitMsg{sessionData{Name: name, Dosage: dose, Frequency: freq, StartTime: start, Doses: doses}}
			}
		}
	}

	for i := range s.inputs {
		if i == s.focused {
			s.inputs[i].Focus()
		} else {
			s.inputs[i].Blur()
		}
	}
	var cmd tea.Cmd
	s.inputs[s.focused], cmd = s.inputs[s.focused].Update(msg)
	return s, cmd
}

type formSubmitMsg struct{ data sessionData }

func (s formScreen) view() string {
	var body string
	labels := [5]string{"Medication name", "Dose", "Frequency", "Start time (+3h on submit)", "Doses"}
	for i := 0; i < 5; i++ {
		body += StyleFieldLabel.Render(labels[i]) + "\n"
		body += s.inputs[i].View() + "\n"
	}
	if s.err != "" {
		body += "\n" + StyleError.Render(s.err)
	}
	body += "\n" + StyleSubtle.Render("tab/enter to advance · ctrl+c to quit")
	return StyleTitle.Render("prescriptioncli — new prescription") + "\n" + body
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./cmd/prescriptioncli/...`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add cmd/prescriptioncli/screens.go
git commit -m "feat(prescriptioncli): prescription form with HH:MM validation"
```

---

## Task 7: Confirm, submitting, and done screens

**Files:**
- Modify: `cmd/prescriptioncli/screens.go`
- Modify: `cmd/prescriptioncli/model.go`

**Interfaces:**
- `confirmScreen` shows a `lipgloss` summary box of all `sessionData` fields plus the doctor fallback UUID.
  - `y` → `submitCmd(api, userID, data) tea.Cmd` and `transitionMsg(stageSubmitting)`.
  - `n`/esc → `transitionMsg(stageForm)`.
- `submittingScreen` shows a `spinner.Model` + "Sending prescription…". Updates its spinner on `spinner.TickMsg`.
- `doneScreen` shows green success (id + scheduled time) or red error with retry hint.
  - `r` on error → `transitionMsg(stageForm)`.
  - `q`/esc → `tea.Quit`.

Add messages:
- `type prescriptionCreatedMsg struct{ resp *PrescriptionResponse }`
- `type prescriptionFailedMsg struct{ err error }`

Add `submitCmd` in `screens.go`.

- [ ] **Step 1: Replace `confirmScreen`, `submittingScreen`, `doneScreen` in `cmd/prescriptioncli/screens.go`**

```go
import (
	"github.com/charmbracelet/bubbles/spinner"
)

type confirmScreen struct {
	data     sessionData
	userID   string
	medicID  string
	shifted  string
	err      error
	quitting bool
}

func newConfirmScreen(d sessionData) confirmScreen {
	shifted, err := shiftStartTime(d.StartTime)
	return confirmScreen{data: d, shifted: shifted, err: err}
}

func (s confirmScreen) update(msg tea.Msg) (confirmScreen, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "y", "Y":
			if s.err != nil {
				return s, nil
			}
			return s, tea.Batch(transitionMsg(stageSubmitting), submitCmd(currentAPI(), sessionRef{userID: s.userID, data: s.data, medicID: s.medicID}))
		case "n", "N", "esc":
			return s, transitionMsg(stageForm)
		}
	}
	return s, nil
}

func (s confirmScreen) view() string {
	if s.err != nil {
		return StyleError.Render(fmt.Sprintf("invalid start time: %v", s.err))
	}
	summary := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s (+3h -> %s)\n%s %d\n%s %s\n%s %s",
		StyleSummaryKey.Render("Medication:"), StyleSummaryValue.Render(s.data.Name),
		StyleSummaryKey.Render("Dose:"), StyleSummaryValue.Render(s.data.Dosage),
		StyleSummaryKey.Render("Frequency:"), StyleSummaryValue.Render(s.data.Frequency),
		StyleSummaryKey.Render("Start time:"), StyleSummaryValue.Render(s.data.StartTime), StyleSummaryValue.Render(s.shifted),
		StyleSummaryKey.Render("Doses:"), s.data.Doses,
		StyleSummaryKey.Render("User ID:"), StyleSummaryValue.Render(s.userID),
		StyleSummaryKey.Render("Doctor ID:"), StyleSummaryValue.Render(s.medicID),
	)
	return StyleTitle.Render("prescriptioncli — confirm") + "\n" +
		StyleBox.Render(summary) + "\n" +
		StyleSubtle.Render("y to submit · n to go back · ctrl+c to quit")
}

type sessionRef struct {
	userID  string
	data    sessionData
	medicID string
}

type submitStartMsg struct{ ref sessionRef }
type prescriptionCreatedMsg struct{ resp *PrescriptionResponse }
type prescriptionFailedMsg struct{ err error }

func submitCmd(api *API, ref sessionRef) tea.Cmd {
	return func() tea.Msg {
		resp, err := api.CreatePrescription(context.Background(), Prescription{
			UserID:    ref.userID,
			MedicID:   ref.medicID,
			Name:      ref.data.Name,
			Dosage:    ref.data.Dosage,
			Frequency: ref.data.Frequency,
			StartTime: ref.data.StartTime,
			Doses:     ref.data.Doses,
		})
		if err != nil {
			return prescriptionFailedMsg{err: err}
		}
		return prescriptionCreatedMsg{resp: resp}
	}
}

type submittingScreen struct {
	spinner spinner.Model
}

func newSubmittingScreen() submittingScreen {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	return submittingScreen{spinner: sp}
}

func (s submittingScreen) update(msg tea.Msg) (submittingScreen, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

func (s submittingScreen) view() string {
	return StyleTitle.Render("prescriptioncli — submitting") + "\n\n" +
		s.spinner.View() + " Sending prescription…\n"
}

type doneScreen struct {
	success bool
	id      string
	shifted string
	err     string
}

func (s doneScreen) update(msg tea.Msg) (doneScreen, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		if km.String() == "q" || km.String() == "esc" || km.String() == "ctrl+c" {
			return s, tea.Quit
		}
		if !s.success && (km.String() == "r" || km.String() == "R") {
			return s, transitionMsg(stageForm)
		}
	}
	return s, nil
}

func (s doneScreen) view() string {
	if s.success {
		body := StyleSuccess.Render("Prescription created!") + "\n\n"
		body += StyleSummaryKey.Render("ID:") + " " + s.id + "\n"
		body += StyleSummaryKey.Render("Scheduled:") + " " + s.shifted + "\n"
		body += "\n" + StyleSubtle.Render("press q to quit")
		return StyleTitle.Render("prescriptioncli — done") + "\n" + StyleBox.Render(body)
	}
	body := StyleError.Render("Failed to create prescription") + "\n\n"
	body += s.err + "\n\n"
	body += StyleSubtle.Render("r to retry · q to quit")
	return StyleTitle.Render("prescriptioncli — error") + "\n" + StyleBox.Render(body)
}
```

- [ ] **Step 2: Add `currentMedicID()` accessor in `model.go`**

All cross-cutting handlers (`stageChangeMsg`, `loginSuccessMsg`, `formSubmitMsg`, `prescriptionCreatedMsg`, `prescriptionFailedMsg`) were already wired in Task 4's corrected `Model.Update`. We just need the `currentMedicID()` accessor so the root can stash the doctor UUID onto the confirm screen during a `stageChangeMsg → stageConfirm` transition:

```go
var currentMedicIDPtr *string

func currentMedicID() string {
	if currentMedicIDPtr == nil {
		return defaultMedicID
	}
	return *currentMedicIDPtr
}

func SetCurrentMedicID(p *string) { currentMedicIDPtr = p }
```

- [ ] **Step 3: Verify build**

Run: `go build ./cmd/prescriptioncli/...`
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add cmd/prescriptioncli/model.go cmd/prescriptioncli/screens.go
git commit -m "feat(prescriptioncli): confirm, submitting, and done screens"
```

---

## Task 8: Wire `main.go` to the bubbletea program

**Files:**
- Modify: `cmd/prescriptioncli/main.go`

**Interfaces:**
- `main()` builds `api := New(*apiURL, *secret)`, sets `currentAPIPtr` and `currentMedicIDPtr`, then `p := tea.NewProgram(NewModel(api))` and runs it.

- [ ] **Step 1: Replace `cmd/prescriptioncli/main.go`**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// TODO: replace with the UUID you seeded for Dr. Test Silva
// (the doctor row must already exist in postgres before this CLI can submit).
const defaultMedicID = "11111111-1111-1111-1111-111111111111"

const defaultAPI = "http://localhost:8080/api/v1"

func main() {
	secret := flag.String("secret", "", "DEMO_PRESCRIPTION_SECRET value (raw, sent as Authorization header)")
	apiURL := flag.String("api", defaultAPI, "API base URL")
	medicID := flag.String("medic-id", defaultMedicID, "doctor UUID creating the prescription")
	flag.Parse()

	if *secret == "" {
		fmt.Fprintln(os.Stderr, "error: -secret is required (raw DEMO_PRESCRIPTION_SECRET value)")
		flag.Usage()
		os.Exit(2)
	}

	api := New(*apiURL, *secret)
	SetCurrentAPI(&api)
	SetCurrentMedicID(medicID)

	p := tea.NewProgram(NewModel(api), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify build**

Run: `go build -o bin/prescriptioncli ./cmd/prescriptioncli`
Expected: build succeeds.

- [ ] **Step 3: Smoke test**

Run: `./bin/prescriptioncli -secret x`
Expected: program launches, shows login screen, accepts text input, tab cycles fields, enter triggers login (will fail against the placeholder URL — that's fine, the error should appear inline in red). Press `ctrl+c` to quit.

- [ ] **Step 4: Run unit tests**

Run: `go test -v ./cmd/prescriptioncli/...`
Expected: all 6 tests in `api_test.go` still pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/prescriptioncli/main.go
git commit -m "feat(prescriptioncli): wire main.go to bubbletea program"
```

---

## Task 9: GitHub Action workflow

**Files:**
- Create: `.github/workflows/build-prescription-cli.yml`

**Interfaces:**
- Triggers: `push` to `main` (paths-filter on `cmd/prescriptioncli/**`, `go.mod`, `go.sum`, the workflow file) + `workflow_dispatch` with boolean `release` input (default false).
- Matrix: `ubuntu-latest` + `windows-latest`, Go 1.26.3.
- Always uploads artifacts named `prescriptioncli-linux` / `prescriptioncli-windows.exe`.
- On `workflow_dispatch` with `release=true`: create/update GitHub Release `prescription-cli-latest` and attach both binaries.

- [ ] **Step 1: Create `.github/workflows/build-prescription-cli.yml`**

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
        required: false
        default: false

permissions:
  contents: write

jobs:
  build:
    name: build (${{ matrix.os }})
    runs-on: ${{ matrix.os }}
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, windows-latest]
    steps:
      - name: checkout
        uses: actions/checkout@v4

      - name: setup-go
        uses: actions/setup-go@v5
        with:
          go-version: "1.26.3"
          cache: true

      - name: build
        shell: bash
        run: |
          set -euo pipefail
          if [ "${{ matrix.os }}" = "windows-latest" ]; then
            OUT=prescriptioncli-windows.exe
          else
            OUT=prescriptioncli-linux
          fi
          go build -trimpath -ldflags="-s -w" -o "$OUT" ./cmd/prescriptioncli

      - name: upload-artifact
        uses: actions/upload-artifact@v4
        with:
          name: prescriptioncli-${{ matrix.os }}
          path: |
            prescriptioncli-linux
            prescriptioncli-windows.exe
          if-no-files-found: error

  release:
    name: publish release
    needs: build
    if: github.event_name == 'workflow_dispatch' && inputs.release == true
    runs-on: ubuntu-latest
    steps:
      - name: checkout
        uses: actions/checkout@v4

      - name: download-linux
        uses: actions/download-artifact@v4
        with:
          name: prescriptioncli-ubuntu-latest
          path: dist/

      - name: download-windows
        uses: actions/download-artifact@v4
        with:
          name: prescriptioncli-windows-latest
          path: dist/

      - name: publish release
        uses: softprops/action-gh-release@v2
        with:
          tag_name: prescription-cli-latest
          name: prescription-cli (latest)
          body: |
            Latest build of the `prescriptioncli` demo runner.

            - Linux: `prescriptioncli-linux`
            - Windows: `prescriptioncli-windows.exe`

            Usage:
            ```
            ./prescriptioncli-linux -secret "$DEMO_PRESCRIPTION_SECRET"
            ```
          files: |
            dist/prescriptioncli-linux
            dist/prescriptioncli-windows.exe
          fail_on_unmatched_files: true
```

- [ ] **Step 2: Validate YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/build-prescription-cli.yml'))"`
Expected: no output, exit code 0.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/build-prescription-cli.yml
git commit -m "ci: build prescriptioncli on push/main + manual dispatch"
```

---

## Task 10: Lint, full test sweep, final commit

- [ ] **Step 1: Lint**

Run: `go vet ./... && golangci-lint run ./cmd/prescriptioncli/...`
Expected: no errors.

- [ ] **Step 2: Full test sweep**

Run: `go test ./...`
Expected: all packages pass (including the 6 API tests).

- [ ] **Step 3: Build production binary**

Run: `go build -trimpath -ldflags="-s -w" -o bin/prescriptioncli ./cmd/prescriptioncli`
Expected: build succeeds.

- [ ] **Step 4: Manual smoke test against a running API**

Prereq: a local API instance + a seeded doctor row whose UUID matches the `defaultMedicID` constant.

```bash
DEMO_PRESCRIPTION_SECRET=devsecret \
  ./bin/prescriptioncli \
  -secret devsecret \
  -medic-id 11111111-1111-1111-1111-111111111111
```

Walk through login → form (try `"08:00"` for start time, expect confirm screen to show `11:00`) → confirm → success. Verify the row landed in the DB.

- [ ] **Step 5: Final commit if any fixes were needed**

```bash
git add cmd/prescriptioncli/
git commit -m "chore(prescriptioncli): lint and smoke-test fixes" --allow-empty
```

---

## Self-review

**Spec coverage:**
- §2 decisions (location, no shared token, bubbletea+bubbles+lipgloss, single Model with stages, net/http, hardcoded UUID with TODO, raw Authorization, +3h, always prompt doses, HTTP tests only, branch name, CI triggers, matrix, release behavior) — all reflected in the tasks above.
- §3 CLI surface — Task 1 + Task 8.
- §4 file layout — Tasks 1–9.
- §5 bubbletea flow + stages — Tasks 4–7.
- §6 HTTP layer — Task 2.
- §7 +3h helper — Task 2 (`shiftStartTime`).
- §8 testing — Task 2.
- §9 GH Action — Task 9.
- §10 error handling — covered by `decodeAPIError`, inline `s.err` in login/form, retry-on-error in done screen.
- §11 out-of-scope — not addressed in any task (correct).
- §12 manual smoke test — Task 10.

**Placeholder scan:** No "TBD", "fill in", or generic "add validation". All code shown.

**Type consistency:** `sessionData`, `Prescription`, `PrescriptionResponse`, `APIError`, `loginSuccessMsg`, `loginErrMsg`, `formSubmitMsg`, `prescriptionCreatedMsg`, `prescriptionFailedMsg`, `stageChangeMsg`, `stageLogin/Form/Confirm/Submitting/Done`, `loginScreen`, `formScreen`, `confirmScreen`, `submittingScreen`, `doneScreen`, `currentAPI()`, `currentMedicID()`, `SetCurrentAPI`, `SetCurrentMedicID`, `transitionMsg` — names and signatures are consistent across tasks.

**One thing to verify during Task 5/7 execution:** the `confirmScreen.medicID` field is populated via `currentMedicID()` inside the `stageChangeMsg` handler in `Model.Update`. This wiring is already explicit in Task 4's corrected `Model.Update`:

```go
case stageChangeMsg:
    m.stage = msg.to
    switch m.stage {
    case stageConfirm:
        m.confirm = newConfirmScreen(m.data)
        m.confirm.userID = m.data.UserID
        m.confirm.medicID = currentMedicID()
    case stageSubmitting:
        m.submit = newSubmittingScreen()
    }
    return m, nil
```

Task 4 must land the corrected `Model.Update` (with the cross-cutting handler block) before Task 7 can compile.
