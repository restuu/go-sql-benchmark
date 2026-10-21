# Plan: Core adapters (sqlc, sqlx, gorm, ent, go-jet) + cross-adapter correctness

## Summary
Add five library adapters behind the Phase 2 `bench.Adapter` contract, a value-level correctness check that runs the adapters against each other (and against golden spot values), and the codegen wiring for sqlc, ent and go-jet. Every (library, scenario) cell ends up a measured number or an explicit `na` (`bench.ErrUnsupported`).

## User Story
As the benchmark owner, I want sqlc, sqlx, gorm, ent and go-jet measured on the same six scenarios as `database/sql`, so that the Phase 5 run compares real libraries and I can trust that they all read and write the same values.

## Problem → Solution
Only `database_sql` is registered, and the runner verifies row counts only (a library that maps `title` to `fax` still passes) → five more adapters, a `bench.Verifier` side-interface plus `internal/correctness` that compares values, and a `make gen` / `make gen-check` pair that keeps generated code reproducible.

## Metadata
- **Complexity**: Large
- **Source PRD**: `.claude/PRPs/prds/go-sql-benchmark.prd.md`
- **PRD Phase**: 3 — Core adapters
- **Estimated Files**: about 25 hand-written (5 adapters, 5 adapter tests, `adaptertest`, `correctness`, `bench/verify.go`, `databasesql/verify.go`, 3 ent schemas, `sqlc.yaml`, `queries.sql`, `tools/go.mod`, Makefile, CLAUDE.md edits) + generated code (sqlc `db/`, ent `entgen/`, jet `gen/`), committed

---

## UX Design
N/A — internal change. The only user-visible changes are `-libs` accepting five more names, `make gen` / `make gen-check`, and `make bench` failing fast with a mismatch list if a library returns wrong values.

### Interaction Changes
| Touchpoint | Before | After | Notes |
|---|---|---|---|
| `go run ./cmd/bench -h` | `-libs` lists `database_sql` | lists `database_sql, ent, gorm, jet, sqlc, sqlx` | default `-libs` is every registered name, so a default run now has 6 libraries |
| `make bench` | counts checked in the cell probe only | correctness pre-flight, then the matrix | pre-flight reseeds when done |
| `make gen` / `make gen-check` | n/a | regenerate sqlc/ent/jet; fail if committed output drifts | jet needs the DB (`db-up`) |

---

## Mandatory Reading

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `internal/bench/bench.go` | 1-118 | `Adapter` interface, `Model`, `NewModel`, `PoolConfig`, `ErrUnsupported`, `ExpectedRows` (1/1/1/1/10/40) |
| P0 | `internal/adapters/databasesql/databasesql.go` | 1-147 | Reference adapter: SQL text, `Init`/`Close`/re-Init, `affected`, error wrapping, scan-every-column |
| P0 | `internal/adapters/databasesql/databasesql_test.go` | 1-150 | Integration test shape the shared contract suite generalises |
| P0 | `cmd/bench/main.go` | 31-34, 67-113 | `registry`, `run()` order (seedcheck → hostInfo → `runner.Run`), `hostInfo`/`moduleVersion` |
| P1 | `internal/runner/cell.go` | 28-60, 100-120 | Per-cell flow: reseed, probe, fresh pool, delete id pool; what ids each scenario uses |
| P1 | `internal/runner/worker.go` | 120-260 | Which ids/cities workers pass (`1..10000` models, `1..1000` customers, `city 0..49`) |
| P1 | `internal/runner/store.go` | 17-80 | `Store.Reseed`, `Store.InsertDeletable` (reuse in correctness; do not re-implement) |
| P1 | `db/schema.sql`, `db/seed.sql` | all | Source of truth for sqlc, jet generation and golden values |
| P1 | `internal/seedcheck/seedcheck.go` | 17-30 | `ExpectedRows` (change together with the seed) |
| P2 | `CLAUDE.md` | Runner section | Contract text that Task 12 extends |
| P2 | `.claude/PRPs/reports/runner-and-reference-adapter-report.md` | Open follow-ups | Context only; do not fix those here |

## External Documentation

| Topic | Source | Key Takeaway |
|---|---|---|
| `go tool` with a separate modfile | spike on Go 1.27.3 (this plan) | `go tool -modfile=tools/go.mod sqlc|ent|jet` works; main `go.mod` stays at `go 1.27.3` with no `toolchain` line and no `lib/pq`/`mysql`/`sqlite3` entries. Putting the same `tool` directives in the main `go.mod` adds ~80 lines incl. `lib/pq` |
| sqlc | v1.31.1 (requires Go 1.26) | builds with `CGO_ENABLED=0`; use `sql_package: "database/sql"` (not `pgx/v5`) so the generated code runs on the pooled `*sql.DB` over `pgx/stdlib` |
| ent | v0.14.6 | schema in `schema/`, `ent generate --target <dir> <schema dir>`; generated `migrate` package pulls `ariga.io/atlas` into the build (accepted, off the hot path) |
| go-jet | v2.16.1 | generator needs a live DB: `jet -source=postgres -dsn=... -path=<dir>`; runtime package `github.com/go-jet/jet/v2/postgres` has no `lib/pq` dependency |
| gorm | gorm v1.31.2, `gorm.io/driver/postgres` v1.6.3 (Go 1.25) | pass the shared `*sql.DB` via `postgres.Config{Conn: sqlDB}` so the driver is `pgx/stdlib` like everyone else |
| sqlx | v1.4.0 (last release 2024-04-15) | `sqlx.NewDb(sqlDB, "pgx")`: bind type `$n` is chosen for driver name `pgx`. Label "no updates since 2024" is Phase 4 metadata, not this phase |

---

## Patterns to Mirror

### NAMING_CONVENTION
// SOURCE: `internal/adapters/databasesql/databasesql.go:1-4, 29-35`
```go
// Package databasesql is the reference adapter: plain database/sql over the pgx stdlib driver.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package databasesql
...
// Adapter implements bench.Adapter with database/sql. It is not safe to call Init concurrently.
type Adapter struct{ db *sql.DB }

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "database_sql" }
```
New packages: `internal/adapters/<lib>`, exported `Adapter` + `New() bench.Adapter`, `Name()` = registry key (`sqlc`, `sqlx`, `gorm`, `ent`, `jet`). Import the library under an alias (`sqlxlib`, `gormlib`, `entgen`, `jetpg`) because the directory name equals the library name. Each adapter file keeps the upstream attribution comment when code is derived from efectn/go-orm-benchmarks (the Go code is rewritten for the new contract, so say "Derived in part from").

### INIT_CLOSE (pool config, ping, re-Init after Close)
// SOURCE: `internal/adapters/databasesql/databasesql.go:37-62`
```go
func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("databasesql: open: %w", err)
	}
	db.SetMaxOpenConns(pool.MaxOpen)
	db.SetMaxIdleConns(pool.MaxIdle)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("databasesql: ping: %w", err)
	}
	a.db = db
	return nil
}

func (a *Adapter) Close() error {
	if a.db == nil {
		return nil
	}
	err := a.db.Close()
	a.db = nil
	if err != nil {
		return fmt.Errorf("databasesql: close: %w", err)
	}
	return nil
}
```
Every new adapter opens its own `sql.Open("pgx", dsn)`, applies `SetMaxOpenConns/SetMaxIdleConns`, pings, then wraps the `*sql.DB` in the library. Close the `*sql.DB` (closing the library handle must close it) and nil the fields so `Init` works again.

### ERROR_HANDLING
// SOURCE: `internal/adapters/databasesql/databasesql.go:79-86, 137-147`
```go
	if err != nil {
		return 0, fmt.Errorf("databasesql: select: %w", err)
	}
	return 1, nil
...
func affected(op string, res sql.Result, err error) (int, error) {
	if err != nil {
		return 0, fmt.Errorf("databasesql: %s: %w", op, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("databasesql: %s: rows affected: %w", op, err)
	}
	return int(n), nil
}
```
Prefix = package name, `%w`, lower-case op. A missing row on SELECT is an error (not a 0 count). UPDATE/DELETE of a missing id returns `0, nil`. JOIN/UNION with no match return `0, nil`.

### READ_SCANS_EVERY_COLUMN
// SOURCE: `internal/adapters/databasesql/databasesql.go:88-112`
Reads iterate rows, scan all selected columns, count, check `rows.Err()`. Library-idiomatic result handling is allowed (sqlx `Select` into a slice, gorm `Find`, sqlc `:many`), because materialising results is part of what is being compared. Do not add work the library would not do (no sorting, no extra conversion in the benchmarked method).

### TEST_STRUCTURE (table-driven, Arrange/Act/Assert, integration skip)
// SOURCE: `internal/adapters/databasesql/databasesql_test.go:18-47, 49-88`
```go
func newInitialised(t *testing.T) (bench.Adapter, runner.Store, context.Context) {
	t.Helper()
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	...
}

func TestAdapterIntegration(t *testing.T) {
	a, store, ctx := newInitialised(t)
	tests := []struct {
		name string
		call func() (int, error)
		want int
	}{
		{"insert", func() (int, error) { return a.Insert(ctx, bench.NewModel()) }, 1},
		...
		{"update missing id affects nothing", func() (int, error) { return a.Update(ctx, missingID, bench.NewModel()) }, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (call from tt)

			// Act
			got, err := tt.call()

			// Assert
			...
```
Integration tests are named `...Integration` (so `make test-db` runs them via `-run Integration`), skip without `BENCH_DSN`, reseed in `t.Cleanup`, and use `-p 1` (shared DB).

### REGISTRY
// SOURCE: `cmd/bench/main.go:31-34`
```go
// registry maps -libs names to adapter constructors; Phase 3 adapters register here.
var registry = map[string]func() bench.Adapter{
	"database_sql": databasesql.New,
}
```

---

## Decisions made by this plan (flag to the user; change before implementing if they disagree)

1. **"Natively expressible" rule.** A scenario is supported if it is written with the library's own query API or, for SQL-first libraries (sqlc, sqlx), its own SQL files/strings. A scenario that would need the library's raw-SQL escape hatch (`gorm.Raw`, ent `QueryContext`/low-level `sql.Selector`, `jet.RawStatement`) is `ErrUnsupported`. Defaults:

| Library | INSERT | UPDATE | DELETE | SELECT | JOIN | UNION |
|---|---|---|---|---|---|---|
| sqlc | ok | ok | ok | ok | ok (SQL in `queries.sql`) | ok |
| sqlx | ok | ok | ok | ok | ok (SQL string) | ok |
| gorm | ok | ok | ok | ok | ok (`Joins("Customer")` belongs-to) | **na** (no UNION in the query API) |
| ent | ok | ok | ok | ok | ok via edge eager-load `WithCustomer()` (2 queries, not one SQL JOIN) | **na** (typed entity API cannot combine two entities; `Selector.Union` is the low-level builder) |
| jet | ok | ok | ok | ok | ok (`INNER_JOIN`) | ok (`UNION`) |

   Ent JOIN as 2 queries and the strictness of the rule are judgement calls; each is a one-line change in the adapter plus this table if the user wants a stricter or looser reading. Record the final table in `CLAUDE.md` (Task 12) so Phase 4 can label cells.
2. **Correctness lives beside the hot path, not on it.** `Adapter` is unchanged. Adapters also implement an optional `bench.Verifier` that returns values through the same library query as the benchmarked method; the benchmarked method discards the result, the verifier maps it to `bench.Model`/`JoinRow`/`UnionRow`. Mapping cost therefore never enters allocs/op.
3. **Reference for value comparison = `databasesql`**, itself checked against hard-coded golden spot values (so "all libraries agree on the same wrong thing" is caught).
4. **Correctness runs as a pre-flight in `cmd/bench run()`** (always, for the selected libs) and as an integration test over the registry. No opt-out flag.
5. **Codegen tools in `tools/go.mod`**, not the main `go.mod`: `go tool -modfile=tools/go.mod ...`. Generated code is committed; `make gen-check` fails on drift.
6. **Registry names**: `sqlc`, `sqlx`, `gorm`, `ent`, `jet` (match upstream result keys; go-jet is `jet`).
7. **Driver**: every adapter uses `sql.Open("pgx", dsn)` (extended protocol, pgx stdlib defaults, no explicit `Prepare`), never gorm's own pgx connection and never `PreferSimpleProtocol`, so differences are the library's.
8. **Write semantics that must not leak between libraries**: UPDATE of a missing id affects 0 rows and never inserts; DELETE of a missing id affects 0 rows; SELECT of a missing id errors.

## Files to Change

| File | Action | Justification |
|---|---|---|
| `tools/go.mod`, `tools/go.sum` | CREATE | pins sqlc v1.31.1, ent v0.14.6, jet v2.16.1 as `tool` directives, isolated from the main module |
| `Makefile` | UPDATE | `gen`, `gen-sqlc`, `gen-ent`, `gen-jet`, `gen-check` |
| `go.mod`, `go.sum` | UPDATE | add sqlx, gorm, gorm postgres driver, ent, go-jet runtime deps; re-check no `toolchain` line |
| `internal/bench/verify.go` (+ `verify_test.go`) | CREATE | `JoinRow`, `UnionRow`, `Verifier` |
| `internal/adapters/databasesql/verify.go` | CREATE | reference `Verifier` (new file; `databasesql.go` untouched) |
| `internal/adaptertest/adaptertest.go` | CREATE | shared integration contract suite for the five new adapters |
| `internal/adapters/sqlx/sqlx.go`, `sqlx_test.go` | CREATE | adapter |
| `internal/adapters/sqlc/{sqlc.go, sqlc_test.go, sqlc.yaml, queries.sql}` | CREATE | adapter + codegen input |
| `internal/adapters/sqlc/db/*` | CREATE (generated) | `go tool sqlc generate` output |
| `internal/adapters/gorm/gorm.go`, `gorm_test.go` | CREATE | adapter |
| `internal/adapters/ent/{ent.go, ent_test.go, schema/*.go}` | CREATE | adapter + schema (Model, Customer, Order) |
| `internal/adapters/ent/entgen/*` | CREATE (generated) | `ent generate` output |
| `internal/adapters/jet/{jet.go, jet_test.go}` | CREATE | adapter |
| `internal/adapters/jet/gen/**` | CREATE (generated) | `jet` output (`model`, `table` packages) |
| `internal/correctness/{correctness.go, correctness_test.go}` | CREATE | cross-adapter + golden + write-path checks |
| `cmd/bench/main.go` | UPDATE | register 5 adapters, pre-flight check, library versions in `hostInfo` |
| `cmd/bench/correctness_test.go` | CREATE | integration test over the registry |
| `CLAUDE.md` | UPDATE | contract (Verifier), n/a table, layout, `make gen*`, graph note |
| `.claude/PRPs/prds/go-sql-benchmark.prd.md` | UPDATE | Phase 3 → in-progress + plan link (done by this planning step) |

## NOT Building
- The 14 extra libraries (Phase 6) and any native (non-`database/sql`) pgx adapter.
- Report generation, ranking, sqlx "no updates since 2024" label (Phase 4).
- Fixes for the Phase 2 follow-ups (CHECKPOINT, `Errors`/`ErrorRate` validation, benchcmp gaps, partial results on abort, p95 stability). Do not touch `internal/runner` or `internal/result`.
- Migrating `databasesql_test.go` onto `adaptertest` (list as a follow-up; leave Phase 2 tests alone).
- `InsertMulti`, `ListModels`, prepared-statement variants, transactions, ent/gorm auto-migration, a `Supplier` entity in ent or gorm (their UNION is `na`).
- A skip flag for the pre-flight.

---

## Step-by-Step Tasks

Work order matters: shared pieces first (2-4), then adapters from simplest to most codegen-heavy (5-9), then the cross-adapter check (10), wiring (11), docs (12), full validation (13). Each Go task: write the test first and show it failing (RED), then implement (GREEN), `go test -race`, `gofmt -l .` empty, `go vet`.

### Task 1: Tooling module and Makefile
- **ACTION**: Create `tools/go.mod` (`module github.com/restuu/go-sql-benchmark/tools`, `go 1.27.3`) with `go get -tool -modfile=tools/go.mod` for `github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1`, `entgo.io/ent/cmd/ent@v0.14.6`, `github.com/go-jet/jet/v2/cmd/jet@v2.16.1`. Add Makefile targets.
- **IMPLEMENT**:
  ```make
  GOTOOL := go tool -modfile=tools/go.mod
  gen: gen-sqlc gen-ent gen-jet
  gen-sqlc: ; $(GOTOOL) sqlc generate -f internal/adapters/sqlc/sqlc.yaml
  gen-ent:  ; $(GOTOOL) ent generate --target internal/adapters/ent/entgen ./internal/adapters/ent/schema
  gen-jet: db-up ; $(GOTOOL) jet -source=postgres -dsn='$(BENCH_DSN)' -path=internal/adapters/jet/gen
  gen-check: gen ; git diff --exit-code -- internal/adapters/sqlc/db internal/adapters/ent/entgen internal/adapters/jet/gen
  ```
  Add the new names to `.PHONY`. Keep the `tools/` directory out of `./...` problems: a nested module is excluded from `go build ./...` automatically.
- **MIRROR**: Makefile style (`@` only where output is noise, one-line recipes, `.PHONY` list at line 11).
- **GOTCHA**: the jet generator rewrites its output directory (`<path>/<dbname>/<schema>/{model,table}`); with dbname `bench` the packages land in `internal/adapters/jet/gen/bench/public/`. `gen-check` also fails on *untracked* files if you only use `git diff`; add `git status --porcelain` check or `git add -N`. `go tool` compile of the three tools takes a few minutes the first time. Re-check `grep -E '^(go|toolchain)' go.mod tools/go.mod` prints only `go 1.27.3` lines.
- **VALIDATE**: `go tool -modfile=tools/go.mod sqlc version` prints `v1.31.1`; `go tool -modfile=tools/go.mod ent --help` and `... jet -h` run; `make -n gen` parses; `go build ./...` still passes (tools module not part of `./...`).

### Task 2: `bench.Verifier` types
- **ACTION**: Create `internal/bench/verify.go`.
- **IMPLEMENT**:
  ```go
  // JoinRow is one row of the JOIN scenario (orders JOIN customers), in the reference column order.
  type JoinRow struct {
  	OrderID, CustomerID int
  	Status              string
  	Total               int
  	CreatedAt           time.Time
  	Name, Email, City   string
  }
  // UnionRow is one row of the UNION scenario.
  type UnionRow struct{ Name, Email, City string }
  // Verifier is implemented by adapters so the correctness check can compare the values their
  // queries return. It is never called by the runner. A scenario the library cannot express
  // returns ErrUnsupported, as on Adapter.
  type Verifier interface {
  	ReadModel(ctx context.Context, id int) (Model, error)
  	ReadJoin(ctx context.Context, customerID int) ([]JoinRow, error)
  	ReadUnion(ctx context.Context, city string) ([]UnionRow, error)
  }
  ```
  Add sort helpers `SortJoinRows` (by OrderID) and `SortUnionRows` (by Name) so comparison is order-independent (queries have no ORDER BY: adding one would cost time the other libraries do not pay).
- **MIRROR**: `bench.go` doc-comment style; table-driven test with AAA for the sort helpers.
- **IMPORTS**: `context`, `slices`, `time`.
- **GOTCHA**: keep `Model` unchanged. No `ORDER BY` in any adapter query.
- **VALIDATE**: `go test -race -cover ./internal/bench` (stays ≥ 80%, was 100%).

### Task 3: Shared contract suite `internal/adaptertest`
- **ACTION**: Create `RunContract(t *testing.T, newAdapter func() bench.Adapter, unsupported ...bench.Scenario)`. It is a non-test file importing `testing` (like `httptest`), so adapter packages can call it.
- **IMPLEMENT**: Generalise `databasesql_test.go` lines 18-137: skip without `BENCH_DSN`; reseed via `runner.NewPGStore`; table of the 10 cases (insert/update/delete/select/join/union + missing-id update/delete 0, join unknown customer 0, union unknown city 0); select of a missing id must error; re-`Init` after `Close`; `Init` against an unreachable DSN errors; for each scenario in `unsupported` the method must return `errors.Is(err, bench.ErrUnsupported)` and every other scenario must work. Use `missingID = 99_999_999`.
- **MIRROR**: `TEST_STRUCTURE` above, with `// Arrange`, `// Act`, `// Assert` inside each subtest.
- **GOTCHA**: `t.Cleanup` reseeds so later packages see seeded data; callers must not run in parallel (`-p 1` in `make test-db` already). Call `t.Helper()` in helpers.
- **VALIDATE**: compiles with `go vet ./internal/adaptertest`; first real exercise is Task 5.

### Task 4: Reference `Verifier` on `databasesql`
- **ACTION**: Create `internal/adapters/databasesql/verify.go` (new file) implementing `ReadModel`, `ReadJoin`, `ReadUnion` with the same SQL constants and scans as `databasesql.go:79-135`, returning values. Add `var _ bench.Verifier = (*Adapter)(nil)`.
- **MIRROR**: `databasesql.go:79-135` (same constants `selectSQL`, `joinSQL`, `unionSQL`).
- **GOTCHA**: do not edit `databasesql.go`; reuse its unexported constants from the new file in the same package.
- **VALIDATE**: add an integration test in a new `verify_test.go`: `ReadModel(1)` equals `NewModel()` with `ID: 1`; `ReadJoin(7)` has 10 rows, all `CustomerID == 7`, `Name == "customer 7"`; `ReadUnion("city 3")` has 40 rows. `make test-db` for that package.

### Task 5: sqlx adapter
- **ACTION**: `go get github.com/jmoiron/sqlx@v1.4.0`; create `internal/adapters/sqlx/sqlx.go`.
- **IMPLEMENT**: `sqlxlib.NewDb(sqlDB, "pgx")`. Same SQL constants as the reference (explicit columns, `"right"` quoted). `Insert/Update/Delete`: `ExecContext` + `RowsAffected`. `Select`: `GetContext(ctx, &row, selectSQL, id)` into a struct with `db:"..."` tags (`db:"right"`, etc.). `Join`: `SelectContext` into `[]joinRow` (tags on all 8 columns), return `len`. `Union`: `SelectContext` into `[]unionRow`. Verifier methods call the same private query helpers and map rows to `bench` types.
- **MIRROR**: `INIT_CLOSE`, `ERROR_HANDLING`, `READ_SCANS_EVERY_COLUMN`; `Name()` = `"sqlx"`.
- **GOTCHA**: sqlx errors if a selected column has no destination field (that is why every column needs a tag and `SELECT *` must not be used). Use `GetContext`/`SelectContext`/`ExecContext` (context variants) so cancellation works in the runner's stuck-phase handling. Closing `sqlx.DB` closes the underlying `*sql.DB`.
- **VALIDATE**: `adaptertest.RunContract(t, New)` in `sqlx_test.go` (RED before implementation: all cases fail), then GREEN; `go test -race -cover ./internal/adapters/sqlx` ≥ 80%.

### Task 6: sqlc adapter and codegen
- **ACTION**: Create `internal/adapters/sqlc/sqlc.yaml`, `queries.sql`, run `make gen-sqlc`, write `sqlc.go`.
- **IMPLEMENT**: `sqlc.yaml`: `version: "2"`, engine `postgresql`, `schema: "../../../db/schema.sql"` (single source of truth; the leading `DROP TABLE IF EXISTS` is accepted by sqlc, verify), `queries: "queries.sql"`, `gen.go: {package: db, sql_package: "database/sql", out: "db"}`. Queries (explicit columns, no `SELECT *`): `CreateModel :execrows`, `UpdateModel :execrows` (params in schema order then id), `DeleteModel :execrows`, `GetModel :one`, `JoinOrders :many` (same SELECT/JOIN as the reference), `UnionByCity :many`. Adapter: `db.New(sqlDB)`; `Insert/Update/Delete` return the `int64` from `:execrows`; `Select` calls `GetModel`; `Join`/`Union` return `len(rows)`. Verifier maps `db.JoinOrdersRow`/`db.UnionByCityRow`/`db.Model` to bench types.
- **MIRROR**: `INIT_CLOSE`; upstream `bench/sqlc/sqlc.yaml` + `queries.sql` (attribution comment in the adapter).
- **GOTCHA**: upstream uses `pgx/v5` and a single `pgx.Conn` (not concurrency-safe, not our driver decision); we use `database/sql`. Generated types: `SERIAL`/`INTEGER` → `int32` (cast to `int`), `BIGINT` → `int64`, `TIMESTAMPTZ` → `time.Time`. `:execrows` is required: `:exec` discards the affected count and the runner checks it. A sqlc `:many` allocates its slice; that is part of the library.
- **VALIDATE**: `make gen-sqlc` leaves `git status` showing only new generated files; `RunContract(t, New)`; `go test -race -cover`.

### Task 7: gorm adapter
- **ACTION**: `go get gorm.io/gorm@v1.31.2 gorm.io/driver/postgres@v1.6.3`; create `internal/adapters/gorm/gorm.go`.
- **IMPLEMENT**: open `*sql.DB` as usual, then `gormlib.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gormlib.Config{SkipDefaultTransaction: true, PrepareStmt: false, Logger: logger.Default.LogMode(logger.Silent)})`. Structs with `TableName()`: `modelRow` → `models`, `customerRow` → `customers`, `orderRow` → `orders` with `Customer customerRow \`gorm:"foreignKey:CustomerID"\``. Operations:
  - Insert: `db.WithContext(ctx).Create(&row)`; return `int(res.RowsAffected)`.
  - Update: `db.WithContext(ctx).Model(&modelRow{}).Where("id = ?", id).Select("name","title","fax","web","age","right","counter").Updates(&row)`.
  - Delete: `db.WithContext(ctx).Delete(&modelRow{}, id)`.
  - Select: `db.WithContext(ctx).Take(&row, id)`.
  - Join: `db.WithContext(ctx).Joins("Customer").Where("orders.customer_id = ?", id).Find(&orders)`; return `len`.
  - Union: `return 0, bench.ErrUnsupported`; `ReadUnion` too.
  - Close: `sqlDB.Close()`; nil both fields.
- **MIRROR**: `INIT_CLOSE`, `ERROR_HANDLING`; upstream `bench/gorm.go` (`SkipDefaultTransaction`, silent logger).
- **GOTCHA** (these silently corrupt results):
  - Do **not** use `Save`: with a non-zero primary key gorm falls back to `INSERT ... ON CONFLICT` when the UPDATE matches nothing, which breaks "update of a missing id affects 0 rows" and grows the table.
  - `Updates(struct)` skips zero values, so `Right == false` would never be written. `Select(<columns>)` forces every column. The correctness check uses `Right: false` to catch this.
  - `First` adds `ORDER BY`; use `Take`.
  - `Delete(&modelRow{}, id)` with an int id is a primary-key condition; a missing id gives `RowsAffected == 0`, not an error.
  - `Joins("Customer")` aliases customer columns as `Customer__name`; gorm maps them back; `orders.customer_id` in `Where` is the table name, not the alias.
  - `SkipDefaultTransaction: true` is applied to every write (otherwise each write is wrapped in a transaction and the comparison is unfair); record this in the CLAUDE.md notes.
- **VALIDATE**: `RunContract(t, New, bench.ScenarioUnion)`; `go test -race -cover`.

### Task 8: ent adapter and codegen
- **ACTION**: `go get entgo.io/ent@v0.14.6`; write `internal/adapters/ent/schema/{model,customer,order}.go`; `make gen-ent`; write `ent.go`.
- **IMPLEMENT**: schema `Model`: fields `name,title,fax,web` (String), `age` (Int), `right` (Bool), `counter` (Int64); `entsql.Annotation{Table: "models"}`. `Customer`: `name,email,city`; table `customers`; `edge.To("orders", Order.Type)`. `Order`: `customer_id` (Int), `status` (String), `total` (Int), `created_at` (Time); table `orders`; `edge.From("customer", Customer.Type).Ref("orders").Field("customer_id").Unique().Required()`. Adapter: `entsql.OpenDB(dialect.Postgres, sqlDB)` + `entgen.NewClient(entgen.Driver(drv))`. Operations:
  - Insert: `client.Model.Create().SetName(...)...Exec(ctx)`; return 1.
  - Update: bulk form `client.Model.Update().Where(model.ID(id)).SetName(...)...Save(ctx)`; returns the affected count.
  - Delete: `client.Model.Delete().Where(model.ID(id)).Exec(ctx)`; returns the affected count.
  - Select: `client.Model.Get(ctx, id)` (missing → `NotFoundError`, an error as required).
  - Join: `client.Order.Query().Where(order.CustomerID(id)).WithCustomer().All(ctx)`; return `len`.
  - Union: `bench.ErrUnsupported`.
  - Close: `client.Close()` (closes the `*sql.DB`).
- **MIRROR**: `INIT_CLOSE`; upstream `bench/ent.go` + `bench/ent/schema/model.go`.
- **GOTCHA**:
  - `UpdateOneID(id)` and `DeleteOneID(id)` return `NotFound` instead of `0, nil` and `UpdateOne` issues a follow-up SELECT; use the bulk `Update()`/`Delete()` builders.
  - The schema package is for codegen only; the DB schema comes from `db/schema.sql`. Never call `client.Schema.Create`.
  - `ent generate` default target is the parent of the schema dir, hence the explicit `--target .../entgen`; the generated package is imported under alias `entgen`.
  - Generated code imports `ariga.io/atlas` (via `migrate`); `go mod tidy` will add it. Check `grep -E '^(go|toolchain)' go.mod` afterwards.
  - `WithCustomer()` runs a second query (`SELECT ... FROM customers WHERE id IN (...)`); this is the documented ent way and the reason the table above flags it.
- **VALIDATE**: `make gen-ent` produces only new files; `RunContract(t, New, bench.ScenarioUnion)`; `go test -race -cover ./internal/adapters/ent` (generated `entgen` is excluded from the coverage target; the adapter package itself must be ≥ 80%).

### Task 9: go-jet adapter and codegen
- **ACTION**: `go get github.com/go-jet/jet/v2@v2.16.1`; `make db-up gen-jet`; write `jet.go`.
- **IMPLEMENT**: generated `model` (`model.Models`, `model.Customers`, `model.Orders`) and `table` packages under `internal/adapters/jet/gen/bench/public/`. Adapter (dot-import `table`, `jetpg "github.com/go-jet/jet/v2/postgres"`):
  - Insert: `Models.INSERT(Models.MutableColumns).MODEL(row).ExecContext(ctx, db)` then `RowsAffected()`.
  - Update: `Models.UPDATE(Models.MutableColumns).MODEL(row).WHERE(Models.ID.EQ(jetpg.Int(int64(id)))).ExecContext(ctx, db)`.
  - Delete: `Models.DELETE().WHERE(Models.ID.EQ(jetpg.Int(int64(id)))).ExecContext(ctx, db)`.
  - Select: `SELECT(Models.AllColumns).FROM(Models).WHERE(...).QueryContext(ctx, db, &dest)` (`dest model.Models`); missing row returns `qrm.ErrNoRows` → error.
  - Join: `SELECT(Orders.AllColumns, Customers.Name, Customers.Email, Customers.City).FROM(Orders.INNER_JOIN(Customers, Customers.ID.EQ(Orders.CustomerID))).WHERE(Customers.ID.EQ(jetpg.Int(...)))` into a `[]struct{ model.Orders; Name, Email, City string }`-style destination, return `len`.
  - Union: `jetpg.UNION(selectCustomers, selectSuppliers)` (distinct `UNION`, matches the reference) into `[]struct{Name, Email, City string}`.
- **MIRROR**: `INIT_CLOSE`; upstream `bench/jet.go` (attribution), `bench/jet/gen.go` for the generator flags (ours: `-source=postgres -dsn`).
- **GOTCHA**: jet builds the statement on every call; that is the library's cost, do not cache statements. Generated model field names: `model.Models.Right`, `Counter int64`, `Age int32`. Destination struct field names must match jet's column mapping or rows silently come back zeroed: assert values in the test, not only counts. `Int(...)` binds parameters (`$n`), it does not inline literals; confirm by logging `stmt.Sql()` once during development, then remove the logging.
- **VALIDATE**: `RunContract(t, New)` (nothing unsupported); `go test -race -cover ./internal/adapters/jet` ≥ 80% (generated `gen/` excluded).

### Task 10: `internal/correctness`
- **ACTION**: Create `Check(ctx, store runner.Store, ref Reference, subjects []Subject) error` returning a joined error listing every mismatch (library, scenario, field, got, want), plus unit tests with in-memory fakes (no DB) and one DB integration test.
- **IMPLEMENT** (what it verifies, in order):
  1. **Golden spot checks on the reference** (guards against all libraries agreeing on a wrong value): `ReadModel(1) == NewModel()+ID 1`; `ReadJoin(7)`: 10 rows, all `CustomerID 7`, `Name "customer 7"`, `Email "customer7@example.com"`, `City "city 7"`, lowest order `{OrderID 6, Status "shipped", Total 7, CreatedAt 2026-01-01T00:06:00Z}`; `ReadUnion("city 3")`: 40 rows incl. `{"customer 3","customer3@example.com","city 3"}` and `{"supplier 3","supplier3@example.com","city 3"}`. Derive these from `db/seed.sql` formulas and confirm against the live DB while writing the test; comment which seed lines they mirror, "change both together".
  2. **Read agreement**: for each subject, `ReadModel` at ids `{1, 5000, 10000}`, `ReadJoin` at customers `{1, 7, 1000}`, `ReadUnion` at cities `{"city 0", "city 3", "city 49"}`, compared with the reference after `SortJoinRows`/`SortUnionRows`; times compared with `time.Time.Equal`. A `Verifier` returning `ErrUnsupported` must match the subject's `Adapter` also returning it (otherwise error).
  3. **Write agreement** (reseed before each subject, reseed again at the end): with a distinctive model `{Name "n", Title "t", Fax "f", Web "w", Age 7, Right false, Counter 42}`: `Insert` then reference `ReadModel(10001)` equals it; `Update(5, distinct)` then reference `ReadModel(5)` equals it; a row from `store.InsertDeletable` then `Delete` → reference `ReadModel(id)` errors and a second `Delete` returns 0.
- **MIRROR**: `TEST_STRUCTURE`; table-driven fakes for the compare logic (`Compare` helpers are pure and tested without a DB: equal, one field differs, row missing, extra row, order differs, unsupported on one side).
- **GOTCHA**: `Right: false` and distinct `Fax`/`Title` values are what catch wrong mappings; `NewModel()` alone cannot (its values equal the seed). The insert id `10001` relies on `Reseed` doing `RESTART IDENTITY` (`db/seed.sql:1`). A subject that does not implement `bench.Verifier` is an error (every registered adapter must). Do not import `cmd/bench` (cycle); `Check` takes adapters as arguments.
- **VALIDATE**: `go test -race -cover ./internal/correctness` ≥ 80% without DB; deliberately break one adapter mapping (e.g. swap `Title`/`Fax` in the sqlx Update) and confirm `Check` names it, then revert.

### Task 11: Wire into `cmd/bench`
- **ACTION**: Update `cmd/bench/main.go`; add `cmd/bench/correctness_test.go`.
- **IMPLEMENT**: register `"sqlc": sqlc.New, "sqlx": sqlx.New, "gorm": gorm.New, "ent": ent.New, "jet": jet.New` (update the registry comment). In `run()`, after `seedcheck.Check`/`hostInfo` and before `runner.Run`: build the reference (`databasesql.New()`, always, even if not in `-libs`) and subjects for `cfg.libs`, `Init` each with a small `PoolConfig{MaxOpen: 2, MaxIdle: 2}`, call `correctness.Check`, `Close` all, then `store.Reseed`; on failure return the joined mismatch error (exit 1). Extend `hostInfo` `Libraries` with `moduleVersion` for `github.com/jmoiron/sqlx`, `gorm.io/gorm`, `entgo.io/ent`, `github.com/go-jet/jet/v2` (sqlc is generated code with no runtime module; Phase 4 reads the tool version from `tools/go.mod`). Test: `TestRegistryCorrectnessIntegration` iterates `registry` and calls `correctness.Check`; `TestRegistryNamesMatchAdapters` (no DB) asserts `registry[k]().Name() == k`.
- **MIRROR**: `cmd/bench/main.go:67-113`.
- **GOTCHA**: `registryNames()` feeds `-libs` validation and defaults, so `flags_test.go` expectations on the default list may need updating (read `cmd/bench/flags_test.go` first; change only the expected list). Keep `run()` under 50 lines by extracting `preflight(ctx, dsn, db, libs) error`.
- **VALIDATE**: `go run ./cmd/bench -h` lists six libs; `make test-db`; `go run ./cmd/bench -libs sqlx -scenarios select -profiles low -low-duration 2s -repeats 1 -out results/smoke-p3.json` completes and the progress output shows the pre-flight.

### Task 12: Documentation
- **ACTION**: Update `CLAUDE.md` (commands: `make gen*`; layout: `tools/`, `internal/adapters/<lib>` generated dirs, `internal/adaptertest`, `internal/correctness`; contract: `Verifier` and the "no ORDER BY" rule; the native/`na` table from this plan; "Value-level cross-adapter correctness is not checked yet" line replaced), update the PRD Phase 3 row to complete with plan and report links when done, write `.claude/PRPs/reports/core-adapters-report.md`, move this plan to `plans/completed/`, run `graphify update .`.
- **GOTCHA**: CLAUDE.md "Decisions" already says "A scenario a library cannot express is 'n/a (raw SQL fallback)'"; the table refines it, do not contradict it.
- **VALIDATE**: `grep -n "not checked yet" CLAUDE.md` returns nothing.

### Task 13: Full validation and review
- **ACTION**: run everything in "Validation Commands"; then `ecc:go-reviewer` on the diff; fix CRITICAL/HIGH; list the rest in the report. Do not commit; the user decides.

---

## Testing Strategy

### Unit Tests (no DB)

| Test | Input | Expected | Edge? |
|---|---|---|---|
| `SortJoinRows`/`SortUnionRows` | shuffled rows | ordered by OrderID / Name | no |
| `Compare` model | equal; one field differs; `Right` differs | nil; mismatch naming the field | yes |
| `Compare` join/union | equal; missing row; extra row; different order; time with another zone, same instant | nil / error / error / nil / nil | yes |
| `Check` with a fake subject that swaps `Title`/`Fax` | fake Verifier | error naming library and scenario | yes |
| `Check` with a subject not implementing `Verifier` | plain `Adapter` | error | yes |
| `Check` with `ErrUnsupported` on both Adapter and Verifier | gorm-like fake | passes | yes |
| `Check` with `ErrUnsupported` on Verifier only | inconsistent fake | error | yes |
| `TestRegistryNamesMatchAdapters` | registry | `Name()` equals key for all six | no |

### Integration Tests (`BENCH_DSN`, named `...Integration`, `-p 1`)
- `adaptertest.RunContract` once per new adapter: the 10 call cases, missing-id select errors, unreachable `Init` errors, re-`Init` after `Close`, unsupported scenarios return `ErrUnsupported`.
- `databasesql` `Verifier` golden values.
- `correctness.Check` over the whole registry (in `cmd/bench`).

### Edge Cases Checklist
- [ ] UPDATE/DELETE of a missing id → 0 rows, no insert (gorm `Save`, ent `UpdateOneID` traps)
- [ ] `Right == false` is written by UPDATE (gorm zero-value skip)
- [ ] SELECT of a missing id → error
- [ ] JOIN unknown customer, UNION unknown city → 0
- [ ] `Init` after `Close`; `Init` on unreachable DB returns an error and leaks no pool
- [ ] Concurrent use under 100 workers (a pass of the stress profile on each library, Task 13)
- [ ] `ErrUnsupported` path produces an `na` cell in the matrix (already covered by `runner` tests with a fake; confirm one real run with gorm/ent UNION)
- [ ] Generated code reproducible (`make gen-check`)

---

## Validation Commands

### Static analysis
```bash
make build vet && gofmt -l . && grep -E '^(go|toolchain)' go.mod tools/go.mod
```
EXPECT: no output from `gofmt -l .`; only `go 1.27.3` lines, no `toolchain`.

### Unit tests
```bash
make test
```
EXPECT: all packages pass without a DB; integration tests skip.

### Integration tests
```bash
make test-db
go test -p 1 -race -cover -count=1 ./internal/... ./cmd/...
```
EXPECT: pass; each new hand-written package ≥ 80% coverage.

### Codegen reproducibility
```bash
make gen-check
```
EXPECT: exit 0, no diff, no untracked files in the generated directories.

### Smoke benchmark (every library, both profiles, short windows)
```bash
make bench BENCH_ARGS="-repeats 1 -low-duration 3s -stress-duration 3s -delete-pool 20000 -out results/smoke-p3.json"
go run ./cmd/benchcmp results/smoke-p3.json results/smoke-p3.json
```
EXPECT: pre-flight passes; 6 libraries × 6 scenarios × 2 profiles = 72 cells, each `ok` except `gorm/union` and `ent/union` (both profiles) = `na`; error rate 0 in every `ok` cell; `make db-check` OK afterwards. (benchcmp against itself is only a schema sanity check.)

### Manual validation
- [ ] Break one mapping on purpose, `make bench` aborts with a readable mismatch, revert.
- [ ] `git status` shows no stray files (`results/` is ignored).

---

## Acceptance Criteria
- [ ] All 13 tasks completed
- [ ] sqlc, sqlx, gorm, ent, jet registered; `-libs` accepts them
- [ ] Every cell is a number or an explicit `na`; the `na` set equals the table in "Decisions"
- [ ] `correctness.Check` passes for all registered adapters and fails on a deliberately broken mapping
- [ ] `make gen-check`, `make test`, `make test-db`, `make vet`, `gofmt -l .` clean; `go.mod` has no `toolchain` line
- [ ] No lib/pq in the main `go.mod`
- [ ] `go-reviewer` run, CRITICAL/HIGH fixed, rest listed in the report
- [ ] CLAUDE.md updated; report written; PRD row updated

## Completion Checklist
- [ ] New code follows the patterns above (package prefix in errors, `%w`, no magic numbers)
- [ ] Functions under 50 lines, files under 800 lines (generated code exempt)
- [ ] Tests are table-driven with `// Arrange`, `// Act`, `// Assert`
- [ ] Attribution comments kept where code derives from efectn/go-orm-benchmarks
- [ ] No debug output left in code
- [ ] `internal/runner`, `internal/result`, `databasesql.go` and `databasesql_test.go` untouched
- [ ] Unrelated findings listed at the end of the report, not fixed

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A library maps a column wrongly and still passes the row-count check | M | wrong benchmark | `Verifier` + distinctive write model + golden spot values (Task 10) |
| gorm `Save` / ent `UpdateOneID` semantics inflate or fail writes | H if used | wrong numbers | explicit gotchas in Tasks 7-8; contract suite covers missing-id update/delete |
| Dependency bump moves `go.mod` to a `toolchain` line (gorm driver needs Go 1.25, ent pulls atlas, x/tools) | M | breaks the native-toolchain rule | check `grep -E '^(go|toolchain)' go.mod` after every `go get`/`go mod tidy` |
| `go tool` builds of sqlc/ent/jet are slow or break on Go 1.27 later | L | codegen blocked | spike passed; tools pinned in `tools/go.mod`; generated code is committed so benchmarks never need the tools |
| sqlc rejects `schema.sql` because of `DROP TABLE IF EXISTS` | L | codegen blocked | fall back to a `-- sqlc:` filtered copy only if it fails; try the real file first (single source of truth) |
| Stress profile (100 workers) exposes a library bug the contract tests do not | M | aborted run | smoke benchmark in Task 13 covers all libraries under stress; compose `max_connections=300` already covers 100 + extras |
| Pre-flight adds time or leaves the DB unseeded on failure | L | confusing state | `Check` reseeds at the end and on failure paths (`defer`); runner reseeds per cell anyway |
| Phase 2 p95 instability makes cross-library comparisons noisy | H | Phase 5 concern | out of scope here; do not draw rankings from the smoke run |

## Notes
- Open question for the user (defaults chosen): ent JOIN as eager-load vs a real SQL JOIN via the `sql/modifier` feature; strictness of the "native" rule for gorm/ent UNION. Both only change the table and one adapter method.
- Suggested commit split when the user approves: (1) `chore: add tools module and gen targets`, (2) `feat: add correctness check and Verifier`, (3) one `feat:` per adapter including generated code, (4) `docs:`. Generated code should be reviewable per commit, not mixed.
- Unchanged by design: the runner measures `Select` etc. through the unchanged `Adapter`; `Verifier` is invoked only by `internal/correctness`.
