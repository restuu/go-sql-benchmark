# Go SQL library benchmark

Run started 2026-10-24T01:50:50Z (UTC). Numbers are only comparable on the same machine.

## Environment

- OS/arch: darwin/arm64
- Go: go1.27.3
- CPUs: 8 (GOMAXPROCS 8)
- PostgreSQL: 15.19
- Driver shared by every database/sql-based library: pgx v5.11.0

| Library | Version | Label | Version used |
|---|---|---|---|
| database/sql | - | - | standard library |
| sqlx | v1.4.0 | no updates since 2024 | v1.4.0 |
| sqlc | - | - | generator v1.31.1, pinned in tools/go.mod |
| ent | v0.14.6 | - | v0.14.6 |
| go-jet | v2.16.1 | - | v2.16.1 |
| gorm | v1.31.2 | - | v1.31.2 |

## Run configuration

Repeats per cell: 3 (the median is reported). Seed: 1792806648074747000.

| Profile | Workers | Pacing | Warm-up | Window | DELETE ids |
|---|---:|---:|---:|---:|---:|
| low | 4 | 10ms | 1s | 5s | 5000 |
| stress | 100 | unpaced | 2s | 10s | 400000 |

## Methodology

- Every database/sql-based library uses the same pgx/stdlib driver (pgx), never lib/pq.
- Numbers are only comparable on the machine and run they were measured on; the host is listed above.
- Each cell is the median of the repeats listed in the run configuration. Before the matrix runs, every adapter is checked against the database/sql reference on returned values, and every read scans every selected column.
- Latency is the per-call service time of successful operations with exact nearest-rank percentiles; it is not corrected for coordinated omission.
- B/op, allocs/op and CPU ns/op are deltas over the measured window divided by ops and include harness and GC cost.
- B/op and allocs/op also include each library's idiomatic result handling: sqlx, sqlc, gorm, ent and jet materialise rows, while the database/sql reference scans into locals, so part of the difference is the cost of that handling.
- A scenario counts as supported only when the library's own query API (or, for the SQL-first sqlc and sqlx, its SQL) expresses it; needing the raw-SQL escape hatch makes the cell n/a and it is not ranked.
- A library's score averages the ranks of its own ranked cells only, and a rank is absolute (1 is best, N is worst among the libraries in that cell). A library with n/a or excluded cells is therefore judged on fewer, possibly easier, scenarios; the Ranked cells column shows how many.
- Stress-profile p99 on macOS includes a constant 100-200 ms host-to-VM stall (Postgres runs in an OrbStack VM) and is not a property of the library.
- Run-to-run stability (two full runs on this machine, 2026-10-24, see .claude/PRPs/reports/full-run-report.md): p95 differed by 20% or less in 62 of 72 cells; the other 10 are all low-profile cells and reached up to 48%, while every stress cell stayed under 9%. The same five libraries made the top 5 in both runs, but sqlx and sqlc swapped ranks 2 and 3, so the order of database/sql, sqlx and sqlc is within noise. The cause is environmental (loaded macOS host, host-VM network crossing, WAL fsync on the VM disk); see .claude/PRPs/reports/p95-stability-report.md.

Reading the tables:

- `p95 vs database/sql` is relative to the baseline's p95 in the same table; negative is faster.
- `p95 spread` is (max - min) / median of the repeats' p95; `*` marks a spread above 10%.
- `!` after a library means an error rate above 1%: the cell is shown but not ranked.
- `[n]` points to the numbered note under the table.
- `n/a (...)` means the library cannot express the scenario; the cell is not ranked.

## Results

### Profile: low

#### insert

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 2.6 ms | 5.3 ms | 8.7 ms | 399 | 873 | 14.0 | 229664 | 0.00 | baseline | 39.9% * |
| sqlx | 2.3 ms | 4.1 ms | 6.4 ms | 399 | 963 | 14.3 | 209520 | 0.00 | -22.9% | 10.0% * |
| sqlc | 2.6 ms | 4.8 ms | 7.6 ms | 399 | 1031 | 15.2 | 255189 | 0.00 | -9.5% | 14.0% * |
| ent | 2.4 ms | 4.5 ms | 6.9 ms | 399 | 4432 | 98.2 | 284626 | 0.00 | -15.5% | 5.2% |
| go-jet | 2.6 ms | 4.8 ms | 9.3 ms | 399 | 3577 | 87.2 | 258217 | 0.00 | -8.8% | 126.5% * |
| gorm [1] | 2.5 ms | 5.2 ms | 7.9 ms | 399 | 6878 | 77.2 | 315801 | 0.00 | -1.9% | 51.1% * |

1. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.

#### update

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 2.8 ms | 5.4 ms | 9.2 ms | 399 | 929 | 16.0 | 261750 | 0.00 | baseline | 55.6% * |
| sqlx | 2.3 ms | 4.3 ms | 6.1 ms | 399 | 929 | 16.0 | 194610 | 0.00 | -21.7% | 30.8% * |
| sqlc | 2.5 ms | 4.5 ms | 7.4 ms | 399 | 1049 | 16.0 | 223670 | 0.00 | -16.8% | 4.9% |
| ent | 2.5 ms | 5.1 ms | 7.6 ms | 399 | 4965 | 101.1 | 257390 | 0.00 | -7.1% | 24.6% * |
| go-jet | 2.3 ms | 6.0 ms | 11 ms | 399 | 3989 | 100.1 | 238327 | 0.00 | +11.0% | 80.9% * |
| gorm [1] | 2.7 ms | 5.5 ms | 7.8 ms | 399 | 7902 | 80.1 | 293546 | 0.00 | +1.4% | 47.8% * |

1. gorm runs with SkipDefaultTransaction (by default every write is wrapped in a transaction) and Updates takes an explicit column list, because Updates(struct) skips zero values.

#### delete

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 2.6 ms | 5.5 ms | 8.7 ms | 399 | 518 | 11.2 | 232609 | 0.00 | baseline | 21.4% * |
| sqlx | 2.4 ms | 4.0 ms | 6.2 ms | 399 | 473 | 11.0 | 186117 | 0.00 | -27.6% | 36.6% * |
| sqlc | 2.6 ms | 4.3 ms | 7.5 ms | 399 | 525 | 11.2 | 226682 | 0.00 | -21.2% | 41.2% * |
| ent | 2.6 ms | 5.0 ms | 7.0 ms | 399 | 2121 | 44.0 | 256924 | 0.00 | -9.1% | 30.6% * |
| go-jet | 2.5 ms | 3.9 ms | 6.7 ms | 399 | 1553 | 36.0 | 248402 | 0.00 | -28.4% | 113.5% * |
| gorm [1] | 2.7 ms | 6.1 ms | 8.0 ms | 399 | 4181 | 49.0 | 277213 | 0.00 | +10.3% | 35.5% * |

1. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.

#### select

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 827 µs | 2.0 ms | 5.1 ms | 399 | 1952 | 50.0 | 255316 | 0.00 | baseline | 21.0% * |
| sqlx | 839 µs | 2.3 ms | 6.1 ms | 399 | 2364 | 55.1 | 276526 | 0.00 | +13.0% | 9.6% |
| sqlc | 797 µs | 1.8 ms | 4.8 ms | 399 | 1977 | 51.0 | 269226 | 0.00 | -10.2% | 20.5% * |
| ent | 887 µs | 2.1 ms | 5.2 ms | 399 | 5575 | 145.1 | 344560 | 0.00 | +6.4% | 45.6% * |
| go-jet | 968 µs | 2.9 ms | 6.4 ms | 399 | 12089 | 221.7 | 422074 | 0.00 | +42.3% | 23.7% * |
| gorm | 925 µs | 2.3 ms | 5.7 ms | 399 | 6071 | 100.5 | 357973 | 0.00 | +16.0% | 20.5% * |

#### join

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 1.1 ms | 3.0 ms | 7.7 ms | 399 | 3641 | 184.3 | 438640 | 0.00 | baseline | 46.1% * |
| sqlx | 1.1 ms | 2.5 ms | 6.7 ms | 399 | 9195 | 207.5 | 513218 | 0.00 | -16.2% | 15.0% * |
| sqlc | 1.1 ms | 2.3 ms | 4.8 ms | 399 | 8104 | 193.7 | 520894 | 0.00 | -23.9% | 125.8% * |
| ent [1] | 1.4 ms | 3.6 ms | 7.7 ms | 399 | 13880 | 346.5 | 491556 | 0.00 | +19.7% | 21.8% * |
| go-jet | 1.2 ms | 3.1 ms | 7.1 ms | 399 | 37471 | 692.0 | 589368 | 0.00 | +1.2% | 28.2% * |
| gorm | 1.1 ms | 2.8 ms | 5.7 ms | 399 | 17419 | 407.2 | 635830 | 0.00 | -8.5% | 17.3% * |

1. ent JOIN is edge eager-loading (WithCustomer): two queries, orders then their customers, not one SQL JOIN. Read its JOIN numbers with that in mind.

#### union

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 1.4 ms | 2.7 ms | 6.5 ms | 399 | 5335 | 269.8 | 331930 | 0.00 | baseline | 7.2% |
| sqlx | 1.4 ms | 2.7 ms | 6.6 ms | 399 | 15927 | 358.9 | 412930 | 0.00 | +0.0% | 0.7% |
| sqlc | 1.3 ms | 2.9 ms | 6.7 ms | 399 | 14744 | 315.8 | 387638 | 0.00 | +5.5% | 57.3% * |
| ent [1] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| go-jet | 1.5 ms | 3.3 ms | 6.6 ms | 399 | 48864 | 794.1 | 617502 | 0.00 | +21.6% | 4.4% |
| gorm [2] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |

1. n/a: ent's query API cannot express UNION, and using raw SQL would not measure ent. The cell is not ranked.
2. n/a: gorm's query API cannot express UNION without raw SQL. The cell is not ranked.

### Profile: stress

#### insert

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 2.7 ms | 4.2 ms | 9.2 ms | 21234 | 872 | 14.0 | 40680 | 0.00 | baseline | 9.4% |
| sqlx | 2.7 ms | 4.2 ms | 11 ms | 21273 | 872 | 14.0 | 40890 | 0.00 | -0.2% | 3.4% |
| sqlc | 2.7 ms | 4.2 ms | 9.0 ms | 21640 | 984 | 15.0 | 40477 | 0.00 | -1.5% | 50.4% * |
| ent | 3.0 ms | 4.6 ms | 22 ms | 19685 | 4388 | 98.0 | 50150 | 0.00 | +8.5% | 32.6% * |
| go-jet | 2.8 ms | 4.4 ms | 13 ms | 21008 | 3528 | 87.0 | 49101 | 0.00 | +4.6% | 21.3% * |
| gorm [1] | 3.0 ms | 4.8 ms | 9.7 ms | 19736 | 6827 | 77.0 | 53021 | 0.00 | +13.6% | 7.5% |

1. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.

#### update

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 2.8 ms | 4.4 ms | 9.4 ms | 20136 | 928 | 16.0 | 41260 | 0.00 | baseline | 3.5% |
| sqlx | 2.9 ms | 4.5 ms | 19 ms | 19981 | 928 | 16.0 | 40499 | 0.00 | +1.3% | 4.2% |
| sqlc | 3.2 ms | 5.1 ms | 10 ms | 18526 | 1048 | 16.0 | 40971 | 0.00 | +16.1% | 36.6% * |
| ent | 3.1 ms | 5.0 ms | 11 ms | 19218 | 4919 | 101.0 | 46185 | 0.00 | +12.1% | 12.0% * |
| go-jet | 3.1 ms | 5.0 ms | 12 ms | 19547 | 3943 | 100.0 | 49752 | 0.00 | +13.9% | 14.2% * |
| gorm [1] | 3.2 ms | 5.0 ms | 14 ms | 19604 | 7856 | 80.0 | 48658 | 0.00 | +12.2% | 8.8% |

1. gorm runs with SkipDefaultTransaction (by default every write is wrapped in a transaction) and Updates takes an explicit column list, because Updates(struct) skips zero values.

#### delete

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 2.8 ms | 4.4 ms | 9.1 ms | 20939 | 472 | 11.0 | 39535 | 0.00 | baseline | 3.8% |
| sqlx | 2.7 ms | 4.2 ms | 13 ms | 20850 | 472 | 11.0 | 40352 | 0.00 | -4.3% | 3.1% |
| sqlc | 2.8 ms | 4.3 ms | 11 ms | 20459 | 480 | 11.0 | 39915 | 0.00 | -1.6% | 17.3% * |
| ent | 2.9 ms | 4.5 ms | 23 ms | 20397 | 2120 | 44.0 | 42832 | 0.00 | +1.8% | 6.8% |
| go-jet | 2.9 ms | 4.7 ms | 19 ms | 19820 | 1552 | 36.0 | 43384 | 0.00 | +6.1% | 8.3% |
| gorm [1] | 3.0 ms | 4.8 ms | 11 ms | 20530 | 4176 | 49.0 | 44100 | 0.00 | +10.2% | 14.2% * |

1. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.

#### select

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 860 µs | 1.7 ms | 7.5 ms | 32862 | 1947 | 50.0 | 44511 | 0.00 | baseline | 0.4% |
| sqlx | 851 µs | 1.8 ms | 18 ms | 32761 | 2347 | 55.0 | 45242 | 0.00 | +2.2% | 4.0% |
| sqlc | 834 µs | 1.7 ms | 15 ms | 32738 | 1971 | 51.0 | 45488 | 0.00 | -0.2% | 6.8% |
| ent | 907 µs | 1.8 ms | 12 ms | 31849 | 5564 | 145.0 | 50239 | 0.00 | +6.6% | 6.6% |
| go-jet | 953 µs | 2.0 ms | 13 ms | 31681 | 12006 | 221.0 | 59241 | 0.00 | +15.4% | 4.3% |
| gorm | 889 µs | 1.9 ms | 201 ms | 31717 | 5996 | 100.0 | 50604 | 0.00 | +9.9% | 20.8% * |

#### join

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 1.2 ms | 2.7 ms | 201 ms | 25346 | 3560 | 183.7 | 91296 | 0.00 | baseline | 2.5% |
| sqlx | 1.3 ms | 2.8 ms | 201 ms | 25441 | 9108 | 206.7 | 95328 | 0.00 | +3.4% | 10.3% * |
| sqlc | 1.3 ms | 2.8 ms | 201 ms | 25276 | 8015 | 192.9 | 92796 | 0.00 | +5.1% | 13.6% * |
| ent [1] | 1.9 ms | 3.8 ms | 203 ms | 15198 | 13820 | 345.9 | 108629 | 0.00 | +42.9% | 8.0% |
| go-jet | 1.3 ms | 2.9 ms | 201 ms | 23303 | 37360 | 691.1 | 103377 | 0.00 | +8.2% | 96.1% * |
| gorm | 1.4 ms | 3.1 ms | 201 ms | 23072 | 17289 | 406.2 | 119634 | 0.00 | +15.2% | 48.6% * |

1. ent JOIN is edge eager-loading (WithCustomer): two queries, orders then their customers, not one SQL JOIN. Read its JOIN numbers with that in mind.

#### union

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 1.8 ms | 5.3 ms | 202 ms | 20660 | 5263 | 269.2 | 53296 | 0.00 | baseline | 1.4% |
| sqlx | 1.8 ms | 5.1 ms | 202 ms | 18684 | 15858 | 358.3 | 64156 | 0.00 | -5.4% | 6.4% |
| sqlc | 1.7 ms | 5.0 ms | 202 ms | 19841 | 14675 | 315.2 | 57079 | 0.00 | -6.5% | 10.0% |
| ent [1] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| go-jet | 2.0 ms | 5.5 ms | 202 ms | 17426 | 48745 | 793.2 | 102697 | 0.00 | +3.8% | 11.5% * |
| gorm [2] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |

1. n/a: ent's query API cannot express UNION, and using raw SQL would not measure ent. The cell is not ranked.
2. n/a: gorm's query API cannot express UNION without raw SQL. The cell is not ranked.

## Top 5

Score is the mean rank over p95, B/op and CPU ns/op in every ranked cell, weighted (p95 1, B/op 1, CPU ns/op 1); lower is better.

Libraries with equal scores share a rank.

| Rank | Library | Score | Ranked cells |
|---:|---|---:|---:|
| 1 | database/sql | 2.06 | 12/12 |
| 2 | sqlx | 2.11 | 12/12 |
| 3 | sqlc | 2.33 | 12/12 |
| 4 | ent | 4.27 | 10/12 |
| 5 | go-jet | 4.58 | 12/12 |

**database/sql**

- fastest p95 in 2 of 12 ranked cells
- weakest on p95: mean rank 3.0
- slowest cell: insert/low (p95 rank 6 of 6)

**sqlx**

- fastest p95 in 4 of 12 ranked cells
- weakest on CPU ns/op: mean rank 2.2
- slowest cell: select/low (p95 rank 4 of 6)

**sqlc**

- fastest p95 in 5 of 12 ranked cells
- weakest on B/op: mean rank 2.5
- slowest cell: update/stress (p95 rank 6 of 6)

**ent**

- weakest on B/op: mean rank 4.6
- slowest cell: join/low (p95 rank 6 of 6)
- judged on 10 of 12 cells

**go-jet**

- fastest p95 in 1 of 12 ranked cells
- weakest on B/op: mean rank 4.7
- slowest cell: update/low (p95 rank 6 of 6)

Also measured:

- gorm: score 5.43, ranked cells 10/12

## Pros and cons

### database/sql

**Pros**

- No dependency beyond the driver; it is the reference every other adapter is checked against.
- Full control of the SQL and of what is scanned.

**Cons**

- Hand-written SQL strings and row scanning, with no compile-time check of either.

### sqlx

**Pros**

- Thin layer over database/sql: the SQL stays hand-written and struct scanning is added.
- Every scenario is expressible with plain SQL.

**Cons**

- No updates since 2024.
- Scanning into structs is reflection-based.

### sqlc

**Pros**

- Queries are checked against the schema when the code is generated, and the generated code is typed.
- Plain SQL, so every scenario including UNION is expressible.

**Cons**

- Needs a generation step (make gen-sqlc) and committed generated code.
- Each query lives in an SQL file; there is no runtime query builder.

### ent

**Pros**

- Typed query API generated from a schema definition.

**Cons**

- JOIN is edge eager-loading: two queries rather than one SQL JOIN.
- UNION is not expressible without raw SQL, so that cell is n/a.
- Needs a generation step (make gen-ent) and a large amount of committed generated code.

### go-jet

**Pros**

- Type-safe SQL builder; every scenario, including UNION, is expressed with its own API.

**Cons**

- Code is generated from the live database (make gen-jet needs it running).
- Pulls in lib/pq for array literals, although no adapter uses that driver.

### gorm

**Pros**

- Query API covers INSERT, UPDATE, DELETE, SELECT and JOIN (Joins with a relation produces one SQL JOIN).

**Cons**

- UNION is not expressible without raw SQL, so that cell is n/a.
- Defaults need care to benchmark fairly: every write is wrapped in a transaction unless SkipDefaultTransaction is set, and Save or Updates(struct) misreport or skip zero values and missing rows unless guarded.

