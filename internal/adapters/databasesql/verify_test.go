package databasesql

import (
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

func TestVerifierIntegration(t *testing.T) {
	// Arrange
	a, _, ctx := newInitialised(t)
	v, ok := a.(bench.Verifier)
	if !ok {
		t.Fatal("databasesql adapter does not implement bench.Verifier")
	}

	// Act
	model, modelErr := v.ReadModel(ctx, 1)
	join, joinErr := v.ReadJoin(ctx, 7)
	union, unionErr := v.ReadUnion(ctx, "city 3")
	_, missingErr := v.ReadModel(ctx, missingID)

	// Assert
	if modelErr != nil || joinErr != nil || unionErr != nil {
		t.Fatalf("reads = %v, %v, %v", modelErr, joinErr, unionErr)
	}
	want := bench.NewModel()
	want.ID = 1
	if model != want {
		t.Errorf("ReadModel(1) = %+v, want %+v", model, want)
	}
	if len(join) != 10 {
		t.Errorf("ReadJoin(7) rows = %d, want 10", len(join))
	}
	for _, r := range join {
		if r.CustomerID != 7 || r.Name != "customer 7" {
			t.Errorf("ReadJoin(7) row = %+v, want customer 7", r)
		}
	}
	if len(union) != 40 {
		t.Errorf("ReadUnion(city 3) rows = %d, want 40", len(union))
	}
	if missingErr == nil {
		t.Error("ReadModel() of a missing id = nil, want error")
	}
}
