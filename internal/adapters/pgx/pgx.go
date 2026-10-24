// Package pgx holds the two native jackc/pgx adapters, both on a pgxpool. "pgx" materialises rows
// with pgx's own CollectRows/RowToStructByName helpers; "pgx_pool" is the lean driver floor, a
// plain Query plus Scan into locals. Neither goes through database/sql. Upstream's single pgx.Conn
// is not safe for the runner's concurrent workers, so both use a pool sized by bench.PoolConfig.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package pgx

import (
	"context"
	"fmt"
	"math"
	"time"

	pgxlib "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

const (
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

// modelRow, joinRow and unionRow carry the db tags RowToStructByName maps columns by.
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

// base is the pool and the write scenarios, identical for both adapters. Statements run through
// pgx's default statement cache; there is no explicit Prepare. It is not safe to call Init concurrently.
type base struct {
	name string
	pool *pgxpool.Pool
}

func (b *base) Name() string { return b.name }

func (b *base) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	if pool.MaxOpen < 1 || pool.MaxOpen > math.MaxInt32 {
		return fmt.Errorf("%s: MaxOpen %d out of range [1, %d]", b.name, pool.MaxOpen, math.MaxInt32)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("%s: parse dsn: %w", b.name, err)
	}
	cfg.MaxConns = int32(pool.MaxOpen)
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("%s: open: %w", b.name, err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return fmt.Errorf("%s: ping: %w", b.name, err)
	}
	b.pool = p
	return nil
}

func (b *base) Close() error {
	if b.pool == nil {
		return nil
	}
	b.pool.Close()
	b.pool = nil
	return nil
}

func (b *base) Insert(ctx context.Context, m bench.Model) (int, error) {
	tag, err := b.pool.Exec(ctx, insertSQL, m.Name, m.Title, m.Fax, m.Web, m.Age, m.Right, m.Counter)
	return b.affected("insert", tag.RowsAffected(), err)
}

func (b *base) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	tag, err := b.pool.Exec(ctx, updateSQL, m.Name, m.Title, m.Fax, m.Web, m.Age, m.Right, m.Counter, id)
	return b.affected("update", tag.RowsAffected(), err)
}

func (b *base) Delete(ctx context.Context, id int) (int, error) {
	tag, err := b.pool.Exec(ctx, deleteSQL, id)
	return b.affected("delete", tag.RowsAffected(), err)
}

func (b *base) affected(op string, n int64, err error) (int, error) {
	if err != nil {
		return 0, fmt.Errorf("%s: %s: %w", b.name, op, err)
	}
	return int(n), nil
}

// Adapter is "pgx": reads are collected into structs by pgx's row helpers.
type Adapter struct{ base }

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised "pgx" adapter.
func New() bench.Adapter { return &Adapter{base{name: "pgx"}} }

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
	rows, err := a.pool.Query(ctx, selectSQL, id)
	if err != nil {
		return modelRow{}, fmt.Errorf("pgx: select: %w", err)
	}
	row, err := pgxlib.CollectOneRow(rows, pgxlib.RowToStructByName[modelRow])
	if err != nil {
		return modelRow{}, fmt.Errorf("pgx: select: %w", err)
	}
	return row, nil
}

func (a *Adapter) joinRows(ctx context.Context, customerID int) ([]joinRow, error) {
	rows, err := a.pool.Query(ctx, joinSQL, customerID)
	if err != nil {
		return nil, fmt.Errorf("pgx: join: %w", err)
	}
	out, err := pgxlib.CollectRows(rows, pgxlib.RowToStructByName[joinRow])
	if err != nil {
		return nil, fmt.Errorf("pgx: join: %w", err)
	}
	return out, nil
}

func (a *Adapter) unionRows(ctx context.Context, city string) ([]unionRow, error) {
	rows, err := a.pool.Query(ctx, unionSQL, city)
	if err != nil {
		return nil, fmt.Errorf("pgx: union: %w", err)
	}
	out, err := pgxlib.CollectRows(rows, pgxlib.RowToStructByName[unionRow])
	if err != nil {
		return nil, fmt.Errorf("pgx: union: %w", err)
	}
	return out, nil
}
