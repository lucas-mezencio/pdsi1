# User Dose Schedule Endpoint Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /api/v1/users/{userId}/doses` returning the user's full medication schedule (planned doses from active prescriptions, overlaid with TAKEN/MISSED status from `dose_records`), and document every dose-related endpoint in the OpenAPI spec.

**Architecture:** Domain layer adds `Prescription.ExpandSchedule(loc) []ScheduledDose` and a `BrazilLocation` constant (UTC-3). Application layer `DoseRecordQueryHandler.ListScheduleForUser` loads active prescriptions + dose records, overlays in memory keyed by `(prescription_id, scheduled_at, medicament_name)`, preserves orphan records. HTTP handler is added as a manual extended route alongside `/users/{userId}/dose-records`. OpenAPI spec gets a new `DoseSchedule` tag, `ScheduledDose` schema, the new path, and documentation blocks for the three already-implemented dose-record endpoints.

**Tech Stack:** Go 1.26.3, chi router, oapi-codegen (for spec validation), PostgreSQL (via existing repos).

**Spec:** `docs/superpowers/specs/2026-07-29-user-dose-schedule-design.md`

## Global Constraints

- Go module path: `github.com.br/lucas-mezencio/pdsi1`
- Brazil timezone: UTC-3 hardcoded via `prescription.BrazilLocation = time.FixedZone("BRT", -3*60*60)`
- Domain types carry JSON tags directly (matches existing `DoseRecord` pattern)
- Each commit independently green: `go test ./...` + `golangci-lint run` pass
- The scheduler timezone stays on server-local (accepted limitation, do NOT change)
- The `/users/{userId}/doses` route is added as a manual extended route, NOT in the generated OpenAPI server. Routes are documented in the spec only.

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/domain/prescription/scheduled_dose.go` (create) | `ScheduledDose` struct (with JSON tags), `BrazilLocation` constant, `Prescription.ExpandSchedule` method, helpers `intervalForMedicament` and `firstScheduleTimeAfter`. |
| `internal/domain/prescription/scheduled_dose_test.go` (create) | Table-driven tests for `ExpandSchedule`: empty, once-daily, twice-daily, BRT wall-clock verification, `CreatedAt` anchor respected. |
| `internal/application/queries/dose_record_queries.go` (modify) | Add `prescriptionRepo` field, accept it in constructor, add `ListDoseScheduleQuery` struct, add `ListScheduleForUser` method, add `overlayKey` helper. |
| `internal/application/queries/dose_record_schedule_test.go` (create) | Tests for `ListScheduleForUser`: empty, all-PENDING reconstruction, TAKEN overlay, orphan record preserved, RBAC forbidden. |
| `internal/api/extended_server.go` (modify) | Add `ListDoseSchedule` handler. No constructor change. |
| `internal/api/router.go` (modify) | Register `r.Get("/users/{userId}/doses", ext.ListDoseSchedule)`. |
| `internal/api/list_dose_schedule_handler_test.go` (create) | Smoke test for local-UUID vs Firebase-UID invariant + round-trip JSON shape. |
| `docs/api.yaml` (modify) | Add `DoseSchedule` tag, `ScheduledDose` schema, `/users/{userId}/doses` path, documentation blocks for `/users/{userId}/dose-records` (GET), `/dose-records/{doseRecordId}/confirm` (POST), `/dose-records/{doseRecordId}/miss` (POST). |
| `cmd/api/main.go` (modify) | Pass `prescriptionRepo` to `NewDoseRecordQueryHandler`. |
| `internal/api/gen/types.gen.go` and `server.gen.go` (regenerated) | Re-run `go generate ./internal/api/gen/...` after docs change. |

---

## Task 1: Domain — `ScheduledDose`, `BrazilLocation`, `Prescription.ExpandSchedule`

**Files:**
- Create: `internal/domain/prescription/scheduled_dose.go`
- Create: `internal/domain/prescription/scheduled_dose_test.go`

**Interfaces:**
- Produces: `prescription.ScheduledDose` struct (exported fields + JSON tags), `prescription.BrazilLocation *time.Location`, `(*prescription.Prescription).ExpandSchedule(loc *time.Location) []ScheduledDose`. Tasks 2 and 3 consume these.

- [ ] **Step 1: Write the failing test**

Create `internal/domain/prescription/scheduled_dose_test.go`:

```go
package prescription

import (
    "testing"
    "time"
)

func newTestPrescription(createdAt time.Time, meds ...Medicament) *Prescription {
    return &Prescription{
        ID:          "pres-1",
        UserID:      "user-1",
        MedicID:     "doc-1",
        Medicaments: meds,
        Active:      true,
        CreatedAt:   createdAt,
        UpdatedAt:   createdAt,
    }
}

func TestBrazilLocation(t *testing.T) {
    if BrazilLocation == nil {
        t.Fatal("BrazilLocation must not be nil")
    }
    name, offset := time.Date(2026, 7, 29, 8, 0, 0, 0, BrazilLocation).Zone()
    if name != "BRT" {
        t.Errorf("zone name = %q, want BRT", name)
    }
    if offset != -3*60*60 {
        t.Errorf("offset = %d, want -10800", offset)
    }
}

func TestPrescription_ExpandSchedule_EmptyMedicaments(t *testing.T) {
    p := newTestPrescription(time.Now())
    got := p.ExpandSchedule(BrazilLocation)
    if len(got) != 0 {
        t.Errorf("expected empty, got %d entries", len(got))
    }
}

func TestPrescription_ExpandSchedule_OnceDailyThreeDoses(t *testing.T) {
    // Created 2026-07-29 00:00 BRT, once daily at 08:00, 3 doses.
    createdAt := time.Date(2026, 7, 29, 0, 0, 0, 0, BrazilLocation)
    p := newTestPrescription(createdAt, Medicament{
        Name:      "AAS",
        Dosage:    "100mg",
        Frequency: "24:00",
        Times:     []string{"08:00"},
        Doses:     3,
    })

    got := p.ExpandSchedule(BrazilLocation)
    if len(got) != 3 {
        t.Fatalf("expected 3 slots, got %d", len(got))
    }
    for i, slot := range got {
        if slot.Status != DoseStatusPending {
            t.Errorf("slot %d status = %q, want PENDING", i, slot.Status)
        }
        if slot.MedicamentName != "AAS" {
            t.Errorf("slot %d medicament = %q, want AAS", i, slot.MedicamentName)
        }
        if slot.Dosage != "100mg" {
            t.Errorf("slot %d dosage = %q, want 100mg", i, slot.Dosage)
        }
        if slot.ScheduledAt.Location().String() != "BRT" {
            t.Errorf("slot %d location = %q, want BRT", i, slot.ScheduledAt.Location().String())
        }
        // 08:00 BRT must hold the expected offset.
        _, off := slot.ScheduledAt.Zone()
        if off != -3*60*60 {
            t.Errorf("slot %d offset = %d, want -10800", i, off)
        }
        if slot.ScheduledAt.Hour() != 8 || slot.ScheduledAt.Minute() != 0 {
            t.Errorf("slot %d wall clock = %02d:%02d, want 08:00", i, slot.ScheduledAt.Hour(), slot.ScheduledAt.Minute())
        }
    }

    // Slots must be one day apart.
    for i := 1; i < len(got); i++ {
        gap := got[i].ScheduledAt.Sub(got[i-1].ScheduledAt)
        if gap != 24*time.Hour {
            t.Errorf("gap[%d-%d] = %v, want 24h", i, i-1, gap)
        }
    }
}

func TestPrescription_ExpandSchedule_TwiceDailyFourDoses(t *testing.T) {
    createdAt := time.Date(2026, 7, 29, 0, 0, 0, 0, BrazilLocation)
    p := newTestPrescription(createdAt, Medicament{
        Name:      "Lisinopril",
        Dosage:    "10mg",
        Frequency: "12:00",
        Times:     []string{"08:00", "20:00"},
        Doses:     4,
    })

    got := p.ExpandSchedule(BrazilLocation)
    if len(got) != 4 {
        t.Fatalf("expected 4 slots, got %d", len(got))
    }
    // Expected wall clocks in BRT: 08:00 day1, 20:00 day1, 08:00 day2, 20:00 day2.
    wantHours := []int{8, 20, 8, 20}
    for i, slot := range got {
        if slot.ScheduledAt.Hour() != wantHours[i] {
            t.Errorf("slot %d hour = %d, want %d", i, slot.ScheduledAt.Hour(), wantHours[i])
        }
    }
}

func TestPrescription_ExpandSchedule_AnchorsAtCreatedAt(t *testing.T) {
    // Created 2026-07-29 06:00 BRT (before 08:00) — first slot = 08:00 same day.
    createdAt := time.Date(2026, 7, 29, 6, 0, 0, 0, BrazilLocation)
    p := newTestPrescription(createdAt, Medicament{
        Name: "AAS", Dosage: "100mg",
        Frequency: "24:00", Times: []string{"08:00"}, Doses: 2,
    })
    got := p.ExpandSchedule(BrazilLocation)
    if len(got) != 2 {
        t.Fatalf("expected 2 slots, got %d", len(got))
    }
    first := got[0].ScheduledAt
    if first.Day() != 29 || first.Hour() != 8 {
        t.Errorf("first slot = day %d hour %d, want day 29 hour 8", first.Day(), first.Hour())
    }
}

func TestPrescription_ExpandSchedule_SkipsPastSlots(t *testing.T) {
    // Created far in the past, 1 dose — depending on `now`, the slot may be past.
    createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, BrazilLocation)
    p := newTestPrescription(createdAt, Medicament{
        Name: "AAS", Dosage: "100mg",
        Frequency: "24:00", Times: []string{"08:00"}, Doses: 1,
    })
    got := p.ExpandSchedule(BrazilLocation)
    // Slot is in 2020 — past now — must be filtered out.
    if len(got) != 0 {
        t.Errorf("expected past slot to be filtered, got %d entries", len(got))
    }
}

func TestPrescription_ExpandSchedule_MultipleMedicaments(t *testing.T) {
    createdAt := time.Date(2026, 7, 29, 0, 0, 0, 0, BrazilLocation)
    p := newTestPrescription(createdAt,
        Medicament{Name: "AAS", Dosage: "100mg", Frequency: "24:00", Times: []string{"08:00"}, Doses: 2},
        Medicament{Name: "Lisinopril", Dosage: "10mg", Frequency: "12:00", Times: []string{"08:00", "20:00"}, Doses: 2},
    )
    got := p.ExpandSchedule(BrazilLocation)
    // 2 + 2 = 4 slots.
    if len(got) != 4 {
        t.Errorf("expected 4 slots (2+2), got %d", len(got))
    }
}
```

Note: `user` is imported but unused — remove the unused import. The corrected header:

```go
package prescription

import (
    "testing"
    "time"
)
```

(The `user` import line was a stray from copy-paste — drop it.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/prescription/ -run TestBrazilLocation -v`
Expected: FAIL — `BrazilLocation` undefined (`undefined: BrazilLocation`).

- [ ] **Step 3: Write the minimum implementation**

Create `internal/domain/prescription/scheduled_dose.go`:

```go
package prescription

import "time"

// BrazilLocation is the fixed timezone used for medication schedule
// wall-clock interpretation in CareConnect. Brazil / BRT, UTC-3, no DST.
// Defined as a package-level constant so the API and scheduler agree
// if/when we want to unify them on a single source of truth.
var BrazilLocation = time.FixedZone("BRT", -3*60*60)

// ScheduledDose is one slot in a user's medication schedule. It is
// produced by expanding a prescription's medicaments and overlaid with
// existing dose_records so that confirmed / missed status wins. JSON
// tags match the existing DoseRecord pattern so the API can return the
// struct directly without a DTO.
type ScheduledDose struct {
    PrescriptionID string     `json:"prescription_id"`
    MedicamentName string     `json:"medicament_name"`
    Dosage         string     `json:"dosage"`
    ScheduledAt    time.Time  `json:"scheduled_at"`
    Status         DoseStatus `json:"status"`
    DoseRecordID   *string    `json:"dose_record_id,omitempty"`
    ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
}

// ExpandSchedule emits one ScheduledDose per planned dose for the
// prescription, anchored to CreatedAt and stepped by the medicament's
// frequency / times. Wall-clock times in medicament.Times are
// interpreted in loc. Slots strictly in the past (before now) are
// skipped so callers can rely on the result to represent upcoming or
// current doses.
func (p *Prescription) ExpandSchedule(loc *time.Location) []ScheduledDose {
    now := time.Now()
    var out []ScheduledDose
    for _, m := range p.Medicaments {
        interval := intervalForMedicament(m)
        first, err := firstScheduleTimeAfter(p.CreatedAt, m.Times, interval, loc)
        if err != nil || len(m.Times) == 0 {
            continue
        }
        for i := 0; i < m.Doses; i++ {
            at := first.Add(time.Duration(i) * interval)
            if at.Before(now) {
                continue
            }
            out = append(out, ScheduledDose{
                PrescriptionID: p.ID,
                MedicamentName: m.Name,
                Dosage:         m.Dosage,
                ScheduledAt:    at,
                Status:         DoseStatusPending,
            })
        }
    }
    return out
}

// intervalForMedicament returns the duration between scheduled doses for
// a medicament, derived from its Frequency string. Mirrors the scheduler
// helper at internal/infrastructure/scheduler/redis_scheduler.go but lives
// in the domain so the API can reuse it without pulling the scheduler
// package as a dependency.
func intervalForMedicament(m Medicament) time.Duration {
    if m.Frequency != "" {
        d, err := parseClockDuration(m.Frequency)
        if err == nil && d > 0 {
            return d
        }
    }
    return 24 * time.Hour
}

// firstScheduleTimeAfter returns the first time at or after start at
// which any of times falls, interpreted in loc. The result is the
// earliest of each `parseClockTime(t)` on or after start.
func firstScheduleTimeAfter(start time.Time, times []string, interval time.Duration, loc *time.Location) (time.Time, error) {
    if interval <= 0 {
        interval = 24 * time.Hour
    }
    var earliest time.Time
    for _, ts := range times {
        h, mi, s, err := parseClockTime(ts)
        if err != nil {
            return time.Time{}, err
        }
        candidate := time.Date(start.Year(), start.Month(), start.Day(), h, mi, s, 0, loc)
        for candidate.Before(start) {
            candidate = candidate.Add(interval)
        }
        if earliest.IsZero() || candidate.Before(earliest) {
            earliest = candidate
        }
    }
    if earliest.IsZero() {
        return time.Time{}, ErrInvalidTimes
    }
    return earliest, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/domain/prescription/ -v`
Expected: PASS for all the new tests in `scheduled_dose_test.go` plus any pre-existing tests in the package.

- [ ] **Step 5: Lint + commit**

Run:

```bash
go vet ./...
golangci-lint run ./internal/domain/prescription/...
```

Expected: no errors.

Then:

```bash
git add internal/domain/prescription/scheduled_dose.go internal/domain/prescription/scheduled_dose_test.go
git commit -m "feat(domain): add ScheduledDose + Prescription.ExpandSchedule + BrazilLocation"
```

---

## Task 2: Application — `DoseRecordQueryHandler.ListScheduleForUser`

**Files:**
- Modify: `internal/application/queries/dose_record_queries.go` (add field, constructor arg, new method + helper)
- Modify: `cmd/api/main.go` (pass `prescriptionRepo` to `NewDoseRecordQueryHandler`)
- Create: `internal/application/queries/dose_record_schedule_test.go`

**Interfaces:**
- Consumes: `prescription.ScheduledDose`, `prescription.BrazilLocation`, `(*prescription.Prescription).ExpandSchedule`. `prescription.Repository.FindActiveByUserID(ctx, userID) ([]*prescription.Prescription, error)` is already on the existing repo. `prescription.DoseRecordRepository.FindByUserID(ctx, userID) ([]*prescription.DoseRecord, error)` is already on the existing repo.
- Produces: `queries.ListDoseScheduleQuery{UserID, CallerID string}`, `(*queries.DoseRecordQueryHandler).ListScheduleForUser(ctx, q) ([]*prescription.ScheduledDose, error)`. Task 3 consumes these.

- [ ] **Step 1: Write the failing test**

Create `internal/application/queries/dose_record_schedule_test.go`:

```go
package queries

import (
    "context"
    "errors"
    "testing"
    "time"

    "github.com.br/lucas-mezencio/pdsi1/internal/application"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

type stubPrescriptionRepo struct {
    active []*prescription.Prescription
}

func (s *stubPrescriptionRepo) Save(_ context.Context, _ *prescription.Prescription) error {
    return nil
}
func (s *stubPrescriptionRepo) FindAll(_ context.Context) ([]*prescription.Prescription, error) {
    return nil, nil
}
func (s *stubPrescriptionRepo) FindByID(_ context.Context, _ string) (*prescription.Prescription, error) {
    return nil, prescription.ErrPrescriptionNotFound
}
func (s *stubPrescriptionRepo) FindByUserID(_ context.Context, _ string) ([]*prescription.Prescription, error) {
    return nil, nil
}
func (s *stubPrescriptionRepo) FindByMedicID(_ context.Context, _ string) ([]*prescription.Prescription, error) {
    return nil, nil
}
func (s *stubPrescriptionRepo) FindActive(_ context.Context) ([]*prescription.Prescription, error) {
    return nil, nil
}
func (s *stubPrescriptionRepo) FindActiveByUserID(_ context.Context, userID string) ([]*prescription.Prescription, error) {
    if userID == "user-1" {
        return s.active, nil
    }
    return nil, nil
}
func (s *stubPrescriptionRepo) Delete(_ context.Context, _ string) error { return nil }
func (s *stubPrescriptionRepo) Exists(_ context.Context, _ string) (bool, error) {
    return false, nil
}

type stubDoseRepo struct {
    records []*prescription.DoseRecord
}

func (s *stubDoseRepo) Save(_ context.Context, _ *prescription.DoseRecord) error {
    return nil
}
func (s *stubDoseRepo) FindByID(_ context.Context, _ string) (*prescription.DoseRecord, error) {
    return nil, prescription.ErrDoseRecordNotFound
}
func (s *stubDoseRepo) FindByUserID(_ context.Context, userID string) ([]*prescription.DoseRecord, error) {
    if userID == "user-1" {
        return s.records, nil
    }
    return nil, nil
}
func (s *stubDoseRepo) FindByPrescriptionID(_ context.Context, _ string) ([]*prescription.DoseRecord, error) {
    return nil, nil
}
func (s *stubDoseRepo) FindPendingBefore(_ context.Context, _ time.Time) ([]*prescription.DoseRecord, error) {
    return nil, nil
}

type allowAllUserRepo struct{}

func (allowAllUserRepo) FindByID(_ context.Context, _ string) (*user.User, error) {
    return nil, user.ErrUserNotFound
}
func (allowAllUserRepo) FindByEmail(_ context.Context, _ string) (*user.User, error) {
    return nil, user.ErrUserNotFound
}
func (allowAllUserRepo) Save(_ context.Context, _ *user.User) error { return nil }
func (allowAllUserRepo) Delete(_ context.Context, _ string) error  { return nil }
func (allowAllUserRepo) FindByFirebaseID(_ context.Context, _ string) (*user.User, error) {
    return nil, user.ErrUserNotFound
}
func (allowAllUserRepo) IsLinked(_ context.Context, _, _ string) (bool, error) {
    return true, nil
}
func (allowAllUserRepo) FindCaregivers(_ context.Context, _ string) ([]*user.User, error) {
    return nil, nil
}
func (allowAllUserRepo) FindCharges(_ context.Context, _ string) ([]*user.User, error) {
    return nil, nil
}
func (allowAllUserRepo) UpdatePasswordHash(_ context.Context, _ string, _ string) error {
    return nil
}

func TestListScheduleForUser_ReconstructsPendingSlots(t *testing.T) {
    createdAt := time.Now().Add(-1 * time.Hour)
    p := &prescription.Prescription{
        ID:        "pres-1",
        UserID:    "user-1",
        MedicID:   "doc-1",
        Active:    true,
        CreatedAt: createdAt,
        Medicaments: []prescription.Medicament{
            {Name: "AAS", Dosage: "100mg", Frequency: "24:00", Times: []string{createdAt.Add(2 * time.Hour).Format("15:04")}, Doses: 2},
        },
    }

    h := NewDoseRecordQueryHandler(&stubDoseRepo{}, allowAllUserRepo{}, &stubPrescriptionRepo{active: []*prescription.Prescription{p}})
    got, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "user-1", CallerID: "user-1"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(got) != 2 {
        t.Fatalf("expected 2 slots, got %d", len(got))
    }
    for i, slot := range got {
        if slot.Status != prescription.DoseStatusPending {
            t.Errorf("slot %d status = %q, want PENDING", i, slot.Status)
        }
        if slot.DoseRecordID != nil {
            t.Errorf("slot %d dose_record_id = %v, want nil", i, *slot.DoseRecordID)
        }
    }
}

func TestListScheduleForUser_OverlaysTakenRecord(t *testing.T) {
    createdAt := time.Now().Add(-1 * time.Hour)
    scheduleTime := createdAt.Add(2 * time.Hour)
    p := &prescription.Prescription{
        ID:        "pres-1",
        UserID:    "user-1",
        MedicID:   "doc-1",
        Active:    true,
        CreatedAt: createdAt,
        Medicaments: []prescription.Medicament{
            {Name: "AAS", Dosage: "100mg", Frequency: "24:00", Times: []string{scheduleTime.Format("15:04")}, Doses: 1},
        },
    }
    confirmed := scheduleTime.Add(5 * time.Minute)
    rec := &prescription.DoseRecord{
        ID:             "rec-1",
        PrescriptionID: "pres-1",
        UserID:         "user-1",
        MedicamentName: "AAS",
        Dosage:         "100mg",
        ScheduledAt:    scheduleTime,
        Status:         prescription.DoseStatusTaken,
        ConfirmedAt:    &confirmed,
    }
    h := NewDoseRecordQueryHandler(&stubDoseRepo{records: []*prescription.DoseRecord{rec}}, allowAllUserRepo{}, &stubPrescriptionRepo{active: []*prescription.Prescription{p}})

    got, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "user-1", CallerID: "user-1"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(got) != 1 {
        t.Fatalf("expected 1 slot, got %d", len(got))
    }
    if got[0].Status != prescription.DoseStatusTaken {
        t.Errorf("status = %q, want TAKEN", got[0].Status)
    }
    if got[0].DoseRecordID == nil || *got[0].DoseRecordID != "rec-1" {
        t.Errorf("dose_record_id = %v, want rec-1", got[0].DoseRecordID)
    }
    if got[0].ConfirmedAt == nil {
        t.Errorf("confirmed_at should be set for TAKEN slot")
    }
}

func TestListScheduleForUser_PreservesOrphanRecord(t *testing.T) {
    // No active prescription for user-1 but they have a TAKEN record.
    takenAt := time.Now().Add(-30 * time.Minute)
    rec := &prescription.DoseRecord{
        ID: "rec-orphan", PrescriptionID: "pres-gone", UserID: "user-1",
        MedicamentName: "OldDrug", Dosage: "5mg",
        ScheduledAt: takenAt, Status: prescription.DoseStatusTaken,
    }
    h := NewDoseRecordQueryHandler(&stubDoseRepo{records: []*prescription.DoseRecord{rec}}, allowAllUserRepo{}, &stubPrescriptionRepo{})

    got, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "user-1", CallerID: "user-1"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(got) != 1 {
        t.Fatalf("expected 1 orphan slot, got %d", len(got))
    }
    if got[0].DoseRecordID == nil || *got[0].DoseRecordID != "rec-orphan" {
        t.Errorf("dose_record_id = %v, want rec-orphan", got[0].DoseRecordID)
    }
    if got[0].MedicamentName != "OldDrug" {
        t.Errorf("medicament_name = %q, want OldDrug", got[0].MedicamentName)
    }
}

func TestListScheduleForUser_RejectsEmptyUserID(t *testing.T) {
    h := NewDoseRecordQueryHandler(&stubDoseRepo{}, allowAllUserRepo{}, &stubPrescriptionRepo{})
    _, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "", CallerID: ""})
    if !errors.Is(err, application.ErrInvalidInput) {
        t.Errorf("err = %v, want ErrInvalidInput", err)
    }
}

type restrictiveUserRepo struct{ allowAllUserRepo }

func (restrictiveUserRepo) IsLinked(_ context.Context, _, _ string) (bool, error) {
    return false, nil
}

func TestListScheduleForUser_ForbiddenForUnlinkedCaller(t *testing.T) {
    h := NewDoseRecordQueryHandler(&stubDoseRepo{}, &restrictiveUserRepo{}, &stubPrescriptionRepo{})
    _, err := h.ListScheduleForUser(context.Background(), ListDoseScheduleQuery{UserID: "user-1", CallerID: "user-2"})
    if !errors.Is(err, application.ErrForbidden) {
        t.Errorf("err = %v, want ErrForbidden", err)
    }
}
```

Note: `user` and `stubUserRepo` types must satisfy `user.Repository`. The codebase's interface lives at `internal/domain/user/repository.go`. If the function set differs, adjust the stubs accordingly — check the interface before running the test. The stubs above are written against the methods used by `DoseRecordQueryHandler` (FindByID, IsLinked, FindCaregivers, FindCharges, plus Save/Delete/Save-related methods used elsewhere). For this test only `IsLinked` matters, so a minimal stub is acceptable; if the interface requires all methods, keep the full stub set.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/application/queries/ -run TestListScheduleForUser -v`
Expected: FAIL — `NewDoseRecordQueryHandler` arity mismatch (current signature is 2 args, new one is 3) and `ListScheduleForUser` undefined.

- [ ] **Step 3: Update `internal/application/queries/dose_record_queries.go`**

Modify the file to:
1. Add `"sort"` and `"time"` imports if not present.
2. Add `prescriptionRepo prescription.Repository` field to the `DoseRecordQueryHandler` struct.
3. Update `NewDoseRecordQueryHandler` to accept `prescription.Repository` as the third arg and store it.
4. Add `ListDoseScheduleQuery` struct.
5. Add `ListScheduleForUser` method.
6. Add `overlayKey` private helper.

The final file (showing only the changed sections; preserve the rest verbatim):

```go
package queries

import (
    "context"
    "errors"
    "sort"
    "strings"
    "time"

    "github.com.br/lucas-mezencio/pdsi1/internal/application"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// ListDoseScheduleQuery retrieves the full planned dose schedule for a user.
type ListDoseScheduleQuery struct {
    UserID   string
    CallerID string
}

// (Existing ListDoseRecordsQuery, ListCaregiversQuery, ListChargesQuery,
// ListCaregiverInvitationsQuery, GetInvitationByTokenQuery stay unchanged.)

// DoseRecordQueryHandler handles dose record read operations.
type DoseRecordQueryHandler struct {
    doseRepo         prescription.DoseRecordRepository
    prescriptionRepo prescription.Repository
    userRepo         user.Repository
}

// NewDoseRecordQueryHandler creates a DoseRecordQueryHandler.
func NewDoseRecordQueryHandler(
    doseRepo prescription.DoseRecordRepository,
    userRepo user.Repository,
    prescriptionRepo prescription.Repository,
) *DoseRecordQueryHandler {
    return &DoseRecordQueryHandler{
        doseRepo:         doseRepo,
        userRepo:         userRepo,
        prescriptionRepo: prescriptionRepo,
    }
}

// ListByUser retrieves dose records for a user (with access control).
// (unchanged)

// checkAccess — unchanged.

// ListScheduleForUser returns the user's full medication schedule:
// every planned dose across active prescriptions, overlaid with existing
// dose_records (TAKEN / MISSED win). Records with no matching
// prescription slot are preserved as orphans.
func (h *DoseRecordQueryHandler) ListScheduleForUser(ctx context.Context, q ListDoseScheduleQuery) ([]*prescription.ScheduledDose, error) {
    if q.UserID == "" {
        return nil, application.ErrInvalidInput
    }
    if err := h.checkAccess(ctx, q.CallerID, q.UserID); err != nil {
        return nil, err
    }

    prescriptions, err := h.prescriptionRepo.FindActiveByUserID(ctx, q.UserID)
    if err != nil {
        return nil, err
    }

    records, err := h.doseRepo.FindByUserID(ctx, q.UserID)
    if err != nil {
        return nil, err
    }

    overlay := make(map[string]*prescription.DoseRecord, len(records))
    for _, r := range records {
        overlay[overlayKey(r.PrescriptionID, r.ScheduledAt, r.MedicamentName)] = r
    }

    out := make([]*prescription.ScheduledDose, 0, len(records))
    seen := make(map[string]struct{}, len(records))
    for _, p := range prescriptions {
        for _, slot := range p.ExpandSchedule(prescription.BrazilLocation) {
            k := overlayKey(slot.PrescriptionID, slot.ScheduledAt, slot.MedicamentName)
            seen[k] = struct{}{}
            if rec, ok := overlay[k]; ok {
                id := rec.ID
                confirmed := rec.ConfirmedAt
                out = append(out, &prescription.ScheduledDose{
                    PrescriptionID: rec.PrescriptionID,
                    MedicamentName: rec.MedicamentName,
                    Dosage:         rec.Dosage,
                    ScheduledAt:    rec.ScheduledAt,
                    Status:         rec.Status,
                    DoseRecordID:   &id,
                    ConfirmedAt:    confirmed,
                })
                continue
            }
            slotCopy := slot
            out = append(out, &slotCopy)
        }
    }

    for _, r := range records {
        k := overlayKey(r.PrescriptionID, r.ScheduledAt, r.MedicamentName)
        if _, ok := seen[k]; ok {
            continue
        }
        id := r.ID
        confirmed := r.ConfirmedAt
        out = append(out, &prescription.ScheduledDose{
            PrescriptionID: r.PrescriptionID,
            MedicamentName: r.MedicamentName,
            Dosage:         r.Dosage,
            ScheduledAt:    r.ScheduledAt,
            Status:         r.Status,
            DoseRecordID:   &id,
            ConfirmedAt:    confirmed,
        })
    }

    sort.Slice(out, func(i, j int) bool {
        return out[i].ScheduledAt.Before(out[j].ScheduledAt)
    })

    return out, nil
}

// overlayKey builds the canonical key used to match a planned slot to
// its materialized dose_record. All times are normalized to UTC to avoid
// zone-name drift between reconstruction and DB read.
func overlayKey(prescriptionID string, scheduledAt time.Time, medicamentName string) string {
    return prescriptionID + "|" + scheduledAt.UTC().Format(time.RFC3339Nano) + "|" + medicamentName
}

// (Existing LinkedUserQueryHandler section unchanged.)
```

- [ ] **Step 4: Update `cmd/api/main.go` call site**

In `cmd/api/main.go`, find:

```go
doseQueries := queries.NewDoseRecordQueryHandler(doseRecordRepo, userRepo)
```

and replace with:

```go
doseQueries := queries.NewDoseRecordQueryHandler(doseRecordRepo, userRepo, prescriptionRepo)
```

`prescriptionRepo` is defined earlier in the same `main()` (line ~140 as `prescriptionRepo := database.NewPrescriptionRepository(db)`).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/application/queries/ -v`
Expected: PASS for all new tests in `dose_record_schedule_test.go` and PASS for any pre-existing tests in the package.

- [ ] **Step 6: Lint + commit**

```bash
go vet ./...
golangci-lint run ./internal/application/queries/... ./cmd/...
git add internal/application/queries/dose_record_queries.go \
        internal/application/queries/dose_record_schedule_test.go \
        cmd/api/main.go
git commit -m "feat(application): add DoseRecordQueryHandler.ListScheduleForUser"
```

---

## Task 3: API — handler + route + smoke test

**Files:**
- Modify: `internal/api/extended_server.go` (add `ListDoseSchedule` method)
- Modify: `internal/api/router.go` (register route)
- Create: `internal/api/list_dose_schedule_handler_test.go`

**Interfaces:**
- Consumes: `queries.ListDoseScheduleQuery`, `(*queries.DoseRecordQueryHandler).ListScheduleForUser`.
- Produces: HTTP handler `(*httpapi.ExtendedServer).ListDoseSchedule`, route `GET /api/v1/users/{userId}/doses`.

- [ ] **Step 1: Write the failing smoke test**

Create `internal/api/list_dose_schedule_handler_test.go`:

```go
package httpapi

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/go-chi/chi/v5"

    "github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

type capturingScheduleDoseRepo struct {
    stubDoseRepoForExport
    gotUserID   string
    gotCallerID string
    returnValue []*prescription.ScheduledDose
}

func (s *capturingScheduleDoseRepo) FindActiveByUserID(_ context.Context, _ string) ([]*prescription.Prescription, error) {
    return nil, nil
}

func (s *capturingScheduleDoseRepo) FindByUserID(ctx context.Context, userID string) ([]*prescription.DoseRecord, error) {
    s.gotUserID = userID
    return nil, nil
}

// TestListDoseSchedule_DoesNotPassFirebaseUIDToIsLinked locks in the
// same local-UUID invariant as TestListDoseRecords_DoesNotPassFirebaseUIDToIsLinked
// (internal/api/list_dose_records_handler_test.go:63) for the new
// schedule endpoint.
func TestListDoseSchedule_DoesNotPassFirebaseUIDToIsLinked(t *testing.T) {
    const (
        localUserUUID = "6b1fb275-2efa-4309-b34a-2f8b8abf6e6c"
        firebaseUID   = "uq1OEy7P0UPOvJIiFwCDNQxMJAW2"
    )

    doseRepo := &capturingScheduleDoseRepo{returnValue: nil}
    userRepo := &recordingUserRepo{}
    prescriptionRepo := &capturingScheduleDoseRepo{} // satisfies prescription.Repository via embedded stubs

    ext := &ExtendedServer{
        userRepo:    userRepo,
        doseQueries: queries.NewDoseRecordQueryHandler(doseRepo, userRepo, prescriptionRepo),
    }

    req := newChiRequestWithParam(http.MethodGet, "/api/v1/users/"+localUserUUID+"/doses", "userId", localUserUUID, localUserUUID)
    rr := httptest.NewRecorder()
    ext.ListDoseSchedule(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
    }
    if doseRepo.gotUserID != localUserUUID {
        t.Fatalf("doseRepo.FindByUserID called with %q, want %q", doseRepo.gotUserID, localUserUUID)
    }
    for _, call := range userRepo.linkedCalls {
        if call.CaregiverID == firebaseUID {
            t.Fatalf("IsLinked received the Firebase UID %q — bug regressed", firebaseUID)
        }
    }

    var decoded []*prescription.ScheduledDose
    if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
        t.Fatalf("response should be a JSON array of ScheduledDose: %v body=%s", err, rr.Body.String())
    }
}
```

The `capturingScheduleDoseRepo` embeds `stubDoseRepoForExport` so it satisfies `prescription.DoseRecordRepository` (which `DoseRecordQueryHandler` needs) and declares `FindActiveByUserID` / `FindByUserID` so it satisfies `prescription.Repository`. Verify `stubDoseRepoForExport` exists in the api test package — search `internal/api/` for it; if missing, replace the embedding with concrete stubs following the patterns in `internal/api/list_dose_records_handler_test.go` and the DoseRecord test stubs.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/api/ -run TestListDoseSchedule -v`
Expected: FAIL — `(*ExtendedServer).ListDoseSchedule` undefined.

- [ ] **Step 3: Add the handler in `internal/api/extended_server.go`**

Find the "--- Dose record endpoints ---" comment block (`extended_server.go:244`) and append after the `MarkDoseMissed` method:

```go
// ListDoseSchedule handles GET /users/{userId}/doses
func (s *ExtendedServer) ListDoseSchedule(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "userId")
    callerID := callerUserID(r)

    items, err := s.doseQueries.ListScheduleForUser(r.Context(), queries.ListDoseScheduleQuery{
        UserID:   userID,
        CallerID: callerID,
    })
    if err != nil {
        writeExtendedError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, items)
}
```

- [ ] **Step 4: Register the route in `internal/api/router.go`**

Find the "Dose records" route block (`router.go:55-58`):

```go
// Dose records
r.Get("/users/{userId}/dose-records", ext.ListDoseRecords)
r.Post("/dose-records/{doseRecordId}/confirm", ext.ConfirmDose)
r.Post("/dose-records/{doseRecordId}/miss", ext.MarkDoseMissed)
```

Insert one line between `ListDoseRecords` and `ConfirmDose`:

```go
// Dose records
r.Get("/users/{userId}/dose-records", ext.ListDoseRecords)
r.Get("/users/{userId}/doses", ext.ListDoseSchedule)
r.Post("/dose-records/{doseRecordId}/confirm", ext.ConfirmDose)
r.Post("/dose-records/{doseRecordId}/miss", ext.MarkDoseMissed)
```

- [ ] **Step 5: Run all tests and verify pass**

```bash
go test ./...
```

Expected: PASS across the board. If the `internal/api` test fails due to missing stubs in `capturingScheduleDoseRepo`, inspect the actual `prescription.Repository` interface and `prescription.DoseRecordRepository` interface in `internal/domain/prescription/repository.go` and add the missing methods.

- [ ] **Step 6: Lint + commit**

```bash
go vet ./...
golangci-lint run
git add internal/api/extended_server.go \
        internal/api/router.go \
        internal/api/list_dose_schedule_handler_test.go
git commit -m "feat(api): expose GET /users/{userId}/doses"
```

---

## Task 4: OpenAPI documentation

**Files:**
- Modify: `docs/api.yaml`

- [ ] **Step 1: Add the `DoseSchedule` tag**

After the last tag entry (around `api.yaml:30` — `Caregivers`), insert:

```yaml
  - name: DoseSchedule
    description: User medication schedule — planned doses overlaid with confirmation history.
```

- [ ] **Step 2: Add the `ScheduledDose` schema**

After the existing `DoseRecord` schema (after `api.yaml:1424`, before `CaregiverInvitation`), insert:

```yaml
    ScheduledDose:
      type: object
      description: |
        A single planned dose in the user's medication schedule. Produced
        by expanding every active prescription's medicaments; existing
        dose records (TAKEN / MISSED) take precedence over reconstructed
        PENDING entries. `dose_record_id` is null until the scheduler
        materializes the underlying record at notification-fire time.
      properties:
        prescription_id:
          type: string
          format: uuid
        medicament_name:
          type: string
          example: "AAS"
        dosage:
          type: string
          example: "100mg"
        scheduled_at:
          type: string
          format: date-time
          description: Planned time in BRT (UTC-3).
        status:
          type: string
          enum: [PENDING, TAKEN, MISSED]
        dose_record_id:
          type: string
          format: uuid
          nullable: true
        confirmed_at:
          type: string
          format: date-time
          nullable: true
      required:
        - prescription_id
        - medicament_name
        - dosage
        - scheduled_at
        - status
```

- [ ] **Step 3: Add the new `/users/{userId}/doses` path**

After the LGPD comment block (`api.yaml:540`), before the "Doctor endpoints" comment (`api.yaml:542`), insert:

```yaml
  # Dose schedule endpoints
  /users/{userId}/doses:
    get:
      tags:
        - DoseSchedule
      summary: List the user's full medication schedule
      description: |
        Returns every dose the user is expected to take across all active
        prescriptions, overlaid with confirmation history. The schedule
        is reconstructed from each active prescription's `created_at` +
        every medicament's `times[]` × `doses`. Existing `dose_records`
        override the reconstructed PENDING status with their actual
        status (TAKEN or MISSED).

        Wall-clock times in the `medicament.times[]` strings are
        interpreted in BRT (UTC-3, no DST) regardless of the server's
        local timezone.
      operationId: listUserDoseSchedule
      parameters:
        - $ref: '#/components/parameters/UserId'
      responses:
        '200':
          description: Full medication schedule sorted by scheduled_at ascending.
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/ScheduledDose'
        '403':
          $ref: '#/components/responses/Forbidden'
        '404':
          $ref: '#/components/responses/NotFound'

  /users/{userId}/dose-records:
    get:
      tags:
        - DoseSchedule
      summary: List dose records for a user (history only)
      description: |
        Returns every persisted dose record for the user, newest first.
        Only doses whose notification has already fired appear here —
        for the full planned schedule use `GET /users/{userId}/doses`.
      operationId: listDoseRecords
      parameters:
        - $ref: '#/components/parameters/UserId'
      responses:
        '200':
          description: Dose records
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/DoseRecord'
        '403':
          $ref: '#/components/responses/Forbidden'

  /dose-records/{doseRecordId}/confirm:
    post:
      tags:
        - DoseSchedule
      summary: Mark a dose as taken
      operationId: confirmDose
      parameters:
        - name: doseRecordId
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Updated dose record
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DoseRecord'
        '403':
          $ref: '#/components/responses/Forbidden'
        '404':
          $ref: '#/components/responses/NotFound'

  /dose-records/{doseRecordId}/miss:
    post:
      tags:
        - DoseSchedule
      summary: Mark a dose as missed
      operationId: markDoseMissed
      parameters:
        - name: doseRecordId
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Updated dose record
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DoseRecord'
        '403':
          $ref: '#/components/responses/Forbidden'
        '404':
          $ref: '#/components/responses/NotFound'
```

- [ ] **Step 4: Verify the YAML**

Run:

```bash
python3 -c "import yaml; yaml.safe_load(open('docs/api.yaml'))"
```

Expected: no error. (If `pyyaml` is missing, use any YAML validator: `npx js-yaml docs/api.yaml` or `docker run --rm -v "$PWD:/data" mikefarah/yq eval-all '. == true' /data/docs/api.yaml`.)

- [ ] **Step 5: Commit**

```bash
git add docs/api.yaml
git commit -m "docs(api): document /users/{userId}/doses + dose-record endpoints"
```

---

## Task 5: Regenerate oapi-codegen

**Files:**
- Regenerated (potentially modified): `internal/api/gen/types.gen.go`, `internal/api/gen/server.gen.go`

- [ ] **Step 1: Run `go generate`**

```bash
cd /home/lucas/projects/care-connect && go generate ./internal/api/gen/...
```

Expected: `oapi-codegen` produces clean output (no errors). The `gen/server.gen.go` file should be unchanged because the new endpoint is not in the generated server (it's a manual extended route). The `gen/types.gen.go` should be unchanged because `ScheduledDose` is not used by any operation in the generated server.

- [ ] **Step 2: Inspect the diff**

```bash
git status
git diff internal/api/gen/
```

Expected: only the generated files appear as modified if anything; no source code changes.

- [ ] **Step 3: Run tests one more time**

```bash
go test ./...
golangci-lint run
```

Expected: PASS.

- [ ] **Step 4: Commit (only if files changed)**

```bash
git diff --quiet internal/api/gen/ || git add internal/api/gen/ && git commit -m "chore(gen): regenerate oapi-codegen types + server"
```

If nothing changed, no commit is needed — skip this step.

---

## Self-Review

**1. Spec coverage:**
- §1 Goal 1 (new endpoint) — Tasks 1, 2, 3.
- §1 Goal 2 (OpenAPI coverage) — Task 4.
- §1 Goal 3 (keep existing endpoints) — Tasks 3 (no removal) + 5 (regen doesn't touch existing).
- §2 decision 1 (path) — Task 3.
- §2 decision 2 (hybrid) — Task 2 overlay logic + Task 1 reconstruction.
- §2 decision 3 (default scope = all active) — Task 2 step 3 (`FindActiveByUserID`).
- §2 decision 4 (one tag) — Task 4 step 1.
- §2 decision 5 (BRT) — Task 1 (`BrazilLocation`).
- §2 decision 6 (scheduler unchanged) — not touched anywhere.
- §4 domain — Task 1.
- §5 application — Task 2.
- §6 HTTP layer — Task 3.
- §7 OpenAPI — Task 4.
- §8 accepted limitation — noted in spec, no code changes.
- §9 testing — Tasks 1, 2, 3.
- §10 branch & commits — Tasks 1–5 each commit independently.

**2. Placeholder scan:** no TBD / TODO / "implement later" / "similar to Task N". All steps show full code or full commands. Good.

**3. Type consistency:**
- `ScheduledDose` fields defined in Task 1 (`PrescriptionID`, `MedicamentName`, `Dosage`, `ScheduledAt`, `Status`, `DoseRecordID *string`, `ConfirmedAt *time.Time`) match every usage in Task 2 overlay and orphan branches. ✓
- `BrazilLocation` referenced by Task 2 (`prescription.BrazilLocation`) defined in Task 1. ✓
- `overlayKey` private helper introduced in Task 2 used only in Task 2. ✓
- `ListDoseScheduleQuery` struct fields used identically in Task 2 implementation and Task 3 handler. ✓
- `ExtendedServer.ListDoseSchedule` signature matches the test's expectation (`rr := httptest.NewRecorder(); ext.ListDoseSchedule(rr, req)`). ✓

All consistent.
---

## Deviations recorded during implementation

The following two deviations from the plan above were applied during the
2026-07-29 implementation. Both were plan defects, not implementation
defects; the implementation choices are correct.

1. **Task 1, Step 1 — hardcoded dates replaced with relative ones.**
   The plan's tests used `time.Date(2026, 7, 29, ...)` for `createdAt`.
   Since the plan was written on 2026-07-29, those slots were all in the
   past the moment the tests ran later that day, and `ExpandSchedule`'s
   past-slot filter dropped them, causing the tests to fail (got 2
   slots instead of 3, etc.). The implementer replaced the hardcoded
   dates with `time.Now().In(BrazilLocation).AddDate(0, 0, 1)` (start
   of tomorrow). This is what `scheduled_dose_test.go` now contains.

2. **Task 2, Step 3 — `overlayKey` truncates to the minute, not
   `RFC3339Nano`.** The plan defined `overlayKey` with
   `scheduledAt.UTC().Format(time.RFC3339Nano)`. In production this is
   fine because both reconstructed slots and scheduler-created records
   use `time.Date(..., 0)` (nanos=0). In tests, however, the
   `DoseRecord.ScheduledAt` was sourced from `time.Now().Add(2 * time.Hour)`
   which carries sub-second nanos; the reconstructed slot had nanos=0,
   so the overlay missed and the orphan pass appended the record
   again. The implementer changed `overlayKey` to
   `Truncate(time.Minute).Format(time.RFC3339)` and updated the
   docstring. This is what `dose_record_queries.go` now contains.

3. **Task 5 — codegen regen skipped.** `go generate ./internal/api/gen/...`
   fails to compile the result against the pinned
   `github.com/oapi-codegen/runtime v1.4.0`. This is a pre-existing
   toolchain skew (codegen v2.8.0 vs runtime v1.4.0) that surfaces the
   fact that the spec gained paths (`/invitations/*`) in commit
   `e523167` without a corresponding regen — the existing
   `*httpapi.Server` doesn't implement those paths because they are
   wired manually as extended routes. The new endpoint
   `/users/{userId}/doses` is itself an extended route, so the regen
   would not have produced a useful diff even if it had built. The
   pre-existing skew is left for a separate PR that bumps the runtime
   or pins codegen to v2.7.0.
