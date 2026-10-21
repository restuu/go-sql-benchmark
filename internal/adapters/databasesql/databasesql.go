// Package databasesql is the reference adapter: plain database/sql over the pgx stdlib driver.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package databasesql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

const (
	insertSQL = `INSERT INTO models (name, title, fax, web, age, "right", counter) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	updateSQL = `UPDATE models SET name = $1, title = $2, fax = $3, web = $4, age = $5, "right" = $6, counter = $7 WHERE id = $8`
	deleteSQL = `DELETE FROM models WHERE id = $1`
	selectSQL = `SELECT id, name, title, fax, web, age, "right", counter FROM models WHERE id = $1`
	joinSQL   = `SELECT o.id, o.customer_id, o.status, o.total, o.created_at, c.name, c.email, c.city
FROM orders o JOIN customers c ON c.id = o.customer_id WHERE c.id = $1`
	unionSQL = `SELECT name, email, city FROM customers WHERE city = $1
UNION
SELECT name, email, city FROM suppliers WHERE city = $1`
)

// Adapter implements bench.Adapter with database/sql. It is not safe to call Init concurrently.
type Adapter struct{ db *sql.DB }

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "database_sql" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("databasesql: open: %w", err)
	}
	db.SetMaxOpenConns(pool.MaxOpen)
	db.SetMaxIdleConns(pool.MaxIdle)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("databasesql: ping: %w", err)
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
		return fmt.Errorf("databasesql: close: %w", err)
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
	var m bench.Model
	err := a.db.QueryRowContext(ctx, selectSQL, id).Scan(&m.ID, &m.Name, &m.Title, &m.Fax, &m.Web, &m.Age, &m.Right, &m.Counter)
	if err != nil {
		return 0, fmt.Errorf("databasesql: select: %w", err)
	}
	return 1, nil
}

func (a *Adapter) Join(ctx context.Context, customerID int) (int, error) {
	rows, err := a.db.QueryContext(ctx, joinSQL, customerID)
	if err != nil {
		return 0, fmt.Errorf("databasesql: join: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		orderID, custID, total int
		status                 string
		createdAt              time.Time
		name, email, city      string
		count                  int
	)
	for rows.Next() {
		if err := rows.Scan(&orderID, &custID, &status, &total, &createdAt, &name, &email, &city); err != nil {
			return 0, fmt.Errorf("databasesql: join: scan: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("databasesql: join: %w", err)
	}
	return count, nil
}

func (a *Adapter) Union(ctx context.Context, city string) (int, error) {
	rows, err := a.db.QueryContext(ctx, unionSQL, city)
	if err != nil {
		return 0, fmt.Errorf("databasesql: union: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		name, email, rowCity string
		count                int
	)
	for rows.Next() {
		if err := rows.Scan(&name, &email, &rowCity); err != nil {
			return 0, fmt.Errorf("databasesql: union: scan: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("databasesql: union: %w", err)
	}
	return count, nil
}

// affected converts an Exec result to the rows-affected count the runner verifies.
func affected(op string, res sql.Result, err error) (int, error) {
	if err != nil {
		return 0, fmt.Errorf("databasesql: %s: %w", op, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("databasesql: %s: rows affected: %w", op, err)
	}
	return int(n), nil
}
