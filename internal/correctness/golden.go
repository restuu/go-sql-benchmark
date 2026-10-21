package correctness

import (
	"context"
	"fmt"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// Golden spot values derived from db/seed.sql; change both together. They guard against every
// library agreeing with the reference on a wrong value.
const (
	goldenCustomer  = 7
	goldenCity      = "city 3"
	goldenOrderID   = 6 // lowest order of customer 7: orders.id i where (i % 1000) + 1 = 7
	goldenOrderTot  = 7 // (6 % 500) + 1
	goldenOrderStat = "shipped"
	goldenOrderMins = 6 // created_at = 2026-01-01 00:00:00+00 + i minutes
)

// checkReference compares the reference adapter's reads with the golden values.
func checkReference(ctx context.Context, ref bench.Verifier) []string {
	var msgs []string
	want := bench.NewModel()
	want.ID = 1
	if got, err := ref.ReadModel(ctx, 1); err != nil {
		msgs = append(msgs, fmt.Sprintf("reference: read model 1: %v", err))
	} else if d := diffModel(got, want); d != "" {
		msgs = append(msgs, "reference: read model 1: "+d)
	}
	msgs = append(msgs, checkGoldenJoin(ctx, ref)...)
	return append(msgs, checkGoldenUnion(ctx, ref)...)
}

func checkGoldenJoin(ctx context.Context, ref bench.Verifier) []string {
	rows, err := ref.ReadJoin(ctx, goldenCustomer)
	if err != nil {
		return []string{fmt.Sprintf("reference: read join %d: %v", goldenCustomer, err)}
	}
	want := bench.ScenarioJoin.ExpectedRows()
	if len(rows) != want {
		return []string{fmt.Sprintf("reference: read join %d: %d rows, want %d", goldenCustomer, len(rows), want)}
	}
	bench.SortJoinRows(rows)
	first := bench.JoinRow{
		OrderID: goldenOrderID, CustomerID: goldenCustomer, Status: goldenOrderStat, Total: goldenOrderTot,
		CreatedAt: time.Date(2026, time.January, 1, 0, goldenOrderMins, 0, 0, time.UTC),
		Name:      "customer 7", Email: "customer7@example.com", City: "city 7",
	}
	if !sameJoinRow(rows[0], first) {
		return []string{fmt.Sprintf("reference: read join %d: first row %+v, want %+v", goldenCustomer, rows[0], first)}
	}
	for _, r := range rows {
		if r.CustomerID != goldenCustomer {
			return []string{fmt.Sprintf("reference: read join %d: row of customer %d", goldenCustomer, r.CustomerID)}
		}
	}
	return nil
}

func checkGoldenUnion(ctx context.Context, ref bench.Verifier) []string {
	rows, err := ref.ReadUnion(ctx, goldenCity)
	if err != nil {
		return []string{fmt.Sprintf("reference: read union %q: %v", goldenCity, err)}
	}
	want := bench.ScenarioUnion.ExpectedRows()
	if len(rows) != want {
		return []string{fmt.Sprintf("reference: read union %q: %d rows, want %d", goldenCity, len(rows), want)}
	}
	for _, g := range []bench.UnionRow{
		{Name: "customer 3", Email: "customer3@example.com", City: goldenCity},
		{Name: "supplier 3", Email: "supplier3@example.com", City: goldenCity},
	} {
		if !containsUnion(rows, g) {
			return []string{fmt.Sprintf("reference: read union %q: missing %+v", goldenCity, g)}
		}
	}
	return nil
}

func containsUnion(rows []bench.UnionRow, want bench.UnionRow) bool {
	for _, r := range rows {
		if r == want {
			return true
		}
	}
	return false
}
