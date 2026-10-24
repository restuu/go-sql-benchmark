package beego

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/adaptertest"
	"github.com/restuu/go-sql-benchmark/internal/bench"
)

func TestContractIntegration(t *testing.T) {
	adaptertest.RunContract(t, New, bench.ScenarioUnion)
}

func TestVerifierIntegration(t *testing.T) {
	adaptertest.RunVerifier(t, New, bench.ScenarioUnion)
}

func TestName(t *testing.T) {
	// Arrange
	a := New()

	// Act
	name := a.Name()

	// Assert
	if name != "beego" {
		t.Fatalf("Name() = %q, want beego", name)
	}
}

var errTest = errors.New("boom")

func TestRecoverRegistrationTurnsPanicIntoError(t *testing.T) {
	// Arrange
	tests := []struct {
		name    string
		fn      func() error
		wantErr string
	}{
		{"panic", func() error { panic("table name duplicate") }, "table name duplicate"},
		{"error", func() error { return errTest }, "boom"},
		{"ok", func() error { return nil }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := recoverRegistration(tt.fn)

			// Assert
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("recoverRegistration() = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("recoverRegistration() = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestSecondAdapterUsesItsOwnDBIntegration(t *testing.T) {
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		t.Skip("BENCH_DSN not set")
	}
	ctx := t.Context()
	cfg := bench.PoolConfig{MaxOpen: 2, MaxIdle: 2}
	// Arrange
	first := New()
	if err := first.Init(ctx, dsn, cfg); err != nil {
		t.Fatalf("first Init() = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close() = %v", err)
	}
	second := New()
	t.Cleanup(func() { _ = second.Close() })

	// Act
	initErr := second.Init(ctx, dsn, cfg)
	rows, selectErr := second.Select(ctx, 1)

	// Assert
	if initErr != nil || selectErr != nil || rows != 1 {
		t.Fatalf("second adapter: Init = %v, Select = %d, %v; want nil, 1, nil", initErr, rows, selectErr)
	}
}
