package databasesql

import (
	"context"
	"fmt"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

var _ bench.Verifier = (*Adapter)(nil)

func (a *Adapter) ReadModel(ctx context.Context, id int) (bench.Model, error) {
	var m bench.Model
	err := a.db.QueryRowContext(ctx, selectSQL, id).Scan(&m.ID, &m.Name, &m.Title, &m.Fax, &m.Web, &m.Age, &m.Right, &m.Counter)
	if err != nil {
		return bench.Model{}, fmt.Errorf("databasesql: read model: %w", err)
	}
	return m, nil
}

func (a *Adapter) ReadJoin(ctx context.Context, customerID int) ([]bench.JoinRow, error) {
	rows, err := a.db.QueryContext(ctx, joinSQL, customerID)
	if err != nil {
		return nil, fmt.Errorf("databasesql: read join: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []bench.JoinRow
	for rows.Next() {
		var r bench.JoinRow
		if err := rows.Scan(&r.OrderID, &r.CustomerID, &r.Status, &r.Total, &r.CreatedAt, &r.Name, &r.Email, &r.City); err != nil {
			return nil, fmt.Errorf("databasesql: read join: scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("databasesql: read join: %w", err)
	}
	return out, nil
}

func (a *Adapter) ReadUnion(ctx context.Context, city string) ([]bench.UnionRow, error) {
	rows, err := a.db.QueryContext(ctx, unionSQL, city)
	if err != nil {
		return nil, fmt.Errorf("databasesql: read union: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []bench.UnionRow
	for rows.Next() {
		var r bench.UnionRow
		if err := rows.Scan(&r.Name, &r.Email, &r.City); err != nil {
			return nil, fmt.Errorf("databasesql: read union: scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("databasesql: read union: %w", err)
	}
	return out, nil
}
