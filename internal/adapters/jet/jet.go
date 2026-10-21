// Package jet is the go-jet adapter: statements built with the generated type-safe SQL builder in
// ./gen (regenerate with `make gen-jet`), running on database/sql over the pgx stdlib driver.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package jet

import (
	"context"
	"database/sql"
	"fmt"

	jetpg "github.com/go-jet/jet/v2/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver

	"github.com/restuu/go-sql-benchmark/internal/adapters/jet/gen/bench/public/model"
	. "github.com/restuu/go-sql-benchmark/internal/adapters/jet/gen/bench/public/table"
	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// joinRow is the destination jet maps an orders+customers row onto, by table-qualified column.
type joinRow struct {
	model.Orders
	Customer model.Customers
}

// Adapter implements bench.Adapter and bench.Verifier with go-jet. It is not safe to call Init concurrently.
type Adapter struct{ db *sql.DB }

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "jet" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("jet: open: %w", err)
	}
	db.SetMaxOpenConns(pool.MaxOpen)
	db.SetMaxIdleConns(pool.MaxIdle)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("jet: ping: %w", err)
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
		return fmt.Errorf("jet: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	res, err := Models.INSERT(Models.MutableColumns).MODEL(toRow(m)).ExecContext(ctx, a.db)
	return affected("insert", res, err)
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	res, err := Models.UPDATE(Models.MutableColumns).MODEL(toRow(m)).
		WHERE(Models.ID.EQ(jetpg.Int(int64(id)))).ExecContext(ctx, a.db)
	return affected("update", res, err)
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	res, err := Models.DELETE().WHERE(Models.ID.EQ(jetpg.Int(int64(id)))).ExecContext(ctx, a.db)
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

func (a *Adapter) selectModel(ctx context.Context, id int) (model.Models, error) {
	var dest model.Models
	err := jetpg.SELECT(Models.AllColumns).FROM(Models).
		WHERE(Models.ID.EQ(jetpg.Int(int64(id)))).QueryContext(ctx, a.db, &dest)
	if err != nil {
		return model.Models{}, fmt.Errorf("jet: select: %w", err)
	}
	return dest, nil
}

func (a *Adapter) joinRows(ctx context.Context, customerID int) ([]joinRow, error) {
	var dest []joinRow
	err := jetpg.SELECT(Orders.AllColumns, Customers.AllColumns).
		FROM(Orders.INNER_JOIN(Customers, Customers.ID.EQ(Orders.CustomerID))).
		WHERE(Customers.ID.EQ(jetpg.Int(int64(customerID)))).QueryContext(ctx, a.db, &dest)
	if err != nil {
		return nil, fmt.Errorf("jet: join: %w", err)
	}
	return dest, nil
}

func (a *Adapter) unionRows(ctx context.Context, city string) ([]model.Customers, error) {
	customers := jetpg.SELECT(Customers.Name, Customers.Email, Customers.City).
		FROM(Customers).WHERE(Customers.City.EQ(jetpg.String(city)))
	suppliers := jetpg.SELECT(Suppliers.Name, Suppliers.Email, Suppliers.City).
		FROM(Suppliers).WHERE(Suppliers.City.EQ(jetpg.String(city)))
	var dest []model.Customers
	err := jetpg.UNION(customers, suppliers).QueryContext(ctx, a.db, &dest)
	if err != nil {
		return nil, fmt.Errorf("jet: union: %w", err)
	}
	return dest, nil
}

func toRow(m bench.Model) model.Models {
	return model.Models{
		Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web,
		Age: int32(m.Age), Right: m.Right, Counter: m.Counter,
	}
}

// affected converts an Exec result to the rows-affected count the runner verifies.
func affected(op string, res sql.Result, err error) (int, error) {
	if err != nil {
		return 0, fmt.Errorf("jet: %s: %w", op, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("jet: %s: rows affected: %w", op, err)
	}
	return int(n), nil
}
