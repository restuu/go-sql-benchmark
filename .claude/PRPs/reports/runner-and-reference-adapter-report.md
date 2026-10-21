# Implementation Report: Runner & Reference Adapter (Phase 2)

## Summary
Custom benchmark runner, validated JSON result format (schema v1), `database/sql` (pgx/stdlib) reference adapter, `cmd/bench` and `cmd/benchcmp`. `make bench` runs 6 scenarios x 2 profiles x N repeats and writes `results/run-<UTC>.json`. Not committed; work is on branch `feat/runner-and-reference-adapter`.

## Assessment vs Reality

| Metric | Predicted (Plan) | Actual |
|---|---|---|
| Complexity | Large | Large |
| Files changed | 17 new, 3 updated | 24 Go files new (incl. 8 test/fake files), 3 updated (Makefile, CLAUDE.md, PRD) |

## Acceptance Criteria

| Criterion | Status |
|---|---|
| Complete JSON for `database_sql` (6 x 2, all metrics), passes `Validate` | Met (`results/smoke2.json`, 12/12 `ok`, error rate 0; full runs `a.json`, `b.json`) |
| Two runs within ~10% p95 per cell | **Not met on this machine; deviations explained and recorded below (tolerance not loosened)** |
| `na` cells representable and tested with a fake | Met (`TestRunMatrix`, `TestRunCellProbeFailures`) |
| `-race` tests pass, vet and gofmt clean | Met |

## Validation Results

| Level | Result |
|---|---|
| `make arch-check build vet`, `gofmt -l .` | Pass, no output |
| `make test` (no DB) | Pass |
| `make test-db` and `go test -p 1 -race -cover ./...` with DB | Pass |
| Coverage | bench 100%, result 94.3%, runner 94.2%, databasesql 82.3%, benchcmp 83.3%, seedcheck 80.8%; `cmd/bench` 55.7% (flag parsing tested; `run()` verified by the smoke and full runs) |
| Negative checks | `-libs nope` exit 2 listing valid names; no `BENCH_DSN` exit 2; DB stopped: exit 1, wrapped `seedcheck: ... connection refused` |
| `go.mod` | only `go 1.27.3`; DB left seeded (`make db-check` OK) |

## Stability finding (success signal)
`benchcmp results/a.json results/b.json` (default profiles, 3 repeats): **9 of 12 cells exceed 10% p95**, even the paced low profile.
- p50 is mostly stable between runs (most cells < 10%, worst `delete/low` 31%); p95 is not.
- Noise is within a run too: `delete/stress` p95 across its 3 repeats was 5.7 / 23.8 / 5.9 ms; a median of 3 does not absorb it.
- Stress p99 sits at about 200 ms in nearly every stress repeat (fixed-timer signature, like a TCP RTO or a Docker Desktop port-forward stall). Stress `elapsed_seconds` overshoots the 10 s window by 1.6-3.3% (tail drain).
- Low-profile p95 (~5-6 ms) is above stress select p95 (~2 ms): idle-wake latency in the Docker VM/proxy path. Do not compare libraries on low-profile tails without checking this.
- Candidate fixes for Phase 5 (untested): `CHECKPOINT` after reseed, longer windows / more repeats, pooled-sample percentiles instead of median of per-repeat p95, an A/B against the native Postgres on 5432.

## Intermittent hang (open risk)
One full run aborted with `ops still in flight 10s after the window end` during `select stress`. It did not recur in two later full runs, a delete->select stress loop, or three smoke runs. **Root cause not established.** The code review ruled out the worker loop, pacing sleep, pool exhaustion, unclosed rows and data races; the likeliest cause is a connection/network stall in the Docker path (same class as the 200 ms p99). Mitigation added (below): the failure is now diagnosable (phase, library, scenario, profile, goroutine profile) and a stuck cell is retried once.

## Code review (`ecc:go-reviewer`): 0 CRITICAL, 2 HIGH, 6 MEDIUM, 9 LOW
Fixed:
- HIGH-1 stuck phase gave no diagnosis: error now names library/scenario/profile/phase and embeds a goroutine profile; `Run` retries a stuck cell once (loud progress line), a second stall fails the run. Not fixed: partial results are not saved if a run aborts.
- HIGH-2 `wg.Wait` blocked forever on an adapter that ignores `ctx`: workers are cancelled at `duration+opTimeout`, given `abandonGrace` (5 s), then abandoned.
- Probe call had no timeout (found while testing HIGH-1): now bounded by `opTimeout`, reported as `errPhaseStuck`.
- MEDIUM-1 zero-op windows (e.g. delete pool consumed by warm-up) are now an error (`errNoOps`); `Validate` rejects `ok` cells with `ops < 1` or `elapsed <= 0`.
- MEDIUM-2 (part): `Load` rejects unknown JSON fields.
- MEDIUM-5 `make test-db` runs packages serially (`-p 1`) because they reseed one DB.
- MEDIUM-6 (part): `benchcmp` rejects a NaN tolerance.

Open follow-ups (not fixed, deliberate):
- MEDIUM-2 rest: no `Errors`/`ErrorRate` cross-check, no profile-config or completeness validation, metric errors lack field names.
- MEDIUM-3 throughput divides by elapsed including the drain tail and counts post-deadline completions (small, systematic).
- MEDIUM-4 no settling (CHECKPOINT) between cells; plausible contributor to p95 variance.
- MEDIUM-6 rest: `benchcmp` does not compare `Config.Profiles`/repeats or error rate.
- LOW: paced catch-up burst after a stall; `time.Sleep` not ctx-aware; warm-up and measured window share the id stream; unpaced latency slice (131072/worker) can grow in-window for few workers; no error-rate circuit breaker; `ErrorRate` of the median is not recomputed from median `Errors`/`Ops`; non-atomic `result.Write`; DB left unseeded on failure/Ctrl-C and only `os.Interrupt` handled; minor convention nits (profile names duplicated in `flags.go`, magic numbers in tests, `fmt.Errorf` without verbs, mutable `seedcheck.ExpectedRows`).

## Deviations from Plan
- Task 3 (result) done before Task 2 (stats): `summarize` returns `result.Metrics`.
- Per-op `context.WithTimeout` replaced by a per-phase bound (`duration + opTimeout`): avoids per-op allocations in the hot loop. Ops are timed on the phase context.
- Added `bench.Cities()`, `-delete-pool` flag, and a final `Reseed` after a successful run (leaves the DB seeded).
- `-workers` applies to the stress profile only (the low profile is fixed at 4).
- Plan's negative check `make db-down && make bench` cannot work (`bench` depends on `db-up`); ran the built binary against the stopped DB instead.
- `go run` can mask exit codes; negative checks used a built binary.
- Flag-parsing, benchcmp-NaN and result-strictness tests were written together with their code (no separate RED run).

## Harness floor
No-op adapter: 0 allocs/op, about 27 B/op (latency slice growth at ~570k ops/worker), about 330 CPU ns/op. Real DB ops are 100-1000x slower, so the floor is negligible, but B/op and CPU/op include harness and GC cost by design.

## Smoke numbers (3 s stress, 2 s low, 1 repeat; one machine, indicative only)
SELECT p50 about 0.9 ms vs INSERT p50 about 2 ms; JOIN/UNION return their expected 10/40 rows (error rate 0). Low-profile CPU/op is about 3x stress (fixed process overhead over few ops).

## Next Steps
- [ ] Review the diff, then commit (nothing committed yet)
- [ ] `graphify update .` and `/graphify . --update` for the doc changes
- [ ] Phase 3 (adapters) and Phase 4 (report) can start; Phase 5 owns stress parameters and the variance findings above
