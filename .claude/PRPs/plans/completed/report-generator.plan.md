# Plan: Report generator (Phase 4)

## Summary
A `cmd/report` command and an `internal/report` package that turn one result JSON (schema v1) plus a hand-written `report/rubric.json` into a deterministic Markdown report: machine/config header, per-scenario tables for each profile with deltas versus `database/sql`, a generated top-5 with reasons, and per-library pros/cons. The cell-level facts that change how numbers must be read (ent JOIN is eager-loading, `na` cells, differing result handling in allocs/op) come from the rubric file and print next to the numbers.

## User Story
As the benchmark owner, I want `make report` to regenerate the comparison from a result file, so that the Phase 5 README is never hand-edited and a second run produces a diffable report.

## Problem → Solution
Results exist only as JSON (`results/*.json`, git-ignored) and `benchcmp` prints p95 deltas between two runs → one command renders tables, rankings, reasons and notes from one run, byte-identical for the same input.

## Metadata
- **Complexity**: Medium (stdlib only, no new dependency)
- **Source PRD**: `.claude/PRPs/prds/go-sql-benchmark.prd.md`
- **PRD Phase**: 4 — Report generator
- **Estimated Files**: about 11 new (`cmd/report/{main,main_test}.go`, `internal/report/{report,rank,table,reasons,rubric}.go` + tests, `report/rubric.json`, `internal/report/testdata/*`) + edits to `Makefile`, `CLAUDE.md`, the PRD row

---

## UX Design
N/A — internal tool. One new command.

### Interaction Changes
| Touchpoint | Before | After | Notes |
|---|---|---|---|
| `go run ./cmd/report -in results/x.json` | n/a | Markdown to stdout (or `-out file`) | never writes `README.md` unless asked; Phase 5 publishes it |
| `make report IN=results/x.json OUT=README.md` | n/a | wraps the command | same shape as `bench-cmp` |

---

## Mandatory Reading

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `internal/result/result.go` | 28-89, 211-227 | the schema; `Load` already validates and rejects unknown fields |
| P0 | `cmd/benchcmp/main.go` | all | CLI shape to mirror: `run(args, stdout, stderr) int`, exit codes, usage errors |
| P0 | `cmd/benchcmp/main_test.go` | 1-67 | table-driven CLI test with Arrange/Act/Assert |
| P1 | `internal/bench/bench.go` | 11-35 | `AllScenarios` is the report order; scenario names |
| P1 | `cmd/bench/main.go` | 41-60 | registry names and `libraryModules` (keys of `Host.Libraries`: pgx, sqlx, gorm, ent, jet) |
| P1 | `CLAUDE.md` | "Adapter notes" | the facts the report must carry |
| P2 | `.claude/PRPs/reports/core-adapters-report.md` | Follow-ups | smoke-run caveat, ent JOIN throughput observation |
| P2 | `results/smoke-p3.json` | all | real data for the fixture (git-ignored, local) |

## External Documentation
No external research needed — stdlib (`encoding/json`, `sort`, `strings`, `text/tabwriter` not needed, Markdown built with `strings.Builder`).

---

## Patterns to Mirror

### CLI_SHAPE
// SOURCE: cmd/benchcmp/main.go:21-58
```go
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run returns the process exit code: 0 within tolerance, 1 beyond it or on a missing cell, 2 on usage or load errors.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("benchcmp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	...
	a, err := result.Load(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
```

### ERROR_HANDLING
// SOURCE: internal/result/result.go:212-226
```go
data, err := os.ReadFile(path)
if err != nil {
	return Run{}, fmt.Errorf("result: read %s: %w", path, err)
}
dec := json.NewDecoder(bytes.NewReader(data))
dec.DisallowUnknownFields()
```
Errors are `report: <op>: %w`.

### CONSTANTS
// SOURCE: cmd/benchcmp/main.go:16-22 — named constants for exit codes and magic numbers (`percentDisplay = 100`).

### TEST_STRUCTURE
// SOURCE: cmd/benchcmp/main_test.go:31-67
```go
tests := []struct{ name string; args []string; wantCode int; wantOut string }{...}
for _, tt := range tests {
	t.Run(tt.name, func(t *testing.T) {
		// Arrange
		// Act
		// Assert
	})
}
```

### DATA_FILES
Hand-written data lives in a committed JSON file loaded with `DisallowUnknownFields` (same discipline as `result.Load`); tests use `t.TempDir()` and a `writeRun`-style helper.

---

## Design (decisions for this phase)

**Inputs**: one result file (`-in`), the rubric (`-rubric`, default `report/rubric.json`), output (`-out`, default stdout).

**Determinism**: no `time.Now`, no map iteration order, no environment. The date printed is `Run.StartedAt` (UTC). Libraries ordered by rank (see below) with name as the tie-break; scenarios in `bench.AllScenarios` order; profiles in `Config.Profiles` order. Cell order in the JSON must not matter (test shuffles).

**Completeness check** (PRD success metric): for every library x `AllScenarios` x config profile there must be a cell (`ok` or `na`). Missing cell → error naming it. A library in the run with no rubric entry → error. A run that does not contain `database_sql` renders without delta columns (shows `-`), not an error (`-libs` can subset).

**Per scenario and profile table** (one table each, 6 x 2): columns `Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread`.
- `na` row: `n/a (<reason>)`, excluded from ranking.
- `p95 vs database/sql`: `+12.3%` relative to the baseline cell's p95; baseline row shows `baseline`.
- `p95 spread`: `(max-min)/median` of the repeats' p95, `-` when `Repeats` is 1. This is the variance column the PRD asks for; a cell above 10% is marked `*` (the same 10% the stability criterion uses, as a named constant).
- cell notes (rubric) become footnote markers `[n]` on the library name and a "Notes" list under the table. This is how the ent JOIN note stays beside the numbers.

**Ranking** (PRD default, equal weights, weights read from the rubric so Phase 5 can change them without code):
- for each (scenario, profile) take the `ok` cells with `error_rate <= maxRankedErrorRate` (named constant 0.01; a cell above it is shown with `!` and excluded from ranking);
- rank each metric ascending (p95, B/op, CPU ns/op); rank = 1 + number of strictly better values (ties share a rank);
- a library's score = weighted mean of its ranks over every ranked metric of every ranked cell (lower is better); `na` and excluded cells simply do not contribute, and the table prints `ranked cells: n/m` so a library with `na` cells is visibly judged on fewer scenarios.
- top 5 = the five lowest scores, tie-break by name. With six core libraries that is 5 of 6; the sixth is listed as "also measured".

**Generated reasons** (from numbers only, no free text): per top-5 library, up to three sentences built from fixed templates: best-ranked count ("fastest p95 in 7 of 11 ranked cells"), weakest metric ("weakest on B/op: rank 5 of 6 on average"), and the single worst cell ("slowest cell: join/stress, rank 6"). Reasons print before the rubric pros/cons, as the PRD specifies.

**Rubric file `report/rubric.json`** (all hand-written content in one place):
```json
{
  "weights": {"p95": 1, "bytes_per_op": 1, "cpu_ns_per_op": 1},
  "methodology": ["<line>", "..."],
  "libraries": [{
    "name": "sqlx", "display": "sqlx", "module": "sqlx",
    "label": "no updates since 2024", "last_release": "v1.4.0 (checked 2026-10-21)",
    "pros": ["..."], "cons": ["..."]
  }],
  "cell_notes": [{"library": "ent", "scenario": "join", "note": "..."}]
}
```
`module` is the key into `Host.Libraries` (empty for `database_sql` and `sqlc`, which print "n/a (generated code)" / "standard library"). `last_release` is dated text because it is outside data the runner can know. `methodology` carries the shared caveats: pgx/stdlib driver for all, numbers are machine-specific, latency is service time not coordinated-omission-corrected, B/op and allocs/op include harness cost and each library's idiomatic result handling (sqlx/sqlc/gorm/ent/jet materialise rows, the reference scans into locals), `na` = needs the raw-SQL escape hatch.

**Alternatives rejected**: `text/template` (logic split from the code that already needs ranking; harder to unit test), YAML rubric (new dependency), a Markdown rubric (not machine-checkable), hard-coding notes in Go (they are data and change in Phase 5/6).

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| `internal/report/rubric.go` | CREATE | `Rubric` types, `LoadRubric`, validation (unknown fields, duplicate libraries, weights >= 0 and not all zero) |
| `internal/report/rank.go` | CREATE | ranking, scores, top-5 |
| `internal/report/reasons.go` | CREATE | reason sentences from ranks |
| `internal/report/table.go` | CREATE | per scenario/profile tables, formatting helpers (durations, deltas, `\|` escaping) |
| `internal/report/report.go` | CREATE | `Render(run, rubric) (string, error)`: completeness check, sections |
| `internal/report/*_test.go`, `testdata/fixture.json`, `testdata/rubric.json`, `testdata/report.golden.md` | CREATE | table-driven tests, golden file |
| `cmd/report/main.go`, `main_test.go` | CREATE | CLI |
| `report/rubric.json` | CREATE | the real rubric for the six core libraries |
| `report/rubric_test.go` (package `report_test`... see Task 8) | CREATE | the real rubric covers the registry and carries the required notes |
| `Makefile` | UPDATE | `report` target, `.PHONY` |
| `CLAUDE.md` | UPDATE | Commands + Layout lines |
| `.claude/PRPs/prds/go-sql-benchmark.prd.md` | UPDATE | Phase 4 row: in-progress then complete |

## NOT Building
- Writing or committing `README.md` (Phase 5, after the full run and the stability check).
- Charts or HTML output.
- Last-release lookup over the network; `last_release` is hand-entered and dated in the rubric.
- The 14 extra libraries (Phase 6); the generator is library-agnostic, only the rubric grows.
- Stability gating inside the generator (`benchcmp` owns it); the report only shows per-cell repeat spread.
- Resolving the open top-5 weighting question; the default equal weights stay, changeable in the rubric.

---

## Step-by-Step Tasks

### Task 1: Rubric types and loader (RED first)
- **ACTION**: tests then `internal/report/rubric.go`.
- **IMPLEMENT**: `Rubric{Weights Weights; Methodology []string; Libraries []LibraryInfo; CellNotes []CellNote}`; `LoadRubric(path)`; `Rubric.library(name)` and `Rubric.notesFor(library, scenario)` lookups; validation: unknown JSON field, duplicate library, empty name, negative or all-zero weights, note referencing an unknown scenario.
- **MIRROR**: ERROR_HANDLING, DATA_FILES.
- **GOTCHA**: `DisallowUnknownFields` so a typo in the rubric fails instead of silently dropping a note.
- **VALIDATE**: `go test -race ./internal/report -run Rubric` fails (RED) then passes.

### Task 2: Ranking
- **ACTION**: tests then `rank.go`.
- **IMPLEMENT**: `rankCells(run, weights) ranking` returning per-library `score`, `ranked`/`total` cell counts, per-metric mean rank, and per-cell ranks (needed by reasons); `top(n)`. Constants `maxRankedErrorRate`, `topCount = 5`.
- **GOTCHA**: ties share a rank; a library alone in a cell (others `na`) gets rank 1 and counts. Division by zero when a library has no ranked cell: score `+Inf`, sorted last, printed `-`.
- **VALIDATE**: table tests: strict order, ties, `na` skipped, error-rate exclusion, weights change the order, all-`na` library.

### Task 3: Reasons
- **ACTION**: tests then `reasons.go`.
- **IMPLEMENT**: `reasonsFor(lib, ranking) []string` with the three fixed templates; no sentence when its input does not exist (a library with no ranked cells gets none).
- **VALIDATE**: exact-string table tests.

### Task 4: Tables
- **ACTION**: tests then `table.go`.
- **IMPLEMENT**: `scenarioTable(run, scenario, profile, ranking, rubric) string`; formatters `fmtDuration` (ns → `µs`/`ms` with 2 significant digits via `time.Duration`), `fmtDelta`, `fmtSpread`, `escapeCell` (`|` and newlines). Spread uses `Repeats` p95 values; `*` above `spreadWarn = 0.10`.
- **GOTCHA**: the baseline's own delta; baseline missing; zero baseline p95 (print `-`, never divide by zero); footnote numbers are assigned per table in encounter order, deterministic.
- **VALIDATE**: table tests per formatter plus one rendered-table string test containing `na`, baseline, `!` and a footnote.

### Task 5: Render
- **ACTION**: tests then `report.go`.
- **IMPLEMENT**: `Render(run result.Run, rb Rubric) (string, error)`: completeness check, sections in order: title and date, machine (`Host`) and versions (each library's `Libraries[module]`, rubric label and `last_release`), run config (profiles in human units, repeats, seed), methodology (rubric lines), per-profile per-scenario tables, top 5 with reasons and "also measured", pros/cons per library (reasons first, then rubric pros/cons).
- **GOTCHA**: shuffled `Cells` must give identical output; no map ranging in output paths (sort keys).
- **VALIDATE**: golden test (`-update` flag rewrites `testdata/report.golden.md`, reviewed by eye), determinism test (render twice, and once with reversed cells), missing cell and missing rubric entry errors.

### Task 6: Fixture
- **ACTION**: create `internal/report/testdata/fixture.json` from the local smoke run, three libraries so it holds an `na` cell and a footnote:
  `jq '.cells |= map(select(.library=="database_sql" or .library=="ent" or .library=="sqlx")) | .host.libraries |= with_entries(select(.key=="pgx" or .key=="ent" or .key=="sqlx"))' results/smoke-p3.json > internal/report/testdata/fixture.json`
  Add `testdata/rubric.json` (three libraries, one ent/join note).
- **GOTCHA**: the fixture must pass `result.Load`; one test asserts that. Give one cell a two-repeat shape by a test-local edit (spread column), not by editing the JSON by hand.
- **VALIDATE**: `go test ./internal/report` loads it.

### Task 7: CLI
- **ACTION**: tests then `cmd/report/main.go`.
- **IMPLEMENT**: flags `-in` (required), `-rubric` (default `report/rubric.json`), `-out` (default stdout); `run(args, stdout, stderr) int`, exit 0 / 1 render error (incomplete run, missing rubric entry) / 2 usage and load errors.
- **MIRROR**: CLI_SHAPE.
- **VALIDATE**: table-driven `run` tests: ok to stdout, ok to file, missing `-in`, bad file, bad rubric, incomplete run → 1.

### Task 8: The real rubric
- **ACTION**: write `report/rubric.json` for `database_sql, ent, gorm, jet, sqlc, sqlx`; add a test that loads it.
- **IMPLEMENT**: pros/cons written from what this repo measured and the Phase 3 notes, no unverifiable claims (no star counts). `cell_notes` must include: ent/join (eager-loading, 2 queries), gorm/union and ent/union (needs raw SQL, `na`), gorm (SkipDefaultTransaction, explicit column `Updates`). `sqlx` label `no updates since 2024`. `methodology` as listed in Design.
- **GOTCHA**: package placement — a test under `report/` would add a Go package for one file. Put the test in `cmd/report/main_test.go` instead (`TestRealRubric`, path `../../report/rubric.json`) and keep `report/` data-only.
- **VALIDATE**: test asserts every name in `cmd/bench` registry has a rubric entry and the ent/join note exists. The registry lives in `package main` of another command, so the test lists the six names as a literal and a comment says to update it with `registry` (Phase 6 will extend both).

### Task 9: Wire up and docs
- **ACTION**: `Makefile` target `report: go run ./cmd/report -in $(IN) $(if $(OUT),-out $(OUT))`, add to `.PHONY`; `CLAUDE.md` Commands (`make report`) and Layout (`internal/report`, `cmd/report`, `report/rubric.json`); PRD Phase 4 row → complete with plan and report links.
- **VALIDATE**: `make -n report IN=x.json` prints the command; `make report IN=results/smoke-p3.json` renders all six scenarios x two profiles with the ent JOIN note and four `na` rows.

### Task 10: Review and graph
- **ACTION**: `ecc:go-reviewer`, fix CRITICAL/HIGH, `graphify update .` then `/graphify . --update` for the plan and report docs. Do not commit until asked.

---

## Testing Strategy

### Unit Tests
| Test | Input | Expected | Edge? |
|---|---|---|---|
| ranking order | 3 libs, distinct metrics | ascending scores | |
| ties | equal p95 | shared rank | yes |
| `na` skipped | ent union `na` | counted in neither numerator nor denominator | yes |
| error-rate exclusion | cell at 2% errors | `!`, not ranked | yes |
| weights | p95 weight only | order follows p95 | |
| delta | baseline 1ms, lib 1.5ms | `+50.0%` | |
| zero baseline | baseline p95 0 | `-` | yes |
| spread | repeats 1.0/1.2/1.1 ms | `~18%`, `*` | yes |
| missing cell | drop one cell | error naming `lib/scenario/profile` | yes |
| missing rubric entry | unknown library | error | yes |
| determinism | reversed `Cells` | identical output | yes |
| markdown escaping | note containing `\|` | escaped | yes |
| golden | fixture + test rubric | matches `report.golden.md` | |
| real rubric | `report/rubric.json` | six libs, ent/join note, sqlx label | |

### Edge Cases Checklist
- [ ] Run without `database_sql` (no delta column values)
- [ ] Single repeat (spread `-`)
- [ ] Library with every cell `na`
- [ ] Empty `pros`/`cons`
- [ ] Rubric typo (unknown field) rejected
- [ ] `-out` into a missing directory (create parents, like `result.Write`)

---

## Validation Commands

### Static analysis
```bash
go vet ./... && gofmt -l . && grep -E '^(go|toolchain)' go.mod
```
EXPECT: no output from vet and gofmt; only `go 1.27.3`.

### Unit tests
```bash
go test -race -cover ./internal/report ./cmd/report
```
EXPECT: pass, coverage at least 80% each.

### Full suite (no DB needed for this phase)
```bash
make test
```
EXPECT: no regressions.

### Manual
- [ ] `make report IN=results/smoke-p3.json` — tables for 6 scenarios x 2 profiles, ent JOIN footnote under the JOIN tables, 4 `na` rows, top 5 with reasons, sqlx label
- [ ] run twice and `diff` the outputs: empty
- [ ] `make report IN=results/a.json` (an older single-library run) renders with only `database_sql`; every row shows `baseline`

---

## Acceptance Criteria
- [ ] README-shaped Markdown is generated from the fixture JSON, byte-identical across runs and cell orders
- [ ] Every library x scenario x profile is a number or an explicit `n/a`; a missing cell is an error
- [ ] ent JOIN eager-loading note, `na` exclusion, sqlx label, library versions, host info and the allocs/op caveat are all in the output
- [ ] Top 5 reasons are generated from numbers and come before the pros/cons
- [ ] Tests written first, `-race`, coverage >= 80% for `internal/report` and `cmd/report`
- [ ] No new dependency; `go.mod` toolchain line unchanged

## Completion Checklist
- [ ] Errors wrapped `report: op: %w`, no magic numbers, functions under 50 lines, files under 800 lines
- [ ] Tests table-driven with Arrange/Act/Assert comments
- [ ] go-reviewer run; CRITICAL/HIGH fixed, rest listed
- [ ] graph refreshed; nothing committed without being asked

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Composite score favours a library with `na` cells (judged on fewer scenarios) | M | misleading top 5 | print `ranked cells: n/m`; reasons mention it; revisit the rule in Phase 5 with the weights question |
| Hand-written rubric claims go stale | M | wrong pros/cons | claims limited to repo-measured facts; `last_release` is dated; Phase 5 re-checks |
| Smoke data (1 repeat, 3 s) read as results | M | wrong conclusions | the report prints repeats and window length in the config section; fixture is test data only |
| Golden file churn hides real diffs | L | missed regressions | `-update` flag is explicit; golden reviewed in the diff |
| Float formatting differs across platforms | L | non-deterministic output | format with fixed `strconv` precision, no `%v` on floats |

## Notes
- Phase 2 p95 instability (9 of 12 cells over 10% between two runs) is not fixed here; the spread column makes it visible in the report, and Phase 5 must not publish before it is resolved.
- Task order is RED then GREEN per task; run `go test -race ./internal/report` after each.
