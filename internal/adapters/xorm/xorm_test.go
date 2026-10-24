package xorm

import (
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/adaptertest"
)

func TestContractIntegration(t *testing.T) {
	adaptertest.RunContract(t, New)
}

func TestVerifierIntegration(t *testing.T) {
	adaptertest.RunVerifier(t, New)
}

func TestName(t *testing.T) {
	// Arrange
	a := New()

	// Act
	name := a.Name()

	// Assert
	if name != "xorm" {
		t.Fatalf("Name() = %q, want xorm", name)
	}
}
