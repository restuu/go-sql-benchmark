// Package gorm is the GORM adapter, running on database/sql over the pgx stdlib driver.
// UNION is reported as unsupported: GORM's query API has no UNION, only raw SQL.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package gorm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
	"gorm.io/driver/postgres"
	gormlib "gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

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

func (modelRow) TableName() string { return "models" }

type customerRow struct {
	ID    int
	Name  string
	Email string
	City  string
}

func (customerRow) TableName() string { return "customers" }

type orderRow struct {
	ID         int
	CustomerID int
	Status     string
	Total      int
	CreatedAt  time.Time
	Customer   customerRow `gorm:"foreignKey:CustomerID"`
}

func (orderRow) TableName() string { return "orders" }

// updateColumns lists every mutable column: Updates(struct) skips zero values (a false Right
// would never be written) unless the columns are selected explicitly.
var updateColumns = []string{"name", "title", "fax", "web", "age", "right", "counter"}

// Adapter implements bench.Adapter and bench.Verifier with GORM. It is not safe to call Init concurrently.
type Adapter struct {
	sqlDB *sql.DB
	db    *gormlib.DB
}

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "gorm" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("gorm: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("gorm: ping: %w", err)
	}
	gdb, err := gormlib.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gormlib.Config{
		SkipDefaultTransaction: true, // otherwise every write is wrapped in a transaction
		PrepareStmt:            false,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("gorm: init: %w", err)
	}
	a.sqlDB, a.db = sqlDB, gdb
	return nil
}

func (a *Adapter) Close() error {
	if a.sqlDB == nil {
		return nil
	}
	err := a.sqlDB.Close()
	a.sqlDB, a.db = nil, nil
	if err != nil {
		return fmt.Errorf("gorm: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	row := toRow(m)
	res := a.db.WithContext(ctx).Create(&row)
	return affected("insert", res)
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	row := toRow(m)
	// Not Save: with a primary key set it inserts when the update matches nothing.
	res := a.db.WithContext(ctx).Model(&modelRow{}).Where("id = ?", id).Select(updateColumns).Updates(&row)
	return affected("update", res)
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	res := a.db.WithContext(ctx).Delete(&modelRow{}, id)
	return affected("delete", res)
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

// Union is unsupported: GORM only offers UNION through raw SQL.
func (a *Adapter) Union(context.Context, string) (int, error) { return 0, bench.ErrUnsupported }

func (a *Adapter) selectModel(ctx context.Context, id int) (modelRow, error) {
	var row modelRow
	// Take, not First: First adds ORDER BY.
	if err := a.db.WithContext(ctx).Take(&row, id).Error; err != nil {
		return modelRow{}, fmt.Errorf("gorm: select: %w", err)
	}
	return row, nil
}

func (a *Adapter) joinOrders(ctx context.Context, customerID int) ([]orderRow, error) {
	var orders []orderRow
	err := a.db.WithContext(ctx).Joins("Customer").Where("orders.customer_id = ?", customerID).Find(&orders).Error
	if err != nil {
		return nil, fmt.Errorf("gorm: join: %w", err)
	}
	return orders, nil
}

func toRow(m bench.Model) modelRow {
	return modelRow{Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web, Age: m.Age, Right: m.Right, Counter: m.Counter}
}

// affected converts a GORM result to the rows-affected count the runner verifies.
func affected(op string, res *gormlib.DB) (int, error) {
	if res.Error != nil {
		return 0, fmt.Errorf("gorm: %s: %w", op, res.Error)
	}
	return int(res.RowsAffected), nil
}
