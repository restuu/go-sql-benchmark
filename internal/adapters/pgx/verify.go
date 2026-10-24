package pgx

import (
	"context"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

func (a *Adapter) ReadModel(ctx context.Context, id int) (bench.Model, error) {
	row, err := a.selectModel(ctx, id)
	if err != nil {
		return bench.Model{}, err
	}
	return bench.Model(row), nil
}

func (a *Adapter) ReadJoin(ctx context.Context, customerID int) ([]bench.JoinRow, error) {
	rows, err := a.joinRows(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := make([]bench.JoinRow, len(rows))
	for i, r := range rows {
		out[i] = bench.JoinRow(r)
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
