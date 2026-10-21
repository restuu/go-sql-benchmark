# Graph Report - benchmark_go_sql  (2026-10-21)

## Corpus Check
- Corpus is ~7,612 words - fits in a single context window. You may not need a graph.

## Summary
- 98 nodes · 153 edges · 9 communities (8 shown, 1 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 22 edges (avg confidence: 0.91)
- Token cost: 117,923 input · 0 output

## Community Hubs (Navigation)
- dbcheck CLI and Go Imports
- PRD Phases and Scope
- Seed Check and Test Conventions
- Docker Postgres Setup
- Schema and Scenarios
- Project Docs and Licensing
- Metrics and Reporting Design
- Stack and Driver Decisions
- Go Module

## God Nodes (most connected - your core abstractions)
1. `internal/seedcheck package design` - 9 edges
2. `Makefile targets (arch-check, db-*, build, vet, test, test-db)` - 8 edges
3. `Check()` - 7 edges
4. `efectn/go-orm-benchmarks (upstream)` - 7 edges
5. `docker-compose.yml design (Postgres 15)` - 7 edges
6. `CLAUDE.md Decisions` - 6 edges
7. `postgres:15-alpine service` - 6 edges
8. `Go arch check (GOARCH vs host arch)` - 5 edges
9. `Core six libraries (database/sql, sqlc, sqlx, gorm, ent, go-jet)` - 5 edges
10. `Phase 1: Foundation` - 5 edges

## Surprising Connections (you probably didn't know these)
- `internal/seedcheck package design` --implements--> `diffCounts()`  [INFERRED]
  .claude/PRPs/plans/completed/foundation.plan.md → internal/seedcheck/seedcheck.go
- `internal/seedcheck package design` --implements--> `Check()`  [INFERRED]
  .claude/PRPs/plans/completed/foundation.plan.md → internal/seedcheck/seedcheck.go
- `CLAUDE.md Decisions` --conceptually_related_to--> `n/a (raw SQL fallback) excluded from rankings`  [INFERRED]
  CLAUDE.md → .claude/PRPs/prds/go-sql-benchmark.prd.md
- `CLAUDE.md Decisions` --conceptually_related_to--> `Custom runner instead of testing.B`  [INFERRED]
  CLAUDE.md → .claude/PRPs/prds/go-sql-benchmark.prd.md
- `CLAUDE.md Decisions` --conceptually_related_to--> `sqlx kept, labelled no updates since 2024`  [INFERRED]
  CLAUDE.md → .claude/PRPs/prds/go-sql-benchmark.prd.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Phase 1 foundation deliverables** — _claude_prps_plans_completed_foundation_plan_makefile, docker_compose, _claude_prps_plans_completed_foundation_plan_seedcheck_package, _claude_prps_plans_completed_foundation_plan_dbcheck_cli, _claude_prps_plans_completed_foundation_plan_license_attribution, claude [EXTRACTED 1.00]
- **Seeded database creation and verification flow** — docker_compose_init_scripts_mount, db_schema, db_seed, _claude_prps_plans_completed_foundation_plan_seedcheck_package, _claude_prps_plans_completed_foundation_plan_dbcheck_cli [INFERRED 0.85]
- **Controls that keep library comparison fair** — _claude_prps_prds_go_sql_benchmark_prd_pgx_stdlib_shared_driver, _claude_prps_prds_go_sql_benchmark_prd_arch_guard, _claude_prps_prds_go_sql_benchmark_prd_measurement_hygiene, _claude_prps_prds_go_sql_benchmark_prd_na_raw_sql_fallback, _claude_prps_prds_go_sql_benchmark_prd_cross_adapter_correctness_check [INFERRED 0.85]

## Communities (9 total, 1 thin omitted)

### Community 0 - "dbcheck CLI and Go Imports"
Cohesion: 0.15
Nodes (6): main(), run(), Check(), diffCounts(), TestCheckIntegration(), TestDiffCounts()

### Community 1 - "PRD Phases and Scope"
Cohesion: 0.14
Nodes (11): Phase 1 out-of-scope list, Input for Phase 2: reseed or insert-then-delete, Adapter interface (one method per scenario), PRD open questions (weights, stress parameters, dataset size), Phase 2: Runner and reference adapter, Phase 3: Core adapters, Phase 4: Report generator, Phase 5: Full run and publish (+3 more)

### Community 2 - "Seed Check and Test Conventions"
Cohesion: 0.20
Nodes (10): Table-driven tests with AAA comments, cmd/dbcheck CLI wrapper design, Seed drift check validation, Makefile targets (arch-check, db-*, build, vet, test, test-db), db/seed.sql deterministic re-runnable seed, internal/seedcheck package design, Deviations from plan, seedcheck coverage 80.8% (needs BENCH_DSN) (+2 more)

### Community 3 - "Docker Postgres Setup"
Cohesion: 0.21
Nodes (8): docker-compose.yml design (Postgres 15), Postgres init scripts run only on empty data dir, Planned directory move to ~/github.com/restuu, Load profiles (low traffic and stress), pg_isready healthcheck, Schema and seed mounted into docker-entrypoint-initdb.d, BENCH_PG_PORT and BENCH_PG_PASSWORD env defaults, postgres:15-alpine service

### Community 4 - "Schema and Scenarios"
Cohesion: 0.28
Nodes (7): db/schema.sql design (models, customers, suppliers, orders), Benchmark scenarios (INSERT, UPDATE, DELETE, SELECT, JOIN, UNION), customers, models, orders, orders_customer_id_idx, suppliers

### Community 5 - "Project Docs and Licensing"
Cohesion: 0.43
Nodes (8): Foundation Plan (Phase 1), MIT LICENSE with upstream attribution, Go SQL Library Benchmark PRD, Phase 1: Foundation, efectn/go-orm-benchmarks (upstream), Foundation Implementation Report, CLAUDE.md (project guide), CLAUDE.md Conventions

### Community 6 - "Metrics and Reporting Design"
Cohesion: 0.25
Nodes (4): Cross-adapter correctness check, Metrics per cell (p50/p95/p99, throughput, B/op, allocs/op, CPU, error rate), README report generator, Top-5 composite ranking

### Community 7 - "Stack and Driver Decisions"
Cohesion: 0.38
Nodes (4): Task 1: Go module and pgx dependency, sqlc and ent pinned via go tool directives, Core six libraries (database/sql, sqlc, sqlx, gorm, ent, go-jet), CLAUDE.md Decisions

## Knowledge Gaps
- **9 isolated node(s):** `github.com/restuu/go-sql-benchmark`, `Cross-adapter correctness check`, `sqlc and ent pinned via go tool directives`, `PRD open questions (weights, stress parameters, dataset size)`, `Planned directory move to ~/github.com/restuu` (+4 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 22 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `internal/seedcheck package design` connect `Seed Check and Test Conventions` to `dbcheck CLI and Go Imports`, `Stack and Driver Decisions`?**
  _High betweenness centrality (0.303) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `internal/seedcheck package design` (e.g. with `Check()` and `diffCounts()`) actually correct?**
  _`internal/seedcheck package design` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/restuu/go-sql-benchmark`, `Cross-adapter correctness check`, `sqlc and ent pinned via go tool directives` to the rest of the system?**
  _9 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `PRD Phases and Scope` be split into smaller, more focused modules?**
  _Cohesion score 0.14285714285714285 - nodes in this community are weakly interconnected._
- **Why does `Makefile targets (arch-check, db-*, build, vet, test, test-db)` connect `Seed Check and Test Conventions` to `PRD Phases and Scope`, `Docker Postgres Setup`?**
  _High betweenness centrality (0.193) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `Check()` (e.g. with `internal/seedcheck package design` and `TestCheckIntegration()`) actually correct?**
  _`Check()` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Why does `cmd/dbcheck CLI wrapper design` connect `Seed Check and Test Conventions` to `dbcheck CLI and Go Imports`?**
  _High betweenness centrality (0.191) - this node is a cross-community bridge._