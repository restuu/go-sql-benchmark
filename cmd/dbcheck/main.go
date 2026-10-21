// Command dbcheck verifies the benchmark database is seeded and runs the expected Postgres version.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/seedcheck"
)

const (
	checkTimeout   = 10 * time.Second
	exitUsageError = 2
	exitCheckError = 1
)

func main() {
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "dbcheck: BENCH_DSN is required")
		os.Exit(exitUsageError)
	}
	if err := run(dsn); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCheckError)
	}
	fmt.Println("seed OK: postgres", seedcheck.WantPostgresMajor, "-", len(seedcheck.ExpectedRows), "tables")
}

func run(dsn string) error {
	db, err := sql.Open("pgx", dsn) // driver registered by seedcheck's stdlib import
	if err != nil {
		return fmt.Errorf("dbcheck: open: %w", err)
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	return seedcheck.Check(ctx, db)
}
