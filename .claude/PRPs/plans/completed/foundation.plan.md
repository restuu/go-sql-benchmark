# Plan: Foundation (Go SQL Benchmark, Phase 1)

## Summary
Create the repo skeleton for the Go SQL benchmark: a Go 1.27.3 module using `pgx/stdlib`, a Postgres 15 docker-compose that comes up seeded for all six scenarios, a Makefile with an architecture guard, MIT LICENSE carrying upstream attribution, and `CLAUDE.md`. A tiny `dbcheck` command proves the seeded DB and the chosen driver work together, so the Phase 1 success signals are executable.

## User Story
As a Go backend engineer building the benchmark, I want a reproducible, native-arch repo base with a seeded Postgres 15, so that later phases (runner, adapters) only add code and never fight the environment.

## Problem → Solution
Empty directory (only the PRD) → `make db-up` yields a seeded Postgres 15, `go build ./...` passes on Go 1.27.3, and `make arch-check` fails on a mismatched Go.

## Metadata
- **Complexity**: Medium
- **Source PRD**: `.claude/PRPs/prds/go-sql-benchmark.prd.md`
- **PRD Phase**: 1 — Foundation
- **Estimated Files**: 11 (+ `go.sum`)

---

## UX Design
N/A — internal change (developer tooling only; no end-user UI).

### Interaction Changes
| Touchpoint | Before | After | Notes |
|---|---|---|---|
| `make arch-check` | n/a | passes on native Go, exits 1 on mismatch | guards CPU/alloc numbers |
| `make db-up` | n/a | seeded Postgres 15 on `127.0.0.1:55432` | waits for healthy |
| `make db-check` | n/a | prints `seed OK`, exits non-zero on drift | uses `pgx/stdlib` |

---

## Mandatory Reading

The repo has no code yet. The "codebase" to mirror is the upstream project, cloned shallowly at
`/private/tmp/claude-501/-Users-restuu-playground-benchmark-go-sql/50c40941-454e-4988-8895-55211750f956/scratchpad/upstream/go-orm-benchmarks`
(re-clone with `git clone --depth 1 https://github.com/efectn/go-orm-benchmarks` if the scratchpad is gone).

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `.claude/PRPs/prds/go-sql-benchmark.prd.md` | all | Decisions log (driver, Go version, Postgres 15, no toolchain line) |
| P0 | upstream `helper/sql.go` | 47-87 | Original `models` DDL and `sql.Open("pgx", ...)` usage |
| P1 | upstream `bench/models.go` | 10-45 | Row values used by `NewModel()` (reuse for seed) |
| P1 | upstream `bench/sqlc/schema.sql`, `sqlc.yaml` | all | Phase 3 will point sqlc at our `db/schema.sql` |
| P2 | upstream `LICENSE` | all | Copyright line that must be preserved |

## External Documentation

| Topic | Source | Key Takeaway |
|---|---|---|
| Postgres image init | docker hub `postgres` | `*.sql` in `/docker-entrypoint-initdb.d` run alphabetically, only on an empty data dir |
| Compose `--wait` | Docker Compose docs | `docker compose up -d --wait` blocks until healthchecks pass |
| pgx stdlib | `github.com/jackc/pgx/v5/stdlib` | `import _ ".../stdlib"` registers driver name `"pgx"` |

No other external research needed — everything else is established Go/Docker usage.

---

## Patterns to Mirror

### NAMING_CONVENTION
// SOURCE: upstream helper/sql.go:47-55, bench/models.go
Lower-case package names (`helper`, `bench`), PascalCase exported identifiers, snake_case SQL, quoted `"right"` column (reserved word). Keep `models` table and its columns identical to upstream so adapters copied in Phase 3 need no changes.

### ERROR_HANDLING
// SOURCE: upstream helper/sql.go:63-72
```go
db, err := sql.Open("pgx", OrmSource)
if err != nil {
	return fmt.Errorf("init_tables: %w", err)
}
defer func() {
	_ = db.Close()
}()
```
Wrap with a package-prefixed message and `%w`. Mirror as `fmt.Errorf("seedcheck: ...: %w", err)`.

### LOGGING_PATTERN
Upstream uses plain `fmt.Println` in `main`. `cmd/dbcheck` prints one OK line to stdout and errors to stderr, exits non-zero. No logging library.

### REPOSITORY_PATTERN / SERVICE_PATTERN
N/A for this phase.

### TEST_STRUCTURE
Go tests, table-driven, with explicit AAA comments (user instinct):
```go
func TestDiffCounts(t *testing.T) {
	tests := []struct {
		name string
		want map[string]int64
		got  map[string]int64
		out  []string
	}{ /* ... */ }
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (inputs from tt)

			// Act
			out := diffCounts(tt.want, tt.got)

			// Assert
			if !slices.Equal(out, tt.out) {
				t.Fatalf("diffCounts() = %v, want %v", out, tt.out)
			}
		})
	}
}
```

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| `go.mod`, `go.sum` | CREATE | Module `github.com/restuu/go-sql-benchmark`, `go 1.27.3`, pgx v5 |
| `db/schema.sql` | CREATE | DDL for `models`, `customers`, `suppliers`, `orders` |
| `db/seed.sql` | CREATE | Deterministic, re-runnable seed via `generate_series` |
| `docker-compose.yml` | CREATE | Postgres 15, healthcheck, init scripts mounted |
| `Makefile` | CREATE | `arch-check`, `db-*`, `build`, `vet`, `test`, `test-db` |
| `internal/seedcheck/seedcheck.go` | CREATE | Row-count + Postgres-major verification |
| `internal/seedcheck/seedcheck_test.go` | CREATE | Unit test (pure diff) + DB integration test (skips without `BENCH_DSN`) |
| `cmd/dbcheck/main.go` | CREATE | CLI wrapper for `make db-check` |
| `LICENSE` | CREATE | MIT, project notice + upstream notice |
| `CLAUDE.md` | CREATE | Commands and non-obvious decisions |

## NOT Building
- No `README.md` (Phase 4 generates it; the generator must include upstream attribution).
- No `make bench` (needs the runner; added in Phase 2/5).
- No `Adapter` interface, runner, metrics, or any library adapter (Phase 2/3).
- No `go tool` directives for sqlc/ent/jet (added with the adapters in Phase 3).
- No `.gitignore` or `git init` (project is not a git repo yet; user decision).
- No Dockerfile for the benchmarker: the harness runs natively on the host, only Postgres is in Docker.

---

## Step-by-Step Tasks

### Task 1: Module and driver dependency
- **ACTION**: Initialise the module.
- **IMPLEMENT**:
  ```bash
  go mod init github.com/restuu/go-sql-benchmark
  go get github.com/jackc/pgx/v5@latest
  ```
  Resulting `go.mod` must have `go 1.27.3` and NO `toolchain` line.
- **MIRROR**: PRD "Architecture Notes" (fresh module, no toolchain line).
- **IMPORTS**: `github.com/jackc/pgx/v5/stdlib` (used in Task 6).
- **GOTCHA**: `go mod tidy` can add a `toolchain` line or raise the `go` line if a dependency demands it; re-check with `grep -E '^(go|toolchain)' go.mod` after every `go get`/`tidy`. Module path assumes GitHub user `restuu` (from `gh api user`); change only if the repo will live elsewhere.
- **VALIDATE**: `grep -E '^(go|toolchain)' go.mod` prints exactly `go 1.27.3`.

### Task 2: Schema
- **ACTION**: Write `db/schema.sql`.
- **IMPLEMENT**:
  ```sql
  -- Single-table scenarios (INSERT/UPDATE/DELETE/SELECT). Same shape as upstream go-orm-benchmarks.
  DROP TABLE IF EXISTS orders, customers, suppliers, models;

  CREATE TABLE models (
      id      SERIAL  NOT NULL,
      name    TEXT    NOT NULL,
      title   TEXT    NOT NULL,
      fax     TEXT    NOT NULL,
      web     TEXT    NOT NULL,
      age     INTEGER NOT NULL,
      "right" BOOLEAN NOT NULL,
      counter BIGINT  NOT NULL,
      CONSTRAINT models_pkey PRIMARY KEY (id)
  );

  -- JOIN scenario: orders JOIN customers. UNION scenario: customers UNION suppliers (same column shape).
  CREATE TABLE customers (
      id    SERIAL PRIMARY KEY,
      name  TEXT NOT NULL,
      email TEXT NOT NULL,
      city  TEXT NOT NULL
  );
  CREATE TABLE suppliers (
      id    SERIAL PRIMARY KEY,
      name  TEXT NOT NULL,
      email TEXT NOT NULL,
      city  TEXT NOT NULL
  );
  CREATE TABLE orders (
      id          SERIAL      PRIMARY KEY,
      customer_id INTEGER     NOT NULL REFERENCES customers (id),
      status      TEXT        NOT NULL,
      total       INTEGER     NOT NULL,
      created_at  TIMESTAMPTZ NOT NULL
  );
  CREATE INDEX orders_customer_id_idx ON orders (customer_id);
  ```
- **MIRROR**: NAMING_CONVENTION (keep `models` identical; drop upstream's obsolete `WITH (OIDS=FALSE)`).
- **GOTCHA**: `"right"` must stay quoted. Phase 3 points `sqlc.yaml` at this file (`schema: "../../db/schema.sql"`), so the `DROP TABLE` line must not break sqlc parsing — sqlc ignores `DROP TABLE IF EXISTS`, but if it complains in Phase 3, move the drop into the seed script.
- **VALIDATE**: Applies cleanly in Task 4 (`make db-up`).

### Task 3: Seed
- **ACTION**: Write `db/seed.sql`, deterministic and re-runnable.
- **IMPLEMENT**:
  ```sql
  TRUNCATE orders, customers, suppliers, models RESTART IDENTITY;

  INSERT INTO models (name, title, fax, web, age, "right", counter)
  SELECT 'Orm Benchmark', 'Just a Benchmark for fun', '99909990', 'http://blog.milkpod29.me', 100, TRUE, 1000
  FROM generate_series(1, 10000);

  INSERT INTO customers (name, email, city)
  SELECT 'customer ' || i, 'customer' || i || '@example.com', 'city ' || (i % 50)
  FROM generate_series(1, 1000) AS i;

  INSERT INTO suppliers (name, email, city)
  SELECT 'supplier ' || i, 'supplier' || i || '@example.com', 'city ' || (i % 50)
  FROM generate_series(1, 1000) AS i;

  INSERT INTO orders (customer_id, status, total, created_at)
  SELECT (i % 1000) + 1,
         (ARRAY['new', 'paid', 'shipped', 'done'])[(i % 4) + 1],
         (i % 500) + 1,
         TIMESTAMPTZ '2026-01-01 00:00:00+00' + i * INTERVAL '1 minute'
  FROM generate_series(1, 10000) AS i;

  ANALYZE;
  ```
  Row counts: models 10000, customers 1000, suppliers 1000, orders 10000 (these are the `ExpectedRows` in Task 5). Dataset size is a PRD open question (Phase 5); keeping all sizes in this one file makes that a one-file change.
- **GOTCHA**: Row values come from upstream `NewModel()` so ORM-inserted rows and seeded rows look alike. `TRUNCATE` lists every table in one statement because `orders` has a FK to `customers`. Phase 2's runner will need to reseed after DELETE scenarios; it can reuse this file (`make db-seed`) — do not add a second seed path.
- **VALIDATE**: `make db-check` (Task 7) reports the exact counts.

### Task 4: docker-compose.yml
- **ACTION**: Postgres 15 with healthcheck and init scripts.
- **IMPLEMENT**:
  ```yaml
  name: go-sql-benchmark
  services:
    postgres:
      image: postgres:15-alpine
      command: ["postgres", "-c", "max_connections=300"]
      ports:
        - "127.0.0.1:${BENCH_PG_PORT:-55432}:5432"
      environment:
        POSTGRES_USER: postgres
        POSTGRES_PASSWORD: ${BENCH_PG_PASSWORD:-bench}
        POSTGRES_DB: bench
      volumes:
        - ./db/schema.sql:/docker-entrypoint-initdb.d/01-schema.sql:ro
        - ./db/seed.sql:/docker-entrypoint-initdb.d/02-seed.sql:ro
      healthcheck:
        test: ["CMD-SHELL", "pg_isready -U postgres -d bench"]
        interval: 2s
        timeout: 3s
        retries: 30
  ```
  No named volume: `down` discards data, every `up` is a fresh seeded DB.
- **MIRROR**: upstream `docker-compose.yml` (postgres service) but pinned to 15 and without the benchmarker container.
- **GOTCHA**: `name: go-sql-benchmark` pins the Compose project name; without it the name follows the directory, and the planned move to `~/github.com/restuu/go-sql-benchmark` would rename the container and network. Host port `5432` is taken by a native Postgres on this machine, `5433` and `5435` by other containers — hence `55432`, bound to `127.0.0.1` only. `max_connections=300` because the default 100 would turn the 100-worker stress profile into connection errors. The init scripts only run on an empty data dir, which is why `db-reset` uses `down -v` (Task 5). `restart` policy deliberately omitted.
- **VALIDATE**: `docker compose config -q` exits 0.

### Task 5: Makefile
- **ACTION**: Targets for arch guard and DB lifecycle. GNU Make 3.81 (macOS default): no `.ONESHELL`, no `$(file ...)`.
- **IMPLEMENT**:
  ```make
  BENCH_PG_PORT     ?= 55432
  BENCH_PG_PASSWORD ?= bench
  BENCH_DSN         ?= postgres://postgres:$(BENCH_PG_PASSWORD)@127.0.0.1:$(BENCH_PG_PORT)/bench?sslmode=disable
  export BENCH_PG_PORT BENCH_PG_PASSWORD BENCH_DSN

  GO_ARCH   := $(shell go env GOARCH)
  HOST_ARCH := $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')

  .PHONY: arch-check db-up db-down db-reset db-seed db-check build vet test test-db

  arch-check:
  	@if [ "$(GO_ARCH)" != "$(HOST_ARCH)" ]; then \
  		echo "Go arch $(GO_ARCH) != host arch $(HOST_ARCH): install the native Go toolchain (benchmark numbers are invalid under emulation)"; \
  		exit 1; \
  	fi
  	@echo "arch OK: $(GO_ARCH)"

  db-up:
  	docker compose up -d --wait

  db-down:
  	docker compose down -v

  db-reset: db-down db-up

  db-seed:
  	docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d bench < db/seed.sql

  db-check:
  	go run ./cmd/dbcheck

  build:
  	go build ./...

  vet:
  	go vet ./...

  test:
  	env -u BENCH_DSN go test ./...

  test-db: db-up
  	go test -count=1 -run Integration ./...
  ```
  (Recipe lines are tab-indented.) `BENCH_DSN` is exported so `go run` / `go test` pick it up. The integration test (Task 6) skips only when `BENCH_DSN` is unset, so `make test` strips it with `env -u` and needs no database, while `test-db` brings the DB up and runs it.
- **GOTCHA**: `uname -m` reports `x86_64` for an Intel/Rosetta shell and `arm64` on Apple Silicon, so map to GOARCH names. `make` re-evaluates `$(shell go env GOARCH)` each run, so `GOARCH=amd64 make arch-check` simulates a mismatch for validation.
- **VALIDATE**: `make arch-check` passes; `GOARCH=amd64 make arch-check` exits non-zero with the message.

### Task 6: seedcheck package (tests first)
- **ACTION**: RED → GREEN → refactor. Write `internal/seedcheck/seedcheck_test.go`, run it (compile failure/FAIL), then write `seedcheck.go`.
- **IMPLEMENT** (`seedcheck.go`):
  ```go
  // Package seedcheck verifies that the benchmark database was seeded as db/seed.sql defines.
  package seedcheck

  import (
  	"context"
  	"database/sql"
  	"fmt"
  	"slices"
  	"sort"
  	"strconv"

  	_ "github.com/jackc/pgx/v5/stdlib"
  )

  // WantPostgresMajor is the Postgres major version the benchmark targets.
  const WantPostgresMajor = 15

  // postgresVersionNumMultiple converts server_version_num (e.g. 150013) to a major version.
  const postgresVersionNumMultiple = 10000

  // ExpectedRows mirrors the row counts generated by db/seed.sql.
  var ExpectedRows = map[string]int64{
  	"models":    10000,
  	"customers": 1000,
  	"suppliers": 1000,
  	"orders":    10000,
  }

  // diffCounts returns one sorted message per table whose count differs or is missing.
  func diffCounts(want, got map[string]int64) []string { ... }

  // Check verifies the Postgres major version and every table's row count.
  func Check(ctx context.Context, db *sql.DB) error { ... }
  ```
  `Check`: `SHOW server_version_num` → `strconv.Atoi` → compare `/ postgresVersionNumMultiple` with `WantPostgresMajor`; then for each table in `ExpectedRows` run `SELECT count(*) FROM <table>` (table names come from the constant map above, never from input — note this in a comment) and pass the collected map to `diffCounts`; return `fmt.Errorf("seedcheck: seed drift: %s", strings.Join(diffs, "; "))` when non-empty. Wrap every query error with `fmt.Errorf("seedcheck: ...: %w", err)`.
- **Tests** (`seedcheck_test.go`, AAA comments in each):
  - `TestDiffCounts` table-driven: equal maps → empty; count mismatch → `"orders: want 10000, got 9"`; missing table → `"suppliers: missing"`; extra table in `got` ignored; output sorted.
  - `TestCheckIntegration`: `dsn := os.Getenv("BENCH_DSN"); if dsn == "" { t.Skip("BENCH_DSN not set") }`; open `sql.Open("pgx", dsn)`, call `Check` with a 10s context, expect nil.
  - Drift case inside the integration test: begin a transaction? `Check` takes `*sql.DB`, so instead keep drift covered by `TestDiffCounts` and the manual check in Validation (delete a row, expect failure, `make db-seed`).
- **MIRROR**: ERROR_HANDLING, TEST_STRUCTURE.
- **GOTCHA**: Do not import `lib/pq`; the registered driver name is `"pgx"`. `slices` is stdlib since Go 1.21.
- **VALIDATE**: `go test ./internal/seedcheck/...` fails before `seedcheck.go` exists and passes after; `go vet ./...` clean.

### Task 7: cmd/dbcheck
- **ACTION**: Thin `main`.
- **IMPLEMENT**:
  ```go
  package main

  func main() {
  	dsn := os.Getenv("BENCH_DSN")
  	if dsn == "" {
  		fmt.Fprintln(os.Stderr, "dbcheck: BENCH_DSN is required")
  		os.Exit(2)
  	}
  	db, err := sql.Open("pgx", dsn)   // driver registered by seedcheck's stdlib import
  	if err != nil { fail(err) }
  	defer db.Close()
  	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  	defer cancel()
  	if err := seedcheck.Check(ctx, db); err != nil { fail(err) }
  	fmt.Println("seed OK: postgres 15,", len(seedcheck.ExpectedRows), "tables")
  }
  ```
  `fail` prints to stderr and `os.Exit(1)` (note `os.Exit` skips deferred calls; call `db.Close()` and `cancel()` explicitly inside `fail` paths or structure as `run() error` + `main` calling `os.Exit` once — prefer `run() error`).
- **MIRROR**: LOGGING_PATTERN.
- **GOTCHA**: `defer` + `os.Exit` leaks the deferral; hence `run() error`. Timeout constant named (`checkTimeout = 10 * time.Second`), no magic numbers.
- **VALIDATE**: `make db-check` prints `seed OK...`.

### Task 8: LICENSE
- **ACTION**: MIT license with two notices.
- **IMPLEMENT**: Standard MIT text with the first line `Copyright (c) 2026 restuu`, followed by a section `This project includes code and schema derived from github.com/efectn/go-orm-benchmarks:` and the verbatim upstream MIT text beginning `Copyright (c) 2021 M. Efe Çetin` (copy from upstream `LICENSE`).
- **GOTCHA**: Copyright holder name is an assumption (OS/GitHub user). Upstream's notice must remain verbatim wherever its adapters are copied (Phase 3).
- **VALIDATE**: `grep -c "Efe Çetin" LICENSE` returns 1.

### Task 9: CLAUDE.md
- **ACTION**: Write only the non-derivable facts, under ~60 lines.
- **IMPLEMENT** sections:
  - Purpose (one paragraph) and "numbers are only comparable on the same machine".
  - Commands: `make arch-check | db-up | db-down | db-reset | db-seed | db-check | build | vet | test | test-db`.
  - Decisions: Go 1.27.3 native arm64, no `toolchain` line, never run under Rosetta; Postgres 15 on host port 55432 (5432 is occupied by a native Postgres on the reference machine); `pgx/stdlib` is the shared driver for every `database/sql`-based library, never `lib/pq`; custom runner, not `testing.B`; sqlx kept and labelled "no updates since 2024"; unsupported scenario = "n/a (raw SQL fallback)", excluded from rankings.
  - Layout: `db/` (schema+seed, single source of truth, also consumed by sqlc in Phase 3), `internal/`, `cmd/`.
  - Conventions: tests are table-driven with `// Arrange`, `// Act`, `// Assert`; upstream code copied into adapters keeps its attribution comment.
  - Pointer to PRD and plans under `.claude/PRPs/`.
- **VALIDATE**: Every command listed in the file exists in the Makefile (`grep` each target).

### Task 10: End-to-end validation
- **ACTION**: Run the Validation Commands below in order and record results.
- **VALIDATE**: All EXPECT lines hold.

---

## Testing Strategy

### Unit Tests
| Test | Input | Expected Output | Edge Case? |
|---|---|---|---|
| `TestDiffCounts/equal` | want == got | empty slice | |
| `TestDiffCounts/mismatch` | orders got 9, want 10000 | `["orders: want 10000, got 9"]` | |
| `TestDiffCounts/missing_table` | suppliers absent from got | `["suppliers: missing"]` | yes |
| `TestDiffCounts/extra_table` | extra key in got | ignored | yes |
| `TestDiffCounts/sorted` | two mismatches | alphabetical order | yes |
| `TestCheckIntegration` | seeded DB via `BENCH_DSN` | nil error | skipped without env |

### Edge Cases Checklist
- [x] Empty input (empty maps)
- [ ] Maximum size input — N/A
- [ ] Invalid types — N/A
- [x] Concurrent access — N/A (read-only checks)
- [x] Network failure: `dbcheck` with the DB down exits non-zero (manual)
- [ ] Permission denied — N/A

Coverage target: `internal/seedcheck` ≥ 80% with `make test-db` (unit + integration); `cmd/dbcheck` is a thin wrapper verified by `make db-check`.

---

## Validation Commands

### Static Analysis
```bash
make arch-check && make build && make vet && gofmt -l .
```
EXPECT: `arch OK: arm64`, zero build/vet errors, `gofmt -l` prints nothing.

### Arch guard negative test
```bash
GOARCH=amd64 make arch-check; echo "exit=$?"
```
EXPECT: message "Go arch amd64 != host arch arm64 ...", `exit=2` (make reports 2 when a recipe fails with 1).

### Unit Tests
```bash
make test
```
EXPECT: pass; integration test skipped.

### Database Validation
```bash
make db-reset
make db-check
make test-db
go test -count=1 -cover -run . ./internal/seedcheck/
```
EXPECT: `db-reset` ends healthy; `db-check` prints `seed OK`; integration test passes; coverage ≥ 80%.

### Drift check (manual)
```bash
docker compose exec -T postgres psql -U postgres -d bench -c "DELETE FROM orders WHERE id = 1"
make db-check        # EXPECT: exit 1, message "orders: want 10000, got 9999"
make db-seed && make db-check   # EXPECT: seed OK again (proves re-runnable seed)
```

### Environment sanity
```bash
grep -E '^(go|toolchain)' go.mod        # EXPECT: only "go 1.27.3"
docker compose ps --format '{{.Service}} {{.Health}}'   # EXPECT: postgres healthy
make db-down                             # EXPECT: container removed, nothing left on 55432
```

### Manual Validation
- [ ] `psql` into the DB shows 4 tables and the `orders_customer_id_idx` index.
- [ ] A native Postgres on 5432 and the existing containers are untouched by `db-up`/`db-down`.

---

## Acceptance Criteria (from PRD Phase 1 success signal)
- [ ] `make db-up` yields a seeded Postgres 15
- [ ] `go build ./...` passes on Go 1.27.3
- [ ] Arch guard fails on a mismatched Go
- [ ] LICENSE carries upstream attribution; `CLAUDE.md` exists
- [ ] Tests written and passing; no vet/gofmt findings

## Completion Checklist
- [ ] Code follows discovered patterns (error wrapping, driver name `"pgx"`)
- [ ] No hardcoded values beyond documented local-only defaults (DB password `bench`, port 55432, overridable by env)
- [ ] Tests follow AAA structure
- [ ] No unnecessary scope additions (no runner, adapters, README)
- [ ] Self-contained — no questions needed during implementation

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `go get` adds a `toolchain` line or raises the `go` line | M | Breaks the "no toolchain download" rule | Grep `go.mod` after every dependency change; fix with `go mod edit -toolchain=none` |
| Port 55432 also taken | L | `db-up` fails | `BENCH_PG_PORT=<free> make db-up`; port is env-driven |
| `max_connections=300` insufficient or too heavy for Docker VM | L | Stress profile errors in Phase 5 | Revisit with stress parameters (PRD open question) |
| `db/schema.sql` `DROP TABLE` confuses sqlc in Phase 3 | L | Codegen fails | Move DROP into the seed script then |
| Docker Desktop/OrbStack VM adds measurement noise | H | Affects later phases, not this one | Already in PRD risks; record host info in Phase 2 |

## Notes
- **Confirmed by user (2026-10-21)**: module path `github.com/restuu/go-sql-benchmark`; JOIN = `orders ⋈ customers`, UNION = `customers ∪ suppliers`; seed sizes 10000/1000/1000/10000. LICENSE holder `restuu` follows from the same GitHub identity.
- **Planned move**: the directory will later move to `~/github.com/restuu/go-sql-benchmark` (no such directory exists there yet). Nothing in the plan depends on the absolute path: Makefile and compose paths are relative, the module path is independent of location, and the Compose project name is pinned. Run `make db-down` before the move so no container is orphaned under the old project name.
- **Input for Phase 2** (not decided here): DELETE and UPDATE consume or mutate seeded `models` rows; at stress throughput 10,000 rows can run out. The runner must either insert-then-delete its own rows or reseed between repeats (`make db-seed` already works for the latter).
- **Input for Phase 4**: the generated README must include the upstream attribution, since Phase 1 intentionally does not create one.
- The project directory is not a git repository; `git init` and a `.gitignore` are left for the user.
