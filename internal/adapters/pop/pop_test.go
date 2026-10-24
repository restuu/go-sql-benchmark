package pop

import (
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
	if name != "pop" {
		t.Fatalf("Name() = %q, want pop", name)
	}
}
