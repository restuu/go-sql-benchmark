# Phase 6 report: extra libraries

Branch `feat/extra-libraries` (uncommitted when written). Result file `results/run-full-15-1.json` (git-ignored, local), log `results/run-full-15-1.log`, conditions `results/run15-conditions.txt`. README regenerated from it with `make report`.

## Outcome

- 15 libraries measured in one run: the core six re-measured plus nine extras (`beego`, `bun`, `pgx`, `pgx_pool`, `pop`, `rel`, `sqlboiler`, `gorm_gen` shown as `gorm/gen`, `xorm`).
- 180 cells (15 libraries x 6 scenarios x 2 profiles): 166 measured, 14 `na`, 0 errors, error rate 0 in every measured cell. The correctness pre-flight passed for all 15 before the matrix.
- `make test`, `make gen-check` and `make test-db` were green before the run; `ecc:go-reviewer` approved the adapters and the report code (no CRITICAL or HIGH).
- Top 5 (equal weights on p95, B/op, CPU ns/op): pgx 3.08, pgx_pool 3.25, database/sql 3.56, sqlx 4.64, beego 4.87 (10 of 12 cells). Scores of libraries with `na` cells average over fewer cells.

## Run conditions

- One run only (no second run), started 11:41 WIB 2026-10-24, ended 13:02 WIB (about 80 minutes). `Run started` in the README is the UTC time of the first cell.
- Load average 2.4 at start (below the 4.5 threshold); `pmset -g log` shows no sleep or wake event between 11:40 and 13:02 (the lines in the conditions file are only Brave video wake-lock assertions just before the start). Load at the end was 31.7, produced by the stress profile itself.
- Other unrelated containers (Postgres, Redis, NSQ stacks) were up in the same OrbStack VM; their load was not measured.
- Stability: with one run there is no run-to-run check. Within a run, 72 of 90 low-profile rows carry a p95 spread above 10% (README marker `*`). That matches the Phase 5 finding (environment-driven p95 instability, gate not met) which is not reopened. Read close ranks (for example the order of pgx, pgx_pool and database/sql, or the group between rank 6 and 13) as ties, not as differences.

## Adapters and the native rule

| Library | Notes |
|---|---|
| beego | ORM on a plain `*sql.DB`; fresh alias per `Init` (global registry); JOIN is `RelatedSel`; UNION `na` |
| bun | measured on every scenario; JOIN is a real SQL JOIN, UNION is `Union` on the select builder |
| pgx | `pgxpool`, `CollectRows`/`RowToStructByName`; all scenarios plain SQL |
| pgx_pool | `pgxpool`, Query plus Scan into locals (driver floor) |
| pop | opens its own pool (same pgx stdlib driver); UNION `na` |
| rel | on the shared pgx stdlib pool; logger replaced by a no-op; UNION `na` |
| sqlboiler | generated models (`make gen-sqlboiler`); JOIN is `qm.InnerJoin` plus `Bind`; UNION `na` |
| gorm_gen | `gorm.io/gen` on GORM (`make gen-gormgen`, adapter dir `gormgen`); UNION `na` |
| xorm | `GonicMapper`; JOIN and UNION (`xorm.io/builder` `Union` passed to `SQL`) measured |

`na` cells (14 = 7 libraries x 2 profiles): UNION for gorm, ent, beego, pop, rel, sqlboiler and gorm_gen. They are excluded from rankings.

Count exceptions (kept visible in the README cell notes and `CLAUDE.md`):

- **pop DELETE** reports no rows affected, so its adapter returns 1 for any error-free call (`bench.DeleteCountUnreported`). The missing-id delete check is replaced by a "row is gone" check in `adaptertest`, and `correctness` skips the second-delete check.
- **Insert returns 1 without a reported count** for sqlboiler, gorm_gen, beego, pop and rel; `correctness.Check` verifies the stored row.
- **ent JOIN** is edge eager-loading (two queries), as noted since Phase 3.

Driver exceptions: `pgx`/`pgx_pool` use their own pool; `pop` opens its own pool on the same pgx stdlib driver; the rest use the shared `*sql.DB`.

## Activity filter and dropped libraries

Checked 2026-10-24 on the default branch (not `pushed_at`) plus the last release.

- **Dropped for inactivity:** `reform` (no default-branch commit since 2022-01-24, last release 2021; its 2026 `pushed_at` is Dependabot branch churn), `go-pg`, `upper/db`, `gorm_prep`.
- **Dropped with evidence (user decision): `zorm`** (`gitee.com/chunanyong/zorm` v1.8.5, last commit 2026-10-18, so not dropped for inactivity). It reads `database/sql`'s private `lastcols` field by reflection. pgx stdlib implements Go 1.27's `driver.RowsColumnScanner`, so `database/sql` never fills `lastcols` (`/usr/local/go/src/database/sql/sql.go` around line 3080) and every zorm read panics with `reflect: slice index out of range` (`zorm@v1.8.5/structFieldInfo.go:507`, recovered into `has=false`). pgx has no option to switch that off. A driver shim and `lib/pq` were offered and declined. zorm also rejects writes outside `zorm.Transaction` unless `DisableTransaction` is set.
- The question "is last commit in 2026 a good filter" is answered: no, see the PRD.

## pgx versus pgx_pool

Both use `pgxpool`. `pgx` collects rows into structs with `CollectRows`/`RowToStructByName` (B/op higher), `pgx_pool` scans into locals like the database/sql reference, so it shows the driver floor. Both use pgx's default statement cache, never an explicit `Prepare`.

## Qualitative measures

Flexibility, ease of use, learning curve (hand-written levels) and community support (dated facts) are rubric entries for all 15 libraries, validated at load, rendered after "Pros and cons", and kept out of `rankCells`, `TopLibraries` and `benchcmp` (a test flips every level and checks nothing before the sections changes). Ease of use and learning curve are the author's judgement; community facts were read from the GitHub and Gitea APIs on 2026-10-24 (`community_checked`). Where a project publishes no release, the tag commit date is used and the entry says so (Go, sqlboiler v4.19.7, gorm/gen v0.3.29).

## Review follow-ups not fixed

From the first `ecc:go-reviewer` pass (batch 1):

1. `pgx` `Init` takes `MaxOpen` as `int32` without rejecting `MaxOpen < 1`.
2. Calling `Init` twice without `Close` leaks the first pool (also true for `databasesql`).
3. pgxpool's health-check goroutine and idle-connection ping are a small asymmetry against database/sql.
4. `pgx` `ReadJoin`/`ReadUnion` return a non-nil empty slice, `pgx_pool` and the reference return nil (harmless).
5. pgx tests miss double Close, `MaxOpen` 0, and `pgx` vs `pgx_pool` agreement on an empty join.

From batches 2-4:

6. `gen-sqlboiler` hardcodes host 127.0.0.1, user `postgres` and db `bench`, while `gen-jet` honours `BENCH_DSN`.
7. `internal/adapters/gormgen/generate/main.go` has `//go:build ignore`, so `make build`/`vet` never compile it (consider `go vet` on that file in `make vet`).
8. The JOIN filter column differs: pop and rel filter on `customers.id`, the others on `orders.customer_id` (same plan via the equivalence class).
9. beego keeps one closed alias per `Init` in its global registry (negligible), and a panic inside its `RegisterModel` `sync.Once` would leave the opened `sql.DB` unclosed.

From the qualitative-code review (LOW, not made):

10. The ease-of-use legend wording is generic for the debugging criterion; `Display` is not escaped in headings and bullets; there is no cross-field check on community dates; the ranking-invariance test does not mutate `Community.Repo` or notes.

Earlier open items stay open and out of scope: partial results on abort, non-atomic `result.Write`, `benchcmp` tie handling, report LOW items. Dynamic-query overhead is a possible later scenario only.
