package gormgen

import (
	"context"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

func (a *Adapter) ReadModel(ctx context.Context, id int) (bench.Model, error) {
	row, err := a.selectModel(ctx, id)
	if err != nil {
		return bench.Model{}, err
	}
	return bench.Model{
		ID: int(row.ID), Name: row.Name, Title: row.Title, Fax: row.Fax, Web: row.Web,
		Age: int(row.Age), Right: row.Right, Counter: row.Counter,
	}, nil
}

func (a *Adapter) ReadJoin(ctx context.Context, customerID int) ([]bench.JoinRow, error) {
	rows, err := a.joinRows(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := make([]bench.JoinRow, len(rows))
	for i, r := range rows {
		out[i] = bench.JoinRow{
			OrderID: int(r.OrderID), CustomerID: int(r.CustomerID), Status: r.Status, Total: int(r.Total),
			CreatedAt: r.CreatedAt, Name: r.Name, Email: r.Email, City: r.City,
		}
	}
	return out, nil
}

// ReadUnion is unsupported, as on Adapter.Union.
func (a *Adapter) ReadUnion(context.Context, string) ([]bench.UnionRow, error) {
	return nil, bench.ErrUnsupported
}
