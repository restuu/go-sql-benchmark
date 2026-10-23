# Investigation Report: p95 stability (Phase 2 follow-up)

## Summary
Two runs of the same matrix on this machine differ by more than 10% p95 in 2 to 9 of 12 cells whatever was changed (2 to 5 on the quietest host). The cause is the environment (a loaded macOS host plus a virtualised Postgres), not the runner. A 10% tolerance on p95 sits at this machine's noise floor: the median pair-to-pair deviation of the current statistic is 9.8%. One component, the 100/200 ms stress p99 stall, has a proven location (the host to OrbStack-VM network crossing). No code or config change in this branch; the stability definition was decided by the user (see Decisions).

Branch `fix/p95-stability`, nothing committed. Raw data: `results/exp/` (git-ignored, local). Probe programs lived in the session scratchpad and are not in the repo.

## Environment
- MacBook Pro M1 Pro (6P+2E), 16 GB, 7.8 GB swap in use, load average 7 to 33 during all runs with ~80% CPU idle, many apps open (Brave, Teams, IDEA). On AC.
- Docker is OrbStack (8 CPUs, 8 GB VM), not Docker Desktop. Postgres 15.19, stock settings except `max_connections=300`.
- Native Postgres on 5432 is 18.6, so it is not a valid A/B for the Docker path (two variables change). Not used.

## Experiments (`database_sql`, default profiles, 3 repeats per run, 3 runs per setup)
Cells over 10% p95, per pair of runs (1-2, 2-3, 1-3), counted with `benchcmp`:

| Id | Variable changed | Result | Verdict |
|---|---|---|---|
| E0 | none (baseline, host to `127.0.0.1:55432`) | 4, 5, 6 of 12 | reference |
| E1 | client inside the VM (linux/arm64 binary, TCP loopback) | 7, 6, 9 of 12; stress p99 stall gone | stall fixed, stability not |
| E2 | checkpoints off (`max_wal_size=32GB`, `checkpoint_timeout=1d`) | 5, 6, 3 of 12 | no effect |
| E3 | autovacuum off | 7, (6), (8) of 12; the pairs with `av-3` are suspect (E6) | no effect |
| E7 | quieter host (user closed Brave/Teams/IDEA, Mac kept awake with `caffeinate -i`) | 5, 4, 2 of 12; worst-pair p95 deviation per cell 2.7% to 18.4% | modest gain, 10% gate still fails |

E2 and E3 were temporary `ALTER SYSTEM` changes on the running container, reset and verified afterwards (`postgresql.auto.conf` empty). `docker-compose.yml` was not touched.

### E1 detail: where the stress stall lives
Stress `select`/`join`/`union` p99 is a constant: about 201 ms through the forwarded port, about 103 ms through the container IP, in 12 of 12 repeats. Little's law (100 workers / 28k ops/s = 3.5 ms mean vs 0.88 ms p50) shows these stalls hold over half of all worker time. Not packet loss: the VM saw 36 retransmits across 539k ops. Layer isolation with the C client `pgbench` (100 connections, point select by id, 10 s):

| Client location | p50 | p95 | p99 | max | ops over 50 ms |
|---|---|---|---|---|---|
| in VM, unix socket | 0.73 ms | 1.8 ms | 3.3 ms | 42 ms | 0 |
| in VM, TCP loopback | 0.92 ms | 3.4 ms | 8.7 ms | 285 ms | 0.03% |
| host to container IP | 0.70 ms | 6.3 ms | 107 ms | 641 ms | 2.4% |
| host to `127.0.0.1:55432` | 0.93 ms | 2.7 ms | 201 ms | 609 ms | 1.3% |

So the stall is neither Go, the harness nor Postgres. The exact mechanism of the 100/200 ms constants is not established (the macOS-side TCP counters read as zeros in the tool sandbox and were not trusted).

### E1 detail: the low-profile tail is idle wake-up
Single connection `SELECT 1`, 1000 samples: p50 0.14 ms unpaced vs 1.4 ms paced at 10 ms through the forward; pure host loopback echo goes from 35 us to 344 us p50 with the same pacing (max 12 ms), and `time.Sleep(10ms)` overshoots by 1 ms at p50. Inside the VM the paced p95 is still 1.2 to 2.0 ms. A busy-core experiment (4 `yes` processes) did not collapse the tail and was inconclusive.

### E4: is a different statistic steadier? (offline, 8 valid host runs, 28 pairs per cell)
| Statistic | pairs over 10% | median pair deviation |
|---|---|---|
| median of 3 p95 (current) | 49% | 9.8% |
| min of 3 p95 | 46% | 9.3% |
| mean of 3 p95 | 57% | 11.5% |
| median of 3 p99 | 49% | 9.8% |
| median of 3 p50 | 19% | 4.2% |
| median CPU ns/op | 12% | 3.7% |

In the VM (3 runs): current 61% / 14.1%, p50 25% / 6.3%.

### E5: checkpoints and autovacuum were active
Postgres checkpointed continuously during runs (15 per hour, WAL-triggered, 50 to 210 s each) and autovacuumed `models`/`orders` 81 times since container creation. Disabling either changed nothing measurable (E2, E3). Write cells have a ~3 ms p50 in every setup (WAL fsync on the VM disk).

### E6: incident, the earlier "ops still in flight" abort
`av-3` hit `ops still in flight after the window end` in `insert stress` (all 100 workers blocked in a network read) and took 50 minutes instead of 6. The macOS power log shows Idle Sleep at 16:29:27, two minutes after the run started, with the machine asleep until 17:16:27. Every other run finished before 16:29 and the log shows no earlier sleep that day. So this occurrence is a sleeping host; `av-3` is unreliable. The abort recorded in the Phase 2 report may have had the same cause, but that earlier event cannot be checked.

### E7: quieter host (H4)
Free memory 71% (was 35%), swap in use 2.7 GB (was 7.8 GB), heavy apps closed, no sleep (power log checked). Load average was still 11 to 14 because of macOS background work (`mediaanalysisd`, Spotlight `mds`), so the host was quieter, not quiet. 3 runs, same command as E0:

| Statistic (3 runs) | pairs over 10%: quiet vs baseline | median pair deviation: quiet vs baseline |
|---|---|---|
| median of 3 p95 | 31% vs 42% | 6.9% vs 8.8% |
| median of 3 p50 | 17% vs 28% | 3.4% vs 4.4% |
| median of 3 p99 | 53% vs 53% | 10.6% vs 11.5% |

Per cell, the worst pair deviation of p95 is 2.7% to 7.1% for stress `select`/`join`/`union` and `update`, 12.7% to 18.4% for the paced low cells and for `insert`/`delete` stress (write p95 is 4 to 6.6 ms, dominated by WAL fsync on the VM disk). 7 of 12 cells still have at least one pair above 10%. A 20% tolerance would pass every cell in this data, but 3 runs is too few to promise that. Stress `select` p99 fell from a constant 201 ms to 8 to 12 ms, while `join` and `union` stress p99 stayed at 201 ms: the stall also depends on host state, and its mechanism is still unknown.

## Not tested
- A truly quiet host (the macOS background load was still present in E7).
- H5, harness effects (tail drain overshoot of 1.6 to 3.3%, catch-up bursts after a stall in paced mode): not measured. By reasoning they are small next to the effects above.
- Full six-library runs and the PRD's real criterion, "top 5 unchanged between two runs". Only `database_sql` was run.

## Decisions (user, 2026-10-23: accepted as proposed)
1. **Stability definition.** Not attainable at 10% p95 here. Gate for Phase 5: p95 within about 20% (pass `-tolerance 0.20` to `benchcmp`; the 0.10 default is unchanged) plus top-5 rank stability across two runs on all six libraries, as the PRD says. The rank check is not implemented yet; it belongs to Phase 5.
2. **Run conditions.** Phase 5 runs with heavy apps closed and the Mac awake, load logged (E7 conditions).
3. **Client placement.** The harness stays on the host. A client inside the VM removes the stall but did not improve spread (E1) and changes the methodology. The rubric methodology must say that stress p99 on macOS includes a 100 to 200 ms host-VM stall and is not a library property. Not written yet (Phase 5 touches `report/rubric.json`).
4. **Keep the Mac awake.** Done: `make bench` runs under `caffeinate -i` when it exists (Makefile only, no Go change).

A VM client on a quiet host (E1 plus E7) was not run.

## Unrelated notes
- `benchcmp`'s deviation is relative to the smaller value (11.7% for 10.29 vs 9.21 ms), so it is asymmetric; fine, but worth documenting.
