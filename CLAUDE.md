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

## Decisions
- Go 1.27.3, native arm64, never under Rosetta. `go.mod` has no `toolchain` line; re-check after every `go get`/`go mod tidy`.
- Postgres 15 on host port 55432 (5432 is a native Postgres on the reference machine); override with `BENCH_PG_PORT`.
- `pgx/stdlib` (driver name `"pgx"`) is the shared driver for every `database/sql`-based library; never `lib/pq`.
- Custom runner, not `testing.B`. sqlx is kept and labelled "no updates since 2024".
- A scenario a library cannot express is "n/a (raw SQL fallback)" and is excluded from rankings.

## Layout
- `db/` — `schema.sql` + `seed.sql`, the single source of truth (sqlc reads it in Phase 3). Row counts live in `internal/seedcheck.ExpectedRows`; change both together.
- `internal/` — packages; `cmd/` — binaries.

## Knowledge graph (graphify)
- `graphify-out/` holds the graph of code + `.claude/PRPs/` docs (`GRAPH_REPORT.md`, `graph.html`, `graph.json`). For architecture or "how does X relate to Y" questions, check `GRAPH_REPORT.md` or run `graphify query "<question>"`, `graphify path "A" "B"`, `graphify explain "X"` before grepping.
- A git post-commit hook (`graphify hook install`; check with `graphify hook status`) rebuilds the code part of the graph after each commit. Doc/plan/PRD changes are ignored by it: run `/graphify . --update` (LLM) for those. Manual code refresh: `graphify update .`. Skip the hook once with `GRAPHIFY_SKIP_HOOK=1`.
- The graph is a snapshot: rebuild it after each phase lands, and trust the code over a stale graph.

## Conventions
- Tests are table-driven with `// Arrange`, `// Act`, `// Assert` comments.
- Code copied from efectn/go-orm-benchmarks keeps its attribution comment; the LICENSE carries the upstream notice.
