// Package bun is the uptrace/bun adapter, running on database/sql over the pgx stdlib driver.
// JOIN uses bun's belongs-to relation, which compiles to a single SQL JOIN; UNION uses bun's
// Union query builder.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package bun

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
	bunlib "github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

type modelRow struct {
	bunlib.BaseModel `bun:"table:models,alias:m"`

	ID      int `bun:",pk,autoincrement"`
	Name    string
	Title   string
	Fax     string
	Web     string
	Age     int
	Right   bool
	Counter int64
}

type customerRow struct {
	bunlib.BaseModel `bun:"table:customers,alias:c"`

	ID    int `bun:",pk"`
	Name  string
	Email string
	City  string
}

type supplierRow struct {
	bunlib.BaseModel `bun:"table:suppliers,alias:s"`

	Name  string
	Email string
	City  string
}

type orderRow struct {
	bunlib.BaseModel `bun:"table:orders,alias:o"`

	ID         int `bun:",pk"`
	CustomerID int
	Status     string
	Total      int
	CreatedAt  time.Time
	Customer   *customerRow `bun:"rel:belongs-to,join:customer_id=id"`
}

// unionRow is a plain destination: the UNION result belongs to no table.
type unionRow struct {
	Name  string
	Email string
	City  string
}

// updateColumns lists every mutable column so a zero value (a false Right) is still written.
var updateColumns = []string{"name", "title", "fax", "web", "age", "right", "counter"}

// Adapter implements bench.Adapter and bench.Verifier with bun. It is not safe to call Init concurrently.
type Adapter struct{ db *bunlib.DB }

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "bun" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("bun: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("bun: ping: %w", err)
	}
	a.db = bunlib.NewDB(sqlDB, pgdialect.New())
	return nil
}

func (a *Adapter) Close() error {
	if a.db == nil {
		return nil
	}
	err := a.db.Close()
	a.db = nil
	if err != nil {
		return fmt.Errorf("bun: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	row := toRow(m)
	res, err := a.db.NewInsert().Model(&row).Exec(ctx)
	return affected("insert", res, err)
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	row := toRow(m)
	res, err := a.db.NewUpdate().Model(&row).Column(updateColumns...).Where("id = ?", id).Exec(ctx)
	return affected("update", res, err)
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	res, err := a.db.NewDelete().Model((*modelRow)(nil)).Where("id = ?", id).Exec(ctx)
	return affected("delete", res, err)
}

func (a *Adapter) Select(ctx context.Context, id int) (int, error) {
	if _, err := a.selectModel(ctx, id); err != nil {
		return 0, err
	}
	return 1, nil
}

func (a *Adapter) Join(ctx context.Context, customerID int) (int, error) {
	orders, err := a.joinOrders(ctx, customerID)
	return len(orders), err
}

func (a *Adapter) Union(ctx context.Context, city string) (int, error) {
	rows, err := a.unionRows(ctx, city)
	return len(rows), err
}

func (a *Adapter) selectModel(ctx context.Context, id int) (modelRow, error) {
	var row modelRow
	if err := a.db.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx); err != nil {
		return modelRow{}, fmt.Errorf("bun: select: %w", err)
	}
	return row, nil
}

func (a *Adapter) joinOrders(ctx context.Context, customerID int) ([]orderRow, error) {
	var orders []orderRow
	err := a.db.NewSelect().Model(&orders).Relation("Customer").Where("o.customer_id = ?", customerID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun: join: %w", err)
	}
	return orders, nil
}

func (a *Adapter) unionRows(ctx context.Context, city string) ([]unionRow, error) {
	customers := a.db.NewSelect().Model((*customerRow)(nil)).Column("name", "email", "city").Where("city = ?", city)
	suppliers := a.db.NewSelect().Model((*supplierRow)(nil)).Column("name", "email", "city").Where("city = ?", city)
	var rows []unionRow
	if err := customers.Union(suppliers).Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("bun: union: %w", err)
	}
	return rows, nil
}

func toRow(m bench.Model) modelRow {
	return modelRow{Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web, Age: m.Age, Right: m.Right, Counter: m.Counter}
}

// affected converts an Exec result to the rows-affected count the runner verifies.
func affected(op string, res sql.Result, err error) (int, error) {
	if err != nil {
		return 0, fmt.Errorf("bun: %s: %w", op, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("bun: %s: rows affected: %w", op, err)
	}
	return int(n), nil
}
