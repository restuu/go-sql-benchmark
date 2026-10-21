# Implementation Report: Core adapters (Phase 3)

## Summary
sqlx, sqlc, gorm, ent and go-jet adapters behind the Phase 2 `bench.Adapter` contract, a value-level cross-adapter correctness check (`bench.Verifier` + `internal/correctness`, run as a pre-flight in `cmd/bench`), a shared adapter integration contract (`internal/adaptertest`) and codegen wiring (`tools/go.mod`, `make gen*`). Branch `feat/core-adapters`, not committed.

## Assessment vs Reality
| Metric | Predicted | Actual |
|---|---|---|
| Complexity | Large | Large |
| Tasks | 13 | 13 done |

## Acceptance Criteria
| Criterion | Status |
|---|---|
| sqlc, sqlx, gorm, ent, jet registered; `-libs` accepts them | Met (`TestRegistryHasCoreLibraries`) |
| Every cell is a number or explicit `na`; `na` set equals the plan's table | Met: smoke run `results/smoke-p3.json` = 72 cells, 68 `ok` (error rate 0 in every one), 4 `na` (gorm and ent UNION, both profiles) |
| `correctness.Check` passes for all adapters and fails on a broken mapping | Met: swapping `title`/`fax` in sqlx Update aborted with `sqlx: update: stored row differs ...`, reverted |
| `make gen-check`, `make test`, full `go test -p 1 -race -cover ./...` with DB, vet, gofmt | Met |
| `go.mod` has no `toolchain` line | Met (`go 1.27.3` in `go.mod` and `tools/go.mod`) |
| No lib/pq in the main `go.mod` | **Not met, explained below** |
| go-reviewer run | Done: 0 CRITICAL, 0 HIGH, 3 MEDIUM fixed, rest listed below |

## Coverage (with DB, `-race`)
bench 100%, correctness 87.1%, result 94.3%, runner 94.2%, jet 87.8%, gorm 87.7%, sqlx 86.8%, sqlc 86.2%, ent 86.0%, databasesql 82.1%, benchcmp 83.3%, seedcheck 80.8%. `cmd/bench` is 64.6% (`run()` is exercised by the smoke runs, as in Phase 2). Generated packages and `adaptertest` (a test helper) are not measured.

## Deviations from the plan
- **lib/pq is in `go.mod` as `// indirect`.** go-jet's runtime `postgres` package imports it for array literals, and ent's generated `migrate` package links it via atlas. Both register an unused `"postgres"` driver; every adapter opens `sql.Open("pgx", dsn)`. Recorded in CLAUDE.md Adapter notes.
- **`tools/go.mod` needed `golang.org/x/tools` v0.52.0** (v0.43.0 made `ent generate` fail with "package context without types" on Go 1.27).
- **`ent generate` needs the target to be an existing package**, so `internal/adapters/ent/entgen/doc.go` is hand-written and kept.
- **sqlc UNION query qualifies columns** (`c.name`, `s.city`): unqualified, sqlc reports `column reference "city" is ambiguous`.
- **`gen-check` compares the working tree before/after regeneration** (shasum of the generated dirs), not against git, so it works before the first commit; the comment says so.
- **Pre-flight additions after review:** timeout (2 min), reseed before each subject's reads, missing-id UPDATE (must affect 0) and SELECT (must fail) checks, delete read-back requires `sql.ErrNoRows`.
- `adaptertest.RunVerifier` added (not in the plan) so each adapter's Verifier is covered by its own package tests.
- `libraryVersions` in `cmd/bench` records sqlx, gorm, ent, jet and pgx module versions in the result host info; sqlc has no runtime module (generator version is in `tools/go.mod`).

## Review findings (`ecc:go-reviewer`)
Fixed (MEDIUM): preflight timeout and fragile reseed ordering, delete read-back accepting any error, missing-id semantics not checked by the pre-flight, read ids vs previous subject's writes.
Open, deliberate:
- MEDIUM: `Store.Reseed` documents "no other connection may be open" but the pre-flight reseeds while idle pools are open. Works because the pools are idle; the timeout turns a hang into an error.
- LOW: `adaptertest` select-missing accepts any error; unreachable-`Init` test cannot detect a leaked pool; the contract's update uses `Right=true` (the pre-flight's `distinct` model covers zero values).
- LOW: `gen-jet` puts the DSN password in argv (local DB).

## Notes for Phase 4 / 5
- ent JOIN is edge eager-loading (2 queries). It is recorded in CLAUDE.md "Adapter notes" and the PRD Phase 4 scope; the report must print it next to ent's JOIN numbers.
- Smoke run was single-repeat with 3 s windows; do not rank from it. Stress JOIN throughput looked lower for ent (about 12.6k ops/s vs about 20-22k for the others), consistent with its 2-query JOIN.
- Phase 2 follow-ups (p95 stability, CHECKPOINT, benchcmp gaps, partial results on abort) are untouched.

## Unrelated observations (not fixed)
- `databasesql_test.go` still has its own copy of the contract; it could use `adaptertest`.
- `graphify-out/` shows modifications after every commit hook rebuild.
