// Package xorm is the xorm adapter, running on database/sql over the pgx stdlib driver. JOIN is
// Table plus Join with an explicit Select; UNION is xorm.io/builder's Union passed to SQL.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package xorm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
	"xorm.io/builder"
	"xorm.io/xorm"
	"xorm.io/xorm/core"
	"xorm.io/xorm/log"
	"xorm.io/xorm/names"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// driverName is the database/sql driver and the xorm dialect key: xorm has a "pgx" PostgreSQL dialect.
const driverName = "pgx"

type modelRow struct {
	ID      int `xorm:"pk autoincr"`
	Name    string
	Title   string
	Fax     string
	Web     string
	Age     int
	Right   bool
	Counter int64
}

func (modelRow) TableName() string { return "models" }

// joinRow is the destination for the aliased orders+customers columns.
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

// unionRow is a plain destination: the UNION result belongs to no table.
type unionRow struct {
	Name  string
	Email string
	City  string
}

// updateColumns lists every mutable column: Update(struct) skips zero values (a false Right) unless
// the columns are named.
var updateColumns = []string{"name", "title", "fax", "web", "age", "right", "counter"}

// Adapter implements bench.Adapter and bench.Verifier with xorm. It is not safe to call Init concurrently.
type Adapter struct {
	sqlDB  *sql.DB
	engine *xorm.Engine
}

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "xorm" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	sqlDB, err := sql.Open(driverName, dsn)
	if err != nil {
		return fmt.Errorf("xorm: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("xorm: ping: %w", err)
	}
	engine, err := xorm.NewEngineWithDB(driverName, dsn, core.FromDB(sqlDB))
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("xorm: init: %w", err)
	}
	engine.SetLogLevel(log.LOG_OFF)
	engine.SetMapper(names.GonicMapper{}) // knows ID and CustomerID; the default SnakeMapper maps ID to i_d
	a.sqlDB, a.engine = sqlDB, engine
	return nil
}

func (a *Adapter) Close() error {
	if a.engine == nil {
		return nil
	}
	err := a.engine.Close()
	a.sqlDB, a.engine = nil, nil
	if err != nil {
		return fmt.Errorf("xorm: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	row := toRow(m)
	n, err := a.engine.Context(ctx).Insert(&row)
	return affected("insert", n, err)
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	row := toRow(m)
	n, err := a.engine.Context(ctx).ID(id).Cols(updateColumns...).Update(&row)
	return affected("update", n, err)
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	n, err := a.engine.Context(ctx).ID(id).Delete(&modelRow{})
	return affected("delete", n, err)
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

func (a *Adapter) Union(ctx context.Context, city string) (int, error) {
	rows, err := a.unionRows(ctx, city)
	return len(rows), err
}

func (a *Adapter) selectModel(ctx context.Context, id int) (modelRow, error) {
	var row modelRow
	has, err := a.engine.Context(ctx).ID(id).Get(&row)
	if err != nil {
		return modelRow{}, fmt.Errorf("xorm: select: %w", err)
	}
	if !has {
		return modelRow{}, fmt.Errorf("xorm: select: model %d: %w", id, sql.ErrNoRows)
	}
	return row, nil
}

func (a *Adapter) joinRows(ctx context.Context, customerID int) ([]joinRow, error) {
	var rows []joinRow
	err := a.engine.Context(ctx).Table("orders").
		Select("orders.id AS order_id, orders.customer_id, orders.status, orders.total, orders.created_at, customers.name, customers.email, customers.city").
		Join("INNER", "customers", "customers.id = orders.customer_id").
		Where("orders.customer_id = ?", customerID).
		Find(&rows)
	if err != nil {
		return nil, fmt.Errorf("xorm: join: %w", err)
	}
	return rows, nil
}

func (a *Adapter) unionRows(ctx context.Context, city string) ([]unionRow, error) {
	customers := builder.Select("name", "email", "city").From("customers").Where(builder.Eq{"city": city})
	suppliers := builder.Select("name", "email", "city").From("suppliers").Where(builder.Eq{"city": city})
	var rows []unionRow
	if err := a.engine.Context(ctx).SQL(customers.Union("distinct", suppliers)).Find(&rows); err != nil {
		return nil, fmt.Errorf("xorm: union: %w", err)
	}
	return rows, nil
}

func toRow(m bench.Model) modelRow {
	return modelRow{Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web, Age: m.Age, Right: m.Right, Counter: m.Counter}
}

// affected converts an xorm count to the rows-affected number the runner verifies.
func affected(op string, n int64, err error) (int, error) {
	if err != nil {
		return 0, fmt.Errorf("xorm: %s: %w", op, err)
	}
	return int(n), nil
}
