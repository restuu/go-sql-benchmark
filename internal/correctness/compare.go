package correctness

import (
	"fmt"
	"slices"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// diffModel returns "" when the models are equal, else a description of the difference.
func diffModel(got, want bench.Model) string {
	if got == want {
		return ""
	}
	return fmt.Sprintf("got %+v, want %+v", got, want)
}

// diffJoin compares two row sets ignoring order and time zone. It does not modify its arguments.
func diffJoin(got, want []bench.JoinRow) string {
	got, want = slices.Clone(got), slices.Clone(want)
	bench.SortJoinRows(got)
	bench.SortJoinRows(want)
	if len(got) != len(want) {
		return fmt.Sprintf("got %d rows, want %d", len(got), len(want))
	}
	for i := range got {
		if !sameJoinRow(got[i], want[i]) {
			return fmt.Sprintf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	return ""
}

func sameJoinRow(a, b bench.JoinRow) bool {
	return a.OrderID == b.OrderID && a.CustomerID == b.CustomerID && a.Status == b.Status &&
		a.Total == b.Total && a.CreatedAt.Equal(b.CreatedAt) &&
		a.Name == b.Name && a.Email == b.Email && a.City == b.City
}

// diffUnion compares two row sets ignoring order. It does not modify its arguments.
func diffUnion(got, want []bench.UnionRow) string {
	got, want = slices.Clone(got), slices.Clone(want)
	bench.SortUnionRows(got)
	bench.SortUnionRows(want)
	if len(got) != len(want) {
		return fmt.Sprintf("got %d rows, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			return fmt.Sprintf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	return ""
}
