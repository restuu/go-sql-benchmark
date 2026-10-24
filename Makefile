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

.PHONY: arch-check db-up db-down db-reset db-seed db-check build vet test test-db bench bench-cmp report gen gen-sqlc gen-ent gen-jet gen-check

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
GEN_DIRS := internal/adapters/sqlc/db internal/adapters/ent/entgen internal/adapters/jet/gen

gen: gen-sqlc gen-ent gen-jet

gen-sqlc:
	$(GOTOOL) sqlc generate -f internal/adapters/sqlc/sqlc.yaml

gen-ent:
	$(GOTOOL) ent generate --target internal/adapters/ent/entgen ./internal/adapters/ent/schema

# jet reads the live schema, so the database must be up.
gen-jet: db-up
	$(GOTOOL) jet -source=postgres -dsn='$(BENCH_DSN)' -path=internal/adapters/jet/gen

# Fails if regenerating changes any generated file in the working tree (commit the result to pin it).
gen-check:
	@sum() { find $(GEN_DIRS) -type f 2>/dev/null | sort | xargs shasum 2>/dev/null; }; \
	before=$$(sum); $(MAKE) --no-print-directory gen; after=$$(sum); \
	if [ "$$before" != "$$after" ]; then echo "generated code drifted: commit the regenerated files"; exit 1; fi; \
	echo "generated code up to date"
