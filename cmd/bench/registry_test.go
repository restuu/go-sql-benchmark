package main

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/runner"
)

func TestRegistryHasCoreLibraries(t *testing.T) {
	// Arrange
	want := []string{"database_sql", "ent", "gorm", "jet", "sqlc", "sqlx"}

	// Act
	got := registryNames()

	// Assert
	if !slices.Equal(got, want) {
		t.Fatalf("registryNames() = %v, want %v", got, want)
	}
}

func TestRegistryNamesMatchAdapters(t *testing.T) {
	for name, newAdapter := range registry {
		t.Run(name, func(t *testing.T) {
			// Arrange
			a := newAdapter()

			// Act
			got := a.Name()

			// Assert
			if got != name {
				t.Fatalf("registry key %q builds an adapter named %q", name, got)
			}
		})
	}
}

func TestLibraryModulesResolve(t *testing.T) {
	for lib, path := range libraryModules {
		t.Run(lib, func(t *testing.T) {
			// Arrange
			// (module path from the table)

			// Act
			version := moduleVersion(path)

			// Assert
			if version == "unknown" {
				t.Fatalf("moduleVersion(%q) = unknown: module not linked into the binary", path)
			}
		})
	}
}

func TestPreflightIntegration(t *testing.T) {
	// Arrange
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), preflightTimeout)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Act
	err = preflight(ctx, dsn, runner.NewPGStore(db), registryNames())

	// Assert
	if err != nil {
		t.Fatalf("preflight() = %v", err)
	}
}
