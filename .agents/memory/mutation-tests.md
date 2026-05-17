# Mutation Test Memory

## Overview
Mutation testing with [gremlins](https://github.com/go-gremlins/gremlins) v0.6.0 validates that unit tests properly detect mutations in business logic.

**Important:** gremlins corrupts the Go test cache between runs. Always run `go clean -testcache` before each gremlins execution.

## Known Issues

### gremlins Cache Corruption Bug
When running gremlins twice on the same package, the second run shows all mutations as TIMED OUT due to cache corruption.

**Workaround:** Always clean test cache before gremlins runs:
```bash
go clean -testcache
gremlins unleash ./internal/domain/
go clean -testcache
gremlins unleash ./internal/application/
```

---

## TIMED OUT Mutations

These mutations cause tests to hang. They likely indicate code that, when mutated, creates infinite loops or blocking operations.

### application/ — 26 TIMED OUT

| File | Line | Mutation Type |
|------|------|--------------|
| commands/auth_commands.go | 71 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 57 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 51 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 131 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 63 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 114 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 109 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 104 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 56 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 54 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 55 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 99 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 104 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 123 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 89 | CONDITIONALS_NEGATION |
| commands/auth_commands.go | 82 | CONDITIONALS_NEGATION |
| commands/doctor_auth_commands.go | 48 | CONDITIONALS_NEGATION |
| commands/doctor_commands.go | 85 | CONDITIONALS_NEGATION |
| commands/user_commands.go | 81 | CONDITIONALS_NEGATION |
| commands/user_commands.go | 106 | CONDITIONALS_NEGATION |
| queries/prescription_queries.go | 59 | CONDITIONALS_NEGATION |
| queries/prescription_queries.go | 63 | CONDITIONALS_NEGATION |
| queries/user_queries.go | 36 | CONDITIONALS_NEGATION |
| queries/user_queries.go | 41 | CONDITIONALS_NEGATION |
| queries/user_queries.go | 53 | CONDITIONALS_NEGATION |
| queries/user_queries.go | 58 | CONDITIONALS_NEGATION |

### domain/ — 40 TIMED OUT

| File | Line | Mutation Type |
|------|------|--------------|
| prescription/medicament.go | 25 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 71 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 71:38 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 76 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 81:27 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 94 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 94:27 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 94:42 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 104 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 114:25 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 114:38 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 114:25 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 119 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 124:27 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 124:27 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 124:42 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 124:42 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 129 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 134:28 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 134:43 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 134:43 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 140 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 149 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 154 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 159:38 | ARITHMETIC_BASE |
| prescription/medicament.go | 179:8 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 171 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 191:13 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 197:24 | ARITHMETIC_BASE |
| prescription/medicament.go | 212:10 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 240:35 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 249:9 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 249:38 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 278 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 283:38 | CONDITIONALS_BOUNDARY |
| prescription/medicament.go | 303 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 303:45 | CONDITIONALS_NEGATION |
| prescription/medicament.go | 307:101 | ARITHMETIC_BASE |
| prescription/prescription.go | 23:68 | CONDITIONALS_NEGATION |

---

## LIVED Mutations

These mutations are NOT detected by tests — the mutated code still passes all tests. This indicates missing test coverage for specific business logic paths.

### application/ — 1 LIVED

| File | Line | Mutation Type | Notes |
|------|------|--------------|-------|
| commands/user_commands.go | 63:20 | CONDITIONALS_NEGATION | May need additional test case |

### domain/ — 16 LIVED

| File | Line | Mutation Type | Notes |
|------|------|--------------|-------|
| prescription/medicament.go | 94:42 | CONDITIONALS_BOUNDARY | Boundary condition not tested |
| prescription/medicament.go | 191:13 | CONDITIONALS_BOUNDARY | Boundary condition not tested |
| prescription/medicament.go | 249:38 | CONDITIONALS_BOUNDARY | Boundary condition not tested |
| prescription/medicament.go | 249:25 | CONDITIONALS_BOUNDARY | Boundary condition not tested |
| prescription/medicament.go | 254:42 | CONDITIONALS_BOUNDARY | Boundary condition not tested |
| prescription/medicament.go | 264:28 | CONDITIONALS_BOUNDARY | Boundary condition not tested |
| prescription/medicament.go | 264:43 | CONDITIONALS_BOUNDARY | Boundary condition not tested |
| prescription/medicament.go | 274:35 | CONDITIONALS_NEGATION | Edge case not tested |
| prescription/medicament.go | 283:25 | CONDITIONALS_BOUNDARY | Boundary condition not tested |
| prescription/medicament.go | 288:42 | CONDITIONALS_BOUNDARY | Boundary condition not tested |
| prescription/medicament.go | 303:29 | CONDITIONALS_NEGATION | Edge case not tested |
| prescription/medicament.go | 307:40 | ARITHMETIC_BASE | Arithmetic mutation not caught |
| prescription/medicament.go | 307:77 | ARITHMETIC_BASE | Arithmetic mutation not caught |
| prescription/medicament.go | 307:64 | ARITHMETIC_BASE | Arithmetic mutation not caught |
| prescription/prescription.go | (see above) | CONDITIONALS_NEGATION | Multiple edge cases |
| user/user.go | 38:10 | CONDITIONALS_NEGATION | Role validation edge case |
| user/user.go | 38:33 | CONDITIONALS_NEGATION | Role validation edge case |

---

## Latest Results (2026-05-17)

### application/
```
Killed: 74, Lived: 1, Not covered: 47
Timed out: 26, Not viable: 0, Skipped: 0
Test efficacy: 98.67%
Mutator coverage: 61.48%
Status: ✅ PASSES threshold-efficacy 80
```

### domain/
```
Killed: 88, Lived: 16, Not covered: 13
Timed out: 40, Not viable: 0, Skipped: 0
Test efficacy: 84.62%
Mutator coverage: 88.89%
Status: ✅ PASSES threshold-efficacy 80
```

---

## TODO

- [ ] Investigate and fix TIMED OUT mutations (add test timeouts or fix infinite loops)
- [ ] Add test coverage for LIVED mutations to improve efficacy
- [ ] Consider adding `--invert-loopctrl` if loop mutations continue to cause issues
- [ ] Investigate Firebase SDK deprecation: `google.golang.org/api/option` deprecated `WithCredentialsFile` and `WithCredentialsJSON` (SA1019). Current workaround uses `option.WithAuthCredentialsJSON(option.ServiceAccount, credentialsJSON)`. Files affected:
  - `internal/infrastructure/firebaseauth/service.go:40`
  - `internal/infrastructure/notification/firebase_sender.go:28`
  See `CLAUDE.md` for full context.