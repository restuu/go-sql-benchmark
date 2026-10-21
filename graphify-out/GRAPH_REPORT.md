# Graph Report - benchmark_go_sql  (2026-10-21)

## Corpus Check
- 117 files · ~84,867 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 4 file(s) not represented in the graph (top: (none) 4)

## Summary
- 1894 nodes · 5071 edges · 92 communities (57 shown, 35 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 466 edges (avg confidence: 0.88)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `2e7af1b8`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- bench/main.go
- Phase 2: Runner and reference adapter
- Seed Check and Test Conventions
- Docker Postgres Setup
- schema.sql
- Project Docs and Licensing
- Metrics and Reporting Design
- efectn/go-orm-benchmarks (upstream)
- Go Module
- Run
- fakeAdapter
- testing.T
- runCell
- Run
- order/where.go
- Plan: Runner & Reference Adapter (Go SQL Benchmark, Phase 2)
- Check
- Implementation Report: Runner & Reference Adapter (Phase 2)
- model/where.go
- customer/where.go
- CustomerQuery
- Next session: Phase 3 (Core adapters), after committing Phase 2
- OrderQuery
- ModelQuery
- ModelUpdateOne
- modelsTable
- context.Context
- hook.go
- entgen/ent.go
- ModelMutation
- OrderUpdateOne
- CustomerUpdateOne
- ModelCreate
- Adapter
- OrderMutation
- Plan: Core adapters (sqlc, sqlx, gorm, ent, go-jet) + cross-adapter correctness
- CustomerMutation
- Model
- Client
- Tx
- CustomerCreate
- OrderCreate
- Verifier
- Adapter
- Condition
- Metrics
- Adapter
- Adapter
- OrderClient
- client.go
- Step-by-Step Tasks
- fakeLib
- config
- CustomerClient
- Customer
- Implementation Report: Core adapters (Phase 3)
- ModelClient
- UnionRow
- Adapter
- txDriver
- Order
- JoinRow
- Schema
- CustomerDelete
- .sqlSave
- ModelDelete
- OrderDelete
- model/model.go
- Customer
- mutation.go
- customer/customer.go
- order/order.go
- newOrderMutation
- Model
- .sqlSave
- .Mutate
- withHooks
- scanWithInterceptors
- github.com/restuu/go-sql-benchmark/tools

## God Nodes (most connected - your core abstractions)
1. `Model` - 92 edges
2. `ModelMutation` - 75 edges
3. `OrderMutation` - 66 edges
4. `CustomerMutation` - 62 edges
5. `Customer` - 59 edges
6. `Order` - 55 edges
7. `CustomerQuery` - 51 edges
8. `OrderQuery` - 51 edges
9. `ModelQuery` - 43 edges
10. `Adapter` - 42 edges

## Surprising Connections (you probably didn't know these)
- `Summary` --references--> `Adapter`  [INFERRED]
  .claude/PRPs/plans/completed/core-adapters.plan.md → internal/bench/bench.go
- `Notes` --references--> `Verifier`  [INFERRED]
  .claude/PRPs/plans/completed/core-adapters.plan.md → internal/bench/verify.go
- `Problem → Solution` --references--> `Verifier`  [INFERRED]
  .claude/PRPs/plans/completed/core-adapters.plan.md → internal/bench/verify.go
- `Risks` --references--> `Verifier`  [INFERRED]
  .claude/PRPs/plans/completed/core-adapters.plan.md → internal/bench/verify.go
- `Task 12: Documentation` --references--> `Verifier`  [INFERRED]
  .claude/PRPs/plans/completed/core-adapters.plan.md → internal/bench/verify.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Phase 1 foundation deliverables** — _claude_prps_plans_completed_foundation_plan_makefile, docker_compose, _claude_prps_plans_completed_foundation_plan_seedcheck_package, _claude_prps_plans_completed_foundation_plan_dbcheck_cli, _claude_prps_plans_completed_foundation_plan_license_attribution, claude [EXTRACTED 1.00]
- **Controls that keep library comparison fair** — _claude_prps_prds_go_sql_benchmark_prd_pgx_stdlib_shared_driver, _claude_prps_prds_go_sql_benchmark_prd_arch_guard, _claude_prps_prds_go_sql_benchmark_prd_measurement_hygiene, _claude_prps_prds_go_sql_benchmark_prd_na_raw_sql_fallback, _claude_prps_prds_go_sql_benchmark_prd_cross_adapter_correctness_check [INFERRED 0.85]
- **Seeded database creation and verification flow** — docker_compose_init_scripts_mount, db_schema, db_seed, _claude_prps_plans_completed_foundation_plan_seedcheck_package, _claude_prps_plans_completed_foundation_plan_dbcheck_cli [INFERRED 0.85]

## Communities (92 total, 35 thin omitted)

### Community 0 - "bench/main.go"
Cohesion: 0.11
Nodes (9): format(), main(), run(), compareCell(), Deviation, relativeDiff(), Suppliers, Config (+1 more)

### Community 1 - "Phase 2: Runner and reference adapter"
Cohesion: 0.22
Nodes (8): Phase 1 out-of-scope list, Input for Phase 2: reseed or insert-then-delete, PRD open questions (weights, stress parameters, dataset size), Phase 2: Runner and reference adapter, Phase 4: Report generator, Phase 5: Full run and publish, Phase 6: Extra libraries (14), Next steps (code review, git init, Phase 2 plan)

### Community 2 - "Seed Check and Test Conventions"
Cohesion: 0.20
Nodes (10): Table-driven tests with AAA comments, cmd/dbcheck CLI wrapper design, Seed drift check validation, Makefile targets (arch-check, db-*, build, vet, test, test-db), db/seed.sql deterministic re-runnable seed, internal/seedcheck package design, Deviations from plan, seedcheck coverage 80.8% (needs BENCH_DSN) (+2 more)

### Community 3 - "Docker Postgres Setup"
Cohesion: 0.24
Nodes (6): docker-compose.yml design (Postgres 15), Planned directory move to ~/github.com/restuu, Load profiles (low traffic and stress), pg_isready healthcheck, BENCH_PG_PORT and BENCH_PG_PASSWORD env defaults, postgres:15-alpine service

### Community 4 - "schema.sql"
Cohesion: 0.21
Nodes (9): Postgres init scripts run only on empty data dir, Phase 3: Core adapters, CLAUDE.md Layout (db/, internal/, cmd/), customers, models, orders, orders_customer_id_idx, suppliers (+1 more)

### Community 5 - "Project Docs and Licensing"
Cohesion: 0.31
Nodes (5): Foundation Plan (Phase 1), Go SQL Library Benchmark PRD, Phase 1: Foundation, Foundation Implementation Report, CLAUDE.md (project guide)

### Community 6 - "Metrics and Reporting Design"
Cohesion: 0.25
Nodes (4): Cross-adapter correctness check, Metrics per cell (p50/p95/p99, throughput, B/op, allocs/op, CPU, error rate), README report generator, Top-5 composite ranking

### Community 7 - "efectn/go-orm-benchmarks (upstream)"
Cohesion: 0.19
Nodes (10): MIT LICENSE with upstream attribution, Task 1: Go module and pgx dependency, db/schema.sql design (models, customers, suppliers, orders), Adapter interface (one method per scenario), Benchmark scenarios (INSERT, UPDATE, DELETE, SELECT, JOIN, UNION), sqlc and ent pinned via go tool directives, Core six libraries (database/sql, sqlc, sqlx, gorm, ent, go-jet), efectn/go-orm-benchmarks (upstream) (+2 more)

### Community 9 - "Run"
Cohesion: 0.16
Nodes (23): Task 3: Result format, validation, compare, Unit Tests, Code review (`ecc:go-reviewer`): 0 CRITICAL, 2 HIGH, 6 MEDIUM, 9 LOW, main(), run(), TestRun(), writeRun(), TestScenarioExpectedRows() (+15 more)

### Community 10 - "fakeAdapter"
Cohesion: 0.09
Nodes (11): cpuTime(), measure(), latencyCapacity(), newPhase(), deleteIDs, fakeAdapter, fakeStore, phase (+3 more)

### Community 11 - "testing.T"
Cohesion: 0.07
Nodes (44): plainAdapter, New(), newInitialised(), TestAdapterIntegration(), TestInitUnreachableDatabaseIntegration(), TestName(), TestReinitAfterCloseIntegration(), TestSelectMissingIDIntegration() (+36 more)

### Community 12 - "runCell"
Cohesion: 0.08
Nodes (39): config, Files to Change, Acceptance Criteria, checkProfile(), names(), parseFlags(), parseList(), parseProfiles() (+31 more)

### Community 13 - "Run"
Cohesion: 0.26
Nodes (12): Task 8: CLI and compare tool, Host, shortenTimeouts(), TestRunCellStuckPhase(), assemble(), Run(), runCellRetryingStuck(), matrixOptions() (+4 more)

### Community 14 - "order/where.go"
Cohesion: 0.05
Nodes (60): NOT Building, Task 6: sqlc adapter and codegen, Customer, JoinOrdersRow, Model, Order, Supplier, UnionByCityRow (+52 more)

### Community 15 - "Plan: Runner & Reference Adapter (Go SQL Benchmark, Phase 2)"
Cohesion: 0.05
Nodes (37): Acceptance Criteria (from PRD Phase 2 success signal), Completion Checklist, Design Decisions (made in this plan; flagged because the PRD left them open), DRIVER_USAGE, Edge Cases Checklist, Environment sanity, ERROR_HANDLING, External Documentation (+29 more)

### Community 16 - "Check"
Cohesion: 0.10
Nodes (25): Task 11: Wire into `cmd/bench`, Task 3: Shared contract suite `internal/adaptertest`, Risks, Task 4: Store (reseed and delete pool), Acceptance Criteria, main(), preflight(), registryNames() (+17 more)

### Community 17 - "Implementation Report: Runner & Reference Adapter (Phase 2)"
Cohesion: 0.12
Nodes (18): Task 2: Pure statistics, Assessment vs Reality, Deviations from Plan, Harness floor, Implementation Report: Runner & Reference Adapter (Phase 2), Intermittent hang (open risk), Next Steps, Smoke numbers (3 s stress, 2 s low, 1 repeat; one machine, indicative only) (+10 more)

### Community 18 - "model/where.go"
Cohesion: 0.04
Nodes (89): Age(), AgeEQ(), AgeGT(), AgeGTE(), AgeIn(), AgeLT(), AgeLTE(), AgeNEQ() (+81 more)

### Community 19 - "customer/where.go"
Cohesion: 0.07
Nodes (56): And(), City(), CityContains(), CityContainsFold(), CityEQ(), CityEqualFold(), CityGT(), CityGTE() (+48 more)

### Community 20 - "CustomerQuery"
Cohesion: 0.09
Nodes (5): CustomerGroupBy, CustomerQuery, CustomerSelect, IsNotFound(), setContextOp()

### Community 21 - "Next session: Phase 3 (Core adapters), after committing Phase 2"
Cohesion: 0.13
Nodes (11): Task 7: database/sql reference adapter, Memory, Next session: Phase 3 (Core adapters), after committing Phase 2, Open items (not for this session unless the user asks), Role, Settled decisions (do not reopen), Start here, State (+3 more)

### Community 22 - "OrderQuery"
Cohesion: 0.09
Nodes (3): OrderGroupBy, OrderQuery, OrderSelect

### Community 23 - "ModelQuery"
Cohesion: 0.10
Nodes (3): ModelGroupBy, ModelQuery, ModelSelect

### Community 25 - "modelsTable"
Cohesion: 0.11
Nodes (12): newCustomersTable(), newCustomersTableImpl(), newModelsTable(), newModelsTableImpl(), newOrdersTable(), newOrdersTableImpl(), newSuppliersTable(), newSuppliersTableImpl() (+4 more)

### Community 26 - "context.Context"
Cohesion: 0.11
Nodes (5): Adapter, CreateModelParams, UpdateModelParams, selector, affected()

### Community 27 - "hook.go"
Cohesion: 0.20
Nodes (6): Customers, Models, Orders, Option, WithMigrateOptions(), WithOptions()

### Community 28 - "entgen/ent.go"
Cohesion: 0.08
Nodes (21): AggregateFunc, clientCtxKey, ConstraintError, NotFoundError, NotLoadedError, NotSingularError, OrderFunc, queryHook (+13 more)

### Community 33 - "Adapter"
Cohesion: 0.09
Nodes (5): DBTX, Queries, New(), rows(), Adapter

### Community 35 - "Plan: Core adapters (sqlc, sqlx, gorm, ent, go-jet) + cross-adapter correctness"
Cohesion: 0.08
Nodes (26): Acceptance Criteria, Codegen reproducibility, Completion Checklist, ERROR_HANDLING, External Documentation, INIT_CLOSE (pool config, ping, re-Init after Close), Integration tests, Interaction Changes (+18 more)

### Community 37 - "Model"
Cohesion: 0.13
Nodes (9): Task 10: `internal/correctness`, fakeStore, world, Models, affected(), toRow(), Model, NewModel() (+1 more)

### Community 38 - "Client"
Cohesion: 0.15
Nodes (11): options, TestingT, Client, NewClient(), Open(), FromContext(), NewContext(), migrateSchema() (+3 more)

### Community 39 - "Tx"
Cohesion: 0.12
Nodes (7): CommitFunc, Committer, Rollbacker, RollbackFunc, Tx, NewTxContext(), TxFromContext()

### Community 42 - "Verifier"
Cohesion: 0.17
Nodes (18): Verifier, diffModel(), TestDiffModel(), checkJoin(), checkReads(), checkSubject(), checkUnion(), compareRead() (+10 more)

### Community 43 - "Adapter"
Cohesion: 0.13
Nodes (5): affected(), Adapter, joinRow, modelRow, unionRow

### Community 44 - "Condition"
Cohesion: 0.16
Nodes (15): Chain, Condition, And(), FixedError(), HasAddedFields(), HasClearedFields(), HasFields(), HasOp() (+7 more)

### Community 45 - "Metrics"
Cohesion: 0.17
Nodes (12): Task 6: Matrix runner, Cell, Metrics, buildCell(), record(), median(), medianMetrics(), TestMedian() (+4 more)

### Community 46 - "Adapter"
Cohesion: 0.17
Nodes (4): Adapter, modelRow, affected(), toRow()

### Community 47 - "Adapter"
Cohesion: 0.17
Nodes (5): Task 9: go-jet adapter and codegen, Customers, Orders, Adapter, joinRow

### Community 50 - "client.go"
Cohesion: 0.18
Nodes (3): Debug(), Option, Log()

### Community 51 - "Step-by-Step Tasks"
Cohesion: 0.14
Nodes (11): Decisions made by this plan (flag to the user; change before implementing if they disagree), Step-by-Step Tasks, Task 12: Documentation, Task 13: Full validation and review, Task 1: Tooling module and Makefile, Task 4: Reference `Verifier` on `databasesql`, Task 5: sqlx adapter, Task 7: gorm adapter (+3 more)

### Community 52 - "fakeLib"
Cohesion: 0.19
Nodes (3): fakeLib, seedJoin(), seedUnion()

### Community 53 - "config"
Cohesion: 0.16
Nodes (6): config, hooks, inters, newConfig(), NewCustomerClient(), NewModelClient()

### Community 54 - "CustomerClient"
Cohesion: 0.21
Nodes (3): CustomerClient, ID(), newCustomerMutation()

### Community 55 - "Customer"
Cohesion: 0.18
Nodes (3): Customer, Model, Order

### Community 56 - "Implementation Report: Core adapters (Phase 3)"
Cohesion: 0.15
Nodes (14): Files to Change, Mandatory Reading, Assessment vs Reality, Coverage (with DB, `-race`), Deviations from the plan, Implementation Report: Core adapters (Phase 3), Notes for Phase 4 / 5, Review findings (`ecc:go-reviewer`) (+6 more)

### Community 58 - "UnionRow"
Cohesion: 0.14
Nodes (11): Edge Cases Checklist, Integration Tests (`BENCH_DSN`, named `...Integration`, `-p 1`), Task 2: `bench.Verifier` types, Testing Strategy, Unit Tests (no DB), TestRegistryNamesMatchAdapters(), UnionRow, SortUnionRows() (+3 more)

### Community 60 - "txDriver"
Cohesion: 0.18
Nodes (4): CommitHook, RollbackHook, txDriver, newTx()

### Community 61 - "Order"
Cohesion: 0.18
Nodes (3): OrderEdges, NewOrderClient(), Order

### Community 62 - "JoinRow"
Cohesion: 0.21
Nodes (7): JoinRow, SortJoinRows(), TestSortJoinRows(), diffJoin(), sameJoinRow(), TestDiffJoin(), checkGoldenJoin()

### Community 63 - "Schema"
Cohesion: 0.21
Nodes (5): Driver(), Create(), Schema, NewSchema(), progress()

### Community 69 - "model/model.go"
Cohesion: 0.33
Nodes (10): ByAge(), ByCounter(), ByFax(), ByID(), ByName(), ByRight(), ByTitle(), ByWeb() (+2 more)

### Community 72 - "mutation.go"
Cohesion: 0.29
Nodes (6): customerOption, modelOption, withCustomer(), withCustomerID(), withModel(), withModelID()

### Community 73 - "customer/customer.go"
Cohesion: 0.38
Nodes (8): ByCity(), ByEmail(), ByID(), ByName(), ByOrders(), ByOrdersCount(), OrderOption, newOrdersStep()

### Community 74 - "order/order.go"
Cohesion: 0.36
Nodes (8): ByCreatedAt(), ByCustomerField(), ByCustomerID(), ByID(), ByStatus(), ByTotal(), OrderOption, newCustomerStep()

### Community 76 - "newOrderMutation"
Cohesion: 0.25
Nodes (4): orderOption, newOrderMutation(), withOrder(), withOrderID()

### Community 80 - ".Mutate"
Cohesion: 0.32
Nodes (3): CustomerFunc, ModelFunc, OrderFunc

### Community 86 - "scanWithInterceptors"
Cohesion: 0.40
Nodes (3): querierAll(), scanWithInterceptors(), withInterceptors()

## Knowledge Gaps
- **10 isolated node(s):** `github.com/restuu/go-sql-benchmark`, `github.com/restuu/go-sql-benchmark/tools`, `PRD open questions (weights, stress parameters, dataset size)`, `Next steps (code review, git init, Phase 2 plan)`, `Deviations from plan` (+5 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 252 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **35 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `ModelMutation` connect `ModelMutation` to `ModelCreate`, `Op`, `ModelDelete`, `CustomerMutation`, `Client`, `entgo.io/ent.Value`, `Tx`, `mutation.go`, `.Client`, `newOrderMutation`, `.sqlSave`, `.SetCounter`, `model/where.go`, `.SetAge`, `ModelClient`, `context.Context`, `ModelUpdateOne`?**
  _High betweenness centrality (0.073) - this node is a cross-community bridge._
- **What connects `github.com/restuu/go-sql-benchmark`, `github.com/restuu/go-sql-benchmark/tools`, `PRD open questions (weights, stress parameters, dataset size)` to the rest of the system?**
  _10 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `bench/main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.11192930780559647 - nodes in this community are weakly interconnected._
- **Why does `CustomerMutation` connect `CustomerMutation` to `Op`, `CustomerDelete`, `.sqlSave`, `Client`, `entgo.io/ent.Value`, `CustomerCreate`, `Tx`, `mutation.go`, `.Client`, `newOrderMutation`, `customer/where.go`, `CustomerClient`, `.SetField`, `.ClearOrders`, `context.Context`, `CustomerUpdateOne`?**
  _High betweenness centrality (0.065) - this node is a cross-community bridge._
- **Should `fakeAdapter` be split into smaller, more focused modules?**
  _Cohesion score 0.09309309309309309 - nodes in this community are weakly interconnected._
- **Why does `OrderMutation` connect `OrderMutation` to `Op`, `OrderDelete`, `entgo.io/ent.Value`, `Tx`, `OrderCreate`, `mutation.go`, `.Client`, `newOrderMutation`, `order/where.go`, `.sqlSave`, `.SetField`, `.SetTotal`, `context.Context`, `OrderUpdateOne`?**
  _High betweenness centrality (0.056) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.06901960784313725 - nodes in this community are weakly interconnected._