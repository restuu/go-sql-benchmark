# Implementation Report: Full run and publish (Phase 5)

## Summary
Two full default runs of all six libraries (6 libraries x 6 scenarios x 2 profiles x 3 repeats, 216 measured windows each) on the reference Mac, then `benchcmp` with the agreed gate. `benchcmp` gained a top-5 rank check, the rubric gained methodology notes, the README was generated from run 1, and the open PRD questions were resolved. **The agreed stability gate was not met**; this report states what was and was not met. The gate was not loosened and no third run was made (user decision, 2026-10-24).

## Run conditions
| | Run 1 | Run 2 |
|---|---|---|
| Start (UTC) | 2026-10-24T01:50:46Z | 2026-10-24T02:25:25Z |
| End (UTC) | 2026-10-24T02:24:11Z | 2026-10-24T02:58:46Z |
| Load average at start (1 min) | 3.59 | 4.39 |
| Load average at end (1 min) | 12.94 | 13.76 |
| Sleep during run | none (last wake 08:47 WIB, before the 08:50 WIB start) | none |
| Result file | `results/run-full-1.json` | `results/run-full-2.json` |

Heavy apps (Brave, Teams, IDEA) were closed; `make bench` wrapped the run in `caffeinate -i`; run 2 waited for the 1-minute load to fall below 4.5. A load spike of 16 at the first check (08:48) decayed on its own before run 1 started. End-of-run load is the benchmark's own stress cells. Raw logs: `results/run{1,2}-conditions.txt`, `results/run-full-{1,2}.log` (git-ignored, local).

## Stability (`make bench-cmp`, tolerance 0.20 plus top-5 check)
| Criterion | Result |
|---|---|
| p95 within 20% per cell | **Not met**: 62 of 72 cells within 20%, 10 beyond |
| The 10 cells | All low profile (4 workers, 10 ms pacing): database/sql insert 30.9%, update 26.0%, join 23.6%; ent update 26.8%, select 26.6%; gorm update 34.4%, delete 42.9%, join 21.9%; jet update 48.3%, delete 32.3% |
| Stress cells | All 36 within 8.7% |
| sqlx, sqlc | No cell beyond 20% (worst 15.8% and 16.5%) |
| Top-5 membership | Met: database/sql, sqlx, sqlc, ent, go-jet in both runs; gorm 6th in both |
| Top-5 order | **Not met**: sqlx and sqlc swap ranks 2 and 3 |

Scores (mean rank over p95, B/op, CPU ns/op; lower is better):

| Run | database/sql | sqlx | sqlc | ent | go-jet |
|---|---:|---:|---:|---:|---:|
| 1 | 2.06 | 2.11 | 2.33 | 4.27 | 4.58 |
| 2 | 2.08 | 2.33 | 2.22 | 4.50 | 4.64 |

database/sql, sqlx and sqlc are within 0.3 of each other in both runs, so their relative order is within run-to-run noise. In 7 of the 10 failing cells run 1 was slower than run 2, which may be drift between the runs; a third run to test that was declined. The cause of the spread is environmental, as in `p95-stability-report.md`; this run does not change that conclusion, but it shows the low profile is more exposed than the stress profile.

Consequence for readers: treat the top-5 set as stable and the order inside the top 3 as a tie, and expect low-profile p95 to move by up to about 48% between runs on this machine. The README methodology section says this.

## Changes
| Area | Change |
|---|---|
| `internal/report/rank.go` | `TopLibraries(run, weights)`: the report's top 5, best first, for use outside the report |
| `cmd/benchcmp` | Top-5 check (membership and order, weights from `-rubric`, default `report/rubric.json`); exit 1 when it differs or when neither run ranks any library; usage text and messages updated |
| `Makefile` | `make bench-cmp` passes `-tolerance $(BENCH_TOL)` (default 0.20); the binary's own default stays 0.10 |
| `report/rubric.json` | Methodology notes: stress p99 includes a 100-200 ms host-VM stall; measured run-to-run stability from these runs |
| `CLAUDE.md` | `bench-cmp` line |
| PRD | Open questions on weighting and stress parameters resolved; Phase 5 row |
| `README.md` | Generated from run 1 (the first run, so no choosing between runs): `make report IN=results/run-full-1.json OUT=README.md`. There was no README before |

Tests: written first (failing), then passing with `-race`; `benchcmp` coverage 90.5%, `report` 99.4%. `ecc:go-reviewer` found no CRITICAL or HIGH issues; fixed: empty-ranking pass, hard-coded "top 5" text, table-driven `TopLibraries` test, malformed-rubric test.

## Decisions recorded (user, 2026-10-24)
- Stability gate reported plainly as not met; no relaxation, no third run.
- Top-5 weighting: equal weights on p95, B/op, CPU ns/op (default kept).
- Stress parameters: defaults kept (100 workers, unpaced, 2 s warm-up, 10 s window, 400k DELETE ids).

## Library versions checked (2026-10-24, `go list -m -u`)
All at the latest upstream version: ent v0.14.6 (2026-03-17), go-jet v2.16.1 (2026-10-06), pgx v5.11.0 (2026-09-07), sqlx v1.4.0 (2024-04-15, which backs its "no updates since 2024" label), gorm v1.31.2 (2026-06-22). The pinned sqlc generator v1.31.1 was not checked. The rubric `version_used` values match `go.mod`.

## Open items (not done, for later)
- Tie handling in the top-5 check: equal scores are cut by library name, so a tie at the 5/6 boundary is arbitrary. A rank-group or membership-only mode was left until a rule is chosen; this run's order swap is exactly that case.
- `benchcmp` now needs `report/rubric.json` (run from the repo root, as `make` does).
- Earlier follow-ups untouched: partial results on abort, non-atomic `result.Write`, report LOW items.
- Phase 6 (14 extra libraries) is out of scope here; its ranking should come from one run containing all 20 libraries, not from adding to today's numbers.
- The knowledge graph was not refreshed; the post-commit hook rebuilds the code part.
