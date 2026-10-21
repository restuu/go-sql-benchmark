// Package sqlc is the sqlc adapter: queries in queries.sql compiled to Go by sqlc, running on
// database/sql over the pgx stdlib driver. Regenerate with `make gen-sqlc`.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package sqlc

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver

	"github.com/restuu/go-sql-benchmark/internal/adapters/sqlc/db"
	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// Adapter implements bench.Adapter and bench.Verifier with sqlc. It is not safe to call Init concurrently.
type Adapter struct {
	sqlDB   *sql.DB
	queries *db.Queries
}

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "sqlc" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("sqlc: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("sqlc: ping: %w", err)
	}
	a.sqlDB = sqlDB
	a.queries = db.New(sqlDB)
	return nil
}

func (a *Adapter) Close() error {
	if a.sqlDB == nil {
		return nil
	}
	err := a.sqlDB.Close()
	a.sqlDB, a.queries = nil, nil
	if err != nil {
		return fmt.Errorf("sqlc: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	n, err := a.queries.CreateModel(ctx, db.CreateModelParams{
		Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web,
		Age: int32(m.Age), Right: m.Right, Counter: m.Counter,
	})
	return rows("insert", n, err)
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	n, err := a.queries.UpdateModel(ctx, db.UpdateModelParams{
		Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web,
		Age: int32(m.Age), Right: m.Right, Counter: m.Counter, ID: int32(id),
	})
	return rows("update", n, err)
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	n, err := a.queries.DeleteModel(ctx, int32(id))
	return rows("delete", n, err)
}

func (a *Adapter) Select(ctx context.Context, id int) (int, error) {
	if _, err := a.queries.GetModel(ctx, int32(id)); err != nil {
		return 0, fmt.Errorf("sqlc: select: %w", err)
	}
	return 1, nil
}

func (a *Adapter) Join(ctx context.Context, customerID int) (int, error) {
	r, err := a.queries.JoinOrders(ctx, int32(customerID))
	if err != nil {
		return 0, fmt.Errorf("sqlc: join: %w", err)
	}
	return len(r), nil
}

func (a *Adapter) Union(ctx context.Context, city string) (int, error) {
	r, err := a.queries.UnionByCity(ctx, city)
	if err != nil {
		return 0, fmt.Errorf("sqlc: union: %w", err)
	}
	return len(r), nil
}

// rows converts the :execrows result to the count the runner verifies.
func rows(op string, n int64, err error) (int, error) {
	if err != nil {
		return 0, fmt.Errorf("sqlc: %s: %w", op, err)
	}
	return int(n), nil
}
