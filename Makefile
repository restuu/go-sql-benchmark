BENCH_PG_PORT     ?= 55432
BENCH_PG_PASSWORD ?= bench
BENCH_DSN         ?= postgres://postgres:$(BENCH_PG_PASSWORD)@127.0.0.1:$(BENCH_PG_PORT)/bench?sslmode=disable
export BENCH_PG_PORT BENCH_PG_PASSWORD BENCH_DSN

GO_ARCH   := $(shell go env GOARCH)
HOST_ARCH := $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')

BENCH_ARGS ?=
BENCH_TOL  ?= 0.20
# macOS idle sleep stalls a run (see .claude/PRPs/reports/p95-stability-report.md, E6); no-op elsewhere.
KEEP_AWAKE := $(shell command -v caffeinate >/dev/null 2>&1 && echo 'caffeinate -i')

.PHONY: arch-check db-up db-down db-reset db-seed db-check build vet test test-db bench bench-cmp report gen gen-sqlc gen-ent gen-jet gen-sqlboiler gen-gormgen gen-check

arch-check:
	@if [ "$(GO_ARCH)" != "$(HOST_ARCH)" ]; then \
		echo "Go arch $(GO_ARCH) != host arch $(HOST_ARCH): install the native Go toolchain (benchmark numbers are invalid under emulation)"; \
		exit 1; \
	fi
	@echo "arch OK: $(GO_ARCH)"

db-up:
	docker compose up -d --wait

db-down:
	docker compose down -v

db-reset: db-down db-up

db-seed:
	docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d bench < db/seed.sql

db-check:
	go run ./cmd/dbcheck

build:
	go build ./...

vet:
	go vet ./...
	go vet internal/adapters/gormgen/generate/main.go

test:
	env -u BENCH_DSN go test ./...

test-db: db-up
	go test -p 1 -count=1 -run Integration ./... # one package at a time: they all reseed the same DB

bench: arch-check db-up
	$(KEEP_AWAKE) go run ./cmd/bench $(BENCH_ARGS)

bench-cmp:
	go run ./cmd/benchcmp -tolerance $(BENCH_TOL) $(A) $(B)

report:
	go run ./cmd/report -in $(IN) $(if $(OUT),-out $(OUT))

# Code generators are pinned in tools/go.mod (tool directives) so the main go.mod stays free of them.
GOTOOL   := go tool -modfile=tools/go.mod
GORMGEN_DIRS := internal/adapters/gormgen/entity internal/adapters/gormgen/query
GEN_DIRS := internal/adapters/sqlc/db internal/adapters/ent/entgen internal/adapters/jet/gen internal/adapters/sqlboiler/models internal/adapters/gormgen/entity internal/adapters/gormgen/query

gen: gen-sqlc gen-ent gen-jet gen-sqlboiler gen-gormgen

gen-sqlc:
	$(GOTOOL) sqlc generate -f internal/adapters/sqlc/sqlc.yaml

gen-ent:
	$(GOTOOL) ent generate --target internal/adapters/ent/entgen ./internal/adapters/ent/schema

# jet reads the live schema, so the database must be up.
gen-jet: db-up
	$(GOTOOL) jet -source=postgres -dsn='$(BENCH_DSN)' -path=internal/adapters/jet/gen

# sqlboiler also reads the live schema. It finds its driver by looking for sqlboiler-psql on PATH,
# so the pinned driver is linked into a temporary directory first.
# It takes the connection as PSQL_* variables and cannot read BENCH_DSN, so a BENCH_DSN other than the
# one built from BENCH_PG_PORT and BENCH_PG_PASSWORD is refused instead of silently reading another database.
gen-sqlboiler: db-up
	@if [ '$(BENCH_DSN)' != 'postgres://postgres:$(BENCH_PG_PASSWORD)@127.0.0.1:$(BENCH_PG_PORT)/bench?sslmode=disable' ]; then \
		echo "gen-sqlboiler cannot use a custom BENCH_DSN; set BENCH_PG_PORT and BENCH_PG_PASSWORD instead"; \
		exit 1; \
	fi
	@bin=$$(mktemp -d) && trap 'rm -rf "$$bin"' EXIT && \
	ln -s "$$($(GOTOOL) -n sqlboiler-psql)" "$$bin/sqlboiler-psql" && \
	PATH="$$bin:$$PATH" PSQL_HOST=127.0.0.1 PSQL_PORT='$(BENCH_PG_PORT)' PSQL_USER=postgres PSQL_PASS='$(BENCH_PG_PASSWORD)' \
	PSQL_DBNAME=bench PSQL_SSLMODE=disable \
	$(GOTOOL) sqlboiler psql --output internal/adapters/sqlboiler/models --pkgname models --no-tests --wipe

# gorm/gen has no CLI, so internal/adapters/gormgen/generate is a small generator program (excluded from
# the build, pinned by go.mod like the generated code's own imports); it reads the live schema too.
# The old output is cleared first so renamed tables leave no stale files, and put back if generation fails.
gen-gormgen: db-up
	@bak=$$(mktemp -d) && trap 'rm -rf "$$bak"' EXIT && \
	cp -R $(GORMGEN_DIRS) "$$bak" && rm -rf $(GORMGEN_DIRS) && \
	{ go run internal/adapters/gormgen/generate/main.go || { rm -rf $(GORMGEN_DIRS); cp -R "$$bak"/* internal/adapters/gormgen/; echo "gen-gormgen failed: previous output restored"; exit 1; }; }

# Fails if regenerating changes any generated file in the working tree (commit the result to pin it).
gen-check:
	@sum() { find $(GEN_DIRS) -type f 2>/dev/null | sort | xargs shasum 2>/dev/null; }; \
	before=$$(sum); $(MAKE) --no-print-directory gen; after=$$(sum); \
	if [ "$$before" != "$$after" ]; then echo "generated code drifted: commit the regenerated files"; exit 1; fi; \
	echo "generated code up to date"
