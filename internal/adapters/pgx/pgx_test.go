package pgx

import (
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
