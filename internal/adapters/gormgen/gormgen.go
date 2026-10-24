// Package gormgen is the gorm/gen adapter: the typed query API generated into ./query (regenerate with
// `make gen-gormgen`) on top of GORM, running on database/sql over the pgx stdlib driver. JOIN uses
// the generated Join with a field-level Select, so it is one SQL JOIN. UNION is reported as
// unsupported: gorm/gen's query API has no UNION, only raw SQL. Insert returns 1 for any error-free
// call: Create does not expose rows affected.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package gormgen

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
	"gorm.io/driver/postgres"
	gormlib "gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/restuu/go-sql-benchmark/internal/adapters/gormgen/entity"
	"github.com/restuu/go-sql-benchmark/internal/adapters/gormgen/query"
	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// joinRow is the destination gorm/gen scans an orders+customers row into, by the aliased column names.
type joinRow struct {
	OrderID    int32
	CustomerID int32
	Status     string
	Total      int32
	CreatedAt  time.Time
	Name       string
	Email      string
	City       string
}

// Adapter implements bench.Adapter and bench.Verifier with gorm/gen. It is not safe to call Init concurrently.
type Adapter struct {
	sqlDB *sql.DB
	q     *query.Query
}

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "gorm_gen" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("gorm_gen: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("gorm_gen: ping: %w", err)
	}
	gdb, err := gormlib.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gormlib.Config{
		SkipDefaultTransaction: true, // otherwise every write is wrapped in a transaction
		PrepareStmt:            false,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("gorm_gen: init: %w", err)
	}
	a.sqlDB, a.q = sqlDB, query.Use(gdb)
	return nil
}

func (a *Adapter) Close() error {
	if a.sqlDB == nil {
		return nil
	}
	err := a.sqlDB.Close()
	a.sqlDB, a.q = nil, nil
	if err != nil {
		return fmt.Errorf("gorm_gen: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	row := toRow(m)
	if err := a.q.Model.WithContext(ctx).Create(&row); err != nil {
		return 0, fmt.Errorf("gorm_gen: insert: %w", err)
	}
	return 1, nil
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	row := toRow(m)
	t := a.q.Model
	// Select every mutable column: Updates(struct) skips zero values (a false Right) otherwise.
	info, err := t.WithContext(ctx).Where(t.ID.Eq(int32(id))).
		Select(t.Name, t.Title, t.Fax, t.Web, t.Age, t.Right, t.Counter).Updates(&row)
	if err != nil {
		return 0, fmt.Errorf("gorm_gen: update: %w", err)
	}
	return int(info.RowsAffected), nil
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	t := a.q.Model
	info, err := t.WithContext(ctx).Where(t.ID.Eq(int32(id))).Delete()
	if err != nil {
		return 0, fmt.Errorf("gorm_gen: delete: %w", err)
	}
	return int(info.RowsAffected), nil
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

// Union is unsupported: gorm/gen only offers UNION through raw SQL.
func (a *Adapter) Union(context.Context, string) (int, error) { return 0, bench.ErrUnsupported }

func (a *Adapter) selectModel(ctx context.Context, id int) (*entity.Model, error) {
	t := a.q.Model
	// Take, not First: First adds ORDER BY.
	row, err := t.WithContext(ctx).Where(t.ID.Eq(int32(id))).Take()
	if err != nil {
		return nil, fmt.Errorf("gorm_gen: select: %w", err)
	}
	return row, nil
}

func (a *Adapter) joinRows(ctx context.Context, customerID int) ([]joinRow, error) {
	o, c := a.q.Order, a.q.Customer
	var rows []joinRow
	err := o.WithContext(ctx).Join(c, c.ID.EqCol(o.CustomerID)).Where(o.CustomerID.Eq(int32(customerID))).
		Select(o.ID.As("order_id"), o.CustomerID, o.Status, o.Total, o.CreatedAt, c.Name, c.Email, c.City).
		Scan(&rows)
	if err != nil {
		return nil, fmt.Errorf("gorm_gen: join: %w", err)
	}
	return rows, nil
}

func toRow(m bench.Model) entity.Model {
	return entity.Model{
		Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web,
		Age: int32(m.Age), Right: m.Right, Counter: m.Counter,
	}
}
