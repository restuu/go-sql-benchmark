# Go SQL library benchmark

Run started 2026-10-24T04:41:14Z (UTC). Numbers are only comparable on the same machine.

Compares 15 libraries on three measures: **latency** (p95, the time within which 95% of calls finished), **memory** (B/op, bytes allocated per operation) and **CPU** (CPU ns/op, CPU time per operation). Each library runs 6 scenarios under 2 load profiles; a **case** is one scenario under one load profile, written scenario/profile, for example join/low.

## Environment

- OS/arch: darwin/arm64
- Go: go1.27.3
- CPUs: 8 (GOMAXPROCS 8)
- PostgreSQL: 15.19
- Driver shared by every database/sql-based library: pgx v5.11.0

| Library | Version | Label | Version used |
|---|---|---|---|
| pgx_pool | v5.11.0 | - | v5.11.0 |
| pgx | v5.11.0 | - | v5.11.0 |
| database/sql | - | - | standard library |
| sqlc | - | - | generator v1.31.1, pinned in tools/go.mod |
| sqlx | v1.4.0 | no updates since 2024 | v1.4.0 |
| sqlboiler | v4.19.7 | - | v4.19.7 |
| beego | v2.3.10 | - | v2.3.10 |
| rel | v0.43.0 | - | v0.43.0 |
| pop | v6.4.1 | - | v6.4.1 |
| ent | v0.14.6 | - | v0.14.6 |
| gorm | v1.31.2 | - | v1.31.2 |
| go-jet | v2.16.1 | - | v2.16.1 |
| xorm | v1.4.3 | - | v1.4.3 |
| bun | v1.3.0 | - | v1.3.0 |
| gorm/gen | v0.3.29 | - | v0.3.29 |

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
- A case is one scenario (insert, update, delete, select, join or union) under one load profile (low or stress), written scenario/profile, for example join/low. Every library is run through the same cases. Each case is measured several times and the median is what is reported and ranked, on three measures: latency (p95), memory (B/op, bytes allocated per operation) and CPU (CPU ns/op). A case the library cannot express is n/a and left out of the ranking.
- pop cannot take an existing *sql.DB, so it opens its own pool through its PostgreSQL dialect, which uses the same pgx stdlib driver; the pool is sized like every other adapter.
- Numbers are only comparable on the machine and run they were measured on; the host is listed above.
- Each case is the median of the repeats listed in the run configuration. Before the matrix runs, every adapter is checked against the database/sql reference on returned values, and every read scans every selected column.
- Latency is the per-call service time of successful operations with exact nearest-rank percentiles; it is not corrected for coordinated omission.
- B/op, allocs/op and CPU ns/op are deltas over the measured window divided by ops and include harness and GC cost.
- B/op and allocs/op also include each library's idiomatic result handling: sqlx, sqlc, gorm, ent and jet materialise rows, while the database/sql reference scans into locals, so part of the difference is the cost of that handling.
- A scenario counts as supported only when the library's own query API (or, for the SQL-first sqlc and sqlx, its SQL) expresses it; needing the raw-SQL escape hatch makes the case n/a and it is not ranked.
- A library's score is the weighted geometric mean, over its ranked cases, of each metric divided by database/sql's value in the same case: 1.00 is the same as database/sql, 2.00 twice as much, 0.50 half, and lower is better. A ratio keeps the size of a difference, so a 1% gap moves the score by about 1% instead of a whole place, and a library's score does not depend on which other libraries were run. A library with n/a or excluded cases is judged on fewer, possibly easier, scenarios; the Ranked cases column shows how many.
- Ease of use and learning curve are the author's judgement from writing and reviewing these adapters and reading each library's documentation, not measurements, and they are not ranked. Community facts (stars, last release, last commit, open issues) were read from the forge APIs on the date shown in that section; a release date is the date the latest release was published, or the tag commit date where the notes say so.
- Stress-profile p99 on macOS includes a constant 100-200 ms host-to-VM stall (Postgres runs in an OrbStack VM) and is not a property of the library.
- Run-to-run stability (two full 15-library runs on this machine, 2026-10-24; the earlier 6-library pair had 62 of 72 cases within 20%, all 10 beyond at the low profile, and all 36 stress cases within 8.7%): p95 differed by 20% or less in 130 of 166 cases. The other 36 are 35 low-profile cases, which reached up to 114% (pgx select), and one stress case (pop insert, 27%); the remaining 82 stress cases stayed under 18%. The top 5 had the same libraries in the same order in both runs, and the five scores moved by 0.05 or less (beego most, 1.31 to 1.36), but sqlc and sqlx are 0.01 apart in both runs, so their order is within noise. The cause is environmental (loaded macOS host, host-VM network crossing, WAL fsync on the VM disk).

Reading the tables:

- `p95 vs database/sql` is relative to the baseline's p95 in the same table; negative is faster.
- `p95 spread` is (max - min) / median of the repeats' p95; `*` marks a spread above 10%.
- `!` after a library means an error rate above 1%: the case is shown but not ranked.
- `[n]` points to the numbered note under the table.
- `n/a (...)` means the library cannot express the scenario; the case is not ranked.

## Results

### Profile: low

#### insert

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.6 ms | 5.1 ms | 8.1 ms | 399 | 572 | 12.0 | 269055 | 0.00 | +8.2% | 17.4% * |
| pgx | 2.5 ms | 4.2 ms | 7.6 ms | 399 | 616 | 12.2 | 222632 | 0.00 | -10.8% | 32.1% * |
| database/sql | 2.7 ms | 4.7 ms | 7.7 ms | 399 | 873 | 14.0 | 238217 | 0.00 | baseline | 264.2% * |
| sqlc | 2.7 ms | 4.8 ms | 7.5 ms | 399 | 1031 | 15.2 | 262986 | 0.00 | +1.0% | 1.0% |
| sqlx | 2.6 ms | 4.7 ms | 6.9 ms | 399 | 918 | 14.2 | 261285 | 0.00 | -1.2% | 11.7% * |
| sqlboiler [1] | 2.7 ms | 5.0 ms | 8.4 ms | 399 | 1969 | 36.2 | 323820 | 0.00 | +6.1% | 16.9% * |
| beego | 2.6 ms | 4.4 ms | 7.0 ms | 399 | 2887 | 50.0 | 293550 | 0.00 | -6.4% | 11.6% * |
| rel | 2.8 ms | 6.0 ms | 9.5 ms | 399 | 3101 | 45.1 | 330510 | 0.00 | +25.5% | 35.3% * |
| pop | 2.8 ms | 5.7 ms | 9.6 ms | 399 | 5762 | 175.1 | 415978 | 0.00 | +20.9% | 25.5% * |
| ent | 2.8 ms | 5.6 ms | 7.3 ms | 399 | 4387 | 98.0 | 341688 | 0.00 | +18.2% | 31.4% * |
| gorm [2] | 2.7 ms | 4.6 ms | 6.6 ms | 399 | 6827 | 77.0 | 332711 | 0.00 | -3.5% | 9.1% |
| go-jet | 2.7 ms | 5.2 ms | 9.2 ms | 399 | 3531 | 87.0 | 337920 | 0.00 | +9.6% | 43.3% * |
| xorm | 2.7 ms | 5.5 ms | 7.8 ms | 399 | 4660 | 94.0 | 364939 | 0.00 | +15.5% | 45.9% * |
| bun | 2.9 ms | 5.8 ms | 9.8 ms | 399 | 6086 | 27.2 | 339448 | 0.00 | +21.8% | 27.8% * |
| gorm/gen [3] | 2.9 ms | 5.6 ms | 10 ms | 399 | 6842 | 77.2 | 391201 | 0.00 | +17.5% | 30.0% * |

1. sqlboiler's Insert reports no rows affected, so the adapter counts any error-free insert as 1 row.
2. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.
3. gorm/gen's Create reports no rows affected, so the adapter counts any error-free insert as 1 row; it runs with SkipDefaultTransaction like the gorm adapter.

#### update

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.9 ms | 5.3 ms | 8.7 ms | 399 | 596 | 14.0 | 270594 | 0.00 | +11.6% | 21.2% * |
| pgx | 2.4 ms | 4.4 ms | 6.8 ms | 399 | 641 | 14.1 | 234686 | 0.00 | -7.4% | 17.5% * |
| database/sql | 2.6 ms | 4.7 ms | 7.4 ms | 399 | 974 | 16.1 | 267742 | 0.00 | baseline | 19.4% * |
| sqlc | 2.7 ms | 6.4 ms | 10 ms | 399 | 1049 | 16.0 | 256903 | 0.00 | +35.2% | 21.6% * |
| sqlx | 2.7 ms | 4.6 ms | 6.8 ms | 399 | 929 | 16.0 | 265106 | 0.00 | -3.3% | 8.2% |
| sqlboiler | 2.6 ms | 4.6 ms | 7.5 ms | 399 | 1231 | 20.1 | 272465 | 0.00 | -3.6% | 10.2% * |
| beego | 2.6 ms | 4.0 ms | 7.0 ms | 399 | 2043 | 32.0 | 252976 | 0.00 | -14.6% | 18.6% * |
| rel | 2.9 ms | 5.4 ms | 7.8 ms | 399 | 2467 | 33.0 | 293692 | 0.00 | +15.0% | 47.8% * |
| pop | 2.8 ms | 5.0 ms | 7.4 ms | 399 | 5779 | 159.0 | 319108 | 0.00 | +5.0% | 2.8% |
| ent | 2.8 ms | 5.0 ms | 7.5 ms | 399 | 4919 | 101.0 | 298962 | 0.00 | +5.0% | 10.7% * |
| gorm [1] | 2.8 ms | 5.2 ms | 8.4 ms | 399 | 7902 | 80.1 | 349931 | 0.00 | +9.0% | 15.1% * |
| go-jet | 2.8 ms | 4.8 ms | 7.1 ms | 399 | 3989 | 100.1 | 314261 | 0.00 | +1.1% | 73.1% * |
| xorm | 2.7 ms | 5.7 ms | 9.3 ms | 399 | 5982 | 157.0 | 331545 | 0.00 | +21.3% | 19.1% * |
| bun | 2.9 ms | 5.2 ms | 8.0 ms | 399 | 5961 | 19.0 | 294619 | 0.00 | +10.9% | 31.6% * |
| gorm/gen | 3.0 ms | 6.1 ms | 9.8 ms | 399 | 15429 | 290.0 | 444888 | 0.00 | +28.1% | 11.3% * |

1. gorm runs with SkipDefaultTransaction (by default every write is wrapped in a transaction) and Updates takes an explicit column list, because Updates(struct) skips zero values.

#### delete

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.8 ms | 5.5 ms | 7.9 ms | 399 | 412 | 9.0 | 253830 | 0.00 | +13.2% | 7.4% |
| pgx | 2.4 ms | 4.7 ms | 8.1 ms | 399 | 412 | 9.0 | 205084 | 0.00 | -2.5% | 15.0% * |
| database/sql | 2.7 ms | 4.8 ms | 7.4 ms | 399 | 473 | 11.0 | 252604 | 0.00 | baseline | 14.4% * |
| sqlc | 2.7 ms | 6.3 ms | 11 ms | 399 | 525 | 11.2 | 237626 | 0.00 | +29.4% | 28.1% * |
| sqlx | 2.8 ms | 5.8 ms | 9.1 ms | 399 | 473 | 11.0 | 253008 | 0.00 | +20.5% | 15.2% * |
| sqlboiler | 2.7 ms | 4.9 ms | 7.3 ms | 399 | 601 | 13.0 | 248918 | 0.00 | +2.2% | 33.8% * |
| beego | 2.5 ms | 4.8 ms | 7.4 ms | 399 | 886 | 19.2 | 228782 | 0.00 | -1.3% | 10.9% * |
| rel | 2.9 ms | 5.6 ms | 8.8 ms | 399 | 881 | 21.0 | 272979 | 0.00 | +16.1% | 41.3% * |
| pop [1] | 2.8 ms | 6.5 ms | 9.9 ms | 399 | 1090 | 32.0 | 257864 | 0.00 | +34.3% | 29.6% * |
| ent | 2.8 ms | 4.8 ms | 6.7 ms | 399 | 2121 | 44.0 | 285870 | 0.00 | +0.0% | 9.5% |
| gorm [2] | 2.7 ms | 5.3 ms | 8.2 ms | 399 | 4224 | 49.2 | 267244 | 0.00 | +9.8% | 30.2% * |
| go-jet | 2.7 ms | 5.9 ms | 9.5 ms | 399 | 1553 | 36.0 | 255937 | 0.00 | +21.0% | 42.6% * |
| xorm | 2.8 ms | 5.0 ms | 7.4 ms | 399 | 3692 | 102.0 | 289071 | 0.00 | +2.4% | 16.2% * |
| bun | 2.8 ms | 4.6 ms | 6.9 ms | 399 | 5141 | 13.1 | 275494 | 0.00 | -4.4% | 57.0% * |
| gorm/gen | 2.8 ms | 5.0 ms | 7.4 ms | 399 | 4480 | 52.0 | 306909 | 0.00 | +4.1% | 3.9% |

1. pop's Destroy reports no rows affected, so the adapter counts any error-free delete as 1 row; the benchmark only deletes existing ids.
2. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.

#### select

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 778 µs | 2.2 ms | 9.8 ms | 399 | 1305 | 17.2 | 238340 | 0.00 | -1.1% | 28.0% * |
| pgx | 806 µs | 3.2 ms | 7.0 ms | 399 | 1258 | 18.0 | 228220 | 0.00 | +42.3% | 77.8% * |
| database/sql | 831 µs | 2.2 ms | 5.2 ms | 399 | 1995 | 50.1 | 281865 | 0.00 | baseline | 40.4% * |
| sqlc | 798 µs | 1.7 ms | 5.3 ms | 399 | 1991 | 51.2 | 293308 | 0.00 | -22.3% | 69.6% * |
| sqlx | 816 µs | 1.8 ms | 6.5 ms | 399 | 2367 | 55.2 | 297813 | 0.00 | -20.9% | 8.7% |
| sqlboiler | 815 µs | 1.9 ms | 4.4 ms | 399 | 2170 | 53.1 | 304418 | 0.00 | -16.8% | 26.0% * |
| beego | 874 µs | 2.0 ms | 6.5 ms | 399 | 3329 | 103.0 | 309084 | 0.00 | -8.9% | 16.8% * |
| rel | 862 µs | 2.4 ms | 6.9 ms | 399 | 3530 | 74.1 | 332536 | 0.00 | +6.3% | 96.1% * |
| pop | 879 µs | 1.9 ms | 6.4 ms | 399 | 3880 | 87.2 | 360258 | 0.00 | -14.3% | 123.8% * |
| ent | 866 µs | 2.3 ms | 5.9 ms | 399 | 5572 | 145.1 | 359600 | 0.00 | +2.0% | 28.2% * |
| gorm | 870 µs | 2.7 ms | 5.4 ms | 399 | 6044 | 100.4 | 363423 | 0.00 | +19.1% | 28.2% * |
| go-jet | 946 µs | 3.2 ms | 7.4 ms | 399 | 12081 | 221.6 | 429824 | 0.00 | +41.0% | 65.5% * |
| xorm | 900 µs | 2.4 ms | 7.8 ms | 399 | 7748 | 208.5 | 381459 | 0.00 | +5.8% | 22.6% * |
| bun | 1.4 ms | 3.5 ms | 6.7 ms | 399 | 8051 | 72.9 | 446226 | 0.00 | +53.8% | 23.7% * |
| gorm/gen | 845 µs | 2.0 ms | 5.2 ms | 399 | 6169 | 98.4 | 350000 | 0.00 | -8.7% | 22.8% * |

#### join

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 908 µs | 2.2 ms | 6.0 ms | 399 | 2882 | 63.5 | 261638 | 0.00 | -1.7% | 25.5% * |
| pgx | 930 µs | 4.1 ms | 6.7 ms | 399 | 8387 | 87.5 | 265488 | 0.00 | +83.4% | 41.6% * |
| database/sql | 1.2 ms | 2.3 ms | 5.4 ms | 399 | 3645 | 184.6 | 528180 | 0.00 | baseline | 20.8% * |
| sqlc | 1.1 ms | 2.1 ms | 5.1 ms | 399 | 8143 | 193.9 | 522258 | 0.00 | -7.6% | 28.7% * |
| sqlx | 1.2 ms | 2.4 ms | 7.2 ms | 399 | 9191 | 207.6 | 557628 | 0.00 | +6.7% | 33.5% * |
| sqlboiler | 1.2 ms | 2.6 ms | 6.8 ms | 399 | 14746 | 260.5 | 595394 | 0.00 | +17.5% | 17.9% * |
| beego [1] | 1.1 ms | 2.7 ms | 6.8 ms | 399 | 12101 | 534.8 | 433597 | 0.00 | +17.7% | 15.2% * |
| rel | 1.1 ms | 2.7 ms | 6.1 ms | 399 | 20123 | 353.4 | 507500 | 0.00 | +21.5% | 59.3% * |
| pop | 1.3 ms | 2.4 ms | 6.2 ms | 399 | 13993 | 294.7 | 696742 | 0.00 | +8.4% | 208.7% * |
| ent [2] | 1.4 ms | 5.7 ms | 14 ms | 399 | 13858 | 346.4 | 537611 | 0.00 | +154.5% | 55.6% * |
| gorm | 1.4 ms | 3.2 ms | 7.9 ms | 399 | 17400 | 407.3 | 555930 | 0.00 | +40.2% | 19.8% * |
| go-jet | 1.2 ms | 2.9 ms | 7.8 ms | 399 | 37457 | 692.0 | 638178 | 0.00 | +27.1% | 85.8% * |
| xorm | 1.2 ms | 3.1 ms | 7.0 ms | 399 | 23678 | 610.6 | 560959 | 0.00 | +39.3% | 2.1% |
| bun | 1.5 ms | 3.2 ms | 7.3 ms | 399 | 13295 | 210.5 | 481428 | 0.00 | +40.7% | 19.9% * |
| gorm/gen | 1.4 ms | 2.9 ms | 8.4 ms | 399 | 23547 | 485.9 | 744287 | 0.00 | +28.9% | 13.8% * |

1. beego JOIN is RelatedSel on a foreign-key field (one SQL statement, customer loaded with the order).
2. ent JOIN is edge eager-loading (WithCustomer): two queries, orders then their customers, not one SQL JOIN. Read its JOIN numbers with that in mind.

#### union

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 1.3 ms | 2.7 ms | 7.6 ms | 399 | 4640 | 171.0 | 265888 | 0.00 | +6.2% | 22.4% * |
| pgx | 1.2 ms | 3.6 ms | 7.6 ms | 399 | 14660 | 257.0 | 295240 | 0.00 | +43.2% | 65.8% * |
| database/sql | 1.4 ms | 2.5 ms | 6.0 ms | 399 | 5381 | 270.0 | 346100 | 0.00 | baseline | 1.4% |
| sqlc | 1.3 ms | 2.6 ms | 5.9 ms | 399 | 14743 | 315.8 | 393184 | 0.00 | +3.0% | 13.9% * |
| sqlx | 1.5 ms | 2.7 ms | 7.2 ms | 399 | 15928 | 358.9 | 421506 | 0.00 | +8.3% | 8.9% |
| sqlboiler [1] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| beego [2] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| rel [3] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| pop [4] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| ent [5] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| gorm [6] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| go-jet | 1.4 ms | 4.2 ms | 8.9 ms | 399 | 48839 | 794.0 | 681392 | 0.00 | +66.5% | 41.1% * |
| xorm [7] | 1.5 ms | 3.5 ms | 9.7 ms | 399 | 39568 | 1010.0 | 602618 | 0.00 | +40.2% | 48.4% * |
| bun | 1.3 ms | 3.4 ms | 8.1 ms | 399 | 20433 | 337.2 | 471253 | 0.00 | +34.7% | 34.7% * |
| gorm/gen [8] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |

1. n/a: sqlboiler's query mods cannot express UNION without raw SQL. The case is not ranked.
2. n/a: beego's query API cannot express UNION without raw SQL. The case is not ranked.
3. n/a: rel's query API cannot express UNION without raw SQL. The case is not ranked.
4. n/a: pop's query API cannot express UNION without RawQuery. The case is not ranked.
5. n/a: ent's query API cannot express UNION, and using raw SQL would not measure ent. The case is not ranked.
6. n/a: gorm's query API cannot express UNION without raw SQL. The case is not ranked.
7. xorm UNION is xorm.io/builder's Union (the query builder xorm itself uses) passed to SQL, so it is measured, not n/a.
8. n/a: gorm/gen's query API cannot express UNION without raw SQL. The case is not ranked.

### Profile: stress

#### insert

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.7 ms | 4.4 ms | 7.9 ms | 22759 | 570 | 12.0 | 45826 | 0.00 | +4.0% | 11.2% * |
| pgx | 2.7 ms | 4.3 ms | 8.1 ms | 21268 | 570 | 12.0 | 40857 | 0.00 | +3.7% | 7.6% |
| database/sql | 2.7 ms | 4.2 ms | 7.2 ms | 23152 | 872 | 14.0 | 46318 | 0.00 | baseline | 3.3% |
| sqlc | 2.8 ms | 4.3 ms | 7.3 ms | 23385 | 984 | 15.0 | 46388 | 0.00 | +2.5% | 10.1% * |
| sqlx | 2.7 ms | 4.3 ms | 7.7 ms | 22775 | 872 | 14.0 | 46151 | 0.00 | +2.6% | 2.4% |
| sqlboiler [1] | 2.9 ms | 4.5 ms | 7.3 ms | 21056 | 1923 | 36.0 | 51983 | 0.00 | +8.1% | 14.4% * |
| beego | 2.8 ms | 4.3 ms | 8.9 ms | 20549 | 2884 | 50.0 | 50020 | 0.00 | +2.2% | 2.3% |
| rel | 2.9 ms | 4.5 ms | 8.2 ms | 21295 | 3099 | 45.0 | 53674 | 0.00 | +8.2% | 20.0% * |
| pop | 3.2 ms | 5.7 ms | 12 ms | 19717 | 5756 | 175.0 | 66623 | 0.00 | +35.1% | 21.1% * |
| ent | 2.9 ms | 4.5 ms | 7.7 ms | 21508 | 4388 | 98.0 | 54644 | 0.00 | +7.0% | 35.7% * |
| gorm [2] | 3.0 ms | 4.7 ms | 12 ms | 19543 | 6828 | 77.0 | 56729 | 0.00 | +13.4% | 19.0% * |
| go-jet | 2.8 ms | 4.6 ms | 11 ms | 21347 | 3528 | 87.0 | 51073 | 0.00 | +8.8% | 19.6% * |
| xorm | 2.8 ms | 4.5 ms | 7.8 ms | 21127 | 4660 | 94.0 | 56425 | 0.00 | +6.5% | 7.4% |
| bun | 2.9 ms | 4.6 ms | 7.2 ms | 21625 | 6044 | 27.0 | 53325 | 0.00 | +9.2% | 15.5% * |
| gorm/gen [3] | 2.9 ms | 4.5 ms | 7.2 ms | 21175 | 6796 | 77.0 | 56478 | 0.00 | +7.6% | 11.2% * |

1. sqlboiler's Insert reports no rows affected, so the adapter counts any error-free insert as 1 row.
2. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.
3. gorm/gen's Create reports no rows affected, so the adapter counts any error-free insert as 1 row; it runs with SkipDefaultTransaction like the gorm adapter.

#### update

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 3.0 ms | 4.7 ms | 7.6 ms | 22673 | 594 | 14.0 | 47102 | 0.00 | +6.8% | 33.2% * |
| pgx | 3.0 ms | 4.8 ms | 8.4 ms | 20130 | 594 | 14.0 | 42884 | 0.00 | +8.3% | 10.6% * |
| database/sql | 2.9 ms | 4.4 ms | 7.4 ms | 22276 | 928 | 15.9 | 47026 | 0.00 | baseline | 3.6% |
| sqlc | 2.9 ms | 4.5 ms | 6.8 ms | 22271 | 1048 | 16.0 | 47150 | 0.00 | +2.8% | 20.7% * |
| sqlx | 2.9 ms | 4.6 ms | 7.3 ms | 22070 | 928 | 15.9 | 46762 | 0.00 | +3.2% | 6.7% |
| sqlboiler | 2.9 ms | 4.6 ms | 8.1 ms | 20371 | 1184 | 20.0 | 46801 | 0.00 | +4.6% | 9.1% |
| beego | 2.9 ms | 4.6 ms | 9.2 ms | 20511 | 2040 | 31.9 | 43177 | 0.00 | +4.1% | 3.4% |
| rel | 3.0 ms | 4.8 ms | 9.3 ms | 21395 | 2464 | 32.9 | 50030 | 0.00 | +8.8% | 18.4% * |
| pop | 3.1 ms | 4.9 ms | 12 ms | 19607 | 5588 | 158.9 | 55776 | 0.00 | +10.9% | 18.2% * |
| ent | 3.0 ms | 4.6 ms | 8.8 ms | 21154 | 4919 | 100.9 | 51147 | 0.00 | +3.8% | 4.8% |
| gorm [1] | 3.2 ms | 5.3 ms | 12 ms | 18426 | 7856 | 79.9 | 52768 | 0.00 | +19.6% | 19.7% * |
| go-jet | 3.0 ms | 4.8 ms | 10 ms | 20857 | 3943 | 99.9 | 51344 | 0.00 | +7.6% | 15.8% * |
| xorm | 3.0 ms | 5.3 ms | 13 ms | 20013 | 5977 | 156.9 | 55711 | 0.00 | +19.4% | 8.5% |
| bun | 3.3 ms | 5.3 ms | 9.9 ms | 21027 | 5960 | 19.0 | 49217 | 0.00 | +19.7% | 7.0% |
| gorm/gen | 3.3 ms | 5.2 ms | 10 ms | 19520 | 15095 | 290.0 | 75603 | 0.00 | +17.2% | 2.6% |

1. gorm runs with SkipDefaultTransaction (by default every write is wrapped in a transaction) and Updates takes an explicit column list, because Updates(struct) skips zero values.

#### delete

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.8 ms | 4.4 ms | 7.6 ms | 23078 | 410 | 9.0 | 45873 | 0.00 | +3.6% | 36.9% * |
| pgx | 2.8 ms | 4.4 ms | 7.3 ms | 21183 | 410 | 9.0 | 42650 | 0.00 | +3.7% | 10.6% * |
| database/sql | 2.8 ms | 4.2 ms | 7.3 ms | 23240 | 472 | 11.0 | 45863 | 0.00 | baseline | 2.6% |
| sqlc | 2.8 ms | 4.5 ms | 8.1 ms | 22718 | 480 | 11.0 | 47457 | 0.00 | +7.8% | 11.4% * |
| sqlx | 2.8 ms | 4.3 ms | 7.3 ms | 23228 | 472 | 11.0 | 44738 | 0.00 | +2.2% | 7.6% |
| sqlboiler | 2.9 ms | 4.4 ms | 7.9 ms | 20858 | 600 | 13.0 | 44663 | 0.00 | +5.3% | 12.5% * |
| beego | 2.8 ms | 4.3 ms | 8.1 ms | 21252 | 840 | 19.0 | 40827 | 0.00 | +3.4% | 3.9% |
| rel | 2.9 ms | 4.6 ms | 8.8 ms | 22450 | 880 | 21.0 | 46716 | 0.00 | +10.1% | 14.7% * |
| pop [1] | 2.8 ms | 4.4 ms | 9.5 ms | 22462 | 1088 | 32.0 | 46804 | 0.00 | +5.6% | 17.4% * |
| ent | 3.0 ms | 4.5 ms | 9.6 ms | 20557 | 2120 | 44.0 | 47404 | 0.00 | +8.1% | 22.5% * |
| gorm [2] | 2.8 ms | 4.4 ms | 10 ms | 20561 | 4176 | 49.0 | 48474 | 0.00 | +4.1% | 30.9% * |
| go-jet | 2.8 ms | 4.5 ms | 9.6 ms | 22271 | 1552 | 36.0 | 46893 | 0.00 | +8.0% | 16.4% * |
| xorm | 2.8 ms | 4.7 ms | 11 ms | 21470 | 3688 | 102.0 | 50894 | 0.00 | +11.4% | 10.1% * |
| bun | 3.1 ms | 4.9 ms | 9.6 ms | 21722 | 5096 | 13.0 | 49399 | 0.00 | +15.5% | 5.7% |
| gorm/gen | 2.8 ms | 4.4 ms | 9.1 ms | 21308 | 4480 | 52.0 | 49014 | 0.00 | +4.6% | 1.3% |

1. pop's Destroy reports no rows affected, so the adapter counts any error-free delete as 1 row; the benchmark only deletes existing ids.
2. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.

#### select

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 806 µs | 1.7 ms | 10 ms | 33883 | 1212 | 17.0 | 37637 | 0.00 | -2.1% | 18.3% * |
| pgx | 801 µs | 1.6 ms | 5.0 ms | 33702 | 1260 | 18.0 | 38198 | 0.00 | -4.9% | 9.4% |
| database/sql | 852 µs | 1.7 ms | 14 ms | 33432 | 1947 | 50.0 | 43870 | 0.00 | baseline | 4.2% |
| sqlc | 829 µs | 1.7 ms | 200 ms | 32968 | 1971 | 51.0 | 43915 | 0.00 | +1.0% | 10.8% * |
| sqlx | 839 µs | 1.7 ms | 5.5 ms | 33260 | 2347 | 55.0 | 45475 | 0.00 | -1.7% | 1.5% |
| sqlboiler | 838 µs | 1.7 ms | 201 ms | 32708 | 2155 | 53.0 | 46577 | 0.00 | -0.9% | 8.8% |
| beego | 862 µs | 1.7 ms | 4.7 ms | 33537 | 3322 | 102.9 | 45793 | 0.00 | +0.4% | 1.6% |
| rel | 867 µs | 1.8 ms | 5.4 ms | 31960 | 3515 | 74.0 | 46643 | 0.00 | +3.9% | 6.0% |
| pop | 904 µs | 1.9 ms | 201 ms | 31827 | 3765 | 87.0 | 52830 | 0.00 | +11.8% | 9.8% |
| ent | 909 µs | 1.8 ms | 4.5 ms | 32890 | 5564 | 145.0 | 50707 | 0.00 | +7.3% | 0.4% |
| gorm | 911 µs | 1.9 ms | 5.1 ms | 33152 | 5996 | 100.0 | 49941 | 0.00 | +12.7% | 7.4% |
| go-jet | 934 µs | 1.9 ms | 201 ms | 30777 | 12006 | 221.0 | 59357 | 0.00 | +12.8% | 37.2% * |
| xorm | 908 µs | 1.8 ms | 201 ms | 31931 | 7660 | 208.0 | 55753 | 0.00 | +6.8% | 5.0% |
| bun | 2.4 ms | 4.7 ms | 204 ms | 11956 | 8236 | 79.5 | 112672 | 0.00 | +175.1% | 6.4% |
| gorm/gen | 918 µs | 1.9 ms | 7.5 ms | 32495 | 6124 | 98.0 | 50143 | 0.00 | +10.8% | 7.0% |

#### join

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 908 µs | 2.0 ms | 201 ms | 29814 | 2880 | 63.5 | 39963 | 0.00 | -24.4% | 16.4% * |
| pgx | 935 µs | 2.0 ms | 201 ms | 29789 | 8384 | 87.5 | 45384 | 0.00 | -23.8% | 6.3% |
| database/sql | 1.2 ms | 2.7 ms | 201 ms | 26474 | 3557 | 183.7 | 88487 | 0.00 | baseline | 18.5% * |
| sqlc | 1.2 ms | 2.7 ms | 201 ms | 25238 | 8015 | 193.0 | 93246 | 0.00 | +1.0% | 4.8% |
| sqlx | 1.3 ms | 2.8 ms | 201 ms | 25352 | 9107 | 206.7 | 95404 | 0.00 | +2.9% | 22.8% * |
| sqlboiler | 1.4 ms | 3.0 ms | 201 ms | 24134 | 14204 | 259.1 | 105871 | 0.00 | +11.0% | 10.9% * |
| beego [1] | 1.1 ms | 2.5 ms | 201 ms | 26874 | 11998 | 533.4 | 72683 | 0.00 | -8.2% | 1.1% |
| rel | 1.1 ms | 2.4 ms | 201 ms | 27207 | 20039 | 352.6 | 70555 | 0.00 | -10.8% | 3.8% |
| pop | 1.4 ms | 3.0 ms | 201 ms | 24454 | 13462 | 293.7 | 109774 | 0.00 | +12.5% | 5.3% |
| ent [2] | 1.9 ms | 3.6 ms | 203 ms | 15622 | 13820 | 345.9 | 107072 | 0.00 | +34.7% | 15.0% * |
| gorm | 1.5 ms | 3.3 ms | 201 ms | 23145 | 17288 | 406.2 | 117755 | 0.00 | +22.0% | 4.8% |
| go-jet | 1.3 ms | 2.9 ms | 201 ms | 24477 | 37356 | 691.1 | 100183 | 0.00 | +6.8% | 15.3% * |
| xorm | 1.2 ms | 2.6 ms | 201 ms | 25669 | 23576 | 609.7 | 85288 | 0.00 | -2.8% | 2.1% |
| bun | 2.1 ms | 5.0 ms | 203 ms | 14626 | 13152 | 210.8 | 101637 | 0.00 | +86.7% | 7.1% |
| gorm/gen | 1.4 ms | 3.2 ms | 201 ms | 23300 | 22771 | 484.8 | 120504 | 0.00 | +20.8% | 11.0% * |

1. beego JOIN is RelatedSel on a foreign-key field (one SQL statement, customer loaded with the order).
2. ent JOIN is edge eager-loading (WithCustomer): two queries, orders then their customers, not one SQL JOIN. Read its JOIN numbers with that in mind.

#### union

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 1.6 ms | 4.9 ms | 201 ms | 21550 | 4641 | 171.0 | 37099 | 0.00 | +0.8% | 23.4% * |
| pgx | 1.7 ms | 5.1 ms | 202 ms | 21157 | 14657 | 257.0 | 46411 | 0.00 | +3.1% | 5.7% |
| database/sql | 1.8 ms | 4.9 ms | 202 ms | 20775 | 5261 | 269.2 | 53366 | 0.00 | baseline | 9.8% |
| sqlc | 1.7 ms | 5.2 ms | 202 ms | 20334 | 14674 | 315.2 | 58495 | 0.00 | +6.3% | 10.4% * |
| sqlx | 1.7 ms | 4.9 ms | 202 ms | 20153 | 15854 | 358.2 | 63010 | 0.00 | +0.8% | 1.7% |
| sqlboiler [1] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| beego [2] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| rel [3] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| pop [4] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| ent [5] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| gorm [6] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| go-jet | 2.0 ms | 5.9 ms | 202 ms | 18303 | 48745 | 793.2 | 103352 | 0.00 | +20.4% | 8.0% |
| xorm [7] | 1.9 ms | 5.3 ms | 202 ms | 19137 | 39475 | 1009.2 | 86170 | 0.00 | +7.7% | 22.2% * |
| bun | 1.8 ms | 5.1 ms | 202 ms | 19410 | 20345 | 336.3 | 73582 | 0.00 | +3.7% | 13.0% * |
| gorm/gen [8] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |

1. n/a: sqlboiler's query mods cannot express UNION without raw SQL. The case is not ranked.
2. n/a: beego's query API cannot express UNION without raw SQL. The case is not ranked.
3. n/a: rel's query API cannot express UNION without raw SQL. The case is not ranked.
4. n/a: pop's query API cannot express UNION without RawQuery. The case is not ranked.
5. n/a: ent's query API cannot express UNION, and using raw SQL would not measure ent. The case is not ranked.
6. n/a: gorm's query API cannot express UNION without raw SQL. The case is not ranked.
7. xorm UNION is xorm.io/builder's Union (the query builder xorm itself uses) passed to SQL, so it is measured, not n/a.
8. n/a: gorm/gen's query API cannot express UNION without raw SQL. The case is not ranked.

## Top 5

Score is the weighted geometric mean, over every ranked case, of each metric's ratio to database/sql (p95 1, B/op 1, CPU ns/op 1). 1.00 means the same as database/sql, 2.00 twice as much, 0.50 half; lower is better. A case needs a measured database/sql result to be scored.

Libraries with equal scores share a rank.

Only libraries that are not tied to one database are ranked; the others are under Honorable mention.

| Rank | Library | Score | Ranked cases |
|---:|---|---:|---:|
| 1 | database/sql | 1.00 | 12/12 |
| 2 | sqlc | 1.15 | 12/12 |
| 3 | sqlx | 1.16 | 12/12 |
| 4 | sqlboiler | 1.25 | 10/12 |
| 5 | beego | 1.31 | 10/12 |

**database/sql**

- the baseline: every ratio is 1.00 by definition

**sqlc**

- ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 1.04x, B/op 1.43x, CPU ns/op 1.03x
- slowest case: update/low (p95 1.35x database/sql)

**sqlx**

- ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 1.01x, B/op 1.44x, CPU ns/op 1.05x
- slowest case: delete/low (p95 1.20x database/sql)

**sqlboiler**

- ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 1.03x, B/op 1.74x, CPU ns/op 1.09x
- slowest case: join/low (p95 1.17x database/sql)
- judged on 10 of 12 cases

**beego**

- ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 0.98x, B/op 2.36x, CPU ns/op 0.97x
- slowest case: join/low (p95 1.18x database/sql)
- judged on 10 of 12 cases

Also measured:

- rel: score 1.49, ranked cases 10/12
- pop: score 1.73, ranked cases 10/12
- ent: score 1.79, ranked cases 10/12
- gorm: score 2.02, ranked cases 10/12
- go-jet: score 2.05, ranked cases 12/12
- xorm: score 2.06, ranked cases 12/12
- bun: score 2.10, ranked cases 12/12
- gorm/gen: score 2.25, ranked cases 10/12

## Honorable mention

Measured and shown in every table, but kept out of the ranking above. Score is computed the same way as in the Top 5, so it is comparable; lower is better.

| Library | Score | Ranked cases | Why not ranked |
|---|---:|---:|---|
| pgx_pool | 0.85 | 12/12 | pgx is a PostgreSQL-only driver API, so code written against it cannot move to another database. |
| pgx | 0.98 | 12/12 | pgx is a PostgreSQL-only driver API, so code written against it cannot move to another database. |

## Pros and cons

### pgx_pool

**Pros**

- Lowest-level adapter here: pgxpool with Query and Scan into locals, so it shows the cost of the driver and pool alone.
- Every scenario, including UNION, is plain SQL.

**Cons**

- Hand-written SQL and manual scanning of every column.
- PostgreSQL only.

### pgx

**Pros**

- Native PostgreSQL driver API without database/sql; rows are collected into structs with `CollectRows`/`RowToStructByName`.
- Every scenario, including UNION, is plain SQL.

**Cons**

- Hand-written SQL and no query builder.
- PostgreSQL only.

### database/sql

**Pros**

- No dependency beyond the driver; it is the reference every other adapter is checked against.
- Full control of the SQL and of what is scanned.

**Cons**

- Hand-written SQL strings and row scanning, with no compile-time check of either.

### sqlc

**Pros**

- Queries are checked against the schema when the code is generated, and the generated code is typed.
- Plain SQL, so every scenario including UNION is expressible.

**Cons**

- Needs a generation step (make gen-sqlc) and committed generated code.
- Each query lives in an SQL file; there is no runtime query builder.

### sqlx

**Pros**

- Thin layer over database/sql: the SQL stays hand-written and struct scanning is added.
- Every scenario is expressible with plain SQL.

**Cons**

- No updates since 2024.
- Scanning into structs is reflection-based.

### sqlboiler

**Pros**

- Typed models, column constants and where helpers generated from the live schema; Update and Delete report rows affected.
- JOIN is qm.InnerJoin bound onto a struct with Bind: one SQL statement.

**Cons**

- UNION is not expressible without raw SQL, so that case is n/a.
- Needs a generation step against the live database (make gen-sqlboiler, which also needs the sqlboiler-psql driver binary on PATH) and a large amount of committed generated code.
- Insert reports no rows affected, so the adapter counts any error-free insert as 1 row.
- The JOIN clause and the Bind destination (alias-prefixed columns) are written by hand as strings.

### beego

**Pros**

- The ORM (Ormer, QuerySeter) runs on a plain *sql.DB through AddAliasWthDB, so it shares the pgx stdlib driver; JOIN is RelatedSel on a foreign-key field, loaded in the same query.

**Cons**

- UNION is not expressible without raw SQL, so that case is n/a.
- Global registries: models register once before first use and a database alias name cannot be reused, so the adapter registers a fresh alias on every Init.
- A query that never calls Limit is capped at 1000 rows by default.

### rel

**Pros**

- Query API covers INSERT, UPDATE (UpdateAny), DELETE (DeleteAny), SELECT and JOIN (JoinOn, one SQL statement); UpdateAny and DeleteAny report rows affected.

**Cons**

- UNION is not expressible without raw SQL, so that case is n/a.
- The PostgreSQL adapter (go-rel/postgres v0.12.0) has had no release since 2024-05.
- Logs every statement by default; the adapter installs a no-op instrumenter.

### pop

**Pros**

- Query API covers INSERT, UPDATE (UpdateQuery reports rows affected), DELETE, SELECT and JOIN (Join with an ON clause, one SQL statement).

**Cons**

- Destroy and Query.Delete report no rows affected, so the adapter's DELETE returns 1 for any error-free call; a missing id would also report 1.
- UNION is not expressible without RawQuery, so that case is n/a.
- Opens its own pool (no *sql.DB injection), though on the same pgx stdlib driver.
- Create, Update and Destroy run model lifecycle callbacks (before/after hooks).

### ent

**Pros**

- Typed query API generated from a schema definition.

**Cons**

- JOIN is edge eager-loading: two queries rather than one SQL JOIN.
- UNION is not expressible without raw SQL, so that case is n/a.
- Needs a generation step (make gen-ent) and a large amount of committed generated code.

### gorm

**Pros**

- Query API covers INSERT, UPDATE, DELETE, SELECT and JOIN (Joins with a relation produces one SQL JOIN).

**Cons**

- UNION is not expressible without raw SQL, so that case is n/a.
- Defaults need care to benchmark fairly: every write is wrapped in a transaction unless SkipDefaultTransaction is set, and Save or Updates(struct) misreport or skip zero values and missing rows unless guarded.

### go-jet

**Pros**

- Type-safe SQL builder; every scenario, including UNION, is expressed with its own API.

**Cons**

- Code is generated from the live database (make gen-jet needs it running).
- Pulls in lib/pq for array literals, although no adapter uses that driver.

### xorm

**Pros**

- Query API covers INSERT, UPDATE, DELETE, SELECT, JOIN (Table plus Join, one SQL statement) and UNION (xorm.io/builder's Union passed to SQL); writes report rows affected.
- Takes an existing *sql.DB (core.FromDB) and has a pgx dialect.

**Cons**

- The default SnakeMapper maps the field ID to the column i_d; the adapter needs GonicMapper.
- Update(struct) skips zero values (a false Right) unless the columns are named, and Get reports a missing row as found=false, not as an error.
- Conditions, join clauses and the select list are strings checked at run time.

### bun

**Pros**

- Query builder covers every scenario: JOIN is a belongs-to relation compiled to one SQL JOIN, UNION is Union on the select builder, and writes report rows affected.

**Cons**

- Models embed bun.BaseModel and carry bun: struct tags, and PostgreSQL needs the separate pgdialect module.

### gorm/gen

**Pros**

- Typed, field-level query API (fields, conditions, Join, Select) generated on top of GORM from the live schema; Update and Delete report rows affected.
- JOIN is the generated Join with a field Select: one SQL statement.

**Cons**

- UNION is not expressible without raw SQL, so that case is n/a.
- Create reports no rows affected, so the adapter counts any error-free insert as 1 row.
- No generator CLI in the module: make gen-gormgen runs a small program (internal/adapters/gormgen/generate) against the live database, and the generated code is committed.
- gorm.io/gen v0.3.29 requires gorm.io/plugin/dbresolver v1.5.3, which does not compile against gorm v1.31.2; go.mod pins dbresolver v1.6.2.
- Same GORM defaults as the gorm adapter need care (SkipDefaultTransaction, an explicit column Select so Updates(struct) writes zero values).

## Flexibility (qualitative)

Hand-written assessment of how each library handles dynamic queries, for example a search endpoint that adds conditions, joins or ordering only when a parameter is present. It is not part of the score, the Top 5 or the stability check, and no query composition was benchmarked.

Levels: `native` = the library's own API composes it; `partial` = works with caveats (see notes); `raw SQL` = needs string SQL or an escape hatch; `static only` = the query is fixed at build time, so use several queries.

| Library | optional filters | dynamic ordering | conditional join | dynamic projection | in-list / batch | dynamic pagination |
|---|---|---|---|---|---|---|
| database/sql | raw SQL | raw SQL | raw SQL | raw SQL | raw SQL | raw SQL |
| sqlc | partial | partial | static only | static only | native | native |
| sqlx | raw SQL | raw SQL | raw SQL | raw SQL | native | raw SQL |
| gorm | native | native | native | native | native | native |
| ent | native | native | native | native | native | native |
| go-jet | native | native | native | native | partial | native |
| pgx | raw SQL | raw SQL | raw SQL | raw SQL | native | raw SQL |
| pgx_pool | raw SQL | raw SQL | raw SQL | raw SQL | native | raw SQL |
| bun | native | native | native | native | native | native |
| beego | native | native | partial | native | native | native |
| pop | native | native | native | native | partial | native |
| rel | native | native | native | native | native | native |
| sqlboiler | native | native | native | native | partial | native |
| gorm/gen | native | native | native | native | native | native |
| xorm | native | native | native | native | native | native |

**database/sql**

- Every dynamic part is built by hand: append WHERE fragments, ORDER BY and LIMIT/OFFSET to a string and collect the arguments in a slice. Nothing limits what can be composed, but nothing checks it either (no compile-time or schema check), and column names for ORDER BY must be allow-listed by the caller.
- Variable-length IN: on PostgreSQL pass one array argument (`id = ANY($1)`) through pgx, or generate the placeholders yourself.

**sqlc**

- Every query is fixed SQL at generation time; there is no runtime WHERE, ORDER BY, JOIN or column list. Dynamic search means either one query per shape or SQL written to tolerate absent parameters.
- Optional filters: `sqlc.narg('x')` makes a parameter nullable, so `WHERE (col = sqlc.narg('x') OR sqlc.narg('x') IS NULL)` or `coalesce(sqlc.narg('x'), col)` skips the filter when it is NULL. This works, but the generated parameter struct carries every filter and the plan is one generic plan for all shapes.
- Dynamic ordering only through a `CASE` expression over a fixed list of columns written in the SQL. Variable-length IN on PostgreSQL is `col = ANY($1::int[])` with a slice argument (`sqlc.slice` is for MySQL and SQLite). LIMIT and OFFSET are ordinary parameters.
- A query that is truly dynamic needs hand-written SQL outside sqlc.

**sqlx**

- Same string-building model as database/sql for filters, ordering, joins, projection and pagination: fully flexible, unchecked, and ORDER BY columns need an allow-list.
- `sqlx.In` expands a slice argument into one placeholder per element (then `Rebind` for the driver; an empty slice is an error), and `NamedExec` accepts a slice of structs or maps for batch inserts.
- Named queries (`:name`) bind from a struct or map but do not make parts of the SQL optional.

**gorm**

- Chained `Where`, `Order`, `Select`/`Omit`, `Joins`, `Preload`, `Limit` and `Offset` can each be applied inside an `if`, and `Scopes(func(*gorm.DB) *gorm.DB)` packages a reusable conditional step.
- Caveat: a struct passed to `Where` skips zero-value fields (0, '', false), so it cannot filter on a zero value; use a map or an explicit `Where("col = ?", v)`. `Order` with a plain string is raw SQL; pass `clause.OrderByColumn` for a checked column.
- Variable-length IN: `Where("id IN ?", slice)`, a slice of primary keys in `Where`, and `CreateInBatches` for batch inserts. `Updates` with a map or `Select` columns chooses the updated columns at runtime.
- Most of the query is built from strings and interface{} values, so mistakes surface at run time, not compile time.

**ent**

- Generated, typed predicates are collected in a `[]predicate.X` and passed to `Where(preds...)`, so optional filters are an `append` inside an `if`; `And`, `Or` and `Not` combine them.
- Conditional eager loading: add `WithXxx()` only when needed (it issues extra queries, not a JOIN). `HasXxxWith(...)` filters through an edge and renders as a subquery, not a JOIN of selected columns.
- Ordering: generated `ByXxx()` options (or `ent.Asc(field)`) passed to `Order`, chosen at runtime from a switch. Projection: `Select(fields...)` with `Scan` into a custom struct, or `Select` on the entity (unselected fields stay zero). `Limit` and `Offset` apply directly; `FieldIn`/`IDIn` take a slice and `CreateBulk` batches inserts.
- Every shape stays inside the generated API, but the schema must be regenerated when the model changes.

**go-jet**

- Typed builder: start from `Bool(true)` and `AND`/`OR` in `BoolExpression`s only when a parameter is present; the FAQ shows exactly this pattern for dynamic conditions.
- Dynamic projection by appending columns or expressions to a `ProjectionList`; joins are values (`table.INNER_JOIN(other, on)`) that can be reassigned conditionally; `ORDER_BY` takes expressions chosen at runtime; `LIMIT` and `OFFSET` take values.
- Variable-length IN is partial: the slice has to be converted to `[]Expression` (one `Int(...)`/`String(...)` per element) before `IN(...)`.
- Every shape stays inside the builder, but column and table types come from code generated against the live database.

**pgx**

- Plain SQL strings with `$n` or `@name` (`pgx.NamedArgs`) arguments: filters, ordering, joins, projection and pagination are all built by hand, with no builder and no compile-time check. Column names for ORDER BY need an allow-list.
- Variable-length input is native: a Go slice encodes as a PostgreSQL array (`id = ANY($1)`), `pgx.Batch` sends several queries in one round trip and `CopyFrom` bulk-loads rows.

**pgx_pool**

- Plain SQL strings with `$n` or `@name` (`pgx.NamedArgs`) arguments: filters, ordering, joins, projection and pagination are all built by hand, with no builder and no compile-time check. Column names for ORDER BY need an allow-list.
- Variable-length input is native: a Go slice encodes as a PostgreSQL array (`id = ANY($1)`), `pgx.Batch` sends several queries in one round trip and `CopyFrom` bulk-loads rows.

**bun**

- SelectQuery methods return the query, so `Where`, `WhereGroup`, `Join`, `Relation`, `Column`/`ExcludeColumn`, `Limit` and `Offset` can each be applied inside an `if`; `Apply(fn...)` packages a reusable conditional step.
- `bun.In(slice)` expands a slice into an IN list, and `NewInsert().Model(&slice)` inserts a slice in one statement.
- Caveat: `Order` takes a column and direction as a string and `OrderExpr` takes raw SQL, so user-supplied column names still need an allow-list; conditions are strings with `?` placeholders, checked at run time.

**beego**

- QuerySeter methods (`Filter`, `OrderBy`, `Limit`, `RelatedSel`) chain, so each can be applied inside an `if`; `NewCondition().And(...)`/`Or(...)` with `SetCond` builds nested AND/OR groups at run time. `OrderBy("-col")` sorts descending.
- Joins: `RelatedSel` only joins relations declared on the model (rel(fk) and similar); an ad-hoc join needs the string-based QueryBuilder or raw SQL, hence partial.
- Projection by passing column names to `All`/`One`; IN lists through the `__in` operator (`Filter("id__in", ids...)`); `InsertMulti(bulk, slice)` batch-inserts.
- Caveat: `Limit` defaults to 1000 rows when not called, and filter field names are strings checked at run time.

**pop**

- Query methods (`Where`, `Order`, `Join`/`LeftJoin`/`InnerJoin`, `Select`, `Limit`, `Paginate`) chain on a `*Query`, so each can be applied inside an `if`; `Scope(fn)` packages a reusable step.
- `Where("id in (?)", ids)` expands a slice into an IN list (pop uses it internally for association preloading). Batch insert is partial: `Create` accepts a slice but issues one INSERT per element.
- Caveat: conditions, ordering and join clauses are strings with `?` placeholders, checked at run time; `Order` with user input needs an allow-list.

**rel**

- Queries and filters are plain values (`rel.Where`, `where.Eq`/`And`/`Or`/`Not`, `rel.SortAsc`, `rel.Select`, `Join`/`JoinOn`/`JoinWith`, `rel.Limit`, `rel.Offset`), so a `[]rel.Querier` can be assembled with `if`s and passed to `FindAll`.
- `where.In`, `InInt` and `InString` take value lists; `InsertAll` inserts a slice.
- Caveat: field names are strings checked at run time, and `Wheref`/`Joinf`/`where.Fragment` accept raw SQL fragments.

**sqlboiler**

- Queries take a list of `qm.QueryMod` values (`qm.Where`, the generated `ModelWhere.Col.EQ(...)` helpers, `qm.OrderBy`, `qm.InnerJoin`, `qm.Load`, `qm.Select`, `qm.Limit`, `qm.Offset`), so a `[]qm.QueryMod` can be assembled with `if`s and passed to `models.Xs(mods...)`.
- Projection: `qm.Select` with `Bind` into a custom struct, or `boil.Whitelist`/`Blacklist` to choose the columns an insert or update writes. IN lists: `qm.WhereIn("id in ?", args...)` or the generated `IN(slice)` helper.
- Batch insert is not offered (no InsertAll in v4.19.7): one `Insert` per row, and `Upsert` takes one row; that is why in_list_batch is partial.
- Caveat: `qm.OrderBy`, `qm.InnerJoin` and the `qm.Where` clause are raw strings checked at run time, so user-supplied column names need an allow-list.

**gorm/gen**

- Conditions are typed `gen.Condition` values (`t.Col.Eq(v)`, `Or`, `Not`), so a `[]gen.Condition` can be assembled with `if`s and passed to `Where(conds...)`; `Scopes(func(gen.Dao) gen.Dao)` packages a reusable conditional step.
- Ordering: the generated query struct has `GetFieldByName(name)`, which looks a generated column up at run time and returns an order expression (an unknown name is reported, not injected). `Select`/`Omit` take field lists, `Join`/`LeftJoin` take generated tables, `Limit` and `Offset` apply directly.
- Variable-length input: `t.Col.In(values...)`, `Create(values...)` and `CreateInBatches` for batch inserts.
- Caveat: only tables and columns that were generated are reachable; a changed schema needs regeneration.

**xorm**

- Each engine call returns a `*Session` that keeps chaining, so `Where`, `And`, `Or`, `Join`, `Select`/`Cols`/`Omit`, `OrderBy`/`Asc`/`Desc` and `Limit` can each be applied inside an `if`; conditions can also be composed as `builder.Cond` values (`builder.Eq`, `builder.And`, `builder.Or`) and passed to `Where`.
- Variable-length input: `In(col, slice...)` and `InsertMulti(slice)` for batch inserts. `Limit(n, start)` covers pagination.
- Caveat: `OrderBy`, `Join` and plain-string `Where` take SQL text, so user-supplied column names need an allow-list.

## Ease of use (qualitative)

Hand-written judgement of the day-to-day work a library asks for. It is not part of the score, the Top 5 or the stability check. Levels for setup effort, query writing and type safety: `good` = little work or checked early, `fair` = some manual work or checked only partly, `poor` = mostly manual or checked only at run time. Levels for debugging: `good` = the SQL that ran and its errors are visible with little setup, `fair` = visible after configuring logging or hooks, `poor` = hard to see what ran.

Criteria: setup effort = codegen, config and boilerplate before the first query; query writing = code a typical query takes; type safety = how many mistakes the compiler catches; debugging = SQL visibility, error messages and logging hooks.

| Library | setup effort | query writing | type safety | debugging |
|---|---|---|---|---|
| database/sql | good | poor | poor | good |
| sqlc | fair | good | good | good |
| sqlx | good | fair | poor | good |
| gorm | good | good | fair | fair |
| ent | poor | good | good | fair |
| go-jet | fair | good | good | good |
| pgx | good | fair | poor | good |
| pgx_pool | good | poor | poor | good |
| bun | good | good | fair | good |
| beego | fair | fair | poor | fair |
| pop | fair | good | poor | fair |
| rel | good | good | fair | good |
| sqlboiler | poor | good | good | good |
| gorm/gen | poor | good | good | fair |
| xorm | fair | good | poor | fair |

**database/sql**

- No setup beyond the driver, but every query needs its own SQL string, argument list and Scan call.
- The SQL that runs is the SQL you wrote, so there is nothing to translate when debugging; a wrong column or placeholder count only fails at run time.

**sqlc**

- Needs a sqlc config, annotated .sql files and a generate step with committed output; after that a query is a typed function call.
- Queries are checked against the schema at generation time, and the generated code is plain, readable Go.

**sqlx**

- Adds struct scanning (Get, Select, named queries) on top of database/sql, so there is less Scan code; SQL stays in strings with no compile-time check.

**gorm**

- Little setup: models with struct tags and a Open call; chained query methods keep typical queries short.
- Columns and conditions are strings, so mistakes surface at run time; defaults such as the implicit write transaction and zero-value skipping in Updates(struct) surprise newcomers.

**ent**

- Needs a schema definition in Go, a code generator and a large body of generated code before the first query.
- The generated builders are typed and discoverable, but a query goes through generated layers (edge loading instead of a JOIN) that you have to inspect to see the SQL.

**go-jet**

- Needs a generate step against a live database; queries are then written with a builder that mirrors SQL.
- Builder expressions are typed against the generated tables, and a statement can print its SQL and arguments.

**pgx**

- Pool config plus plain SQL; CollectRows with RowToStructByName removes most Scan code, but the column to field match is checked at run time.
- Errors are PostgreSQL errors with codes, and a query tracer is built in.

**pgx_pool**

- Pool config plus plain SQL, with Query/Scan into locals like database/sql: the most code per query, and nothing is checked before run time.
- The SQL that runs is the SQL you wrote, and PostgreSQL error codes are available directly.

**bun**

- Models embed bun.BaseModel with struct tags, and PostgreSQL needs the separate pgdialect module; no generator.
- Conditions and columns are strings, and a query hook (bundebug) prints each statement.

**beego**

- Models must be registered globally before use and the database alias is a global name, which is easy to get wrong in tests and multi-database programs.
- Filters are strings such as customer__id, so typos only fail at run time; a query without Limit is capped at 1000 rows by default.

**pop**

- Needs connection details and db struct tags; the pop CLI and migrations are optional here.
- Where clauses are strings; Destroy reports no rows affected and model lifecycle callbacks run on writes, which hides what a call does.

**rel**

- Repository style with changesets; no generator, and struct fields map to columns by convention.
- Conditions use typed helpers (where.Eq) but column names are strings; every statement is logged unless an instrumenter is replaced.

**sqlboiler**

- Needs the live database, the sqlboiler-psql driver binary and a config file to generate a large body of models.
- Generated where helpers and column constants make most queries typed; JOIN clauses are still strings, and boil.DebugMode prints the SQL.

**gorm/gen**

- gorm.io/gen has no CLI: a generator program has to be written and run against the live database, and the module needed a dependency pin to build.
- Generated query objects are typed, but they sit on top of GORM, so its defaults (implicit transactions, zero values in Updates) apply and its debug logging is the only SQL view.

**xorm**

- The default name mapper turns ID into i_d, so a mapper has to be set; otherwise setup is small and no generator is needed.
- Conditions and join clauses are strings; Update(struct) skips zero values and Get reports a missing row as found=false, both easy to misread.

## Learning curve (qualitative)

Hand-written judgement of how long a Go developer who knows SQL needs before writing production queries. It is not part of the score. `gentle` = SQL and the Go standard library are enough, `moderate` = one new API or tool to learn, `steep` = a framework, DSL or several tools to learn.

| Library | Learning curve |
|---|---|
| database/sql | gentle |
| sqlc | moderate |
| sqlx | gentle |
| gorm | moderate |
| ent | steep |
| go-jet | moderate |
| pgx | moderate |
| pgx_pool | gentle |
| bun | moderate |
| beego | moderate |
| pop | moderate |
| rel | moderate |
| sqlboiler | moderate |
| gorm/gen | steep |
| xorm | moderate |

**database/sql**

- SQL and the standard library are enough; the API is small and stable.

**sqlc**

- Write SQL plus sqlc query annotations and configuration; the generated API itself is trivial.

**sqlx**

- Anyone who knows database/sql already knows most of it; the db struct tag is the only new concept.

**gorm**

- Large API with several ways to do one thing (Save, Updates, Select, Omit); the defaults matter for correctness.

**ent**

- A schema DSL, the generator, edges and eager loading are all new concepts on top of SQL.

**go-jet**

- SQL knowledge transfers directly; the builder API and the generator are the new parts.

**pgx**

- A native API unlike database/sql (pgxpool, Rows, generic collect helpers) to learn on top of SQL.

**pgx_pool**

- Same shape as database/sql (Query, Next, Scan), so the step up from it is small.

**bun**

- A SQL-shaped query builder plus model tags and relations; close to SQL but its own vocabulary.

**beego**

- The ORM is a subset of a framework with its own conventions (aliases, QuerySeter lookups); the documentation is spread over the framework.

**pop**

- Part of the Buffalo toolkit, so some documentation assumes the framework; the query API itself is small.

**rel**

- Repository, changeset and association concepts to learn; smaller community means fewer worked examples.

**sqlboiler**

- The generator and its config are the main new thing; the query mods (qm) are small and typed.

**gorm/gen**

- Needs GORM first, then the generator and its generated query API.

**xorm**

- A session-style API with its own conventions (mappers, Cols, Table plus Join); documentation is thinner than GORM.

## Community support (facts)

Repository facts read on 2026-10-24, with no rating and no part in the score; they go stale, so re-check before relying on them. Open issues exclude pull requests.

| Library | Repository | Stars | Last release | Last commit | Open issues |
|---|---|---|---|---|---|
| database/sql | golang/go | 139506 | 2026-10-20 | 2026-10-23 | 9834 |
| sqlc | sqlc-dev/sqlc | 18406 | 2026-04-22 | 2026-10-23 | 585 |
| sqlx | jmoiron/sqlx | 17739 | 2024-04-23 | 2024-05-30 | 317 |
| gorm | go-gorm/gorm | 39986 | 2026-06-25 | 2026-10-23 | 457 |
| ent | ent/ent | 17208 | 2026-03-23 | 2026-10-14 | 502 |
| go-jet | go-jet/jet | 3820 | 2026-10-06 | 2026-10-09 | 33 |
| pgx | jackc/pgx | 14354 | 2026-09-07 | 2026-10-21 | 182 |
| pgx_pool | jackc/pgx | 14354 | 2026-09-07 | 2026-10-21 | 182 |
| bun | uptrace/bun | 5002 | 2026-10-03 | 2026-10-03 | 27 |
| beego | beego/beego | 32429 | 2026-04-03 | 2026-03-09 | 7 |
| pop | gobuffalo/pop | 1525 | 2026-09-16 | 2026-10-18 | 91 |
| rel | go-rel/rel | 786 | 2026-08-15 | 2026-09-07 | 30 |
| sqlboiler | aarondl/sqlboiler | 6979 | 2026-01-01 | 2026-07-12 | 98 |
| gorm/gen | go-gorm/gen | 2561 | 2026-08-25 | 2026-09-02 | 16 |
| xorm | gitea.com/xorm/xorm | 409 | 2026-09-30 | 2026-10-23 | 399 |

- **database/sql**: Counts cover the whole Go project, not database/sql alone; the release date is the go1.27.3 tag commit (the repository publishes no GitHub releases).
- **pgx**: Same repository as the pgx_pool entry.
- **pgx_pool**: Same repository as the pgx entry.
- **beego**: The v2.3.10 release was published after the last commit on master, which is why the release date is later. Module is github.com/beego/beego/v2; the repository covers the whole web framework, not only the ORM.
- **rel**: The PostgreSQL adapter go-rel/postgres is a separate module that has had no release since 2024-05.
- **sqlboiler**: The release date is the v4.19.7 tag commit; the latest GitHub release is v4.19.5 (2025-06-27).
- **gorm/gen**: The release date is the v0.3.29 tag commit.
- **xorm**: Hosted on Gitea rather than GitHub; figures come from the Gitea API (open issues exclude pull requests).
