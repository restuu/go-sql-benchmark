package ent

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
		ID: m.ID, Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web,
		Age: m.Age, Right: m.Right, Counter: m.Counter,
	}, nil
}

func (a *Adapter) ReadJoin(ctx context.Context, customerID int) ([]bench.JoinRow, error) {
	orders, err := a.joinOrders(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := make([]bench.JoinRow, len(orders))
	for i, o := range orders {
		out[i] = bench.JoinRow{
			OrderID: o.ID, CustomerID: o.CustomerID, Status: o.Status, Total: o.Total, CreatedAt: o.CreatedAt,
			Name: o.Edges.Customer.Name, Email: o.Edges.Customer.Email, City: o.Edges.Customer.City,
		}
	}
	return out, nil
}

func (a *Adapter) ReadUnion(context.Context, string) ([]bench.UnionRow, error) {
	return nil, bench.ErrUnsupported
}
