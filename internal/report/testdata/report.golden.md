# Go SQL library benchmark

Run started 2026-10-21T04:58:02Z (UTC). Numbers are only comparable on the same machine.

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
| ent | v0.14.6 | - | v0.14.6 |

## Run configuration

Repeats per cell: 2 (the median is reported). Seed: 1792558680234027000.

| Profile | Workers | Pacing | Warm-up | Window | DELETE ids |
|---|---:|---:|---:|---:|---:|
| low | 4 | 10ms | 1s | 3s | 5000 |
| stress | 100 | unpaced | 2s | 3s | 100000 |

## Methodology

- Fixture methodology line one.
- Fixture methodology line two with a | pipe.

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
| database/sql | 2.8 ms | 9.9 ms | 24 ms | 399 | 874 | 14.0 | 195348 | 0.00 | baseline | 26.1% * |
| sqlx | 2.3 ms | 9.0 ms | 17 ms | 399 | 873 | 14.0 | 233713 | 0.00 | -9.4% | 0.0% |
| ent | 2.6 ms | 9.3 ms | 14 ms | 399 | 4389 | 98.0 | 326083 | 0.00 | -6.5% | 0.0% |

#### update

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 2.4 ms | 8.3 ms | 14 ms | 399 | 930 | 16.0 | 222732 | 0.00 | baseline | 0.0% |
| sqlx | 3.0 ms | 9.2 ms | 14 ms | 399 | 929 | 16.0 | 288111 | 0.00 | +11.5% | 0.0% |
| ent | 3.6 ms | 10 ms | 17 ms | 399 | 4918 | 101.0 | 321677 | 0.00 | +22.9% | 0.0% |

#### delete

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 2.3 ms | 7.3 ms | 14 ms | 399 | 474 | 11.0 | 176521 | 0.00 | baseline | 0.0% |
| sqlx | 3.1 ms | 9.9 ms | 13 ms | 399 | 473 | 11.0 | 248954 | 0.00 | +34.5% | 0.0% |
| ent | 2.5 ms | 8.5 ms | 16 ms | 399 | 2121 | 44.0 | 260175 | 0.00 | +15.9% | 0.0% |

#### select

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 989 µs | 5.8 ms | 12 ms | 398 | 1976 | 50.2 | 255293 | 0.00 | baseline | 0.0% |
| sqlx | 989 µs | 5.3 ms | 8.9 ms | 399 | 2388 | 55.3 | 361768 | 0.00 | -8.0% | 0.0% |
| ent | 1.0 ms | 6.4 ms | 11 ms | 396 | 5579 | 145.1 | 313550 | 0.00 | +11.8% | 0.0% |

#### join

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 1.1 ms | 5.6 ms | 11 ms | 399 | 3613 | 184.6 | 423869 | 0.00 | baseline | 0.0% |
| sqlx | 1.4 ms | 6.7 ms | 11 ms | 399 | 9162 | 207.6 | 507652 | 0.00 | +21.0% | 0.0% |
| ent [1] | 1.3 ms | 9.8 ms | 18 ms | 399 | 13858 | 346.6 | 421037 | 0.00 | +75.6% | 0.0% |

1. ent JOIN is edge eager-loading: two queries, not one SQL JOIN.

#### union

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 1.2 ms | 5.2 ms | 10 ms | 399 | 5300 | 269.5 | 214091 | 0.00 | baseline | 0.0% |
| sqlx | 1.3 ms | 7.2 ms | 12 ms | 399 | 15888 | 358.5 | 452082 | 0.00 | +37.0% | 0.0% |
| ent | n/a (unsupported) | - | - | - | - | - | - | - | - | - |

### Profile: stress

#### insert

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 3.1 ms | 6.2 ms | 16 ms | 18414 | 874 | 14.0 | 40335 | 0.00 | baseline | 0.0% |
| sqlx | 3.2 ms | 5.8 ms | 11 ms | 19230 | 873 | 14.0 | 45008 | 0.00 | -5.3% | 0.0% |
| ent | 3.5 ms | 6.9 ms | 203 ms | 15613 | 4389 | 98.1 | 57365 | 0.00 | +12.7% | 0.0% |

#### update

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 3.6 ms | 7.6 ms | 20 ms | 16128 | 930 | 16.0 | 39679 | 0.00 | baseline | 0.0% |
| sqlx | 3.5 ms | 6.6 ms | 18 ms | 17090 | 928 | 16.0 | 46633 | 0.00 | -13.7% | 0.0% |
| ent | 3.9 ms | 10.0 ms | 205 ms | 12997 | 4920 | 101.0 | 45528 | 0.00 | +31.1% | 0.0% |

#### delete

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 2.9 ms | 5.0 ms | 202 ms | 18740 | 473 | 11.0 | 46400 | 0.00 | baseline | 0.0% |
| sqlx | 3.0 ms | 5.3 ms | 203 ms | 17626 | 473 | 11.0 | 45223 | 0.00 | +6.4% | 0.0% |
| ent | 4.1 ms | 12 ms | 204 ms | 12015 | 2121 | 44.0 | 38185 | 0.00 | +148.3% | 0.0% |

#### select

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 909 µs | 2.0 ms | 22 ms | 31041 | 1947 | 50.0 | 40999 | 0.00 | baseline | 0.0% |
| sqlx | 877 µs | 1.9 ms | 7.2 ms | 26375 | 2348 | 55.0 | 44726 | 0.00 | -4.4% | 0.0% |
| ent | 1.2 ms | 3.6 ms | 201 ms | 24245 | 5564 | 145.0 | 44849 | 0.00 | +83.5% | 0.0% |

#### join

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 1.3 ms | 3.8 ms | 201 ms | 22047 | 3558 | 183.7 | 88866 | 0.00 | baseline | 0.0% |
| sqlx | 1.3 ms | 3.0 ms | 201 ms | 21705 | 9106 | 206.7 | 92975 | 0.00 | -22.5% | 0.0% |
| ent [1] | 2.1 ms | 4.3 ms | 203 ms | 12636 | 13820 | 345.9 | 103165 | 0.00 | +10.7% | 0.0% |

1. ent JOIN is edge eager-loading: two queries, not one SQL JOIN.

#### union

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| database/sql | 1.8 ms | 4.9 ms | 202 ms | 17341 | 5262 | 269.2 | 57691 | 0.00 | baseline | 0.0% |
| sqlx | 1.7 ms | 4.4 ms | 202 ms | 17675 | 15856 | 358.2 | 67541 | 0.00 | -10.2% | 0.0% |
| ent | n/a (unsupported) | - | - | - | - | - | - | - | - | - |

## Top 5

Score is the mean rank over p95, B/op and CPU ns/op in every ranked cell, weighted (p95 1, B/op 1, CPU ns/op 1); lower is better.

Libraries with equal scores share a rank.

| Rank | Library | Score | Ranked cells |
|---:|---|---:|---:|
| 1 | database/sql | 1.47 | 12/12 |
| 2 | sqlx | 1.75 | 12/12 |
| 3 | ent | 2.73 | 10/12 |

**database/sql**

- fastest p95 in 5 of 12 ranked cells
- weakest on p95: mean rank 1.7
- slowest cell: insert/low (p95 rank 3 of 3)

**sqlx**

- fastest p95 in 7 of 12 ranked cells
- weakest on CPU ns/op: mean rank 2.2
- slowest cell: delete/low (p95 rank 3 of 3)

**ent**

- weakest on B/op: mean rank 3.0
- slowest cell: update/low (p95 rank 3 of 3)
- judged on 10 of 12 cells

## Pros and cons

### database/sql

**Pros**

- Reference implementation.

**Cons**

- Hand-written scanning.

### sqlx

**Cons**

- Unmaintained.

### ent

**Pros**

- Typed graph API.

