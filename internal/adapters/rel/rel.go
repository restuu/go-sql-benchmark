// Package rel is the go-rel/rel adapter, running on database/sql over the pgx stdlib driver
// (postgres.New takes the existing *sql.DB). Writes go through the query API (UpdateAny,
// DeleteAny) because those report rows affected, where Update(entity) would return a NotFound
// error for a missing id. JOIN is a query-builder JOIN, one SQL statement. UNION is reported as
// unsupported: rel's query API has no UNION. go-rel/postgres has had no release since 2024-05.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package rel

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	rellib "github.com/go-rel/postgres"
	"github.com/go-rel/rel"
	"github.com/go-rel/rel/where"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

const (
	modelsTable = "models"
	ordersTable = "orders"
)

// modelRow maps to models by field name (snake_case); the field ID is the primary key.
type modelRow struct {
	ID      int
	Name    string
	Title   string
	Fax     string
	Web     string
	Age     int
	Right   bool
	Counter int64
}

func (modelRow) Table() string { return modelsTable }

// joinRow is the destination of the orders-customers JOIN; the columns are aliased by the query.
type joinRow struct {
	OrderID    int
	CustomerID int
	Status     string
	Total      int
	CreatedAt  time.Time
	Name       string
	Email      string
	City       string
}

func (joinRow) Table() string { return ordersTable }

// Adapter implements bench.Adapter and bench.Verifier with rel. It is not safe to call Init concurrently.
type Adapter struct {
	sqlDB *sql.DB
	repo  rel.Repository
}

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "rel" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("rel: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("rel: ping: %w", err)
	}
	repo := rel.New(rellib.New(sqlDB))
	repo.Instrumentation(silent) // rel logs every statement by default
	a.sqlDB, a.repo = sqlDB, repo
	return nil
}

// silent is a no-op instrumenter: errors are returned to the caller, not reported here.
func silent(context.Context, string, string, ...any) func(error) { return func(error) {} }

func (a *Adapter) Close() error {
	if a.sqlDB == nil {
		return nil
	}
	err := a.sqlDB.Close()
	a.sqlDB, a.repo = nil, nil
	if err != nil {
		return fmt.Errorf("rel: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	row := toRow(0, m)
	if err := a.repo.Insert(ctx, &row); err != nil {
		return 0, fmt.Errorf("rel: insert: %w", err)
	}
	return 1, nil // Insert reports no count; an error-free insert is one row
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	n, err := a.repo.UpdateAny(ctx, rel.From(modelsTable).Where(where.Eq("id", id)),
		rel.Set("name", m.Name), rel.Set("title", m.Title), rel.Set("fax", m.Fax), rel.Set("web", m.Web),
		rel.Set("age", m.Age), rel.Set("right", m.Right), rel.Set("counter", m.Counter))
	if err != nil {
		return 0, fmt.Errorf("rel: update: %w", err)
	}
	return n, nil
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	n, err := a.repo.DeleteAny(ctx, rel.From(modelsTable).Where(where.Eq("id", id)))
	if err != nil {
		return 0, fmt.Errorf("rel: delete: %w", err)
	}
	return n, nil
}

func (a *Adapter) Select(ctx context.Context, id int) (int, error) {
	if _, err := a.selectModel(ctx, id); err != nil {
		return 0, err
	}
	return 1, nil
}

func (a *Adapter) Join(ctx context.Context, customerID int) (int, error) {
	rows, err := a.joinRows(ctx, customerID)
	return len(rows), err
}

// Union is unsupported: rel's query API has no UNION.
func (a *Adapter) Union(context.Context, string) (int, error) { return 0, bench.ErrUnsupported }

func (a *Adapter) selectModel(ctx context.Context, id int) (modelRow, error) {
	var row modelRow
	if err := a.repo.Find(ctx, &row, where.Eq("id", id)); err != nil {
		return modelRow{}, fmt.Errorf("rel: select: %w", err)
	}
	return row, nil
}

func (a *Adapter) joinRows(ctx context.Context, customerID int) ([]joinRow, error) {
	var rows []joinRow
	query := rel.Select("orders.id AS order_id", "orders.customer_id", "orders.status", "orders.total",
		"orders.created_at", "customers.name", "customers.email", "customers.city").
		From(ordersTable).
		JoinOn("customers", "orders.customer_id", "customers.id").
		Where(where.Eq("customers.id", customerID))
	if err := a.repo.FindAll(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("rel: join: %w", err)
	}
	return rows, nil
}

func toRow(id int, m bench.Model) modelRow {
	return modelRow{ID: id, Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web, Age: m.Age, Right: m.Right, Counter: m.Counter}
}
