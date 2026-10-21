package correctness

import (
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

func TestDiffModel(t *testing.T) {
	base := bench.Model{ID: 1, Name: "n", Title: "t", Fax: "f", Web: "w", Age: 7, Right: true, Counter: 42}
	swapped := base
	swapped.Title, swapped.Fax = base.Fax, base.Title
	falseRight := base
	falseRight.Right = false
	tests := []struct {
		name string
		got  bench.Model
		want bench.Model
		diff bool
	}{
		{"equal", base, base, false},
		{"title and fax swapped", swapped, base, true},
		{"right differs", falseRight, base, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (models from tt)

			// Act
			msg := diffModel(tt.got, tt.want)

			// Assert
			if (msg != "") != tt.diff {
				t.Fatalf("diffModel() = %q, want diff=%v", msg, tt.diff)
			}
		})
	}
}

func TestDiffJoin(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 6, 0, 0, time.UTC)
	row := func(id int) bench.JoinRow {
		return bench.JoinRow{OrderID: id, CustomerID: 7, Status: "new", Total: 1, CreatedAt: at, Name: "customer 7", Email: "e", City: "c"}
	}
	otherZone := row(1)
	otherZone.CreatedAt = at.In(time.FixedZone("x", 7*3600))
	wrongName := row(2)
	wrongName.Name = "other"
	tests := []struct {
		name string
		got  []bench.JoinRow
		want []bench.JoinRow
		diff bool
	}{
		{"equal", []bench.JoinRow{row(1), row(2)}, []bench.JoinRow{row(1), row(2)}, false},
		{"order independent", []bench.JoinRow{row(2), row(1)}, []bench.JoinRow{row(1), row(2)}, false},
		{"same instant in another zone", []bench.JoinRow{otherZone}, []bench.JoinRow{row(1)}, false},
		{"missing row", []bench.JoinRow{row(1)}, []bench.JoinRow{row(1), row(2)}, true},
		{"extra row", []bench.JoinRow{row(1), row(2)}, []bench.JoinRow{row(1)}, true},
		{"field differs", []bench.JoinRow{row(1), wrongName}, []bench.JoinRow{row(1), row(2)}, true},
		{"both empty", nil, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (rows from tt)

			// Act
			msg := diffJoin(tt.got, tt.want)

			// Assert
			if (msg != "") != tt.diff {
				t.Fatalf("diffJoin() = %q, want diff=%v", msg, tt.diff)
			}
		})
	}
}

func TestDiffUnion(t *testing.T) {
	a := bench.UnionRow{Name: "a", Email: "a@x", City: "c"}
	b := bench.UnionRow{Name: "b", Email: "b@x", City: "c"}
	wrongEmail := b
	wrongEmail.Email = "other"
	tests := []struct {
		name string
		got  []bench.UnionRow
		want []bench.UnionRow
		diff bool
	}{
		{"equal", []bench.UnionRow{a, b}, []bench.UnionRow{a, b}, false},
		{"order independent", []bench.UnionRow{b, a}, []bench.UnionRow{a, b}, false},
		{"missing row", []bench.UnionRow{a}, []bench.UnionRow{a, b}, true},
		{"field differs", []bench.UnionRow{a, wrongEmail}, []bench.UnionRow{a, b}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (rows from tt)

			// Act
			msg := diffUnion(tt.got, tt.want)

			// Assert
			if (msg != "") != tt.diff {
				t.Fatalf("diffUnion() = %q, want diff=%v", msg, tt.diff)
			}
		})
	}
}
