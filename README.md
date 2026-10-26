# Go SQL library benchmark

> **Read this first.** These results are not final. They come from one machine and one environment (listed under Environment below); another machine, operating system, Docker setup, Go version or library version, or even a busy moment on the same machine, can give different numbers, and this document may be updated after it was generated. Treat it as a brief summary of how the benchmark works and what it showed in the author's working environment, not as a verdict on any library. Run the benchmark yourself before relying on it; see "Run it yourself" below.

Run started 2026-10-24T04:41:14Z (UTC). Numbers are only comparable on the same machine.

Compares 15 libraries on three measures: **latency** (p95, the time within which 95% of calls finished), **memory** (B/op, bytes allocated per operation) and **CPU** (CPU ns/op, CPU time per operation). Each library runs 6 scenarios under 2 load profiles; a **case** is one scenario under one load profile, written scenario/profile, for example join/low. Union is measured but not scored (see Union below); the other 5 scenarios, 10 cases, make up every score.

This is the summary. The per-case tables, pros and cons, and the qualitative sections (flexibility, ease of use, learning curve, type safety, community, a separate qualitative ranking and a combined ranking) are in the [detailed report](REPORT.md).

## How to read the numbers

Every library does the same small database jobs (add a row, read a row, and so on). For each job we measure three things, and lower is better for all three:

- **p95 (latency)**: how long one call takes, counting the slow ones. 95 out of 100 calls were at least this fast, so it shows what a user occasionally waits, which an average would hide.
- **B/op (memory)**: bytes of memory the program asked for on each call ("op" is one call). More memory means more work later for Go's garbage collector, the part that frees memory.
- **CPU ns/op (processor time)**: nanoseconds of processor time spent on each call. One nanosecond is a billionth of a second.

Each number is compared with plain `database/sql`, Go's built-in way to talk to a SQL database, which is the baseline. That is what the `x` figures mean: `1.00x` is the same as the baseline, `1.04x` is 4% more, `0.50x` is half.

So "p95 1.04x, B/op 1.43x, CPU ns/op 1.03x" reads: this library's calls take 4% longer, use 43% more memory and use 3% more processor time than `database/sql`.

A library's **score** combines its three ratios over every scored case into one number (see the [glossary](#glossary) for how). `1.00` is the same as `database/sql`, `2.00` is twice as costly, `0.50` half. Lower is better. Other terms, such as scenario, load profile and ORM, are in the [glossary](#glossary).

## Top 5

Score is the weighted geometric mean, over every ranked case, of each metric's ratio to database/sql (p95 1, B/op 1, CPU ns/op 1). 1.00 means the same as database/sql, 2.00 twice as much, 0.50 half; lower is better. Union is not scored, and a case needs a measured database/sql result to be scored.

Libraries with equal scores share a rank.

Only libraries that are not tied to one database are ranked; the others are under Honorable mention.

| Rank | Library | Score | Ranked cases |
|---:|---|---:|---:|
| 1 | database/sql | 1.00 | 10/10 |
| 2 | sqlx | 1.09 | 10/10 |
| 3 | sqlc | 1.10 | 10/10 |
| 4 | sqlboiler | 1.25 | 10/10 |
| 5 | beego | 1.31 | 10/10 |

**database/sql**

- the baseline: every ratio is 1.00 by definition

**sqlx**

- ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 1.01x, B/op 1.25x, CPU ns/op 1.03x
- slowest case: delete/low (p95 1.20x database/sql)

**sqlc**

- ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 1.04x, B/op 1.25x, CPU ns/op 1.01x
- slowest case: update/low (p95 1.35x database/sql)

**sqlboiler**

- ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 1.03x, B/op 1.74x, CPU ns/op 1.09x
- slowest case: join/low (p95 1.17x database/sql)

**beego**

- ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 0.98x, B/op 2.36x, CPU ns/op 0.97x
- slowest case: join/low (p95 1.18x database/sql)

Also measured:

- rel: score 1.49, ranked cases 10/10
- pop: score 1.73, ranked cases 10/10
- ent: score 1.79, ranked cases 10/10
- go-jet: score 1.91, ranked cases 10/10
- xorm: score 1.99, ranked cases 10/10
- gorm: score 2.02, ranked cases 10/10
- bun: score 2.16, ranked cases 10/10
- gorm/gen: score 2.25, ranked cases 10/10

## Honorable mention

Measured and shown in every table of the detailed report, but kept out of the ranking above. Score is computed the same way as in the Top 5, so it is comparable; lower is better.

| Library | Score | Ranked cases | Why not ranked |
|---|---:|---:|---|
| pgx_pool | 0.85 | 10/10 | pgx is a PostgreSQL-only driver API, so code written against it cannot move to another database. |
| pgx | 0.91 | 10/10 | pgx is a PostgreSQL-only driver API, so code written against it cannot move to another database. |

## Feature matrix

Which scenarios each library expresses with its own query API. `n/a` means it needs hand-written SQL for that scenario.

| Library | insert | update | delete | select | join | union |
|---|---|---|---|---|---|---|
| pgx_pool | native | native | native | native | native | native |
| pgx | native | native | native | native | native | native |
| database/sql | native | native | native | native | native | native |
| sqlx | native | native | native | native | native | native |
| sqlc | native | native | native | native | native | native |
| sqlboiler | native | native | native | native | native | n/a |
| beego | native | native | native | native | native [1] | n/a |
| rel | native | native | native | native | native | n/a |
| pop | native | native | native | native | native | n/a |
| ent | native | native | native | native | native [2] | n/a |
| go-jet | native | native | native | native | native | native |
| xorm | native | native | native | native | native | native [3] |
| gorm | native | native | native | native | native | n/a |
| bun | native | native | native | native | native | native |
| gorm/gen | native | native | native | native | native | n/a |

1. beego JOIN is RelatedSel on a foreign-key field (one SQL statement, customer loaded with the order).
2. ent JOIN is edge eager-loading (WithCustomer): two queries, orders then their customers, not one SQL JOIN. Read its JOIN numbers with that in mind.
3. xorm UNION is xorm.io/builder's Union (the query builder xorm itself uses) passed to SQL, so it is measured, not n/a.

## Union

UNION is measured for every library that can express it natively but is not part of any score above. This ranking uses the same ratio to database/sql, over the union cases only (every load profile).

| Rank | Library | Score | Ranked cases | p95 | B/op | CPU ns/op |
|---:|---|---:|---:|---:|---:|---:|
| 1 | database/sql | 1.00 | 2/2 | 1.00x | 1.00x | 1.00x |
| 2 | sqlc | 1.48 | 2/2 | 1.05x | 2.76x | 1.12x |
| 3 | sqlx | 1.55 | 2/2 | 1.05x | 2.99x | 1.20x |
| 4 | bun | 1.84 | 2/2 | 1.18x | 3.83x | 1.37x |
| 5 | xorm | 2.48 | 2/2 | 1.23x | 7.43x | 1.68x |
| 6 | go-jet | 2.94 | 2/2 | 1.42x | 9.17x | 1.95x |
| - | pgx_pool | 0.87 | 2/2 | 1.03x | 0.87x | 0.73x |
| - | pgx | 1.42 | 2/2 | 1.22x | 2.76x | 0.86x |

No ranked union result: beego, ent, gorm, gorm/gen, pop, rel, sqlboiler.

## Run it yourself

You need macOS or Linux, a native Go 1.27.3 toolchain for your CPU (not under emulation), Docker with Compose, and GNU Make. The stress profile runs 100 workers against Postgres, so give Docker a few CPUs and a few GB of memory.

```sh
git clone https://github.com/restuu/go-sql-benchmark && cd go-sql-benchmark
make arch-check      # fails if Go's architecture differs from the host's: emulated runs are invalid
make bench           # starts Postgres 15 in Docker, runs every library, writes results/run-<UTC>.json
make report IN=results/run-<UTC>.json OUT=results/summary.md   # writes this summary plus results/REPORT.md (the detailed report)
```

- `results/` is git-ignored, so your own runs and reports never touch the committed `README.md` and `REPORT.md`. `OUT` and the detailed report must be different files in one directory; `DETAILS=<path>` sets the second one.
- A full run takes about 80 minutes with the defaults. Close heavy applications and stop unrelated Docker containers first, and keep the machine awake and otherwise idle; numbers are only comparable on the same machine.
- For a quick smoke run of two libraries, `make bench BENCH_ARGS="-libs database_sql,sqlx -repeats 1 -low-duration 2s -stress-duration 3s"`. `go run ./cmd/bench -h` lists every flag. Keep all scenarios in a run you want to render, because the report needs every scenario of every library in it.
- `make bench-cmp A=results/a.json B=results/b.json` compares two runs: per-case p95 deviation plus a check that the Top 5 libraries and their order match.
- The database listens on host port 55432 (override with `BENCH_PG_PORT`, and the password with `BENCH_PG_PASSWORD`). `make db-down` and `make db-reset` discard its data (also needed before a new `BENCH_PG_PASSWORD` takes effect); each run reseeds it.

### Add your own library

Fork the repository (keep the `LICENSE` notice) and add the library as one more adapter:

1. Create `internal/adapters/<name>/` with a type that implements `bench.Adapter` (`Name`, `Init`, `Close`, `Insert`, `Update`, `Delete`, `Select`, `Join`, `Union`; see `internal/bench/bench.go`) and `bench.Verifier` (`ReadModel`, `ReadJoin`, `ReadUnion`). The existing adapters are the templates: `sqlx` for a SQL-first library, `gorm` for an ORM, `sqlc` or `ent` for generated code.
2. Follow the adapter contract. Each method runs one scenario call and returns the rows affected or scanned (1, 1, 1, 1, 10 and 40 for insert, update, delete, select, join and union). Reads must scan every selected column, and there is no explicit `Prepare`. UPDATE or DELETE of a missing id affects 0 rows, SELECT of a missing id is an error, and UPDATE writes every column including zero values. `Init` must work again after `Close`. Return `bench.ErrUnsupported` for a scenario the library cannot express with its own query API; that case shows as "n/a" and is left out of the ranking. Apply the `bench.PoolConfig` that `Init` receives (`MaxOpen`, `MaxIdle`). Libraries on `database/sql` use `sql.Open("pgx", dsn)` so they share one driver; run `go get` for the library and check that `go.mod` still has no `toolchain` line.
3. Add the contract tests by copying `internal/adapters/sqlx/sqlx_test.go` (`adaptertest.RunContract` and `adaptertest.RunVerifier`). Keep `Integration` in the test names, because `make test-db` only runs those, and pass the scenarios your adapter leaves unsupported as extra arguments, as `internal/adapters/beego/beego_test.go` does for `bench.ScenarioUnion`.
4. Register the constructor in `registry` in `cmd/bench/main.go`, and in `libraryModules` if the library is a Go module whose version should appear in the report.
5. Add an entry for it in `report/rubric.json`: display name, `url` (its https home page, linked in the report), pros and cons, flexibility, ease of use, type safety, learning curve and community facts. The report refuses to render if a library in the run has no entry.
6. Check it with `make vet test`, then `make test-db` (runs the contract tests and a value-level correctness check against `database/sql` before every benchmark), then run `database_sql` plus your library and render the report: `make bench BENCH_ARGS="-libs database_sql,<name>"`. Scores are ratios to `database_sql`, so keep it in the run.

### Using an AI coding assistant

If you would rather have an AI coding assistant do the work, open the repository in it and paste one of these prompts.

To run the benchmark and summarise it:

```text
Run the benchmark in this repository on my machine and summarise the result. Run `make arch-check`
first and stop if it fails. Before the measured run, ask me to close heavy applications and stop
unrelated Docker containers. Then run `make bench` in the background (about 80 minutes), render the
report with `make report IN=<the new results file> OUT=results/summary.md`, and tell me the Top 5, the honorable mentions, and any case whose p95
spread is marked with `*`. Do not edit any code.
```

To add a library you want to compare:

```text
Add the Go SQL library <module path> to this benchmark. Follow "Add your own library" in README.md:
write the adapter in internal/adapters/<name>/ the way the existing adapters do (sqlx is the
simplest), return bench.ErrUnsupported for scenarios the library cannot express with its own API,
add the contract tests, register it in cmd/bench/main.go, and add an honest entry in
report/rubric.json with community facts read from the library's repository. Run `make vet test` and
`make test-db`, then a short run of database_sql plus the new library, and show me the report.
Ask me before making any judgement call about a scenario.
```

To score the qualitative measures your own way (different weights, level values or community formula):

```text
Change how the qualitative ranking is scored in this repository. The weights, the level-to-number
scales and the community formula are constants at the top of internal/report/qualscore.go, and the
levels themselves are in report/rubric.json. I want: <describe your weights, scales or formula, for
example "ease of use counts double" or "include stars in the community score">. Update the
constants or the formula, update internal/report/qualscore_test.go, and make sure the explanation
the report prints under "Qualitative ranking" still matches the new rules. Refresh the golden files
with `go test ./internal/report -run TestRenderGolden -update`, run `make vet test`, regenerate the
reports with `make report IN=<the results file> OUT=results/summary.md`, and show me the ranking
before and after. Do not change any level in report/rubric.json unless I ask; those are judgements.
```

## Environment

- OS/arch: darwin/arm64
- Go: go1.27.3
- CPUs: 8 (GOMAXPROCS 8)
- PostgreSQL: 15.19
- Driver shared by every database/sql-based library: pgx v5.11.0

| Library | Version | Label | Version used | Link |
|---|---|---|---|---|
| pgx_pool | v5.11.0 | - | v5.11.0 | <https://github.com/jackc/pgx> |
| pgx | v5.11.0 | - | v5.11.0 | <https://github.com/jackc/pgx> |
| database/sql | - | - | standard library | <https://pkg.go.dev/database/sql> |
| sqlx | v1.4.0 | no updates since 2024 | v1.4.0 | <https://github.com/jmoiron/sqlx> |
| sqlc | - | - | generator v1.31.1, pinned in tools/go.mod | <https://github.com/sqlc-dev/sqlc> |
| sqlboiler | v4.19.7 | maintenance mode | v4.19.7 | <https://github.com/aarondl/sqlboiler> |
| beego | v2.3.10 | - | v2.3.10 | <https://github.com/beego/beego> |
| rel | v0.43.0 | - | v0.43.0 | <https://github.com/go-rel/rel> |
| pop | v6.4.1 | - | v6.4.1 | <https://github.com/gobuffalo/pop> |
| ent | v0.14.6 | - | v0.14.6 | <https://github.com/ent/ent> |
| go-jet | v2.16.1 | - | v2.16.1 | <https://github.com/go-jet/jet> |
| xorm | v1.4.3 | - | v1.4.3 | <https://gitea.com/xorm/xorm> |
| gorm | v1.31.2 | - | v1.31.2 | <https://github.com/go-gorm/gorm> |
| bun | v1.3.0 | - | v1.3.0 | <https://github.com/uptrace/bun> |
| gorm/gen | v0.3.29 | - | v0.3.29 | <https://github.com/go-gorm/gen> |

## Run configuration

Repeats per case: 3 (the median is reported). Seed: 1792816868404869000.

| Profile | Workers | Pacing | Warm-up | Window | DELETE ids |
|---|---:|---:|---:|---:|---:|
| low | 4 | 10ms | 1s | 5s | 5000 |
| stress | 100 | unpaced | 2s | 10s | 400000 |

## Methodology

- Every database/sql-based library uses the same pgx/stdlib driver (pgx), never lib/pq.
- pgx and pgx_pool bypass database/sql and use pgxpool with the pool sized like every other adapter. pgx collects rows into structs with its CollectRows/RowToStructByName helpers; pgx_pool scans into locals like the database/sql reference, so it shows the driver floor. Both use pgx's default statement cache, never an explicit Prepare.
- The ranking and the Top 5 cover database-agnostic libraries. pgx and pgx_pool are PostgreSQL-only, so they are honorable mentions: they are measured and shown in every table, but they take no rank from the other libraries, and their score is computed as if they had joined the ranking.
- A case is one scenario (insert, update, delete, select, join or union) under one load profile (low or stress), written scenario/profile, for example join/low. Every library is run through the same cases. Each case is measured several times and the median is what is reported and ranked, on three measures: latency (p95), memory (B/op, bytes allocated per operation) and CPU (CPU ns/op). A case the library cannot express is n/a. Union is measured and reported on its own but is not part of the score, because some libraries cannot express it without hand-written SQL (see the feature matrix) and a score over different cases would not be comparable.
- pop cannot take an existing *sql.DB, so it opens its own pool through its PostgreSQL dialect, which uses the same pgx stdlib driver; the pool is sized like every other adapter.
- Numbers are only comparable on the machine and run they were measured on; the host is listed above.
- Each case is the median of the repeats listed in the run configuration. Before the matrix runs, every adapter is checked against the database/sql reference on returned values, and every read scans every selected column.
- Latency is the per-call service time of successful operations with exact nearest-rank percentiles; it is not corrected for coordinated omission.
- B/op, allocs/op and CPU ns/op are deltas over the measured window divided by ops and include harness and GC cost.
- B/op and allocs/op also include each library's idiomatic result handling: sqlx, sqlc, gorm, ent and jet materialise rows, while the database/sql reference scans into locals, so part of the difference is the cost of that handling.
- A scenario counts as supported only when the library's own query API (or, for the SQL-first sqlc and sqlx, its SQL) expresses it; needing the raw-SQL escape hatch makes the case n/a.
- A library's score is the weighted geometric mean, over its ranked cases, of each metric divided by database/sql's value in the same case: 1.00 is the same as database/sql, 2.00 twice as much, 0.50 half, and lower is better. A ratio keeps the size of a difference, so a 1% gap moves the score by about 1% instead of a whole place, and a library's score does not depend on which other libraries were run. The scored cases are insert, update, delete, select and join under both load profiles, the same for every library; a library with error-excluded cases is judged on fewer, and the Ranked cases column shows how many.
- Ease of use, type safety and learning curve are the author's judgement from writing and reviewing these adapters and reading each library's documentation, not measurements, and they are not ranked. Community facts (stars, last release, last commit, open issues) were read from the forge APIs on the date shown in that section; a release date is the date the latest release was published, or the tag commit date where the notes say so.
- Stress-profile p99 on macOS includes a constant 100-200 ms host-to-VM stall (Postgres runs in an OrbStack VM) and is not a property of the library.
- Run-to-run stability (two full 15-library runs on this machine, 2026-10-24; the earlier 6-library pair had 62 of 72 cases within 20%, all 10 beyond at the low profile, and all 36 stress cases within 8.7%): p95 differed by 20% or less in 130 of 166 cases. The other 36 are 35 low-profile cases, which reached up to 114% (pgx select), and one stress case (pop insert, 27%); the remaining 82 stress cases stayed under 18%. The top 5 had the same libraries in the same order in both runs, and the five scores moved by 0.05 or less (beego most, 1.31 to 1.36), but sqlc and sqlx are 0.01 apart in both runs, so their order is within noise. The cause is environmental (loaded macOS host, host-VM network crossing, WAL fsync on the VM disk).

## Glossary

Plain-language meanings of the terms used in this report. `REPORT.md` uses the same terms.

### What is measured

- **latency**: the time one call to the database takes, from asking until the answer is back.
- **percentile**: sort all the measured times from fastest to slowest and look at one position. **p50** is the middle one (half the calls were faster), **p95** is the time 95 out of 100 calls beat, and **p99** is the time 99 out of 100 beat, so it reflects the rare slow calls.
- **p50**: the median latency: half the calls were faster, half slower.
- **p95**: the main latency figure in this report. Unlike an average, a few slow calls show up in it.
- **p99**: see percentile. Stress-profile p99 on macOS also includes a constant 100 to 200 ms stall between the host and the Docker virtual machine, which is not the library's doing.
- **throughput** (**ops/s**): how many calls finish per second, added up over all workers. Higher is better; it is shown for context and is not part of the score.
- **B/op**: bytes of memory allocated per operation (one call).
- **allocs/op**: how many separate memory allocations one call makes. Fewer is usually cheaper.
- **CPU ns/op**: processor time per call, in nanoseconds (billionths of a second).
- **garbage collector**: the part of Go that frees memory the program no longer uses. The more a call allocates, the more work the garbage collector has to do, and that work is included in the CPU figure.
- **ratio**: a measurement divided by the same measurement for `database/sql` in the same case. `1.04x` means 4% more than the baseline; `0.50x` means half.
- **baseline**: `database/sql`, the reference every other library is compared against.
- **geometric mean**: a way of averaging ratios. Twice as slow in one case and twice as fast in another cancel out to 1.00, which an ordinary average would not give.
- **score**: for each library, the weighted geometric mean of its p95, B/op and CPU ns/op ratios over all the cases it was judged on (the weights are listed under Top 5). `1.00` equals `database/sql`; lower is better. It never depends on the other libraries in the run.
- **median**: the middle value. Each case is run several times and the middle result is reported, so one unlucky run does not decide it.
- **spread** (**p95 spread**): how far apart the p95 of the repeats of a case were, as a share of the median. A `*` next to a figure means the repeats disagreed by more than 10%, so read it as approximate.
- **p95 vs database/sql**: the library's p95 compared with `database/sql`'s p95 in the same table, as a percentage. `+20.0%` means 20% slower than `database/sql`, `-20.0%` means 20% faster.
- **error rate** (**err %**): the share of calls that failed, as a percentage. A `!` next to a library marks a case above 1%; it is shown but not ranked.
- **n/a**: the library cannot do that scenario with its own query features (only with hand-written SQL), so the case is left out of the ranking.

### What the benchmark does

- **scenario**: one kind of database job. There are six, listed next.
- **insert**: add one new row to a table.
- **update**: change every column of one existing row.
- **delete**: remove one row.
- **select**: read one row by its id.
- **join**: read the orders of one customer together with that customer's details, which live in two tables (10 rows).
- **union**: combine two lists, the customers and the suppliers of one city, into one result (40 rows). It is reported separately and is not part of the score, because some libraries cannot express it without hand-written SQL.
- **load profile**: how hard the database is pushed. **low** is a quiet application: 4 callers, each pausing 10 ms between calls. **stress** is a busy one: 100 callers hitting the database without pausing.
- **low**: the quiet load profile (4 workers, 10 ms between calls).
- **stress**: the busy load profile (100 workers, no pauses).
- **worker**: one caller running calls one after another. More workers means more calls at the same moment.
- **warm-up**: the first seconds of each run, which are thrown away so start-up effects do not count.
- **measured window**: the seconds after the warm-up during which the numbers are recorded.
- **case**: one scenario under one load profile, written scenario/profile, for example join/low.
- **connection pool**: a set of ready database connections that callers share, so each call does not open its own.

### Kinds of library

- **database/sql**: Go's standard package for SQL databases. You write the SQL yourself.
- **driver**: the piece that speaks to the database. Every `database/sql`-based library here uses the same one (`pgx`), so only the library on top differs.
- **SQL-first**: you write plain SQL and the library helps run it and read the rows back.
- **query builder**: you build the query with Go code (functions and method calls) instead of writing SQL text.
- **ORM** (object-relational mapper): you work with Go structs and the library writes the SQL for you.
- **code generation**: a tool reads your database or SQL and writes Go code for you ahead of time (`make gen` regenerates it here).
- **PostgreSQL**: the database used for every measurement (version 15).
- **honorable mention**: a library that is measured and shown but not ranked against the others, because it only works with PostgreSQL (`pgx`, `pgx_pool`), so code written for it cannot move to another database.

### Qualitative terms (judgements, not measurements)

- **dynamic query**: a query whose shape changes at run time, for example a search that adds a filter only when the user fills in that field.
- **raw SQL**: SQL written as plain text inside the Go code. A library that needs it for something has no built-in way to do that.
- **type safety**: how many mistakes the Go compiler can catch before the program runs, such as a misspelled column name or a wrong type.
