# Go SQL Library Benchmark

## Problem Statement

Go engineers choosing a Postgres data-access library (database/sql, sqlc, sqlx, gorm, ent, go-jet, ...) have no reproducible, readable comparison. The closest public project, `efectn/go-orm-benchmarks`, measures only single-table CRUD with sequential `testing.B`, reports only `ns/op`, `B/op`, `allocs/op`, has no JOIN/UNION/DELETE, no stress mode, no CPU or latency percentiles, and ends in a raw text dump with no summary. The cost is library choices made from blog-post folklore instead of evidence.

## Evidence

- Upstream inspected on 2026-10-21 (`efectn/go-orm-benchmarks`, MIT, last push 2024-10-15): operations are `Insert, InsertMulti, Update, Read, ReadSlice` on one `models` table; `results.md` is plain tab-aligned text; runs sequentially; Postgres 14.
- Upstream does not compile on Go 1.27.3 as pinned (`golang.org/x/tools v0.25.0`, `tokeninternal.go:64`); it builds after bumping x/tools to v0.52.0. Existing comparisons are therefore also going stale.
- The requester (a Go backend engineer) reports using sqlx on a previous project and wants evidence for the next choice. Wider demand: Assumption - needs validation (e.g. does anyone else re-run it).

## Proposed Solution

A fresh Go module (attribution to upstream, MIT preserved) with its own runner instead of `testing.B`. The runner executes one shared scenario set (INSERT, UPDATE, DELETE, SELECT single-table, JOIN of 2+ tables, UNION of 2+ tables) against Postgres 15 in Docker, under a low-traffic profile and a stress profile, and records latency percentiles, throughput, allocations, CPU time and error rate to JSON. A report generator turns the JSON into `README.md`: per-scenario tables, deltas versus `database/sql`, a top-5 section with generated reasoning, and per-library pros/cons mixing computed numbers with a hand-written qualitative rubric. Chosen over forking because the missing pieces (runner, scenarios, reporting) are most of the work and a 24-library history adds nothing; the six upstream adapters and schema are copied with attribution.

## Key Hypothesis

We believe a reproducible harness with generated rankings and pros/cons will let a Go engineer choose a SQL library with evidence.
We'll know we're right when `make bench` on the reference machine regenerates the README end to end, and a second run keeps the top 5 unchanged with per-library p95 within about 10%.

## What We're NOT Building

- Databases other than Postgres - scope and fairness.
- Connection-pool tuning sweeps - one fixed, documented pool config.
- Prepared-statement/caching variants beyond each library's default behavior - keeps comparison to "how people actually use it".
- CI or cloud runners - numbers are only comparable on the same machine.
- A web dashboard - README is the deliverable.
- Measuring developer experience - stays a hand-written rubric, never presented as benchmark output.

## Success Metrics

| Metric | Target | How Measured |
|--------|--------|--------------|
| End-to-end reproducibility | `make bench` regenerates README with no manual steps | Run on clean checkout |
| Run-to-run stability | Top 5 unchanged; per-library p95 within 10% across 2 runs | Compare two JSON result sets |
| Scenario coverage (Phase 1 libs) | 6 libs x 6 scenarios x 2 profiles; every cell is a number or explicit "n/a" | Report generator completeness check |
| Metrics per cell | p50/p95/p99, throughput, B/op, allocs/op, CPU time, error rate | JSON schema validation |
| Correctness | Every adapter returns identical row counts/values per scenario | Cross-adapter verification test before timing |

## Open Questions

- [ ] Weighting of the top-5 composite score (default: equal-weight average of per-scenario ranks on p95, B/op, CPU time).
- [ ] Stress parameters: worker count (default 100), duration, dataset size.
- [ ] Do the 14 extra libraries share the same stress config as the core 6?
- [ ] Is "last commit in 2026" a good-enough activity filter (dependency-bump pushes count)?

---

## Users & Context

**Primary User**
- **Who**: Go backend engineer or tech lead picking or auditing a data-access layer.
- **Current behavior**: Reads blog posts and upstream's `results.md`, or guesses from experience.
- **Trigger**: Starting a new service or questioning an existing library choice.
- **Success state**: Picks a library and can point to latency, memory, CPU and trade-offs for realistic queries.

**Job to Be Done**
When I'm choosing a Go SQL library, I want to see latency, memory and CPU for realistic queries under light and heavy load, so I can pick with evidence.

**Non-Users**
People comparing databases other than Postgres, and library maintainers wanting microbenchmarks of internals.

---

## Solution Detail

### Core Capabilities (MoSCoW)

| Priority | Capability | Rationale |
|----------|------------|-----------|
| Must | Postgres 15 in Docker, one-command run, Go arch check | Reproducibility; a mismatched (Rosetta) Go binary was already hit on this machine |
| Must | Scenarios: INSERT, UPDATE, DELETE, SELECT single, JOIN (2+ tables), UNION (2+ tables) | Explicit requirement |
| Must | Profiles: low traffic (1-4 paced workers) and stress (fixed workers, fixed duration) | Explicit requirement |
| Must | Metrics: p50/p95/p99 latency, throughput, B/op, allocs/op, CPU time (getrusage), error rate | Explicit requirement ("other relevant data" = percentiles, throughput, errors) |
| Must | Adapters for database/sql, sqlc, sqlx, gorm, ent, go-jet (sqlx kept, labelled "no updates since 2024") | Explicit requirement |
| Must | README generator: tables, delta vs database/sql, top-5 with reasoning, pros/cons | Explicit requirement |
| Must | `CLAUDE.md`, upstream attribution + MIT LICENSE | Explicit requirement / license |
| Should | Cross-adapter correctness check before timing | Prevents ranking libraries that return wrong data |
| Should | "n/a (raw SQL fallback)" cells excluded from rankings | Keeps comparison fair |
| Should | Per-library "last release" shown in README | Activity filter alone can mislead |
| Could | 14 extra libs: beego, bun, go-pg, pgx, pgx_pool, pop, reform, rel, sqlboiler, upper/db, xorm, zorm, gorm/gen, gorm_prep (active in 2026) | User request; large effort so phased after core |
| Could | pprof CPU profiles saved for stress runs | Deeper CPU diagnosis |
| Won't | Dashboard, CI, other databases | See "What We're NOT Building" |

Excluded by activity filter (no commits in 2026): dbr, godb, gorp, sq.

### MVP Scope

Core 6 libraries x 6 scenarios x 2 profiles, JSON results, generated README with top-5 and pros/cons.

### User Flow

`make bench` -> arch check -> docker compose up (Postgres 15) -> migrate/seed -> cross-adapter correctness check -> run scenarios per library (shuffled order, warm-up, N repeats) -> write `results/*.json` -> generate `README.md`.

---

## Technical Approach

**Feasibility**: HIGH

**Architecture Notes**
- Fresh module, `go 1.27.3`, no `toolchain` line; `make bench` aborts if `go env GOARCH` != `uname -m`.
- Custom runner with a small `Adapter` interface (one method per scenario); upstream `ORMInterface`/`testing.B` is not reused.
- Copy from upstream (attributed): six adapters, schema, sqlc layout, ent and go-jet codegen inputs. Bump x/tools (and other stale pins) for Go 1.27.
- `sqlc` and `ent` pinned via `go tool` directives so no global install is needed; generated code committed.
- Results as JSON per run; report generator is a separate command reading JSON plus a `rubric` data file for qualitative pros/cons.
- Top 5: composite of per-scenario rank (p95, B/op, CPU time) across supported scenarios; reasons generated from the numbers.
- Measurement hygiene: warm-up, shuffled library order, repeated runs with median, fixed GOMAXPROCS and pool size, record variance and host info (8-core arm64, 16 GB, Docker 29.8.2).

**Technical Risks**

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Noisy laptop/Docker measurements (thermal, VM) | H | Warm-up, repeats + median, variance in report, state numbers are machine-specific |
| Libraries without native UNION/JOIN | H | Mark "n/a (raw SQL fallback)"; exclude from rankings |
| Library versions pinned for Go 1.23 fail on 1.27 | M | Bump during Phase 1 (x/tools already confirmed to fix the build) |
| Adapter scope (about 80 new implementations if all 20 libs) | H | Phase the 14 extras after core 6 |
| Driver choice skews results | M | Decided: `pgx/stdlib` for all database/sql-based libs; documented in README. Upstream adapters using `lib/pq` must be switched. |
| CPU time includes Postgres client overhead of the harness | M | Measure the benchmark process only; Postgres runs in Docker |

---

## Implementation Phases

<!--
  STATUS: pending | in-progress | complete
  PARALLEL: phases that can run concurrently (e.g., "with 3" or "-")
  DEPENDS: phases that must complete first (e.g., "1, 2" or "-")
  PRP: link to generated plan file once created
-->

| # | Phase | Description | Status | Parallel | Depends | PRP Plan |
|---|-------|-------------|--------|----------|---------|----------|
| 1 | Foundation | Module on Go 1.27.3, Postgres 15 compose, schema/seed, arch check, Makefile, LICENSE attribution, CLAUDE.md | complete | - | - | [foundation.plan.md](../plans/completed/foundation.plan.md), [report](../reports/foundation-report.md) |
| 2 | Runner & reference adapter | `Adapter` interface, low/stress profiles, metrics capture, JSON output, `database/sql` adapter for all 6 scenarios | complete | - | 1 | [plan](../plans/completed/runner-and-reference-adapter.plan.md), [report](../reports/runner-and-reference-adapter-report.md) (p95 stability not met on this machine; recorded), [p95 investigation](../reports/p95-stability-report.md) (environmental; p95 gate relaxed to about 20% plus top-5 rank stability, decided by the user) |
| 3 | Core adapters | sqlc, sqlx, gorm, ent, go-jet for all scenarios + cross-adapter correctness check | complete | with 4 | 2 | [plan](../plans/completed/core-adapters.plan.md), [report](../reports/core-adapters-report.md) |
| 4 | Report generator | JSON -> README: tables, delta vs baseline, top-5 reasoning, pros/cons from data + rubric file | complete | with 3 | 2 | [plan](../plans/completed/report-generator.plan.md), [report](../reports/report-generator-report.md) |
| 5 | Full run & publish | Full run on reference machine, stability check (2 runs), commit README | pending | - | 3, 4 | - |
| 6 | Extra libraries | 14 active-in-2026 libs, same scenarios, "n/a" handling | pending | - | 5 | - |

### Phase Details

**Phase 1: Foundation**
- **Goal**: A repo that builds on Go 1.27.3 and starts a seeded Postgres 15.
- **Scope**: `go.mod`, `docker-compose.yml`, schema/seed for single, JOIN and UNION tables, `make` targets, arch guard, LICENSE with upstream attribution, `CLAUDE.md`.
- **Success signal**: `make db-up` yields a seeded Postgres 15; `go build ./...` passes; arch guard fails on a mismatched Go.

**Phase 2: Runner & reference adapter**
- **Goal**: Prove the measurement pipeline on the baseline.
- **Scope**: Adapter interface, both profiles, p50/p95/p99, throughput, B/op, allocs/op, getrusage CPU, error rate, JSON schema.
- **Success signal**: `database/sql` produces a complete JSON result; two runs agree within about 10% p95.

**Phase 3: Core adapters**
- **Goal**: The other five required libraries on the same scenarios.
- **Scope**: sqlc, sqlx, gorm, ent, go-jet (copied from upstream with attribution, extended with DELETE/JOIN/UNION); correctness check; "n/a" marking where native support is missing.
- **Success signal**: All adapters pass correctness; every cell is a number or explicit "n/a".

**Phase 4: Report generator**
- **Goal**: Readable summary from results, no hand-editing.
- **Scope**: Tables, delta vs `database/sql`, top-5 with generated reasons placed before each pros/cons list, rubric data file, library metadata (last release, "no updates since 2024" label for sqlx), machine info. Per-cell notes from `CLAUDE.md` "Adapter notes" (e.g. ent JOIN is edge eager-loading, 2 queries) must appear beside the numbers.
- **Success signal**: README generated from a fixture JSON; deterministic output.

**Phase 5: Full run & publish**
- **Goal**: The deliverable README with real numbers.
- **Scope**: Full run, repeat for stability, resolve open questions (weights, stress params, driver) and record them.
- **Success signal**: Success-metric table met.

**Phase 6: Extra libraries**
- **Goal**: Extend coverage to the 14 libraries active in 2026.
- **Scope**: Adapters, codegen where required, rerun and regenerate README.
- **Success signal**: Same completeness check as Phase 3 across all 20.

### Parallelism Notes

Phases 3 and 4 touch different packages (adapters vs report generator) and both depend only on the Phase 2 JSON schema and interface, so they can run concurrently. Phase 6 could split per library after Phase 5.

---

## Decisions Log

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Start point | Fresh module, copy 6 adapters with attribution | GitHub fork of upstream | Missing parts are most of the work; avoid 24-library baggage |
| Library scope | Core 6 first; 14 extras later | All 20 at once | About 80 new implementations is too large before first results |
| sqlx | Keep, label "no updates since 2024" | Drop as stale | User uses it and wants it compared |
| Activity filter (extras) | Last commit in 2026 and not archived | Release date in 2026 | Several active repos have old releases; show last release anyway |
| Unsupported scenario | "n/a (raw SQL fallback)", excluded from rankings | Raw SQL counted | Raw fallback measures the driver, not the library |
| CPU measurement | Process CPU time via getrusage | pprof only | Cheap and comparable; pprof as Could |
| Stress shape | Fixed workers + fixed duration | Fixed request count | Gives throughput and tail latency under saturation |
| Pros/cons | Computed metrics + hand-written rubric file | Fully generated | DX cannot be benchmarked |
| Go version | 1.27.3, no toolchain line, native arm64 | Allow toolchain download | Rosetta/mismatched binary would corrupt CPU/alloc numbers |
| Postgres | 15 in Docker | 14 (upstream) | Requirement |
| Driver (database/sql-based libs) | `pgx/stdlib` (pgx v5) for database/sql, sqlc, sqlx, gorm, ent, go-jet | `lib/pq` (upstream default) | User decision; lib/pq is maintenance-only. One shared driver keeps differences attributable to the library. Native `pgx`/`pgx_pool` (Phase 6) stay separate entries. |

---

## Research Summary

**Market Context**
- `efectn/go-orm-benchmarks` (MIT, 329 stars, last push 2024-10-15): 24 variants, `testing.B`, text output only; gaps are JOIN/UNION/DELETE, concurrency/stress, CPU, percentiles, summary.
- Activity in 2026 (last commit): active - beego, bun, go-pg, pgx, pop, reform, rel, sqlboiler, upper/db, xorm (gitea.com), zorm (gitee), gorm/gen, plus core ent, gorm, sqlc, go-jet. Stale - sqlx (2024-08), dbr (2024-10), gorp (2024-11), godb (2023-12), sq (2024-09). Several active repos have old releases (reform 2021, sqlboiler 2025-06, upper 2025-03).

**Technical Context**
- Upstream layout: `main.go`, `helper/` (suite, SQL, errors), `bench/` per-library adapters (3,750 lines), `bench/sqlc` (queries.sql, schema.sql, sqlc.yaml), ent and jet codegen dirs.
- Go 1.27.3 darwin/arm64 verified native after reinstall; an earlier x86_64 install failed with "bad CPU type in executable".
- Upstream builds on 1.27.3 once `golang.org/x/tools` is bumped to v0.52.0.
- Reference machine: 8 cores, 16 GB, arm64, Docker 29.8.2. `sqlc` and `ent` CLIs not installed globally (will be pinned as Go tools).

---

*Generated: 2026-10-21*
*Status: DRAFT - needs validation*
