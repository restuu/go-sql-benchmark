BENCH_PG_PORT     ?= 55432
BENCH_PG_PASSWORD ?= bench
BENCH_DSN         ?= postgres://postgres:$(BENCH_PG_PASSWORD)@127.0.0.1:$(BENCH_PG_PORT)/bench?sslmode=disable
export BENCH_PG_PORT BENCH_PG_PASSWORD BENCH_DSN

GO_ARCH   := $(shell go env GOARCH)
HOST_ARCH := $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')

.PHONY: arch-check db-up db-down db-reset db-seed db-check build vet test test-db

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
	go test -count=1 -run Integration ./...
