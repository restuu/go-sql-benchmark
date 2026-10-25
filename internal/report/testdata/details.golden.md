# Go SQL library benchmark: detailed report

The ranking, how to run the benchmark, the environment and the methodology are in the [summary](README.md); its [glossary](README.md#glossary) explains the technical terms. Numbers are only comparable on the machine they were measured on, and they are not final.

## Reading the tables

Each table is one scenario under one load profile (a case); each row is one library. Terms such as p95 and B/op are also explained in the [glossary](README.md#glossary) of the summary.

- `Library`: the library measured.
- `p50`, `p95`, `p99`: how long one call took. Sort all the calls from fastest to slowest: p50 is the middle one, p95 is the time 95 out of 100 calls beat, p99 the time 99 out of 100 beat. Lower is better; p95 is the figure used for the score.
- `ops/s`: operations (calls) finished per second across all workers. Higher is better; not part of the score.
- `B/op`: bytes of memory allocated per call. Lower is better.
- `allocs/op`: how many separate memory allocations one call makes. Lower is usually better; not part of the score.
- `CPU ns/op`: processor time spent per call, in nanoseconds (billionths of a second). Lower is better.
- `err %`: the share of calls that failed, as a percentage.
- `p95 vs database/sql`: this library's p95 compared with database/sql's p95 in the same table, as a percentage; `+20.0%` is 20% slower, `-20.0%` is 20% faster.
- `p95 spread` is (max - min) / median of the repeats' p95; `*` marks a spread above 10%, so read that row as approximate.
- `!` after a library means an error rate above 1%: the case is shown but not ranked.
- `[n]` points to the numbered note under the table.
- `n/a (...)` means the library cannot express the scenario; the case is not ranked.

## Feature matrix

Which scenarios each library expresses with its own query API. `n/a` means it needs hand-written SQL for that scenario.

| Library | insert | update | delete | select | join | union |
|---|---|---|---|---|---|---|
| database/sql | native | native | native | native | native | native |
| sqlx | native | native | native | native | native | native |
| ent | native | native | native | native | native [1] | n/a |

1. ent JOIN is edge eager-loading: two queries, not one SQL JOIN.

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

## Pros and cons

### database/sql

<https://pkg.go.dev/database/sql>

**Pros**

- Reference implementation.

**Cons**

- Hand-written scanning.

### sqlx

<https://github.com/jmoiron/sqlx>

**Cons**

- Unmaintained.

### ent

<https://github.com/ent/ent>

**Pros**

- Typed graph API.

## Flexibility (qualitative)

Hand-written assessment of how each library handles dynamic queries, for example a search endpoint that adds conditions, joins or ordering only when a parameter is present. It is not part of the score, the Top 5 or the stability check, and no query composition was benchmarked.

Levels: `native` = the library's own API composes it; `partial` = works with caveats (see notes); `raw SQL` = needs string SQL or an escape hatch; `static only` = the query is fixed at build time, so use several queries.

| Library | optional filters | dynamic ordering | conditional join | dynamic projection | in-list / batch | dynamic pagination |
|---|---|---|---|---|---|---|
| database/sql | native | native | native | native | partial | native |
| ent | native | native | native | partial | native | native |
| sqlx | raw SQL | raw SQL | raw SQL | raw SQL | native | static only |

**database/sql**

- Dynamic SQL is built by hand with a | pipe.

**ent**

- Predicates compose into a slice.

**sqlx**

- unverified: named queries only.

## Ease of use (qualitative)

Hand-written judgement of the day-to-day work a library asks for. It is not part of the score, the Top 5 or the stability check. Levels for setup effort, query writing and type safety: `good` = little work or checked early, `fair` = some manual work or checked only partly, `poor` = mostly manual or checked only at run time. Levels for debugging: `good` = the SQL that ran and its errors are visible with little setup, `fair` = visible after configuring logging or hooks, `poor` = hard to see what ran.

Criteria: setup effort = codegen, config and boilerplate before the first query; query writing = code a typical query takes; type safety = how many mistakes the compiler catches; debugging = SQL visibility, error messages and logging hooks.

| Library | setup effort | query writing | type safety | debugging |
|---|---|---|---|---|
| database/sql | good | poor | poor | good |
| ent | poor | good | good | fair |
| sqlx | good | fair | poor | good |

**database/sql**

- Ease note for database/sql with a | pipe.

**ent**

- Ease note for ent with a | pipe.

**sqlx**

- Ease note for sqlx with a | pipe.

## Learning curve (qualitative)

Hand-written judgement of how long a Go developer who knows SQL needs before writing production queries. It is not part of the score. `gentle` = SQL and the Go standard library are enough, `moderate` = one new API or tool to learn, `steep` = a framework, DSL or several tools to learn.

| Library | Learning curve |
|---|---|
| database/sql | gentle |
| ent | steep |
| sqlx | gentle |

**database/sql**

- Curve note for database/sql.

**ent**

- Curve note for ent.

**sqlx**

- Curve note for sqlx.

## Community support (facts)

Repository facts read on 2026-10-24, with no rating and no part in the score; they go stale, so re-check before relying on them. Open issues exclude pull requests.

| Library | Repository | Stars | Last release | Last commit | Open issues |
|---|---|---|---|---|---|
| database/sql | golang/go | 130000 | 2026-10-01 | 2026-10-23 | 9000 |
| ent | ent/ent | 17000 | 2026-08-01 | 2026-10-20 | 400 |
| sqlx | jmoiron/sqlx | 16000 | 2024-05-01 | 2024-06-01 | 300 |

- **database/sql**: Counts cover the whole Go project.
## Qualitative ranking (judgement)

A second, separate ranking of the three sections above. It is not part of the measured score, the Top 5 or the stability check. The levels behind it are the author's judgement and the score only turns them into numbers, so it looks more objective than it is.

Weights: flexibility 0.40, ease of use 0.35 (the learning curve counts as one more criterion), community 0.25. Levels count linearly from 1 (best) to 0 (worst). Community is the mean of last-commit recency and last-release recency (1 within 30 days of 2026-10-24, 0 after 730 days) and open issues per 1,000 stars (1 with none, 0 at 100 or more); stars are left out because they reward age. Equal totals share a rank and are listed alphabetically.

| Rank | Library | Total | Flexibility | Ease of use | Community |
|---|---|---|---|---|---|
| 1 | database/sql | 0.78 | 0.94 | 0.60 | 0.77 |
| 2 | ent | 0.78 | 0.94 | 0.50 | 0.90 |
| 3 | sqlx | 0.47 | 0.39 | 0.70 | 0.27 |

## Combined ranking (judgement)

A third ranking that adds each library's place in the measured ranking (Top 5 and "Also measured") to its place in the qualitative ranking above; the lowest sum is first. Both rankings count equally. Equal sums share a rank and are listed alphabetically. Only places are added, so how far apart two libraries are in score is lost, and a sum of a few places is within noise. The qualitative half is the author's judgement, so this ranking is too.

| Rank | Library | Measured rank | Qualitative rank | Sum |
|---|---|---|---|---|
| 1 | database/sql | 1 | 1 | 2 |
| 2 | ent | 3 | 2 | 5 |
| 2 | sqlx | 2 | 3 | 5 |
