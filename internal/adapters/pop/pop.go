// Package pop is the gobuffalo/pop adapter. pop cannot take an existing *sql.DB; it opens its own
// pool, but its PostgreSQL dialect uses the same pgx stdlib driver as every other adapter.
//
// Known limits: pop's native Destroy reports no rows affected, so DELETE returns 1 for any
// error-free call (the benchmark only deletes existing ids; a missing id would also report 1).
// UNION is reported as unsupported: pop's query API has no UNION, only RawQuery.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package pop

import (
	"context"
	"fmt"
	"time"

	"github.com/gobuffalo/pop/v6"
	_ "github.com/jackc/pgx/v5/stdlib" // pop's pgx driver

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

type modelRow struct {
	ID      int    `db:"id"`
	Name    string `db:"name"`
	Title   string `db:"title"`
	Fax     string `db:"fax"`
	Web     string `db:"web"`
	Age     int    `db:"age"`
	Right   bool   `db:"right"`
	Counter int64  `db:"counter"`
}

func (modelRow) TableName() string { return "models" }

// joinRow is the destination of the orders-customers JOIN; the query aliases the columns.
type joinRow struct {
	OrderID    int       `db:"order_id"`
	CustomerID int       `db:"customer_id"`
	Status     string    `db:"status"`
	Total      int       `db:"total"`
	CreatedAt  time.Time `db:"created_at"`
	Name       string    `db:"name"`
	Email      string    `db:"email"`
	City       string    `db:"city"`
}

func (joinRow) TableName() string { return "orders" }

// updateColumns lists every mutable column so a zero value (a false Right) is still written.
var updateColumns = []string{"name", "title", "fax", "web", "age", "right", "counter"}

var joinColumns = []string{
	"orders.id AS order_id", "orders.customer_id", "orders.status", "orders.total", "orders.created_at",
	"customers.name", "customers.email", "customers.city",
}

// Adapter implements bench.Adapter and bench.Verifier with pop. It is not safe to call Init concurrently.
type Adapter struct{ conn *pop.Connection }

var (
	_ bench.Verifier              = (*Adapter)(nil)
	_ bench.DeleteCountUnreported = (*Adapter)(nil)
)

// DeleteCountUnreported marks Delete as returning 1 for any error-free call.
func (*Adapter) DeleteCountUnreported() {}

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "pop" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	conn, err := pop.NewConnection(&pop.ConnectionDetails{
		Dialect: "postgres", Driver: "pgx", URL: dsn, Pool: pool.MaxOpen, IdlePool: pool.MaxIdle,
	})
	if err != nil {
		return fmt.Errorf("pop: connection: %w", err)
	}
	if err := conn.Open(); err != nil {
		return fmt.Errorf("pop: open: %w", err)
	}
	// Open does not connect; fail here, not on the first scenario call.
	if err := conn.WithContext(ctx).RawQuery("SELECT 1").Exec(); err != nil {
		_ = conn.Close()
		return fmt.Errorf("pop: ping: %w", err)
	}
	a.conn = conn
	return nil
}

func (a *Adapter) Close() error {
	if a.conn == nil {
		return nil
	}
	err := a.conn.Close()
	a.conn = nil
	if err != nil {
		return fmt.Errorf("pop: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	row := toRow(0, m)
	if err := a.conn.WithContext(ctx).Create(&row); err != nil {
		return 0, fmt.Errorf("pop: insert: %w", err)
	}
	return 1, nil // Create reports no count; an error-free insert is one row
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	row := toRow(0, m)
	n, err := a.conn.WithContext(ctx).Where("id = ?", id).UpdateQuery(&row, updateColumns...)
	if err != nil {
		return 0, fmt.Errorf("pop: update: %w", err)
	}
	return int(n), nil
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	if err := a.conn.WithContext(ctx).Destroy(&modelRow{ID: id}); err != nil {
		return 0, fmt.Errorf("pop: delete: %w", err)
	}
	return 1, nil // Destroy reports no count; see the package comment
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

// Union is unsupported: pop only offers UNION through RawQuery.
func (a *Adapter) Union(context.Context, string) (int, error) { return 0, bench.ErrUnsupported }

func (a *Adapter) selectModel(ctx context.Context, id int) (modelRow, error) {
	var row modelRow
	if err := a.conn.WithContext(ctx).Find(&row, id); err != nil {
		return modelRow{}, fmt.Errorf("pop: select: %w", err)
	}
	return row, nil
}

func (a *Adapter) joinRows(ctx context.Context, customerID int) ([]joinRow, error) {
	var rows []joinRow
	err := a.conn.WithContext(ctx).Select(joinColumns...).
		Join("customers", "customers.id = orders.customer_id").
		Where("customers.id = ?", customerID).
		All(&rows)
	if err != nil {
		return nil, fmt.Errorf("pop: join: %w", err)
	}
	return rows, nil
}

func toRow(id int, m bench.Model) modelRow {
	return modelRow{ID: id, Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web, Age: m.Age, Right: m.Right, Counter: m.Counter}
}
