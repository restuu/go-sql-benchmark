// Package sqlx is the jmoiron/sqlx adapter: raw SQL through sqlx's struct scanning over the pgx
// stdlib driver. sqlx has had no release since 2024-04.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
	sqlxlib "github.com/jmoiron/sqlx"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

const (
	// driverName makes sqlx pick $n placeholders.
	driverName = "pgx"

	insertSQL = `INSERT INTO models (name, title, fax, web, age, "right", counter) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	updateSQL = `UPDATE models SET name = $1, title = $2, fax = $3, web = $4, age = $5, "right" = $6, counter = $7 WHERE id = $8`
	deleteSQL = `DELETE FROM models WHERE id = $1`
	selectSQL = `SELECT id, name, title, fax, web, age, "right", counter FROM models WHERE id = $1`
	joinSQL   = `SELECT o.id AS order_id, o.customer_id, o.status, o.total, o.created_at, c.name, c.email, c.city
FROM orders o JOIN customers c ON c.id = o.customer_id WHERE c.id = $1`
	unionSQL = `SELECT name, email, city FROM customers WHERE city = $1
UNION
SELECT name, email, city FROM suppliers WHERE city = $1`
)

// modelRow, joinRow and unionRow carry the db tags sqlx maps columns by; sqlx fails on a selected
// column without a destination field.
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

type unionRow struct {
	Name  string `db:"name"`
	Email string `db:"email"`
	City  string `db:"city"`
}

// Adapter implements bench.Adapter and bench.Verifier with sqlx. It is not safe to call Init concurrently.
type Adapter struct{ db *sqlxlib.DB }

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "sqlx" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("sqlx: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("sqlx: ping: %w", err)
	}
	a.db = sqlxlib.NewDb(sqlDB, driverName)
	return nil
}

func (a *Adapter) Close() error {
	if a.db == nil {
		return nil
	}
	err := a.db.Close()
	a.db = nil
	if err != nil {
		return fmt.Errorf("sqlx: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	res, err := a.db.ExecContext(ctx, insertSQL, m.Name, m.Title, m.Fax, m.Web, m.Age, m.Right, m.Counter)
	return affected("insert", res, err)
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	res, err := a.db.ExecContext(ctx, updateSQL, m.Name, m.Title, m.Fax, m.Web, m.Age, m.Right, m.Counter, id)
	return affected("update", res, err)
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	res, err := a.db.ExecContext(ctx, deleteSQL, id)
	return affected("delete", res, err)
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
	if err := a.db.GetContext(ctx, &row, selectSQL, id); err != nil {
		return modelRow{}, fmt.Errorf("sqlx: select: %w", err)
	}
	return row, nil
}

func (a *Adapter) joinRows(ctx context.Context, customerID int) ([]joinRow, error) {
	var rows []joinRow
	if err := a.db.SelectContext(ctx, &rows, joinSQL, customerID); err != nil {
		return nil, fmt.Errorf("sqlx: join: %w", err)
	}
	return rows, nil
}

func (a *Adapter) unionRows(ctx context.Context, city string) ([]unionRow, error) {
	var rows []unionRow
	if err := a.db.SelectContext(ctx, &rows, unionSQL, city); err != nil {
		return nil, fmt.Errorf("sqlx: union: %w", err)
	}
	return rows, nil
}

// affected converts an Exec result to the rows-affected count the runner verifies.
func affected(op string, res sql.Result, err error) (int, error) {
	if err != nil {
		return 0, fmt.Errorf("sqlx: %s: %w", op, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("sqlx: %s: rows affected: %w", op, err)
	}
	return int(n), nil
}
