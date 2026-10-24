// Package sqlboiler is the SQLBoiler adapter: generated models in ./models (regenerate with
// `make gen-sqlboiler`), running on database/sql over the pgx stdlib driver. JOIN uses qm.InnerJoin
// bound to a struct, so it is one SQL JOIN. UNION is reported as unsupported: SQLBoiler's query mods
// have no UNION, only raw SQL.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package sqlboiler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver

	"github.com/restuu/go-sql-benchmark/internal/adapters/sqlboiler/models"
	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// joinRow is the destination SQLBoiler binds an orders+customers row onto, by the "order." and
// "customer." alias prefixes.
type joinRow struct {
	Order    models.Order    `boil:"order,bind"`
	Customer models.Customer `boil:"customer,bind"`
}

// joinSelect lists the bound columns; the aliases keep the two ids apart.
var joinSelect = qm.Select(
	`orders.id AS "order.id"`, `orders.customer_id AS "order.customer_id"`, `orders.status AS "order.status"`,
	`orders.total AS "order.total"`, `orders.created_at AS "order.created_at"`,
	`customers.id AS "customer.id"`, `customers.name AS "customer.name"`,
	`customers.email AS "customer.email"`, `customers.city AS "customer.city"`,
)

// Adapter implements bench.Adapter and bench.Verifier with SQLBoiler. It is not safe to call Init concurrently.
type Adapter struct{ db *sql.DB }

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "sqlboiler" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("sqlboiler: open: %w", err)
	}
	db.SetMaxOpenConns(pool.MaxOpen)
	db.SetMaxIdleConns(pool.MaxIdle)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("sqlboiler: ping: %w", err)
	}
	a.db = db
	return nil
}

func (a *Adapter) Close() error {
	if a.db == nil {
		return nil
	}
	err := a.db.Close()
	a.db = nil
	if err != nil {
		return fmt.Errorf("sqlboiler: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	row := toRow(m)
	if err := row.Insert(ctx, a.db, boil.Infer()); err != nil {
		return 0, fmt.Errorf("sqlboiler: insert: %w", err)
	}
	return 1, nil
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	row := toRow(m)
	row.ID = id
	// Infer writes every non-key column, zero values included.
	n, err := row.Update(ctx, a.db, boil.Infer())
	if err != nil {
		return 0, fmt.Errorf("sqlboiler: update: %w", err)
	}
	return int(n), nil
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	n, err := (&models.Model{ID: id}).Delete(ctx, a.db)
	if err != nil {
		return 0, fmt.Errorf("sqlboiler: delete: %w", err)
	}
	return int(n), nil
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

// Union is unsupported: SQLBoiler only offers UNION through raw SQL.
func (a *Adapter) Union(context.Context, string) (int, error) { return 0, bench.ErrUnsupported }

func (a *Adapter) selectModel(ctx context.Context, id int) (*models.Model, error) {
	row, err := models.FindModel(ctx, a.db, id)
	if err != nil {
		return nil, fmt.Errorf("sqlboiler: select: %w", err)
	}
	return row, nil
}

func (a *Adapter) joinRows(ctx context.Context, customerID int) ([]joinRow, error) {
	var rows []joinRow
	err := models.Orders(
		joinSelect,
		qm.InnerJoin("customers ON customers.id = orders.customer_id"),
		models.OrderWhere.CustomerID.EQ(customerID),
	).Bind(ctx, a.db, &rows)
	if err != nil {
		return nil, fmt.Errorf("sqlboiler: join: %w", err)
	}
	return rows, nil
}

func toRow(m bench.Model) models.Model {
	return models.Model{Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web, Age: m.Age, Right: m.Right, Counter: m.Counter}
}
