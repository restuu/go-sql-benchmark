package bench

import (
	"cmp"
	"context"
	"slices"
	"time"
)

// JoinRow is one row of the JOIN scenario (orders JOIN customers), in the reference column order.
type JoinRow struct {
	OrderID, CustomerID int
	Status              string
	Total               int
	CreatedAt           time.Time
	Name, Email, City   string
}

// UnionRow is one row of the UNION scenario.
type UnionRow struct{ Name, Email, City string }

// Verifier is implemented by adapters so the correctness check can compare the values their
// queries return. The runner never calls it. A scenario the library cannot express returns
// ErrUnsupported, as on Adapter.
type Verifier interface {
	ReadModel(ctx context.Context, id int) (Model, error)
	ReadJoin(ctx context.Context, customerID int) ([]JoinRow, error)
	ReadUnion(ctx context.Context, city string) ([]UnionRow, error)
}

// SortJoinRows orders rows by order id. Adapter queries have no ORDER BY (it would cost time the
// other libraries do not pay), so the correctness check sorts before comparing.
func SortJoinRows(rows []JoinRow) {
	slices.SortFunc(rows, func(a, b JoinRow) int { return cmp.Compare(a.OrderID, b.OrderID) })
}

// SortUnionRows orders rows by name, which is unique within one city of the seed.
func SortUnionRows(rows []UnionRow) {
	slices.SortFunc(rows, func(a, b UnionRow) int { return cmp.Compare(a.Name, b.Name) })
}
