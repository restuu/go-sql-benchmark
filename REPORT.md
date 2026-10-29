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
| pgx_pool | native | native | native | native | native | native |
| pgx | native | native | native | native | native | native |
| database/sql | native | native | native | native | native | native |
| sqlc | native | native | native | native | native | native |
| sqlx | native | native | native | native | native | native |
| sqlboiler | native | native | native | native | native | n/a |
| beego | native | native | native | native | native [1] | n/a |
| rel | native | native | native | native | native | n/a |
| pop | native | native | native | native | native | n/a |
| ent | native | native | native | native | native [2] | n/a |
| go-jet | native | native | native | native | native | native |
| xorm | native | native | native | native | native | native [3] |
| gorm | native | native | native | native | native | n/a |
| bun | native | native | native | native | native | native |
| gorm/gen | native | native | native | native | native | n/a |

1. beego JOIN is RelatedSel on a foreign-key field (one SQL statement, customer loaded with the order).
2. ent JOIN is edge eager-loading (WithCustomer): two queries, orders then their customers, not one SQL JOIN. Read its JOIN numbers with that in mind.
3. xorm UNION is xorm.io/builder's Union (the query builder xorm itself uses) passed to SQL, so it is measured, not n/a.

## Results

### Profile: low

#### insert

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.6 ms | 5.3 ms | 7.9 ms | 399 | 572 | 12.0 | 231911 | 0.00 | +24.3% | 29.7% * |
| pgx | 2.5 ms | 5.0 ms | 7.3 ms | 399 | 570 | 12.0 | 234554 | 0.00 | +17.2% | 39.2% * |
| database/sql | 2.5 ms | 4.2 ms | 6.7 ms | 399 | 919 | 14.2 | 223714 | 0.00 | baseline | 22.4% * |
| sqlc | 2.6 ms | 4.9 ms | 8.0 ms | 399 | 1031 | 15.2 | 220685 | 0.00 | +15.2% | 27.6% * |
| sqlx | 2.5 ms | 4.8 ms | 7.4 ms | 399 | 918 | 14.2 | 222830 | 0.00 | +12.6% | 14.1% * |
| sqlboiler [1] | 2.7 ms | 4.7 ms | 7.2 ms | 399 | 1969 | 36.2 | 336344 | 0.00 | +10.7% | 36.3% * |
| beego | 2.7 ms | 5.3 ms | 9.3 ms | 399 | 2892 | 50.1 | 268291 | 0.00 | +24.8% | 34.0% * |
| rel | 2.5 ms | 4.2 ms | 7.0 ms | 399 | 3148 | 45.3 | 285334 | 0.00 | -0.9% | 9.0% |
| pop | 2.6 ms | 4.4 ms | 7.4 ms | 399 | 5762 | 175.1 | 346028 | 0.00 | +4.7% | 37.4% * |
| ent | 2.5 ms | 4.5 ms | 7.3 ms | 399 | 4432 | 98.2 | 309476 | 0.00 | +7.6% | 23.1% * |
| go-jet | 2.5 ms | 4.9 ms | 8.4 ms | 399 | 3576 | 87.2 | 262160 | 0.00 | +14.9% | 23.7% * |
| xorm | 2.7 ms | 5.0 ms | 7.6 ms | 399 | 4661 | 94.0 | 312138 | 0.00 | +18.8% | 23.5% * |
| gorm [2] | 2.6 ms | 4.1 ms | 6.9 ms | 399 | 6873 | 77.2 | 319625 | 0.00 | -1.9% | 16.9% * |
| bun | 2.5 ms | 5.8 ms | 9.4 ms | 399 | 6087 | 27.2 | 283959 | 0.00 | +37.6% | 42.6% * |
| gorm/gen [3] | 2.6 ms | 5.0 ms | 6.8 ms | 399 | 6799 | 77.1 | 319066 | 0.00 | +19.4% | 12.4% * |

1. sqlboiler's Insert reports no rows affected, so the adapter counts any error-free insert as 1 row.
2. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.
3. gorm/gen's Create reports no rows affected, so the adapter counts any error-free insert as 1 row; it runs with SkipDefaultTransaction like the gorm adapter.

#### update

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.7 ms | 5.1 ms | 7.9 ms | 399 | 639 | 14.1 | 250054 | 0.00 | +18.8% | 17.9% * |
| pgx | 2.5 ms | 4.7 ms | 7.1 ms | 399 | 596 | 14.0 | 211808 | 0.00 | +9.1% | 15.0% * |
| database/sql | 2.5 ms | 4.3 ms | 6.6 ms | 399 | 974 | 16.1 | 218335 | 0.00 | baseline | 25.5% * |
| sqlc | 2.6 ms | 4.9 ms | 8.6 ms | 399 | 1049 | 16.0 | 237226 | 0.00 | +14.0% | 51.3% * |
| sqlx | 2.4 ms | 4.6 ms | 7.1 ms | 399 | 929 | 16.0 | 216542 | 0.00 | +6.7% | 1.4% |
| sqlboiler | 2.6 ms | 4.7 ms | 7.9 ms | 399 | 1231 | 20.1 | 246300 | 0.00 | +10.7% | 22.0% * |
| beego | 2.8 ms | 4.9 ms | 7.3 ms | 399 | 2042 | 32.0 | 260632 | 0.00 | +14.4% | 63.4% * |
| rel | 2.6 ms | 4.9 ms | 7.3 ms | 399 | 2465 | 33.0 | 254302 | 0.00 | +14.2% | 25.6% * |
| pop | 2.7 ms | 4.7 ms | 7.9 ms | 399 | 5899 | 159.2 | 301280 | 0.00 | +10.0% | 52.9% * |
| ent | 2.6 ms | 4.7 ms | 7.0 ms | 399 | 4919 | 101.0 | 254067 | 0.00 | +9.1% | 10.4% * |
| go-jet | 2.7 ms | 4.5 ms | 7.6 ms | 399 | 3942 | 100.0 | 267922 | 0.00 | +5.6% | 82.9% * |
| xorm | 2.6 ms | 5.0 ms | 7.3 ms | 399 | 6027 | 157.1 | 286890 | 0.00 | +16.1% | 5.9% |
| gorm [1] | 2.7 ms | 5.3 ms | 8.1 ms | 399 | 7857 | 80.0 | 287075 | 0.00 | +23.7% | 28.7% * |
| bun | 2.6 ms | 5.1 ms | 8.5 ms | 399 | 5961 | 19.0 | 241850 | 0.00 | +18.3% | 10.7% * |
| gorm/gen | 2.9 ms | 5.1 ms | 8.1 ms | 399 | 15605 | 290.2 | 429151 | 0.00 | +20.3% | 19.4% * |

1. gorm runs with SkipDefaultTransaction (by default every write is wrapped in a transaction) and Updates takes an explicit column list, because Updates(struct) skips zero values.

#### delete

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.4 ms | 4.3 ms | 6.7 ms | 399 | 411 | 9.0 | 200013 | 0.00 | -15.1% | 49.0% * |
| pgx | 2.6 ms | 4.9 ms | 7.8 ms | 399 | 412 | 9.0 | 210500 | 0.00 | -2.6% | 8.1% |
| database/sql | 2.5 ms | 5.0 ms | 8.4 ms | 399 | 473 | 11.0 | 219706 | 0.00 | baseline | 13.3% * |
| sqlc | 2.3 ms | 4.6 ms | 7.1 ms | 399 | 480 | 11.0 | 196522 | 0.00 | -8.2% | 17.2% * |
| sqlx | 2.7 ms | 5.0 ms | 8.8 ms | 399 | 518 | 11.2 | 227854 | 0.00 | -0.5% | 29.7% * |
| sqlboiler | 2.5 ms | 4.9 ms | 7.3 ms | 399 | 646 | 13.2 | 215960 | 0.00 | -3.5% | 12.1% * |
| beego | 2.8 ms | 6.0 ms | 9.1 ms | 399 | 886 | 19.2 | 232100 | 0.00 | +20.0% | 7.5% |
| rel | 2.6 ms | 5.0 ms | 10 ms | 399 | 881 | 21.0 | 230540 | 0.00 | +0.2% | 57.9% * |
| pop [1] | 2.6 ms | 6.3 ms | 12 ms | 399 | 1134 | 32.2 | 219384 | 0.00 | +25.6% | 28.3% * |
| ent | 2.5 ms | 4.9 ms | 7.0 ms | 399 | 2166 | 44.2 | 229910 | 0.00 | -3.2% | 31.0% * |
| go-jet | 2.7 ms | 6.2 ms | 9.2 ms | 399 | 1553 | 36.0 | 239078 | 0.00 | +22.9% | 29.0% * |
| xorm | 2.5 ms | 5.5 ms | 9.4 ms | 399 | 3691 | 102.0 | 246457 | 0.00 | +9.1% | 18.9% * |
| gorm [2] | 2.6 ms | 4.7 ms | 7.5 ms | 399 | 4180 | 49.0 | 243344 | 0.00 | -5.6% | 159.3% * |
| bun | 2.6 ms | 4.7 ms | 7.0 ms | 399 | 5141 | 13.2 | 233428 | 0.00 | -7.2% | 25.6% * |
| gorm/gen | 2.5 ms | 4.2 ms | 6.8 ms | 399 | 4525 | 52.2 | 249454 | 0.00 | -15.9% | 10.8% * |

1. pop's Destroy reports no rows affected, so the adapter counts any error-free delete as 1 row; the benchmark only deletes existing ids.
2. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.

#### select

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 783 µs | 2.0 ms | 4.4 ms | 399 | 1210 | 17.0 | 212976 | 0.00 | +0.2% | 14.4% * |
| pgx | 767 µs | 1.9 ms | 4.6 ms | 399 | 1262 | 18.0 | 221621 | 0.00 | -3.6% | 25.0% * |
| database/sql | 793 µs | 2.0 ms | 4.1 ms | 399 | 1952 | 50.0 | 289916 | 0.00 | baseline | 17.9% * |
| sqlc | 1.0 ms | 2.0 ms | 4.3 ms | 399 | 2021 | 51.2 | 301998 | 0.00 | +0.6% | 24.0% * |
| sqlx | 817 µs | 2.3 ms | 4.6 ms | 399 | 2365 | 55.1 | 293125 | 0.00 | +18.0% | 31.0% * |
| sqlboiler | 794 µs | 1.8 ms | 4.0 ms | 399 | 2257 | 53.3 | 288428 | 0.00 | -8.5% | 54.1% * |
| beego | 801 µs | 1.7 ms | 4.4 ms | 399 | 3335 | 103.0 | 261874 | 0.00 | -16.9% | 81.1% * |
| rel | 804 µs | 1.7 ms | 4.1 ms | 399 | 3526 | 74.1 | 288998 | 0.00 | -12.9% | 19.0% * |
| pop | 835 µs | 2.0 ms | 5.0 ms | 399 | 3922 | 87.3 | 328177 | 0.00 | +2.0% | 11.2% * |
| ent | 870 µs | 2.2 ms | 5.4 ms | 399 | 5617 | 145.2 | 333085 | 0.00 | +9.9% | 44.2% * |
| go-jet | 843 µs | 2.0 ms | 5.1 ms | 399 | 12088 | 221.7 | 375566 | 0.00 | -1.9% | 10.9% * |
| xorm | 834 µs | 2.1 ms | 5.3 ms | 399 | 7736 | 208.6 | 347368 | 0.00 | +3.2% | 41.0% * |
| gorm | 835 µs | 1.9 ms | 5.1 ms | 399 | 6080 | 100.4 | 322190 | 0.00 | -5.8% | 8.3% |
| bun | 1.5 ms | 3.0 ms | 6.0 ms | 399 | 8017 | 72.5 | 414598 | 0.00 | +51.4% | 7.0% |
| gorm/gen | 829 µs | 2.0 ms | 4.4 ms | 399 | 6161 | 98.3 | 320503 | 0.00 | -0.8% | 11.3% * |

#### join

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 952 µs | 2.1 ms | 4.6 ms | 399 | 2929 | 63.6 | 234124 | 0.00 | -6.6% | 28.3% * |
| pgx | 925 µs | 2.0 ms | 4.1 ms | 399 | 8433 | 87.6 | 261851 | 0.00 | -11.1% | 9.0% |
| database/sql | 1.1 ms | 2.2 ms | 5.0 ms | 399 | 3642 | 184.2 | 412566 | 0.00 | baseline | 10.7% * |
| sqlc | 1.2 ms | 2.4 ms | 5.9 ms | 399 | 8104 | 193.5 | 451007 | 0.00 | +9.0% | 37.4% * |
| sqlx | 1.1 ms | 2.6 ms | 4.5 ms | 399 | 9188 | 207.2 | 466250 | 0.00 | +17.9% | 18.2% * |
| sqlboiler | 1.2 ms | 3.1 ms | 6.3 ms | 399 | 14891 | 259.8 | 518113 | 0.00 | +36.8% | 25.9% * |
| beego [1] | 1.1 ms | 2.8 ms | 5.8 ms | 399 | 12100 | 533.7 | 402793 | 0.00 | +26.9% | 84.9% * |
| rel | 1.0 ms | 2.2 ms | 4.4 ms | 399 | 20125 | 353.2 | 434170 | 0.00 | -2.1% | 11.0% * |
| pop | 1.2 ms | 2.4 ms | 5.2 ms | 399 | 13934 | 294.3 | 557872 | 0.00 | +7.6% | 12.7% * |
| ent [2] | 1.4 ms | 3.0 ms | 6.4 ms | 399 | 13883 | 346.3 | 489707 | 0.00 | +33.7% | 7.5% |
| go-jet | 1.1 ms | 2.5 ms | 6.5 ms | 399 | 37458 | 691.7 | 578684 | 0.00 | +12.7% | 7.2% |
| xorm | 1.0 ms | 2.3 ms | 5.8 ms | 399 | 23678 | 610.4 | 506351 | 0.00 | +2.0% | 18.1% * |
| gorm | 1.3 ms | 2.6 ms | 6.1 ms | 399 | 17395 | 406.6 | 635416 | 0.00 | +16.0% | 36.0% * |
| bun | 1.5 ms | 3.2 ms | 6.4 ms | 399 | 13394 | 210.3 | 417033 | 0.00 | +41.2% | 9.2% |
| gorm/gen | 1.2 ms | 2.6 ms | 5.4 ms | 399 | 23498 | 485.6 | 634875 | 0.00 | +15.9% | 7.6% |

1. beego JOIN is RelatedSel on a foreign-key field (one SQL statement, customer loaded with the order).
2. ent JOIN is edge eager-loading (WithCustomer): two queries, orders then their customers, not one SQL JOIN. Read its JOIN numbers with that in mind.

#### union

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 1.4 ms | 2.6 ms | 5.4 ms | 399 | 4640 | 171.0 | 239799 | 0.00 | +2.8% | 24.0% * |
| pgx | 1.4 ms | 2.5 ms | 5.0 ms | 399 | 14705 | 257.2 | 286840 | 0.00 | +0.8% | 14.8% * |
| database/sql | 1.4 ms | 2.5 ms | 5.1 ms | 399 | 5335 | 269.8 | 317470 | 0.00 | baseline | 23.3% * |
| sqlc | 1.4 ms | 3.1 ms | 6.4 ms | 399 | 14742 | 315.8 | 369062 | 0.00 | +22.6% | 12.6% * |
| sqlx | 1.4 ms | 2.7 ms | 5.2 ms | 399 | 15927 | 358.9 | 388392 | 0.00 | +7.7% | 33.2% * |
| sqlboiler [1] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| beego [2] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| rel [3] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| pop [4] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| ent [5] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| go-jet | 1.5 ms | 3.2 ms | 7.1 ms | 399 | 48849 | 794.0 | 600072 | 0.00 | +30.1% | 24.9% * |
| xorm [6] | 1.4 ms | 2.8 ms | 5.9 ms | 399 | 39578 | 1010.1 | 524086 | 0.00 | +13.3% | 38.3% * |
| gorm [7] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| bun | 1.4 ms | 3.2 ms | 6.9 ms | 399 | 20457 | 337.5 | 414128 | 0.00 | +28.8% | 7.7% |
| gorm/gen [8] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |

1. n/a: sqlboiler's query mods cannot express UNION without raw SQL.
2. n/a: beego's query API cannot express UNION without raw SQL.
3. n/a: rel's query API cannot express UNION without raw SQL.
4. n/a: pop's query API cannot express UNION without RawQuery.
5. n/a: ent's query API cannot express UNION, and using raw SQL would not measure ent.
6. xorm UNION is xorm.io/builder's Union (the query builder xorm itself uses) passed to SQL, so it is measured, not n/a.
7. n/a: gorm's query API cannot express UNION without raw SQL.
8. n/a: gorm/gen's query API cannot express UNION without raw SQL.

### Profile: stress

#### insert

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.7 ms | 4.2 ms | 24 ms | 20939 | 570 | 12.0 | 43534 | 0.00 | -1.9% | 17.4% * |
| pgx | 2.7 ms | 4.2 ms | 7.4 ms | 21422 | 570 | 12.0 | 40139 | 0.00 | -1.0% | 12.5% * |
| database/sql | 2.8 ms | 4.3 ms | 10 ms | 21491 | 872 | 14.0 | 43660 | 0.00 | baseline | 7.9% |
| sqlc | 2.7 ms | 4.2 ms | 7.2 ms | 21933 | 984 | 15.0 | 43189 | 0.00 | -1.0% | 36.6% * |
| sqlx | 2.6 ms | 4.1 ms | 9.6 ms | 22008 | 872 | 14.0 | 43489 | 0.00 | -4.5% | 1.7% |
| sqlboiler [1] | 2.8 ms | 4.4 ms | 13 ms | 20754 | 1924 | 36.0 | 47295 | 0.00 | +2.0% | 2.1% |
| beego | 2.9 ms | 4.5 ms | 8.3 ms | 20407 | 2884 | 50.0 | 49341 | 0.00 | +4.4% | 50.6% * |
| rel | 2.9 ms | 4.4 ms | 11 ms | 20603 | 3100 | 45.0 | 51129 | 0.00 | +3.1% | 1.3% |
| pop | 3.0 ms | 4.7 ms | 7.6 ms | 20081 | 5756 | 175.0 | 63318 | 0.00 | +8.8% | 15.4% * |
| ent | 3.0 ms | 4.7 ms | 9.1 ms | 20212 | 4389 | 98.1 | 51292 | 0.00 | +9.9% | 22.9% * |
| go-jet | 2.8 ms | 4.3 ms | 11 ms | 21056 | 3528 | 87.0 | 49200 | 0.00 | +0.6% | 10.5% * |
| xorm | 3.0 ms | 4.6 ms | 7.4 ms | 20909 | 4660 | 94.0 | 55790 | 0.00 | +8.0% | 3.4% |
| gorm [2] | 2.8 ms | 4.3 ms | 7.0 ms | 21096 | 6828 | 77.0 | 54752 | 0.00 | -0.1% | 4.7% |
| bun | 2.9 ms | 4.4 ms | 7.9 ms | 20771 | 6045 | 27.1 | 52057 | 0.00 | +1.7% | 20.9% * |
| gorm/gen [3] | 2.8 ms | 4.4 ms | 20 ms | 20143 | 6796 | 77.1 | 55724 | 0.00 | +2.8% | 3.2% |

1. sqlboiler's Insert reports no rows affected, so the adapter counts any error-free insert as 1 row.
2. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.
3. gorm/gen's Create reports no rows affected, so the adapter counts any error-free insert as 1 row; it runs with SkipDefaultTransaction like the gorm adapter.

#### update

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.9 ms | 4.4 ms | 10 ms | 20618 | 594 | 14.0 | 43847 | 0.00 | -2.2% | 28.5% * |
| pgx | 2.8 ms | 4.4 ms | 11 ms | 20322 | 594 | 14.0 | 40260 | 0.00 | -3.2% | 15.2% * |
| database/sql | 3.0 ms | 4.5 ms | 8.5 ms | 21235 | 928 | 15.9 | 43773 | 0.00 | baseline | 5.7% |
| sqlc | 2.9 ms | 4.6 ms | 8.7 ms | 20330 | 1048 | 16.0 | 42052 | 0.00 | +1.2% | 3.8% |
| sqlx | 2.9 ms | 4.6 ms | 10 ms | 20621 | 928 | 15.9 | 43526 | 0.00 | +2.0% | 5.4% |
| sqlboiler | 2.8 ms | 4.4 ms | 17 ms | 20328 | 1184 | 20.0 | 41719 | 0.00 | -2.8% | 6.5% |
| beego | 2.9 ms | 4.7 ms | 13 ms | 20111 | 2040 | 32.0 | 44464 | 0.00 | +5.3% | 164.6% * |
| rel | 3.0 ms | 4.7 ms | 12 ms | 20373 | 2464 | 32.9 | 44182 | 0.00 | +4.1% | 0.7% |
| pop | 3.0 ms | 4.7 ms | 18 ms | 19664 | 5588 | 159.0 | 51609 | 0.00 | +4.6% | 5.0% |
| ent | 3.0 ms | 4.6 ms | 13 ms | 20169 | 4919 | 100.9 | 46674 | 0.00 | +3.3% | 3.0% |
| go-jet | 3.0 ms | 4.7 ms | 11 ms | 20405 | 3943 | 100.0 | 48272 | 0.00 | +4.5% | 3.9% |
| xorm | 3.1 ms | 5.2 ms | 12 ms | 20028 | 5977 | 157.0 | 55049 | 0.00 | +16.0% | 4.9% |
| gorm [1] | 2.9 ms | 4.6 ms | 13 ms | 20051 | 7856 | 80.0 | 52220 | 0.00 | +2.0% | 7.3% |
| bun | 3.2 ms | 5.4 ms | 48 ms | 18173 | 5960 | 19.0 | 46182 | 0.00 | +19.7% | 15.5% * |
| gorm/gen | 3.2 ms | 4.9 ms | 11 ms | 19186 | 15099 | 290.0 | 74120 | 0.00 | +9.3% | 4.1% |

1. gorm runs with SkipDefaultTransaction (by default every write is wrapped in a transaction) and Updates takes an explicit column list, because Updates(struct) skips zero values.

#### delete

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 2.8 ms | 4.4 ms | 8.8 ms | 21014 | 410 | 9.0 | 42048 | 0.00 | -0.8% | 13.9% * |
| pgx | 2.7 ms | 4.3 ms | 12 ms | 20963 | 410 | 9.0 | 40083 | 0.00 | -3.6% | 8.6% |
| database/sql | 2.8 ms | 4.4 ms | 10 ms | 21286 | 472 | 11.0 | 42815 | 0.00 | baseline | 1.5% |
| sqlc | 2.8 ms | 4.3 ms | 7.6 ms | 21062 | 480 | 11.0 | 40912 | 0.00 | -2.0% | 9.5% |
| sqlx | 2.7 ms | 4.2 ms | 7.6 ms | 21339 | 472 | 11.0 | 42043 | 0.00 | -5.7% | 2.5% |
| sqlboiler | 2.7 ms | 4.3 ms | 8.6 ms | 21031 | 600 | 13.0 | 40419 | 0.00 | -2.5% | 4.8% |
| beego | 2.8 ms | 4.3 ms | 11 ms | 20877 | 840 | 19.0 | 40908 | 0.00 | -2.3% | 11.5% * |
| rel | 2.8 ms | 4.6 ms | 11 ms | 21012 | 880 | 21.0 | 41606 | 0.00 | +3.3% | 13.6% * |
| pop [1] | 2.7 ms | 4.2 ms | 13 ms | 21051 | 1088 | 32.0 | 42203 | 0.00 | -4.3% | 4.4% |
| ent | 2.8 ms | 4.4 ms | 10 ms | 20671 | 2120 | 44.0 | 42389 | 0.00 | -1.5% | 8.8% |
| go-jet | 2.8 ms | 4.4 ms | 11 ms | 20673 | 1552 | 36.0 | 44525 | 0.00 | +0.2% | 5.4% |
| xorm | 2.8 ms | 4.7 ms | 12 ms | 20749 | 3688 | 102.0 | 49531 | 0.00 | +5.5% | 6.5% |
| gorm [2] | 2.8 ms | 4.4 ms | 13 ms | 20725 | 4176 | 49.0 | 47556 | 0.00 | -1.6% | 1.1% |
| bun | 3.0 ms | 4.9 ms | 13 ms | 19848 | 5096 | 13.0 | 44889 | 0.00 | +9.8% | 13.8% * |
| gorm/gen | 2.7 ms | 4.3 ms | 14 ms | 20648 | 4480 | 52.0 | 45250 | 0.00 | -3.6% | 13.4% * |

1. pop's Destroy reports no rows affected, so the adapter counts any error-free delete as 1 row; the benchmark only deletes existing ids.
2. gorm runs with SkipDefaultTransaction; by default it wraps every write in a transaction.

#### select

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 798 µs | 1.7 ms | 4.6 ms | 34952 | 1212 | 17.0 | 37282 | 0.00 | -7.1% | 13.4% * |
| pgx | 800 µs | 1.6 ms | 4.9 ms | 35189 | 1260 | 18.0 | 38521 | 0.00 | -8.7% | 7.0% |
| database/sql | 844 µs | 1.8 ms | 5.1 ms | 33832 | 1947 | 50.0 | 43331 | 0.00 | baseline | 6.7% |
| sqlc | 836 µs | 1.7 ms | 12 ms | 33125 | 1972 | 51.0 | 44516 | 0.00 | -3.2% | 2.5% |
| sqlx | 854 µs | 1.7 ms | 16 ms | 33482 | 2348 | 55.0 | 45850 | 0.00 | -2.1% | 7.8% |
| sqlboiler | 834 µs | 1.7 ms | 200 ms | 32986 | 2155 | 53.0 | 45639 | 0.00 | -6.2% | 10.5% * |
| beego | 848 µs | 1.7 ms | 5.1 ms | 33324 | 3322 | 102.9 | 46108 | 0.00 | -3.1% | 2.3% |
| rel | 864 µs | 1.7 ms | 4.3 ms | 33612 | 3515 | 74.0 | 45447 | 0.00 | -2.2% | 4.0% |
| pop | 886 µs | 1.8 ms | 11 ms | 31578 | 3765 | 87.0 | 53038 | 0.00 | +1.3% | 0.9% |
| ent | 888 µs | 1.8 ms | 11 ms | 32909 | 5564 | 145.0 | 50539 | 0.00 | +1.9% | 5.7% |
| go-jet | 914 µs | 1.9 ms | 18 ms | 31535 | 12006 | 221.0 | 59128 | 0.00 | +8.3% | 5.0% |
| xorm | 912 µs | 1.8 ms | 7.7 ms | 32289 | 7660 | 208.0 | 56021 | 0.00 | +3.4% | 2.8% |
| gorm | 867 µs | 1.8 ms | 201 ms | 31714 | 5997 | 100.0 | 52338 | 0.00 | -0.1% | 7.7% |
| bun | 2.4 ms | 5.0 ms | 204 ms | 11631 | 8238 | 79.5 | 115116 | 0.00 | +182.9% | 4.1% |
| gorm/gen | 872 µs | 1.8 ms | 4.6 ms | 33193 | 6125 | 98.0 | 49745 | 0.00 | +0.2% | 2.0% |

#### join

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 908 µs | 2.0 ms | 201 ms | 29545 | 2880 | 63.5 | 40366 | 0.00 | -25.5% | 20.2% * |
| pgx | 944 µs | 2.0 ms | 201 ms | 30832 | 8384 | 87.5 | 43264 | 0.00 | -22.9% | 1.4% |
| database/sql | 1.2 ms | 2.6 ms | 201 ms | 25767 | 3558 | 183.7 | 90060 | 0.00 | baseline | 2.4% |
| sqlc | 1.2 ms | 2.7 ms | 201 ms | 25133 | 8014 | 192.9 | 93038 | 0.00 | +4.8% | 7.9% |
| sqlx | 1.3 ms | 2.8 ms | 201 ms | 25323 | 9106 | 206.7 | 95417 | 0.00 | +5.3% | 7.3% |
| sqlboiler | 1.3 ms | 2.9 ms | 201 ms | 24816 | 14204 | 259.1 | 105996 | 0.00 | +10.3% | 6.5% |
| beego [1] | 1.1 ms | 2.4 ms | 201 ms | 27011 | 11997 | 533.3 | 71930 | 0.00 | -10.3% | 7.7% |
| rel | 1.1 ms | 2.5 ms | 201 ms | 27302 | 20039 | 352.6 | 71195 | 0.00 | -3.7% | 8.1% |
| pop | 1.4 ms | 2.8 ms | 201 ms | 25092 | 13462 | 293.6 | 110365 | 0.00 | +8.4% | 17.2% * |
| ent [2] | 1.9 ms | 3.7 ms | 203 ms | 15876 | 13820 | 345.9 | 104702 | 0.00 | +41.0% | 2.1% |
| go-jet | 1.3 ms | 3.0 ms | 201 ms | 24850 | 37357 | 691.1 | 101180 | 0.00 | +14.3% | 6.6% |
| xorm | 1.2 ms | 2.6 ms | 201 ms | 26302 | 23576 | 609.7 | 84424 | 0.00 | -0.4% | 9.5% |
| gorm | 1.4 ms | 3.1 ms | 201 ms | 23404 | 17288 | 406.1 | 117584 | 0.00 | +19.1% | 5.0% |
| bun | 2.1 ms | 5.0 ms | 204 ms | 13931 | 13155 | 210.8 | 100148 | 0.00 | +89.2% | 8.9% |
| gorm/gen | 1.4 ms | 3.0 ms | 201 ms | 23753 | 22772 | 484.7 | 120020 | 0.00 | +15.2% | 2.4% |

1. beego JOIN is RelatedSel on a foreign-key field (one SQL statement, customer loaded with the order).
2. ent JOIN is edge eager-loading (WithCustomer): two queries, orders then their customers, not one SQL JOIN. Read its JOIN numbers with that in mind.

#### union

| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| pgx_pool | 1.5 ms | 4.5 ms | 202 ms | 21433 | 4642 | 171.0 | 36496 | 0.00 | +6.4% | 20.4% * |
| pgx | 1.6 ms | 4.7 ms | 202 ms | 21252 | 14658 | 257.0 | 46116 | 0.00 | +11.9% | 22.8% * |
| database/sql | 1.6 ms | 4.2 ms | 202 ms | 19642 | 5261 | 269.2 | 55953 | 0.00 | baseline | 28.5% * |
| sqlc | 1.6 ms | 4.2 ms | 202 ms | 19813 | 14673 | 315.2 | 59428 | 0.00 | +0.4% | 17.7% * |
| sqlx | 1.7 ms | 4.2 ms | 202 ms | 19894 | 15853 | 358.2 | 63220 | 0.00 | -0.8% | 14.4% * |
| sqlboiler [1] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| beego [2] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| rel [3] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| pop [4] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| ent [5] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| go-jet | 2.0 ms | 5.4 ms | 202 ms | 18288 | 48744 | 793.2 | 100001 | 0.00 | +28.7% | 10.3% * |
| xorm [6] | 1.8 ms | 5.1 ms | 202 ms | 18680 | 39477 | 1009.2 | 88447 | 0.00 | +20.1% | 8.2% |
| gorm [7] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |
| bun | 1.8 ms | 5.1 ms | 202 ms | 19258 | 20342 | 336.2 | 70062 | 0.00 | +20.8% | 17.1% * |
| gorm/gen [8] | n/a (unsupported) | - | - | - | - | - | - | - | - | - |

1. n/a: sqlboiler's query mods cannot express UNION without raw SQL.
2. n/a: beego's query API cannot express UNION without raw SQL.
3. n/a: rel's query API cannot express UNION without raw SQL.
4. n/a: pop's query API cannot express UNION without RawQuery.
5. n/a: ent's query API cannot express UNION, and using raw SQL would not measure ent.
6. xorm UNION is xorm.io/builder's Union (the query builder xorm itself uses) passed to SQL, so it is measured, not n/a.
7. n/a: gorm's query API cannot express UNION without raw SQL.
8. n/a: gorm/gen's query API cannot express UNION without raw SQL.

## Pros and cons

### pgx_pool

<https://github.com/jackc/pgx>

**Pros**

- Lowest-level adapter here: pgxpool with Query and Scan into locals, so it shows the cost of the driver and pool alone.
- Every scenario, including UNION, is plain SQL.

**Cons**

- Hand-written SQL and manual scanning of every column.
- PostgreSQL only.

### pgx

<https://github.com/jackc/pgx>

**Pros**

- Native PostgreSQL driver API without database/sql; rows are collected into structs with `CollectRows`/`RowToStructByName`.
- Every scenario, including UNION, is plain SQL.

**Cons**

- Hand-written SQL and no query builder.
- PostgreSQL only.

### database/sql

<https://pkg.go.dev/database/sql>

**Pros**

- No dependency beyond the driver; it is the reference every other adapter is checked against.
- Full control of the SQL and of what is scanned.

**Cons**

- Hand-written SQL strings and row scanning, with no compile-time check of either.

### sqlc

<https://github.com/sqlc-dev/sqlc>

**Pros**

- Queries are checked against the schema when the code is generated, and the generated code is typed.
- Plain SQL, so every scenario including UNION is expressible.

**Cons**

- Needs a generation step (make gen-sqlc) and committed generated code.
- Each query lives in an SQL file; there is no runtime query builder.

### sqlx

<https://github.com/jmoiron/sqlx>

**Pros**

- Thin layer over database/sql: the SQL stays hand-written and struct scanning is added.
- Every scenario is expressible with plain SQL.

**Cons**

- No updates since 2024.
- Scanning into structs is reflection-based.

### sqlboiler

<https://github.com/aarondl/sqlboiler>

**Pros**

- Typed models, column constants and where helpers generated from the live schema; Update and Delete report rows affected.
- JOIN is qm.InnerJoin bound onto a struct with Bind: one SQL statement.

**Cons**

- In maintenance mode (https://github.com/aarondl/sqlboiler): no new features, only community bug fixes and compatibility changes, and maintainers usually do not resolve reported issues.
- UNION is not expressible without raw SQL, so that case is n/a.
- Needs a generation step against the live database (make gen-sqlboiler, which also needs the sqlboiler-psql driver binary on PATH) and a large amount of committed generated code.
- Insert reports no rows affected, so the adapter counts any error-free insert as 1 row.
- The JOIN clause and the Bind destination (alias-prefixed columns) are written by hand as strings.

### beego

<https://github.com/beego/beego>

**Pros**

- The ORM (Ormer, QuerySeter) runs on a plain *sql.DB through AddAliasWthDB, so it shares the pgx stdlib driver; JOIN is RelatedSel on a foreign-key field, loaded in the same query.

**Cons**

- UNION is not expressible without raw SQL, so that case is n/a.
- Global registries: models register once before first use and a database alias name cannot be reused, so the adapter registers a fresh alias on every Init.
- A query that never calls Limit is capped at 1000 rows by default.

### rel

<https://github.com/go-rel/rel>

**Pros**

- Query API covers INSERT, UPDATE (UpdateAny), DELETE (DeleteAny), SELECT and JOIN (JoinOn, one SQL statement); UpdateAny and DeleteAny report rows affected.

**Cons**

- UNION is not expressible without raw SQL, so that case is n/a.
- The PostgreSQL adapter (go-rel/postgres v0.12.0) has had no release since 2024-05.
- Logs every statement by default; the adapter installs a no-op instrumenter.

### pop

<https://github.com/gobuffalo/pop>

**Pros**

- Query API covers INSERT, UPDATE (UpdateQuery reports rows affected), DELETE, SELECT and JOIN (Join with an ON clause, one SQL statement).

**Cons**

- Destroy and Query.Delete report no rows affected, so the adapter's DELETE returns 1 for any error-free call; a missing id would also report 1.
- UNION is not expressible without RawQuery, so that case is n/a.
- Opens its own pool (no *sql.DB injection), though on the same pgx stdlib driver.
- Create, Update and Destroy run model lifecycle callbacks (before/after hooks).

### ent

<https://github.com/ent/ent>

**Pros**

- Typed query API generated from a schema definition.

**Cons**

- JOIN is edge eager-loading: two queries rather than one SQL JOIN.
- UNION is not expressible without raw SQL, so that case is n/a.
- Needs a generation step (make gen-ent) and a large amount of committed generated code.

### go-jet

<https://github.com/go-jet/jet>

**Pros**

- Type-safe SQL builder; every scenario, including UNION, is expressed with its own API.

**Cons**

- Code is generated from the live database (make gen-jet needs it running).
- Pulls in lib/pq for array literals, although no adapter uses that driver.

### xorm

<https://gitea.com/xorm/xorm>

**Pros**

- Query API covers INSERT, UPDATE, DELETE, SELECT, JOIN (Table plus Join, one SQL statement) and UNION (xorm.io/builder's Union passed to SQL); writes report rows affected.
- Takes an existing *sql.DB (core.FromDB) and has a pgx dialect.

**Cons**

- The default SnakeMapper maps the field ID to the column i_d; the adapter needs GonicMapper.
- Update(struct) skips zero values (a false Right) unless the columns are named, and Get reports a missing row as found=false, not as an error.
- Conditions, join clauses and the select list are strings checked at run time.

### gorm

<https://github.com/go-gorm/gorm>

**Pros**

- Query API covers INSERT, UPDATE, DELETE, SELECT and JOIN (Joins with a relation produces one SQL JOIN).

**Cons**

- UNION is not expressible without raw SQL, so that case is n/a.
- Defaults need care to benchmark fairly: every write is wrapped in a transaction unless SkipDefaultTransaction is set, and Save or Updates(struct) misreport or skip zero values and missing rows unless guarded.

### bun

<https://github.com/uptrace/bun>

**Pros**

- Query builder covers every scenario: JOIN is a belongs-to relation compiled to one SQL JOIN, UNION is Union on the select builder, and writes report rows affected.

**Cons**

- Models embed bun.BaseModel and carry bun: struct tags, and PostgreSQL needs the separate pgdialect module.

### gorm/gen

<https://github.com/go-gorm/gen>

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

Hand-written judgement of the day-to-day work a library asks for. It is not part of the score, the Top 5 or the stability check. Levels for setup effort and query writing: `good` = little work, `fair` = some manual work, `poor` = mostly manual. Levels for debugging: `good` = the SQL that ran and its errors are visible with little setup, `fair` = visible after configuring logging or hooks, `poor` = hard to see what ran.

Criteria: setup effort = codegen, config and boilerplate before the first query; query writing = code a typical query takes; debugging = SQL visibility, error messages and logging hooks.

| Library | setup effort | query writing | debugging |
|---|---|---|---|
| database/sql | good | poor | good |
| sqlc | fair | good | good |
| sqlx | good | fair | good |
| gorm | good | good | fair |
| ent | poor | good | fair |
| go-jet | fair | good | good |
| pgx | good | fair | good |
| pgx_pool | good | poor | good |
| bun | good | good | good |
| beego | fair | fair | fair |
| pop | fair | good | fair |
| rel | good | good | good |
| sqlboiler | poor | good | good |
| gorm/gen | poor | good | fair |
| xorm | fair | good | fair |

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

## Type safety (qualitative)

Hand-written judgement of how many mistakes the Go compiler catches before the program runs, such as a misspelled column name or a wrong type. It is not part of the measured score, the Top 5 or the stability check, but it is one of the four weighted parts of the qualitative ranking. `good` = checked at compile time, `fair` = checked only partly, `poor` = mostly checked at run time.

| Library | Type safety |
|---|---|
| database/sql | poor |
| sqlc | good |
| sqlx | poor |
| gorm | fair |
| ent | good |
| go-jet | good |
| pgx | poor |
| pgx_pool | poor |
| bun | fair |
| beego | poor |
| pop | poor |
| rel | fair |
| sqlboiler | good |
| gorm/gen | good |
| xorm | poor |

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
## Qualitative ranking (judgement)

A second, separate ranking of the four weighted parts of the qualitative sections above. It is not part of the measured score, the Top 5 or the stability check. The levels behind it are the author's judgement and the score only turns them into numbers, so it looks more objective than it is.

Weights: flexibility 0.28, ease of use 0.24 (the learning curve counts as one more criterion), community 0.17, type safety 0.31. Levels count linearly from 1 (best) to 0 (worst). Community is the mean of last-commit recency and last-release recency (1 within 30 days of 2026-10-24, 0 after 730 days) and open issues per 1,000 stars (1 with none, 0 at 100 or more); stars are left out because they reward age. Equal totals share a rank and are listed alphabetically.

| Rank | Library | Total | Flexibility | Ease of use | Community | Type safety |
|---|---|---|---|---|---|---|
| 1 | go-jet | 0.92 | 0.94 | 0.75 | 0.97 | 1.00 |
| 2 | gorm/gen | 0.84 | 1.00 | 0.38 | 0.95 | 1.00 |
| 3 | ent | 0.82 | 1.00 | 0.38 | 0.81 | 1.00 |
| 4 | bun | 0.81 | 1.00 | 0.88 | 0.98 | 0.50 |
| 5 | rel | 0.79 | 1.00 | 0.88 | 0.85 | 0.50 |
| 6 | sqlc | 0.79 | 0.56 | 0.75 | 0.82 | 1.00 |
| 7 | gorm | 0.77 | 1.00 | 0.75 | 0.92 | 0.50 |
| 8 | pop | 0.55 | 0.94 | 0.62 | 0.80 | 0.00 |
| 9 | xorm | 0.54 | 1.00 | 0.62 | 0.67 | 0.00 |
| 10 | beego | 0.52 | 0.94 | 0.50 | 0.82 | 0.00 |
| 11 | database/sql | 0.40 | 0.33 | 0.75 | 0.77 | 0.00 |
| - | sqlboiler | 0.86 | 0.94 | 0.62 | 0.79 | 1.00 |
| - | pgx | 0.47 | 0.44 | 0.75 | 0.95 | 0.00 |
| - | pgx_pool | 0.47 | 0.44 | 0.75 | 0.95 | 0.00 |
| - | sqlx | 0.38 | 0.44 | 0.88 | 0.27 | 0.00 |

Libraries marked `-` are honorable mentions and are not ranked.

## Combined ranking (judgement)

A third ranking that adds each library's place in the measured ranking (Top 5 and "Also measured") to its place in the qualitative ranking above; the lowest sum is first. Both rankings count equally. Equal sums share a rank and are listed alphabetically. Only places are added, so how far apart two libraries are in score is lost, and a sum of a few places is within noise. The qualitative half is the author's judgement, so this ranking is too.

| Rank | Library | Measured rank | Qualitative rank | Sum |
|---|---|---|---|---|
| 1 | go-jet | 7 | 1 | 8 |
| 1 | sqlc | 2 | 6 | 8 |
| 3 | ent | 6 | 3 | 9 |
| 3 | rel | 4 | 5 | 9 |
| 5 | database/sql | 1 | 11 | 12 |
| 6 | beego | 3 | 10 | 13 |
| 6 | gorm/gen | 11 | 2 | 13 |
| 6 | pop | 5 | 8 | 13 |
| 9 | bun | 10 | 4 | 14 |
| 10 | gorm | 9 | 7 | 16 |
| 11 | xorm | 8 | 9 | 17 |

Honorable mentions are not ranked here either.
