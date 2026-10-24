package pgx

import (
	"context"
	"fmt"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// PoolAdapter is "pgx_pool": reads are a plain Query plus Scan into locals, like the database/sql
// reference, so it shows the cost of the driver and pool alone.
type PoolAdapter struct{ base }

var _ bench.Verifier = (*PoolAdapter)(nil)

// NewPool returns an uninitialised "pgx_pool" adapter.
func NewPool() bench.Adapter { return &PoolAdapter{base{name: "pgx_pool"}} }

func (a *PoolAdapter) Select(ctx context.Context, id int) (int, error) {
	if _, err := a.ReadModel(ctx, id); err != nil {
		return 0, err
	}
	return 1, nil
}

func (a *PoolAdapter) Join(ctx context.Context, customerID int) (int, error) {
	rows, err := a.pool.Query(ctx, joinSQL, customerID)
	if err != nil {
		return 0, fmt.Errorf("pgx_pool: join: %w", err)
	}
	defer rows.Close()

	var (
		r     bench.JoinRow
		count int
	)
	for rows.Next() {
		if err := rows.Scan(&r.OrderID, &r.CustomerID, &r.Status, &r.Total, &r.CreatedAt, &r.Name, &r.Email, &r.City); err != nil {
			return 0, fmt.Errorf("pgx_pool: join: scan: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("pgx_pool: join: %w", err)
	}
	return count, nil
}

func (a *PoolAdapter) Union(ctx context.Context, city string) (int, error) {
	rows, err := a.pool.Query(ctx, unionSQL, city)
	if err != nil {
		return 0, fmt.Errorf("pgx_pool: union: %w", err)
	}
	defer rows.Close()

	var (
		r     bench.UnionRow
		count int
	)
	for rows.Next() {
		if err := rows.Scan(&r.Name, &r.Email, &r.City); err != nil {
			return 0, fmt.Errorf("pgx_pool: union: scan: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("pgx_pool: union: %w", err)
	}
	return count, nil
}

func (a *PoolAdapter) ReadModel(ctx context.Context, id int) (bench.Model, error) {
	var m bench.Model
	err := a.pool.QueryRow(ctx, selectSQL, id).Scan(&m.ID, &m.Name, &m.Title, &m.Fax, &m.Web, &m.Age, &m.Right, &m.Counter)
	if err != nil {
		return bench.Model{}, fmt.Errorf("pgx_pool: select: %w", err)
	}
	return m, nil
}

func (a *PoolAdapter) ReadJoin(ctx context.Context, customerID int) ([]bench.JoinRow, error) {
	rows, err := a.pool.Query(ctx, joinSQL, customerID)
	if err != nil {
		return nil, fmt.Errorf("pgx_pool: read join: %w", err)
	}
	defer rows.Close()

	var out []bench.JoinRow
	for rows.Next() {
		var r bench.JoinRow
		if err := rows.Scan(&r.OrderID, &r.CustomerID, &r.Status, &r.Total, &r.CreatedAt, &r.Name, &r.Email, &r.City); err != nil {
			return nil, fmt.Errorf("pgx_pool: read join: scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgx_pool: read join: %w", err)
	}
	return out, nil
}

func (a *PoolAdapter) ReadUnion(ctx context.Context, city string) ([]bench.UnionRow, error) {
	rows, err := a.pool.Query(ctx, unionSQL, city)
	if err != nil {
		return nil, fmt.Errorf("pgx_pool: read union: %w", err)
	}
	defer rows.Close()

	var out []bench.UnionRow
	for rows.Next() {
		var r bench.UnionRow
		if err := rows.Scan(&r.Name, &r.Email, &r.City); err != nil {
			return nil, fmt.Errorf("pgx_pool: read union: scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgx_pool: read union: %w", err)
	}
	return out, nil
}
