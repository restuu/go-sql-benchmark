package jet

import (
	"context"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

func (a *Adapter) ReadModel(ctx context.Context, id int) (bench.Model, error) {
	m, err := a.selectModel(ctx, id)
	if err != nil {
		return bench.Model{}, err
	}
	return bench.Model{
		ID: int(m.ID), Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web,
		Age: int(m.Age), Right: m.Right, Counter: m.Counter,
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
			OrderID: int(r.ID), CustomerID: int(r.CustomerID), Status: r.Status, Total: int(r.Total),
			CreatedAt: r.CreatedAt, Name: r.Customer.Name, Email: r.Customer.Email, City: r.Customer.City,
		}
	}
	return out, nil
}

func (a *Adapter) ReadUnion(ctx context.Context, city string) ([]bench.UnionRow, error) {
	rows, err := a.unionRows(ctx, city)
	if err != nil {
		return nil, err
	}
	out := make([]bench.UnionRow, len(rows))
	for i, r := range rows {
		out[i] = bench.UnionRow{Name: r.Name, Email: r.Email, City: r.City}
	}
	return out, nil
}
