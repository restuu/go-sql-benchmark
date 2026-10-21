package sqlc

import (
	"context"
	"fmt"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

func (a *Adapter) ReadModel(ctx context.Context, id int) (bench.Model, error) {
	m, err := a.queries.GetModel(ctx, int32(id))
	if err != nil {
		return bench.Model{}, fmt.Errorf("sqlc: read model: %w", err)
	}
	return bench.Model{
		ID: int(m.ID), Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web,
		Age: int(m.Age), Right: m.Right, Counter: m.Counter,
	}, nil
}

func (a *Adapter) ReadJoin(ctx context.Context, customerID int) ([]bench.JoinRow, error) {
	rows, err := a.queries.JoinOrders(ctx, int32(customerID))
	if err != nil {
		return nil, fmt.Errorf("sqlc: read join: %w", err)
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

func (a *Adapter) ReadUnion(ctx context.Context, city string) ([]bench.UnionRow, error) {
	rows, err := a.queries.UnionByCity(ctx, city)
	if err != nil {
		return nil, fmt.Errorf("sqlc: read union: %w", err)
	}
	out := make([]bench.UnionRow, len(rows))
	for i, r := range rows {
		out[i] = bench.UnionRow{Name: r.Name, Email: r.Email, City: r.City}
	}
	return out, nil
}
