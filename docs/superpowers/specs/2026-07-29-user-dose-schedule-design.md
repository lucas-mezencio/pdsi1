# Design: User Dose Schedule Endpoint + OpenAPI Coverage

**Date:** 2026-07-29
**Status:** Approved (design)
**Scope:** New `GET /api/v1/users/{userId}/doses` endpoint that returns the
user's full medication schedule (planned doses overlaid with confirmation
history). Full OpenAPI documentation for the new endpoint and for the three
already-implemented dose-record endpoints.

## 1. Goals

1. Give the client (mobile app, dashboard) a single endpoint that returns
   "everything the user has to take" across every active prescription,
   crossed with "what they've already taken" from `dose_records`.
2. Document every dose-related HTTP endpoint in the OpenAPI spec so Swagger
   UI / generated clients reflect reality.
3. Keep the existing `/dose-records/*` endpoints intact — they continue to
   expose the raw history.

## 2. Decisions recap

| #  | Decision                                       | Choice                                                                 |
|----|------------------------------------------------|------------------------------------------------------------------------|
| 1  | Endpoint name                                  | `GET /api/v1/users/{userId}/doses`                                     |
| 2  | Completeness                                   | Hybrid — reconstruct from active prescriptions, overlay dose_records  |
| 3  | Default time scope                             | All active prescriptions (no `?from` / `?to` in v1)                    |
| 4  | OpenAPI tagging                                | One tag — `DoseSchedule` — for both the new and history endpoints     |
| 5  | Timezone                                       | Brazil (UTC-3) hardcoded for wall-clock interpretation                |
| 6  | Scheduler timezone                             | **Not changed.** Accepted limitation: dispatcher uses server-local tz  |
| 7  | Reconstruction layer                           | Domain (`Prescription.ExpandSchedule`) + application overlay          |
| 8  | HTTP wiring                                    | Manual extended route, mirrors `/users/{userId}/dose-records`         |
| 9  | Branch                                         | `feat/dose-schedule-endpoint` (off `main`)                            |

## 3. Background

The scheduler creates `dose_records` only at the moment a notification fires
(`internal/infrastructure/scheduler/worker.go:136`, via
`DoseRecordStore.CreatePending`). Doses scheduled in the future are therefore
invisible to `/users/{userId}/dose-records` until the scheduler runs.
Reconstruction in the domain layer closes that gap.

The `Prescription.GetAllNotificationTimes` method
(`internal/domain/prescription/prescription.go:74`) already enumerates
`(medicament, time)` pairs but does not step through `Doses` total slots —
that work needs to happen for this endpoint.

## 4. Domain layer

New file: `internal/domain/prescription/scheduled_dose.go`.

```go
package prescription

// ScheduledDose is one slot in a user's medication schedule. It is produced
// by expanding a prescription's medicaments and overlaid with existing
// dose_records so that confirmed/missed status wins.
type ScheduledDose struct {
    PrescriptionID string
    MedicamentName string
    Dosage         string
    ScheduledAt    time.Time
    Status         DoseStatus // PENDING | TAKEN | MISSED
    DoseRecordID   *string    // nil until scheduler materializes
    ConfirmedAt    *time.Time // nil unless Status == TAKEN
}
```

New method on `*Prescription`:

```go
// ExpandSchedule emits one ScheduledDose per planned dose for the
// prescription, anchored to CreatedAt and stepped by the medicament's
// frequency / times. Times are interpreted in loc (wall clock).
func (p *Prescription) ExpandSchedule(loc *time.Location) []ScheduledDose
```

Implementation notes:

- Walks `p.Medicaments`, computing `interval` and `firstTime` exactly the
  same way as `redis_scheduler.go` (`intervalForSchedule`, `nextScheduleTime`).
  Those helpers already exist in `internal/infrastructure/scheduler` and
  could be extracted into the prescription package; if we don't extract,
  duplicate them in the prescription package's `expand.go` (preferred — the
  scheduler package has its own copies anyway).
- For each medicament: emit `medicament.Doses` rows starting at the first
  slot `>= now` (where `now` is the query time). Subsequent slots are
  `firstTime + n*interval` for `n = 0..Doses-1`.
- Timezone: `loc` is passed in by the caller. The Brazil location constant
  lives in the prescription package:

  ```go
  // BrazilLocation is the fixed timezone used for medication schedule
  // wall-clock interpretation. CareConnect targets the Brazilian market
  // (BRT / UTC-3 with no DST). Defined as a package-level constant so the
  // API and scheduler agree if/when we want to unify them.
  var BrazilLocation = time.FixedZone("BRT", -3*60*60)
  ```

## 5. Application layer

New method on `*DoseRecordQueryHandler`
(`internal/application/queries/dose_record_queries.go`):

```go
type ListDoseScheduleQuery struct {
    UserID   string
    CallerID string
}

func (h *DoseRecordQueryHandler) ListScheduleForUser(
    ctx context.Context, q ListDoseScheduleQuery,
) ([]*prescription.ScheduledDose, error)
```

Flow:

1. Validate `UserID` non-empty → `application.ErrInvalidInput` otherwise.
2. `checkAccess(ctx, q.CallerID, q.UserID)` — same helper as `ListByUser`
   (`dose_record_queries.go:72`).
3. `prescriptionRepo.FindActiveByUserID(ctx, q.UserID)` — get every active
   prescription.
4. `doseRepo.FindByUserID(ctx, q.UserID)` — get every dose record.
5. Build a lookup map for overlay:
   `map[string]*prescription.DoseRecord` keyed by
   `key(prescriptionID, scheduledAt, medicamentName)`
   where `key` is `"<presID>|<scheduledAt.UTC().Format(time.RFC3339Nano)>|<medicamentName>"`.
6. For each prescription call `p.ExpandSchedule(prescription.BrazilLocation)`,
   producing planned rows. For each row, if an overlay record exists at the
   same key, copy `ID`, `Status`, `ConfirmedAt` onto the row.
7. For every existing dose record whose key is **not** represented by any
   prescription slot (orphan — prescription deactivated, medicament removed,
   or schedule modified after the record was created), append a row with
   the record's own fields. Keeps the user from losing history they can
   still see elsewhere.
8. Sort result by `ScheduledAt ASC`. Return.

`prescriptionRepo` is not currently injected into `DoseRecordQueryHandler`.
`NewDoseRecordQueryHandler` gains one argument
(`prescription.Repository`) — additive constructor change, all call sites in
`cmd/api/main.go` updated.

## 6. HTTP layer

### New handler — `internal/api/extended_server.go`

```go
// ListDoseSchedule handles GET /users/{userId}/doses
func (s *ExtendedServer) ListDoseSchedule(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "userId")
    callerID := callerUserID(r)

    items, err := s.doseQueries.ListScheduleForUser(r.Context(),
        queries.ListDoseScheduleQuery{
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

Mirror of `ListDoseRecords` (`extended_server.go:247-260`).

### Route registration — `internal/api/router.go`

```go
// Dose records
r.Get("/users/{userId}/dose-records", ext.ListDoseRecords)
r.Get("/users/{userId}/doses", ext.ListDoseSchedule)   // <-- new
r.Post("/dose-records/{doseRecordId}/confirm", ext.ConfirmDose)
r.Post("/dose-records/{doseRecordId}/miss", ext.MarkDoseMissed)
```

No constructor signature changes — `ExtendedServer` already holds
`doseQueries`.

## 7. OpenAPI spec — `docs/api.yaml`

After edits, regenerate with `go generate ./internal/api/gen/...`
(`internal/api/gen/generate.go:1-5` already runs `oapi-codegen`).

### New tag (api.yaml:18-30)

```yaml
- name: DoseSchedule
  description: User medication schedule — planned doses overlaid with confirmation history.
```

### New schema

```yaml
ScheduledDose:
  type: object
  description: |
    A single planned dose in the user's medication schedule. Produced by
    expanding every active prescription's medicaments; existing dose
    records (TAKEN / MISSED) take precedence over reconstructed PENDING
    entries. `dose_record_id` is null until the scheduler materializes the
    underlying record at notification-fire time.
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

### New path

```yaml
/users/{userId}/doses:
  get:
    tags:
      - DoseSchedule
    summary: List the user's full medication schedule
    description: |
      Returns every dose the user is expected to take across all active
      prescriptions, overlaid with confirmation history. The schedule is
      reconstructed from each active prescription's `created_at` + every
      medicament's `times[]` × `doses`. Existing `dose_records` override
      the reconstructed PENDING status with their actual status (TAKEN
      or MISSED).

      Wall-clock times in the `medicament.times[]` strings are interpreted
      in BRT (UTC-3, no DST) regardless of the server's local timezone.
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
```

### New paths for already-implemented endpoints

```yaml
/users/{userId}/dose-records:
  get:
    tags: [DoseSchedule]
    summary: List dose records for a user (history only)
    description: |
      Returns every persisted dose record for the user, newest first.
      Only doses whose notification has already fired appear here — for
      the full planned schedule use `GET /users/{userId}/doses`.
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
    tags: [DoseSchedule]
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
    tags: [DoseSchedule]
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

The existing `DoseRecord` schema (`api.yaml:1388`) is reused as-is.

## 8. Accepted limitation: scheduler timezone

`internal/infrastructure/scheduler/redis_scheduler.go` uses
`startDate.Location()` for `nextScheduleTime`
(`redis_scheduler.go:164`), i.e. the server's local timezone. This PR does
**not** change that. As a result:

- Dose reconstruction in this endpoint always uses BRT (UTC-3).
- The scheduler dispatches notifications in server-local time.

For the current deployment this is fine because the server runs in BRT and
the team accepts that prescription submitters must compensate for any
divergence manually. If the deployment ever moves out of BRT or we want
single-source-of-truth, follow-up work can switch the scheduler to
`prescription.BrazilLocation` — the constant is already in place to make
that a one-line change.

## 9. Testing plan (TDD)

| Layer            | File                                                         | Cases                                                                                                                                                                       |
|------------------|--------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Domain           | `internal/domain/prescription/scheduled_dose_test.go` (new)  | Empty medicaments → empty. Once-daily 3 doses → 3 rows in order. Twice-daily 4 doses → 4 rows. `loc=BRT` produces expected wall-clock timestamps. Anchor respects `CreatedAt`. |
| Application      | `internal/application/queries/dose_record_queries_test.go` (new or extend) | Empty user. One prescription, no records → all PENDING, no `dose_record_id`. One prescription + one TAKEN overlay → status=TAKEN, `dose_record_id` set. Orphan record preserved. RBAC: forbidden / not-found paths. |
| API (smoke)      | `internal/api/list_dose_schedule_handler_test.go` (new)      | Locks in the same local-UUID vs Firebase-UID invariant the existing list-dose-records smoke test enforces (`internal/api/list_dose_records_handler_test.go:63`).                 |
| API (round-trip) | Same file                                                    | `httptest` request → 200 + JSON array with the expected shape and sort order.                                                                                              |
| Regeneration     | Manual                                                       | `go generate ./internal/api/gen/...` — verify `internal/api/gen/types.gen.go` and `server.gen.go` regenerate cleanly.                                                       |

## 10. Branch & commits

Branch: `feat/dose-schedule-endpoint` (off `main`).

1. `feat(domain): add ScheduledDose + Prescription.ExpandSchedule + BrazilLocation`
2. `feat(application): add DoseRecordQueryHandler.ListScheduleForUser`
3. `feat(api): expose GET /users/{userId}/doses`
4. `docs(api): document /users/{userId}/doses + dose-record endpoints`
5. `chore(gen): regenerate oapi-codegen types + server`

Each commit independently green (`go test ./...` + `golangci-lint run`).

## 11. Out of scope (explicit)

- Query filters (`?from`, `?to`, `?status`, `?prescription_id`).
- Pagination.
- Switching the scheduler to BRT (see §8 — accepted limitation).
- Documenting other undocumented extended routes:
  `/users/{userId}/caregivers`, `/users/{userId}/charges`,
  `/users/{userId}/invitations`, `/invitations/...`,
  `/users/me/data-export`, `/users/me/device-tokens/...`. Worth a
  separate PR.
- Tag splitting (`DoseRecords` vs `DoseSchedule`) — single tag chosen.