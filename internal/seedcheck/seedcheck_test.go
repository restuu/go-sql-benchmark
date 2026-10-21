package seedcheck

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"testing"
	"time"
)

const integrationTimeout = 10 * time.Second

func TestDiffCounts(t *testing.T) {
	tests := []struct {
		name string
		want map[string]int64
		got  map[string]int64
		out  []string
	}{
		{
			name: "equal",
			want: map[string]int64{"orders": 10000, "customers": 1000},
			got:  map[string]int64{"orders": 10000, "customers": 1000},
			out:  nil,
		},
		{
			name: "empty maps",
			want: map[string]int64{},
			got:  map[string]int64{},
			out:  nil,
		},
		{
			name: "mismatch",
			want: map[string]int64{"orders": 10000},
			got:  map[string]int64{"orders": 9},
			out:  []string{"orders: want 10000, got 9"},
		},
		{
			name: "missing table",
			want: map[string]int64{"suppliers": 1000},
			got:  map[string]int64{},
			out:  []string{"suppliers: missing"},
		},
		{
			name: "extra table ignored",
			want: map[string]int64{"orders": 10000},
			got:  map[string]int64{"orders": 10000, "extra": 5},
			out:  nil,
		},
		{
			name: "sorted",
			want: map[string]int64{"orders": 10000, "customers": 1000},
			got:  map[string]int64{"orders": 1, "customers": 2},
			out:  []string{"customers: want 1000, got 2", "orders: want 10000, got 1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (inputs from tt)

			// Act
			out := diffCounts(tt.want, tt.got)

			// Assert
			if !slices.Equal(out, tt.out) {
				t.Fatalf("diffCounts() = %v, want %v", out, tt.out)
			}
		})
	}
}

func TestCheckIntegration(t *testing.T) {
	// Arrange
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	// Act
	err = Check(ctx, db)

	// Assert
	if err != nil {
		t.Fatalf("Check() = %v, want nil", err)
	}
}
