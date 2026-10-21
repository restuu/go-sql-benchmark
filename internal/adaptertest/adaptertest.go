// Package adaptertest is the integration contract every library adapter must pass. Call RunContract
// from a test named ...Integration so `make test-db` runs it; it skips without BENCH_DSN.
package adaptertest

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/runner"
)

const (
	integrationTimeout = 30 * time.Second
	missingID          = 99_999_999
	// unreachableDSN points at a closed port; connect_timeout keeps the failing Init fast.
	unreachableDSN = "postgres://postgres:bench@127.0.0.1:1/bench?sslmode=disable&connect_timeout=2"
	contractPool   = 4
)

// RunContract runs the shared adapter checks. Scenarios listed in unsupported must return
// bench.ErrUnsupported; every other scenario must behave as the reference database/sql adapter.
func RunContract(t *testing.T, newAdapter func() bench.Adapter, unsupported ...bench.Scenario) {
	t.Helper()
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	t.Run("calls", func(t *testing.T) { runCalls(t, dsn, newAdapter, unsupported) })
	t.Run("select of a missing id fails", func(t *testing.T) { runSelectMissing(t, dsn, newAdapter) })
	t.Run("init on an unreachable database fails", func(t *testing.T) { runUnreachable(t, newAdapter) })
	t.Run("init works again after close", func(t *testing.T) { runReinit(t, dsn, newAdapter) })
	t.Run("name is stable", func(t *testing.T) {
		// Arrange
		a := newAdapter()

		// Act
		name := a.Name()

		// Assert
		if name == "" {
			t.Fatal("Name() is empty")
		}
	})
}

// initialised returns an adapter connected to dsn and a store on the same database. The cleanup
// closes the adapter and reseeds so later tests and `make db-check` see the seeded state.
func initialised(t *testing.T, dsn string, newAdapter func() bench.Adapter) (bench.Adapter, runner.Store, context.Context) {
	t.Helper()
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
	a := newAdapter()
	if err := a.Init(ctx, dsn, bench.PoolConfig{MaxOpen: contractPool, MaxIdle: contractPool}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() {
		_ = a.Close()
		_ = store.Reseed(context.Background())
		_ = storeDB.Close()
	})
	return a, store, ctx
}

func runCalls(t *testing.T, dsn string, newAdapter func() bench.Adapter, unsupported []bench.Scenario) {
	a, store, ctx := initialised(t, dsn, newAdapter)
	deletable, err := store.InsertDeletable(ctx, 1)
	if err != nil {
		t.Fatalf("InsertDeletable: %v", err)
	}
	tests := []struct {
		name     string
		scenario bench.Scenario
		call     func() (int, error)
		want     int
	}{
		{"insert", bench.ScenarioInsert, func() (int, error) { return a.Insert(ctx, bench.NewModel()) }, 1},
		{"update", bench.ScenarioUpdate, func() (int, error) { return a.Update(ctx, 5, bench.NewModel()) }, 1},
		{"delete", bench.ScenarioDelete, func() (int, error) { return a.Delete(ctx, deletable[0]) }, 1},
		{"select", bench.ScenarioSelect, func() (int, error) { return a.Select(ctx, 5) }, 1},
		{"join", bench.ScenarioJoin, func() (int, error) { return a.Join(ctx, 7) }, 10},
		{"union", bench.ScenarioUnion, func() (int, error) { return a.Union(ctx, "city 3") }, 40},
		{"update of a missing id affects nothing", bench.ScenarioUpdate, func() (int, error) { return a.Update(ctx, missingID, bench.NewModel()) }, 0},
		{"delete of a missing id affects nothing", bench.ScenarioDelete, func() (int, error) { return a.Delete(ctx, missingID) }, 0},
		{"join of an unknown customer is empty", bench.ScenarioJoin, func() (int, error) { return a.Join(ctx, missingID) }, 0},
		{"union of an unknown city is empty", bench.ScenarioUnion, func() (int, error) { return a.Union(ctx, "nowhere") }, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			wantUnsupported := slices.Contains(unsupported, tt.scenario)

			// Act
			got, err := tt.call()

			// Assert
			if wantUnsupported {
				if !errors.Is(err, bench.ErrUnsupported) {
					t.Fatalf("call = %d, %v; want bench.ErrUnsupported", got, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("call = %v", err)
			}
			if got != tt.want {
				t.Fatalf("rows = %d, want %d", got, tt.want)
			}
		})
	}
}

func runSelectMissing(t *testing.T, dsn string, newAdapter func() bench.Adapter) {
	// Arrange
	a, _, ctx := initialised(t, dsn, newAdapter)

	// Act
	_, err := a.Select(ctx, missingID)

	// Assert
	if err == nil {
		t.Fatal("Select() of a missing id = nil, want error")
	}
}

func runUnreachable(t *testing.T, newAdapter func() bench.Adapter) {
	// Arrange
	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()
	a := newAdapter()

	// Act
	err := a.Init(ctx, unreachableDSN, bench.PoolConfig{MaxOpen: 1, MaxIdle: 1})

	// Assert
	if err == nil {
		_ = a.Close()
		t.Fatal("Init() on an unreachable database = nil, want error")
	}
}

func runReinit(t *testing.T, dsn string, newAdapter func() bench.Adapter) {
	// Arrange
	a, _, ctx := initialised(t, dsn, newAdapter)
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

// RunVerifier checks the adapter's bench.Verifier reads against the seeded data. Scenarios in
// unsupported must return bench.ErrUnsupported.
func RunVerifier(t *testing.T, newAdapter func() bench.Adapter, unsupported ...bench.Scenario) {
	t.Helper()
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	a, _, ctx := initialised(t, dsn, newAdapter)
	v, ok := a.(bench.Verifier)
	if !ok {
		t.Fatalf("%s does not implement bench.Verifier", a.Name())
	}
	t.Run("model", func(t *testing.T) {
		// Arrange
		want := bench.NewModel()
		want.ID = 1

		// Act
		got, err := v.ReadModel(ctx, 1)

		// Assert
		if err != nil || got != want {
			t.Fatalf("ReadModel(1) = %+v, %v; want %+v", got, err, want)
		}
	})
	t.Run("missing model fails", func(t *testing.T) {
		// Arrange
		// (missing id)

		// Act
		_, err := v.ReadModel(ctx, missingID)

		// Assert
		if err == nil {
			t.Fatal("ReadModel() of a missing id = nil, want error")
		}
	})
	t.Run("join", func(t *testing.T) {
		// Arrange
		wantUnsupported := slices.Contains(unsupported, bench.ScenarioJoin)

		// Act
		rows, err := v.ReadJoin(ctx, 7)

		// Assert
		if wantUnsupported {
			if !errors.Is(err, bench.ErrUnsupported) {
				t.Fatalf("ReadJoin() = %v, want bench.ErrUnsupported", err)
			}
			return
		}
		if err != nil || len(rows) != bench.ScenarioJoin.ExpectedRows() {
			t.Fatalf("ReadJoin(7) = %d rows, %v; want %d", len(rows), err, bench.ScenarioJoin.ExpectedRows())
		}
		for _, r := range rows {
			if r.CustomerID != 7 || r.Name != "customer 7" {
				t.Fatalf("ReadJoin(7) row = %+v, want customer 7", r)
			}
		}
	})
	t.Run("union", func(t *testing.T) {
		// Arrange
		wantUnsupported := slices.Contains(unsupported, bench.ScenarioUnion)

		// Act
		rows, err := v.ReadUnion(ctx, "city 3")

		// Assert
		if wantUnsupported {
			if !errors.Is(err, bench.ErrUnsupported) {
				t.Fatalf("ReadUnion() = %v, want bench.ErrUnsupported", err)
			}
			return
		}
		if err != nil || len(rows) != bench.ScenarioUnion.ExpectedRows() {
			t.Fatalf("ReadUnion(city 3) = %d rows, %v; want %d", len(rows), err, bench.ScenarioUnion.ExpectedRows())
		}
	})
}
