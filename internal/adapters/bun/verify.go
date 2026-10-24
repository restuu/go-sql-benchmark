package bun

import (
	"context"
	"fmt"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

func (a *Adapter) ReadModel(ctx context.Context, id int) (bench.Model, error) {
	row, err := a.selectModel(ctx, id)
	if err != nil {
		return bench.Model{}, err
	}
	return bench.Model{
		ID: row.ID, Name: row.Name, Title: row.Title, Fax: row.Fax, Web: row.Web,
		Age: row.Age, Right: row.Right, Counter: row.Counter,
	}, nil
}

func (a *Adapter) ReadJoin(ctx context.Context, customerID int) ([]bench.JoinRow, error) {
	orders, err := a.joinOrders(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := make([]bench.JoinRow, len(orders))
	for i, o := range orders {
		if o.Customer == nil {
			return nil, fmt.Errorf("bun: order %d has no customer loaded", o.ID)
		}
		out[i] = bench.JoinRow{
			OrderID: o.ID, CustomerID: o.CustomerID, Status: o.Status, Total: o.Total, CreatedAt: o.CreatedAt,
			Name: o.Customer.Name, Email: o.Customer.Email, City: o.Customer.City,
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
		out[i] = bench.UnionRow(r)
	}
	return out, nil
}
