package runner

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/seedcheck"
)

const storeTestTimeout = 60 * time.Second

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestStoreIntegration(t *testing.T) {
	// Arrange
	db := openIntegrationDB(t)
	store := NewPGStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), storeTestTimeout)
	defer cancel()
	t.Cleanup(func() { _ = store.Reseed(context.Background()) }) // leave the DB seeded

	t.Run("reseed restores the seed after drift", func(t *testing.T) {
		// Arrange
		if _, err := db.ExecContext(ctx, `DELETE FROM models WHERE id <= 100`); err != nil {
			t.Fatalf("drift: %v", err)
		}

		// Act
		err := store.Reseed(ctx)

		// Assert
		if err != nil {
			t.Fatalf("Reseed() = %v", err)
		}
		if err := seedcheck.Check(ctx, db); err != nil {
			t.Fatalf("seedcheck after reseed: %v", err)
		}
	})

	t.Run("insert deletable returns fresh distinct ids", func(t *testing.T) {
		// Arrange
		const n = 100
		seeded := int(seedcheck.ExpectedRows["models"])

		// Act
		ids, err := store.InsertDeletable(ctx, n)

		// Assert
		if err != nil {
			t.Fatalf("InsertDeletable() = %v", err)
		}
		if len(ids) != n {
			t.Fatalf("len(ids) = %d, want %d", len(ids), n)
		}
		if lowest := slices.Min(ids); lowest <= seeded {
			t.Errorf("smallest id %d overlaps the seeded range 1..%d", lowest, seeded)
		}
		slices.Sort(ids)
		if len(slices.Compact(ids)) != n {
			t.Errorf("ids are not distinct")
		}
	})

	t.Run("zero ids", func(t *testing.T) {
		// Arrange
		// (no input)

		// Act
		ids, err := store.InsertDeletable(ctx, 0)

		// Assert
		if err != nil || len(ids) != 0 {
			t.Fatalf("InsertDeletable(0) = %v, %v; want empty, nil", ids, err)
		}
	})

	t.Run("negative count is rejected", func(t *testing.T) {
		// Arrange
		// (no input)

		// Act
		_, err := store.InsertDeletable(ctx, -1)

		// Assert
		if err == nil {
			t.Fatal("InsertDeletable(-1) = nil, want error")
		}
	})
}
