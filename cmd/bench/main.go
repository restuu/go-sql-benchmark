// Command bench runs the benchmark matrix against the seeded database and writes a JSON result.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"slices"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/adapters/databasesql"
	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/result"
	"github.com/restuu/go-sql-benchmark/internal/runner"
	"github.com/restuu/go-sql-benchmark/internal/seedcheck"
)

const (
	checkTimeout   = 10 * time.Second
	exitUsageError = 2
	exitRunError   = 1
	pgxModulePath  = "github.com/jackc/pgx/v5"
)

// registry maps -libs names to adapter constructors; Phase 3 adapters register here.
var registry = map[string]func() bench.Adapter{
	"database_sql": databasesql.New,
}

func main() {
	cfg, err := parseFlags(os.Args[1:], registryNames(), time.Now())
	if errors.Is(err, flag.ErrHelp) {
		fmt.Println(err)
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitUsageError)
	}
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "bench: BENCH_DSN is required")
		os.Exit(exitUsageError)
	}
	if err := run(cfg, dsn); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitRunError)
	}
	fmt.Println(cfg.out)
}

func registryNames() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func run(cfg config, dsn string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	db, err := sql.Open("pgx", dsn) // driver registered by the databasesql import
	if err != nil {
		return fmt.Errorf("bench: open: %w", err)
	}
	defer func() { _ = db.Close() }()

	checkCtx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	if err := seedcheck.Check(checkCtx, db); err != nil {
		return err
	}
	host, err := hostInfo(checkCtx, db)
	if err != nil {
		return err
	}

	factories := make([]func() bench.Adapter, len(cfg.libs))
	for i, name := range cfg.libs {
		factories[i] = registry[name]
	}
	fmt.Fprintf(os.Stderr, "bench: seed %d, %d repeats\n", cfg.seed, cfg.repeats)
	run, err := runner.Run(ctx, runner.NewPGStore(db), runner.Options{
		DSN: dsn, Adapters: factories, Scenarios: cfg.scenarios, Profiles: cfg.profiles,
		Repeats: cfg.repeats, Seed: cfg.seed, Progress: os.Stderr,
	}, host)
	if err != nil {
		return err
	}
	return result.Write(cfg.out, run)
}

func hostInfo(ctx context.Context, db *sql.DB) (result.Host, error) {
	var pgVersion string
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&pgVersion); err != nil {
		return result.Host{}, fmt.Errorf("bench: read server version: %w", err)
	}
	return result.Host{
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, GoVersion: runtime.Version(),
		NumCPU: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0),
		PostgresVersion: pgVersion,
		Libraries:       map[string]string{"pgx": moduleVersion(pgxModulePath)},
	}, nil
}

// moduleVersion returns the version of a dependency linked into this binary, or "unknown".
func moduleVersion(path string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path == path {
			return dep.Version
		}
	}
	return "unknown"
}
