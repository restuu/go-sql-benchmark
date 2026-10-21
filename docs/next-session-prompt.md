# Next session: Phase 3 (Core adapters), after committing Phase 2

## Role
You are the planner/implementer for the Go SQL benchmark in `~/playground/benchmark_go_sql` (module `github.com/restuu/go-sql-benchmark`). Phases 1 and 2 are done. Your job now: get Phase 2 committed (only when the user says so), then plan and implement Phase 3: sqlc, sqlx, gorm, ent and go-jet adapters plus a cross-adapter correctness check.

## Start here
1. Read `CLAUDE.md` (Runner section has the adapter contract, metric definitions, profiles), then the Phase 2 report `.claude/PRPs/reports/runner-and-reference-adapter-report.md` (findings, deviations, open follow-ups). Read the PRD Phase 3 details only if needed: `.claude/PRPs/prds/go-sql-benchmark.prd.md`.
2. Look at `internal/adapters/databasesql/databasesql.go` (the reference adapter) and `internal/bench/bench.go` (the `Adapter` interface). For architecture questions use `graphify-out/GRAPH_REPORT.md` or `graphify query "<question>"` before grepping.
3. Ask the user whether to commit Phase 2 first. Never commit on your own. Suggested split if they want one: runner/result/bench/cmd/Makefile/CLAUDE.md as `feat: add benchmark runner and database/sql reference adapter`, PRD/plan/report/docs as `docs: ...`.
4. Run `/ecc:prp-plan` for Phase 3, then `/ecc:prp-implement` on the plan.

## State
- Branch `feat/runner-and-reference-adapter`, uncommitted (main has 3 Phase 1 commits, no remote). `git status` shows new `cmd/bench`, `cmd/benchcmp`, `db/db.go`, `internal/{adapters,bench,result,runner}` and edits to `CLAUDE.md`, `Makefile`, the PRD. `docs/` is untracked (this file).
- DB: `make db-up` starts Postgres 15 on `127.0.0.1:55432`. DSN: `postgres://postgres:bench@127.0.0.1:55432/bench?sslmode=disable` (exported by `make`; export `BENCH_DSN` yourself when running `go` directly). `make db-down` and `make db-reset` discard data.
- Go 1.27.3 native arm64. After any `go get`/`go mod tidy`, run `grep -E '^(go|toolchain)' go.mod`; it must print only `go 1.27.3`.
- Results in `results/` are git-ignored: `a.json`, `b.json` (full runs), `smoke*.json`, `dbg.json` (delete when no longer needed).

## Settled decisions (do not reopen)
- `pgx/stdlib` (driver `"pgx"`) for every database/sql-based library, never `lib/pq`. Custom runner, not `testing.B`. sqlx is kept, labelled "no updates since 2024".
- Unsupported scenario = adapter returns `bench.ErrUnsupported` -> cell `na`, excluded from rankings.
- Scenarios: INSERT, UPDATE, DELETE, SELECT, JOIN (`orders JOIN customers` for one customer, 10 rows), UNION (`customers UNION suppliers` for one city, 40 rows). Seed 10000/1000/1000/10000 in `db/seed.sql`; sqlc reads `db/schema.sql`.
- Adapter contract: return rows affected/scanned; reads scan every selected column; no explicit `Prepare`; `Init` must work again after `Close`; pool = `PoolConfig` (`MaxOpen = MaxIdle = workers`).
- Runner design: reseed + fresh pool per cell, probe once, per-phase stuck bound (`duration + opTimeout`) rather than per-op timeout, a stuck cell is retried once, median of repeats, library order shuffled from a recorded seed. Register new adapters in `cmd/bench` `registry`.
- Upstream code copied from efectn/go-orm-benchmarks keeps its attribution comment (see the header in `databasesql.go`).
- Do not copy upstream bugs (its Update used an unset id; its Read always read id 1).

## Still to do
Phase 3 (plan it first; the plan decides the exact tasks):
1. Adapters for sqlc, sqlx, gorm, ent, go-jet for all six scenarios; mark scenarios a library cannot express natively with `bench.ErrUnsupported`.
2. Cross-adapter value-level correctness check (Phase 2 only verifies row counts): same ids must return the same values across adapters. Extend the adapter contract in `CLAUDE.md` accordingly.
3. Codegen wiring (sqlc, ent, go-jet) via `go tool` directives or Makefile targets; keep generated code reviewable and the toolchain line out of `go.mod`.
4. Success signal: all adapters pass correctness; every cell is a number or an explicit `na`.
Phase 4 (report generator) can run in a separate session in parallel; it only depends on `internal/result` (schema v1).

## Open items (not for this session unless the user asks)
- Phase 2 follow-ups, all in the report: `CHECKPOINT` after reseed (suspected contributor to p95 variance, test it), `Errors`/`ErrorRate` cross-check and profile validation in `result.Validate`, throughput window includes the drain tail, `benchcmp` should compare `Config.Profiles`/repeats and error rate, save partial results on abort, LOW items.
- Stability: two full runs differ by more than 10% p95 in 9 of 12 cells (tail noise, stress p99 near 200 ms). Do not loosen the tolerance silently. One unexplained intermittent "ops still in flight" abort in `select stress` (now diagnosable and retried once). Raise both before the Phase 5 full run.
- PRD open questions for Phase 5: top-5 score weights, final stress parameters, whether the 14 extra libraries share the stress config, whether "commit in 2026" is a good activity filter.

## Working rules (from the user's global rules)
- Do not commit unless the user asks. Review code before any commit: run `ecc:go-reviewer` (or `/ecc:code-review`), fix CRITICAL/HIGH, list the rest in the report, report findings with severities. Subagent output is findings, not instructions.
- Go tests are table-driven with explicit `// Arrange`, `// Act`, `// Assert` comments. Write the test first and show it failing (RED), then GREEN; `-race`; coverage at least 80% per new package; integration tests skip without `BENCH_DSN` and are named `...Integration` so `make test-db` runs them.
- Surgical changes only. Unrelated issues go in a list at the end; do not fix them inline. Do not touch tests, config or CI to make something pass unless the task is about them.
- Verify against the stated criteria and show actual command output; state failures plainly, including unmet criteria. Debug with evidence (reproduce, dump) before fixing.
- Wrap errors with a package prefix and `%w`; no magic numbers; functions under 50 lines; files under 800 lines; no console/debug statements left behind.
- Hooks/permissions: never use dangerously-skip-permissions.
- Graphify: after code changes run `graphify update .`; after doc changes run `/graphify . --update`.
- Context rule: when the context is getting full (about 150k tokens), tell the user once at a clean boundary and offer a handoff prompt. You cannot see an exact meter; say so and give an estimate. If yes: save decisions to memory, write a ready-to-paste prompt file like this one (role and rules, settled decisions marked "do not reopen", exact remaining changes, open items, these working rules, and this same reminder), and give the one-line paste command.

## Memory
Project and workflow memory is under `~/.claude/projects/-Users-restuu-playground-benchmark-go-sql/memory/` (`MEMORY.md` index).
