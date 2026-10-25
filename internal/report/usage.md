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
5. Add an entry for it in `report/rubric.json`: display name, `url` (its https home page, linked in the report), pros and cons, flexibility, ease of use, learning curve and community facts. The report refuses to render if a library in the run has no entry.
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
