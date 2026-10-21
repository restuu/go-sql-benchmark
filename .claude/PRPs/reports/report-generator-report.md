# Implementation Report: Report generator (Phase 4)

## Summary
`cmd/report` and `internal/report` render one result file (schema v1) plus the hand-written `report/rubric.json` into a deterministic Markdown report: environment and library versions, run configuration, methodology, one table per profile and scenario with deltas against `database/sql`, a generated top 5 with reasons printed before the pros/cons, and a completeness check (a missing cell or rubric entry is an error). Stdlib only, no new dependency. Branch `feat/report-generator`, not committed.

## Assessment vs Reality
| Metric | Predicted | Actual |
|---|---|---|
| Complexity | Medium | Medium |
| Tasks | 10 | 10 done |
| New dependency | none | none; `go.mod` unchanged |

## Tasks
| # | Task | Status |
|---|---|---|
| 1 | Rubric types and loader | Done (`rubric.go`) |
| 2 | Ranking | Done (`rank.go`) |
| 3 | Reasons | Done (`reasons.go`) |
| 4 | Tables | Done (`table.go`) |
| 5 | Render | Done (`report.go`, golden test) |
| 6 | Fixture | Done (`testdata/{fixture,rubric}.json`, `report.golden.md`) |
| 7 | CLI | Done (`cmd/report`) |
| 8 | The real rubric | Done (`report/rubric.json`, 6 libraries; `TestRealRubric` in `cmd/report/main_test.go`) |
| 9 | Wire up and docs | Done (`make report`, CLAUDE.md, PRD row) |
| 10 | Review and graph | Review done; graph refresh below |

## Acceptance Criteria
| Criterion | Status |
|---|---|
| Markdown from the fixture, byte-identical across runs and cell orders | Met (golden test, determinism test with reversed cells; `make report` twice on `results/smoke-p3.json` diffs empty) |
| Every library x scenario x profile is a number or explicit `n/a`; a missing cell is an error | Met (`checkComplete`, test names `lib/scenario/profile`) |
| ent JOIN eager-loading note, `na` exclusion, sqlx label, versions, host info, allocs/op caveat in the output | Met: smoke render has 12 tables, 4 `n/a` rows, the ent JOIN note under both JOIN tables |
| Top 5 reasons generated from numbers, before pros/cons | Met |
| Tests first, `-race`, coverage at least 80% | Met: `internal/report` 99.4%, `cmd/report` 94.3% |
| No new dependency, no `toolchain` line | Met |
| go-reviewer run | Done: 0 CRITICAL, 0 HIGH; M1, M3, M4, M5 fixed, M2 documented; LOW listed below |

## Validation
`go vet ./...`, `gofmt -l .` (empty), `go test -race -cover ./internal/report ./cmd/report`, `make test` all pass. `make report IN=results/smoke-p3.json` renders six libraries; `results/a.json` renders with only `database_sql`.

## Deviations from the plan
- **Ties:** the Top table prints competition ranks from exact score equality (1, 2, 2, 4) and says so. A tie across position 5/6 is still cut by name, but the sixth appears under "Also measured" with its score, so the tie is visible (smoke data: ent and gorm both 4.57).
- **Score scope (M2, documented, not changed):** a rank is absolute and a library's score averages only its own ranked cells, so a library with `na` or excluded cells is judged on fewer, possibly easier, scenarios. A methodology line in the rubric says so; the "Ranked cells" column shows the counts. The real rule belongs to the Phase 5 weighting question.
- **`last_release` became `version_used`** (column "Version used"): upstream release dates were never checked, so the old name and its dated text were wrong. The values are now the version string only. The column overlaps the host-derived "Version" column except for `database/sql` (standard library) and sqlc (generator version); Phase 5 can fold them together.
- **Weights:** rubric validation rejects a weight sum that overflows to +Inf (it would make every score NaN); "weakest on X" only considers metrics with a non-zero weight.
- `fmtSpread` prints one decimal and `*` after a space (`18.2% *`); `n/a` cells print the runner's reason verbatim.
- `Render` also errors on an `ok` cell with no median and on an empty run; rubric validation also rejects a cell note for an unknown library and requires `display`.
- The real-rubric test lives in `cmd/report/main_test.go` (keeps `report/` data-only), with the six registry names as a literal that must be updated with `cmd/bench` `registry`.
- CLAUDE.md graphify line edited because `graphify-out/` is now git-ignored; its four tracked files are staged as deletions.

## Review findings still open (LOW, deliberate)
- `rubric.go` `LoadRubric` accepts trailing JSON after the object; two `fmt.Errorf` calls without arguments should be `errors.New`.
- `escapeCell` ignores lone `\r`, backticks, `<` and `*`; profile names and `n/a` reasons from a run reach headings unescaped.
- Unit-boundary rounding can print "1000 µs".
- `Render` does not call `run.Validate()` or reject duplicate profile names.
- `cmd/report`: `-h` exits 2 (convention 0); a write failure exits 2 (usage code); `os.WriteFile` is not atomic; `-out` equal to `-in` would overwrite the results JSON; the stdout `Fprint` error is ignored.
- Test gaps: baseline cell `na` or high-error, zero-median spread, `-h`.

## Notes for Phase 5
- Smoke data is one repeat with 3 s windows; do not rank from it or publish it.
- Phase 5 must not publish before the Phase 2 p95 instability is resolved (9 of 12 cells over 10% between two runs).
- Open PRD questions: top-5 weights (and the M2 scoring rule) and final stress parameters. Re-check library versions in the rubric before publishing.
- `-out` never defaults to `README.md`; Phase 5 runs `make report IN=... OUT=README.md`.
