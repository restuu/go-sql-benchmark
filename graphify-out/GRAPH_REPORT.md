# Graph Report - benchmark_go_sql  (2026-10-21)

## Corpus Check
- 37 files · ~28,291 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 4 file(s) not represented in the graph (top: (none) 4)

## Summary
- 408 nodes · 1094 edges · 21 communities (19 shown, 2 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 142 edges (avg confidence: 0.9)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `4ded5bc2`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bench/main.go
- PRD Phases and Scope
- Seed Check and Test Conventions
- Docker Postgres Setup
- Schema and Scenarios
- Project Docs and Licensing
- Metrics and Reporting Design
- Stack and Driver Decisions
- Go Module
- result.go
- context.Context
- testing.T
- flags.go
- Run
- phase
- Plan: Runner & Reference Adapter (Go SQL Benchmark, Phase 2)
- Check
- Implementation Report: Runner & Reference Adapter (Phase 2)
- Validation Commands
- Step-by-Step Tasks
- Patterns to Mirror

## God Nodes (most connected - your core abstractions)
1. `runCell()` - 25 edges
2. `Run()` - 20 edges
3. `Adapter` - 19 edges
4. `Plan: Runner & Reference Adapter (Go SQL Benchmark, Phase 2)` - 19 edges
5. `Scenario` - 18 edges
6. `fakeAdapter` - 18 edges
7. `Profile` - 17 edges
8. `Run` - 17 edges
9. `Files to Change` - 17 edges
10. `parseFlags()` - 15 edges

## Surprising Connections (you probably didn't know these)
- `Notes` --references--> `Run`  [INFERRED]
  .claude/PRPs/plans/completed/runner-and-reference-adapter.plan.md → internal/result/result.go
- `Smoke run (fast)` --references--> `Load()`  [INFERRED]
  .claude/PRPs/plans/completed/runner-and-reference-adapter.plan.md → internal/result/result.go
- `Task 2: Pure statistics` --references--> `summarize()`  [INFERRED]
  .claude/PRPs/plans/completed/runner-and-reference-adapter.plan.md → internal/runner/stats.go
- `Files to Change` --references--> `Scenario`  [INFERRED]
  .claude/PRPs/plans/completed/runner-and-reference-adapter.plan.md → internal/bench/bench.go
- `Files to Change` --references--> `Model`  [INFERRED]
  .claude/PRPs/plans/completed/runner-and-reference-adapter.plan.md → internal/bench/bench.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Phase 1 foundation deliverables** — _claude_prps_plans_completed_foundation_plan_makefile, docker_compose, _claude_prps_plans_completed_foundation_plan_seedcheck_package, _claude_prps_plans_completed_foundation_plan_dbcheck_cli, _claude_prps_plans_completed_foundation_plan_license_attribution, claude [EXTRACTED 1.00]
- **Controls that keep library comparison fair** — _claude_prps_prds_go_sql_benchmark_prd_pgx_stdlib_shared_driver, _claude_prps_prds_go_sql_benchmark_prd_arch_guard, _claude_prps_prds_go_sql_benchmark_prd_measurement_hygiene, _claude_prps_prds_go_sql_benchmark_prd_na_raw_sql_fallback, _claude_prps_prds_go_sql_benchmark_prd_cross_adapter_correctness_check [INFERRED 0.85]
- **Seeded database creation and verification flow** — docker_compose_init_scripts_mount, db_schema, db_seed, _claude_prps_plans_completed_foundation_plan_seedcheck_package, _claude_prps_plans_completed_foundation_plan_dbcheck_cli [INFERRED 0.85]

## Communities (21 total, 2 thin omitted)

### Community 1 - "PRD Phases and Scope"
Cohesion: 0.12
Nodes (14): Phase 1 out-of-scope list, Input for Phase 2: reseed or insert-then-delete, db/schema.sql design (models, customers, suppliers, orders), db/seed.sql deterministic re-runnable seed, Adapter interface (one method per scenario), Benchmark scenarios (INSERT, UPDATE, DELETE, SELECT, JOIN, UNION), PRD open questions (weights, stress parameters, dataset size), Phase 2: Runner and reference adapter (+6 more)

### Community 2 - "Seed Check and Test Conventions"
Cohesion: 0.19
Nodes (11): Table-driven tests with AAA comments, cmd/dbcheck CLI wrapper design, Seed drift check validation, Makefile targets (arch-check, db-*, build, vet, test, test-db), internal/seedcheck package design, Deviations from plan, seedcheck coverage 80.8% (needs BENCH_DSN), Foundation validation results (+3 more)

### Community 3 - "Docker Postgres Setup"
Cohesion: 0.21
Nodes (8): docker-compose.yml design (Postgres 15), Postgres init scripts run only on empty data dir, Planned directory move to ~/github.com/restuu, Load profiles (low traffic and stress), pg_isready healthcheck, Schema and seed mounted into docker-entrypoint-initdb.d, BENCH_PG_PORT and BENCH_PG_PASSWORD env defaults, postgres:15-alpine service

### Community 4 - "Schema and Scenarios"
Cohesion: 0.47
Nodes (5): customers, models, orders, orders_customer_id_idx, suppliers

### Community 5 - "Project Docs and Licensing"
Cohesion: 0.36
Nodes (8): Foundation Plan (Phase 1), MIT LICENSE with upstream attribution, Go SQL Library Benchmark PRD, Phase 1: Foundation, efectn/go-orm-benchmarks (upstream), Foundation Implementation Report, CLAUDE.md (project guide), CLAUDE.md Conventions

### Community 6 - "Metrics and Reporting Design"
Cohesion: 0.25
Nodes (4): Cross-adapter correctness check, Metrics per cell (p50/p95/p99, throughput, B/op, allocs/op, CPU, error rate), README report generator, Top-5 composite ranking

### Community 7 - "Stack and Driver Decisions"
Cohesion: 0.38
Nodes (4): Task 1: Go module and pgx dependency, sqlc and ent pinned via go tool directives, Core six libraries (database/sql, sqlc, sqlx, gorm, ent, go-jet), CLAUDE.md Decisions

### Community 9 - "result.go"
Cohesion: 0.06
Nodes (52): Files to Change, Task 6: Matrix runner, Code review (`ecc:go-reviewer`): 0 CRITICAL, 2 HIGH, 6 MEDIUM, 9 LOW, format(), main(), run(), TestRun(), writeRun() (+44 more)

### Community 10 - "context.Context"
Cohesion: 0.06
Nodes (21): Task 5: Worker loop and cell execution (tests first, with fakes), Task 7: database/sql reference adapter, Deviations from Plan, Adapter, affected(), Cities(), Adapter, Model (+13 more)

### Community 11 - "testing.T"
Cohesion: 0.14
Nodes (30): Unit Tests, New(), newInitialised(), TestAdapterIntegration(), TestInitUnreachableDatabaseIntegration(), TestName(), TestReinitAfterCloseIntegration(), TestSelectMissingIDIntegration() (+22 more)

### Community 12 - "flags.go"
Cohesion: 0.15
Nodes (22): config, checkProfile(), names(), parseFlags(), parseList(), parseProfiles(), parseScenarios(), TestParseFlagsDefaults() (+14 more)

### Community 13 - "Run"
Cohesion: 0.23
Nodes (14): Task 8: CLI and compare tool, Host, assemble(), progress(), record(), Run(), runCellRetryingStuck(), matrixOptions() (+6 more)

### Community 14 - "phase"
Cohesion: 0.20
Nodes (4): cpuTime(), measure(), phase, worker

### Community 15 - "Plan: Runner & Reference Adapter (Go SQL Benchmark, Phase 2)"
Cohesion: 0.13
Nodes (15): Acceptance Criteria (from PRD Phase 2 success signal), Completion Checklist, Design Decisions (made in this plan; flagged because the PRD left them open), Edge Cases Checklist, External Documentation, Interaction Changes, Metadata, NOT Building (+7 more)

### Community 16 - "Check"
Cohesion: 0.21
Nodes (12): Risks, Task 4: Store (reseed and delete pool), hostInfo(), main(), moduleVersion(), registryNames(), run(), NewPGStore() (+4 more)

### Community 17 - "Implementation Report: Runner & Reference Adapter (Phase 2)"
Cohesion: 0.18
Nodes (10): Acceptance Criteria, Assessment vs Reality, Harness floor, Implementation Report: Runner & Reference Adapter (Phase 2), Intermittent hang (open risk), Next Steps, Smoke numbers (3 s stress, 2 s low, 1 repeat; one machine, indicative only), Stability finding (success signal) (+2 more)

### Community 18 - "Validation Commands"
Cohesion: 0.22
Nodes (9): Environment sanity, Integration (needs the DB), Manual Validation, Negative checks, Smoke run (fast), Static Analysis, Success signal: stability (PRD Phase 2), Unit Tests (+1 more)

### Community 19 - "Step-by-Step Tasks"
Cohesion: 0.25
Nodes (7): Mandatory Reading, Step-by-Step Tasks, Task 10: End-to-end validation, Task 1: Embed SQL and core types, Task 2: Pure statistics, Task 3: Result format, validation, compare, Task 9: Makefile and CLAUDE.md

### Community 20 - "Patterns to Mirror"
Cohesion: 0.29
Nodes (7): DRIVER_USAGE, ERROR_HANDLING, MAIN_PATTERN / LOGGING_PATTERN, NAMING_CONVENTION, Patterns to Mirror, REFERENCE_SQL, TEST_STRUCTURE

## Knowledge Gaps
- **9 isolated node(s):** `github.com/restuu/go-sql-benchmark`, `PRD open questions (weights, stress parameters, dataset size)`, `Next steps (code review, git init, Phase 2 plan)`, `Deviations from plan`, `seedcheck coverage 80.8% (needs BENCH_DSN)` (+4 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 75 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `internal/seedcheck package design` connect `Seed Check and Test Conventions` to `Check`, `PRD Phases and Scope`, `Stack and Driver Decisions`?**
  _High betweenness centrality (0.176) - this node is a cross-community bridge._
- **Are the 16 inferred relationships involving `runCell()` (e.g. with `Files to Change` and `Task 6: Matrix runner`) actually correct?**
  _`runCell()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/restuu/go-sql-benchmark`, `PRD open questions (weights, stress parameters, dataset size)`, `Next steps (code review, git init, Phase 2 plan)` to the rest of the system?**
  _9 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `PRD Phases and Scope` be split into smaller, more focused modules?**
  _Cohesion score 0.12105263157894737 - nodes in this community are weakly interconnected._
- **Why does `Check()` connect `Check` to `bench/main.go`, `Seed Check and Test Conventions`, `context.Context`, `Run`, `Step-by-Step Tasks`?**
  _High betweenness centrality (0.174) - this node is a cross-community bridge._
- **Are the 6 inferred relationships involving `Run()` (e.g. with `Task 8: CLI and compare tool` and `.Reseed()`) actually correct?**
  _`Run()` has 6 INFERRED edges - model-reasoned connections that need verification._
- **Should `result.go` be split into smaller, more focused modules?**
  _Cohesion score 0.06202435312024353 - nodes in this community are weakly interconnected._