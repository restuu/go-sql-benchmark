# go-sql-benchmark

Benchmarks Go SQL libraries (database/sql, sqlc, sqlx, gorm, ent, go-jet, ...) against one seeded
Postgres 15. Numbers are only comparable on the same machine. Plan and PRD: `.claude/PRPs/`.

## Commands
- `make arch-check` — fails if Go's GOARCH != host arch (emulated runs invalidate results)
- `make db-up | db-down | db-reset` — Postgres 15 via Docker; `db-down` and `db-reset` discard data
- `make db-seed` — re-run `db/seed.sql` against the running DB (truncates, reseeds)
- `make db-check` — verify Postgres major + row counts (`cmd/dbcheck`)
- `make build | vet | test` — `test` strips `BENCH_DSN`, so DB integration tests skip
- `make test-db` — brings the DB up and runs the `Integration` tests
- `make gen | gen-sqlc | gen-ent | gen-jet` — regenerate committed sqlc/ent/jet code with tools pinned in `tools/go.mod` (`go tool -modfile=tools/go.mod`); `gen-jet` reads the live schema so it brings the DB up. `make gen-check` fails if regenerating changes any file.
- `make bench BENCH_ARGS="..."` — arch check, DB up, run the matrix, write `results/run-<UTC>.json` (`go run ./cmd/bench -h` lists flags); reseeds the DB on success
- `make bench-cmp A=results/a.json B=results/b.json` — per-cell p95 deviation plus top-5 library check (membership and order, weights from `report/rubric.json`); exit 1 beyond `BENCH_TOL` (make default 0.20; the binary's own `-tolerance` default is 0.10) or when the top 5 differ
- `make report IN=results/x.json [OUT=README.md]` — render one result file plus `report/rubric.json` to Markdown (stdout without `OUT`); exit 1 if a cell or rubric entry is missing, 2 on usage/load errors. Same input gives byte-identical output.

## Decisions
- Go 1.27.3, native arm64, never under Rosetta. `go.mod` has no `toolchain` line; re-check after every `go get`/`go mod tidy`.
- Postgres 15 on host port 55432 (5432 is a native Postgres on the reference machine); override with `BENCH_PG_PORT`.
- `pgx/stdlib` (driver name `"pgx"`) is the shared driver for every `database/sql`-based library; never `lib/pq`.
- Custom runner, not `testing.B`. sqlx is kept and labelled "no updates since 2024".
- A scenario a library cannot express is "n/a (raw SQL fallback)" and is excluded from rankings.

## Layout
- `db/` — `schema.sql` + `seed.sql`, the single source of truth (sqlc reads it; jet generates from the live DB it seeds). Row counts live in `internal/seedcheck.ExpectedRows`; change both together.
- `internal/` — packages; `cmd/` — binaries.
- `internal/bench` adapter contract + profiles; `internal/runner` matrix, cells, stats, reseed store; `internal/result` JSON schema v1 (the Phase 4 contract); `internal/adapters/<lib>` one package per library (`databasesql`, `sqlx`, `sqlc`, `gorm`, `ent`, `jet`), registered in `cmd/bench` `registry`; generated code lives in `sqlc/db`, `ent/entgen` (from `ent/schema`) and `jet/gen`. `internal/adaptertest` is the shared integration contract each adapter test calls; `internal/correctness` is the cross-adapter value check. `internal/report` + `cmd/report` render a result file to Markdown (ranking, tables, reasons); `report/rubric.json` is the hand-written data (weights, methodology, pros/cons, cell notes) and must list every library in `registry`. `tools/` is a separate module holding the codegen tools.

## Runner
- Adapter contract: one method per scenario returning rows affected/scanned (runner checks 1/1/1/1/10/40 and counts a mismatch as an error); reads must scan every selected column; no explicit `Prepare`; return `bench.ErrUnsupported` for a scenario the library cannot express (cell becomes `na`); `Init` must work again after `Close`.
- Per cell: reseed, probe once, reseed, fresh pool of `workers` conns, warm-up, measured window. Library order is shuffled per repeat from the recorded seed; per-cell result is the median of `-repeats` (default 3).
- Metrics: latency = per-call service time of successful ops (exact nearest-rank p50/p95/p99, not coordinated-omission-corrected); throughput = ops / wall clock; B/op, allocs/op, CPU ns/op = window deltas (`MemStats`, `getrusage`) / ops and include harness and GC cost (harness floor about 0 allocs/op).
- Profiles (flags override, effective values stored in the JSON): low = 4 workers, 10 ms pacing, 1 s warm-up, 5 s window; stress = 100 workers, unpaced, 2 s warm-up, 10 s window, 400k pre-inserted DELETE ids (window ends early if exhausted).
- Value-level correctness: adapters also implement `bench.Verifier` (`ReadModel`, `ReadJoin`, `ReadUnion` return values through the same library query as the benchmarked method; an unsupported scenario returns `ErrUnsupported` on both). `internal/correctness.Check` compares them with the `database_sql` reference (itself checked against golden values from the seed) and replays INSERT/UPDATE/DELETE with a distinctive model (`Right` false). It runs as a pre-flight in `cmd/bench` before every matrix and aborts on any mismatch. Adapter queries have no `ORDER BY`; the check sorts.
- Write semantics every adapter must keep: UPDATE or DELETE of a missing id affects 0 rows (gorm `Save` and ent `UpdateOneID` would not), SELECT of a missing id errors, UPDATE writes every column including zero values.

## Adapter notes (carry into the Phase 4 report)
- Native rule: a scenario is supported if the library's own query API (or, for SQL-first sqlc/sqlx, its SQL) expresses it; needing the raw-SQL escape hatch means `na`. Current `na`: gorm UNION, ent UNION. Every other cell is measured.
- **ent JOIN is edge eager-loading** (`WithCustomer()`): two queries (orders, then their customers), not one SQL JOIN. The report must say so next to ent's JOIN numbers.
- gorm uses `SkipDefaultTransaction` (otherwise every write is wrapped in a transaction), `Joins("Customer")` for JOIN, `Updates` with an explicit column list.
- Reads use each library's idiomatic result handling (sqlx/sqlc/gorm/ent/jet materialise rows; the reference scans into locals), so allocs/op differences include that.
- `lib/pq` is linked into the binary (go-jet array literals; ent's generated `migrate` package via atlas) but registers an unused "postgres" driver only; no adapter uses it. All adapters open `sql.Open("pgx", dsn)`.

## Knowledge graph (graphify)
- `graphify-out/` (git-ignored, local only) holds the graph of code + `.claude/PRPs/` docs (`GRAPH_REPORT.md`, `graph.html`, `graph.json`). For architecture or "how does X relate to Y" questions, check `GRAPH_REPORT.md` or run `graphify query "<question>"`, `graphify path "A" "B"`, `graphify explain "X"` before grepping.
- A git post-commit hook (`graphify hook install`; check with `graphify hook status`) rebuilds the code part of the graph after each commit. Doc/plan/PRD changes are ignored by it: run `/graphify . --update` (LLM) for those. Manual code refresh: `graphify update .`. Skip the hook once with `GRAPHIFY_SKIP_HOOK=1`.
- The graph is a snapshot: rebuild it after each phase lands, and trust the code over a stale graph.

## Conventions
- Tests are table-driven with `// Arrange`, `// Act`, `// Assert` comments.
- Code copied from efectn/go-orm-benchmarks keeps its attribution comment; the LICENSE carries the upstream notice.
