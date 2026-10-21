package databasesql

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/runner"
)

const integrationTimeout = 30 * time.Second

const missingID = 99_999_999

// newInitialised returns an adapter connected to BENCH_DSN, a store on the same database and a
// cleanup that reseeds so later tests and `make db-check` see the seeded state.
func newInitialised(t *testing.T) (bench.Adapter, runner.Store, context.Context) {
	t.Helper()
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	t.Cleanup(cancel)

	storeDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open store db: %v", err)
	}
	store := runner.NewPGStore(storeDB)
	if err := store.Reseed(ctx); err != nil {
		t.Fatalf("reseed: %v", err)
	}
	a := New()
	if err := a.Init(ctx, dsn, bench.PoolConfig{MaxOpen: 4, MaxIdle: 4}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() {
		_ = a.Close()
		_ = store.Reseed(context.Background())
		_ = storeDB.Close()
	})
	return a, store, ctx
}

func TestAdapterIntegration(t *testing.T) {
	a, store, ctx := newInitialised(t)
	deletable, err := store.InsertDeletable(ctx, 1)
	if err != nil {
		t.Fatalf("InsertDeletable: %v", err)
	}
	tests := []struct {
		name string
		call func() (int, error)
		want int
	}{
		{"insert", func() (int, error) { return a.Insert(ctx, bench.NewModel()) }, 1},
		{"update", func() (int, error) { return a.Update(ctx, 5, bench.NewModel()) }, 1},
		{"delete", func() (int, error) { return a.Delete(ctx, deletable[0]) }, 1},
		{"select", func() (int, error) { return a.Select(ctx, 5) }, 1},
		{"join", func() (int, error) { return a.Join(ctx, 7) }, 10},
		{"union", func() (int, error) { return a.Union(ctx, "city 3") }, 40},
		{"update missing id affects nothing", func() (int, error) { return a.Update(ctx, missingID, bench.NewModel()) }, 0},
		{"delete missing id affects nothing", func() (int, error) { return a.Delete(ctx, missingID) }, 0},
		{"join unknown customer is empty", func() (int, error) { return a.Join(ctx, missingID) }, 0},
		{"union unknown city is empty", func() (int, error) { return a.Union(ctx, "nowhere") }, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (call from tt)

			// Act
			got, err := tt.call()

			// Assert
			if err != nil {
				t.Fatalf("call = %v", err)
			}
			if got != tt.want {
				t.Fatalf("rows = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSelectMissingIDIntegration(t *testing.T) {
	// Arrange
	a, _, ctx := newInitialised(t)

	// Act
	_, err := a.Select(ctx, missingID)

	// Assert
	if err == nil {
		t.Fatal("Select() of a missing id = nil, want error")
	}
}

func TestInitUnreachableDatabaseIntegration(t *testing.T) {
	// Arrange
	if os.Getenv("BENCH_DSN") == "" {
		t.Skip("BENCH_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()
	a := New()

	// Act
	err := a.Init(ctx, "postgres://postgres:bench@127.0.0.1:1/bench?sslmode=disable&connect_timeout=2", bench.PoolConfig{MaxOpen: 1, MaxIdle: 1})

	// Assert
	if err == nil {
		t.Fatal("Init() on an unreachable database = nil, want error")
	}
}

func TestReinitAfterCloseIntegration(t *testing.T) {
	// Arrange
	a, _, ctx := newInitialised(t)
	dsn := os.Getenv("BENCH_DSN")
	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Act
	initErr := a.Init(ctx, dsn, bench.PoolConfig{MaxOpen: 2, MaxIdle: 2})
	rows, selectErr := a.Select(ctx, 1)

	// Assert
	if initErr != nil || selectErr != nil || rows != 1 {
		t.Fatalf("re-Init = %v, Select = %d, %v; want nil, 1, nil", initErr, rows, selectErr)
	}
}

func TestName(t *testing.T) {
	// Arrange
	a := New()

	// Act
	name := a.Name()

	// Assert
	if name != "database_sql" {
		t.Fatalf("Name() = %q, want database_sql", name)
	}
}
