package pgx

import (
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/adaptertest"
	"github.com/restuu/go-sql-benchmark/internal/bench"
)

var constructors = []struct {
	name string
	new  func() bench.Adapter
}{
	{"pgx", New},
	{"pgx_pool", NewPool},
}

func TestContractIntegration(t *testing.T) {
	for _, c := range constructors {
		t.Run(c.name, func(t *testing.T) { adaptertest.RunContract(t, c.new) })
	}
}

func TestVerifierIntegration(t *testing.T) {
	for _, c := range constructors {
		t.Run(c.name, func(t *testing.T) { adaptertest.RunVerifier(t, c.new) })
	}
}

func TestName(t *testing.T) {
	for _, c := range constructors {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			a := c.new()

			// Act
			name := a.Name()

			// Assert
			if name != c.name {
				t.Fatalf("Name() = %q, want %q", name, c.name)
			}
		})
	}
}

func TestInitRejectsBadDSN(t *testing.T) {
	for _, c := range constructors {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			a := c.new()

			// Act
			err := a.Init(t.Context(), "://not a dsn", bench.PoolConfig{MaxOpen: 1, MaxIdle: 1})

			// Assert
			if err == nil {
				t.Fatal("Init() with a malformed DSN succeeded")
			}
		})
	}
}

func TestCloseBeforeInit(t *testing.T) {
	for _, c := range constructors {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			a := c.new()

			// Act
			err := a.Close()

			// Assert
			if err != nil {
				t.Fatalf("Close() before Init = %v", err)
			}
		})
	}
}

func TestInitRejectsBadPoolSize(t *testing.T) {
	const validDSN = "postgres://postgres:x@127.0.0.1:1/bench"
	sizes := []struct {
		name    string
		maxOpen int
	}{
		{"zero", 0},
		{"negative", -1},
		{"overflows int32", math.MaxInt32 + 1},
	}
	for _, c := range constructors {
		for _, tt := range sizes {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				// Arrange
				a := c.new()

				// Act
				err := a.Init(t.Context(), validDSN, bench.PoolConfig{MaxOpen: tt.maxOpen, MaxIdle: 1})

				// Assert
				if err == nil || !strings.Contains(err.Error(), "MaxOpen") {
					t.Fatalf("Init(MaxOpen=%d) error = %v, want one naming MaxOpen", tt.maxOpen, err)
				}
			})
		}
	}
}

func TestCloseTwiceIntegration(t *testing.T) {
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	for _, c := range constructors {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			a := c.new()
			if err := a.Init(t.Context(), dsn, bench.PoolConfig{MaxOpen: 2, MaxIdle: 2}); err != nil {
				t.Fatalf("Init() = %v", err)
			}

			// Act
			first := a.Close()
			second := a.Close()

			// Assert
			if first != nil || second != nil {
				t.Fatalf("Close() = %v then %v, want nil twice", first, second)
			}
		})
	}
}

func TestEmptyJoinAgreesIntegration(t *testing.T) {
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	ctx := t.Context()
	read := func(newAdapter func() bench.Adapter) (int, int) {
		t.Helper()
		a := newAdapter()
		if err := a.Init(ctx, dsn, bench.PoolConfig{MaxOpen: 2, MaxIdle: 2}); err != nil {
			t.Fatalf("Init() = %v", err)
		}
		t.Cleanup(func() { _ = a.Close() })
		join, err := a.Join(ctx, -1)
		if err != nil {
			t.Fatalf("Join() = %v", err)
		}
		union, err := a.Union(ctx, "nowhere")
		if err != nil {
			t.Fatalf("Union() = %v", err)
		}
		return join, union
	}

	// Act
	pgxJoin, pgxUnion := read(New)
	poolJoin, poolUnion := read(NewPool)

	// Assert
	if !reflect.DeepEqual([]int{pgxJoin, pgxUnion}, []int{poolJoin, poolUnion}) || pgxJoin != 0 || pgxUnion != 0 {
		t.Fatalf("empty reads: pgx = (%d, %d), pgx_pool = (%d, %d), want (0, 0) for both", pgxJoin, pgxUnion, poolJoin, poolUnion)
	}
}
