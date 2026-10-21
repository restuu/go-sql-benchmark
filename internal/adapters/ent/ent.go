// Package ent is the ent adapter, running on database/sql over the pgx stdlib driver. Code for
// the client is generated from ./schema by `make gen-ent`. UNION is reported as unsupported: ent's
// typed entity API cannot combine two entities. JOIN uses edge eager-loading (WithCustomer), which
// runs two queries (orders, then their customers) rather than one SQL JOIN.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package ent

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver

	"github.com/restuu/go-sql-benchmark/internal/adapters/ent/entgen"
	"github.com/restuu/go-sql-benchmark/internal/adapters/ent/entgen/model"
	"github.com/restuu/go-sql-benchmark/internal/adapters/ent/entgen/order"
	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// Adapter implements bench.Adapter and bench.Verifier with ent. It is not safe to call Init concurrently.
type Adapter struct{ client *entgen.Client }

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "ent" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("ent: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("ent: ping: %w", err)
	}
	a.client = entgen.NewClient(entgen.Driver(entsql.OpenDB(dialect.Postgres, sqlDB)))
	return nil
}

func (a *Adapter) Close() error {
	if a.client == nil {
		return nil
	}
	err := a.client.Close() // closes the *sql.DB the driver wraps
	a.client = nil
	if err != nil {
		return fmt.Errorf("ent: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	err := a.client.Model.Create().
		SetName(m.Name).SetTitle(m.Title).SetFax(m.Fax).SetWeb(m.Web).
		SetAge(m.Age).SetRight(m.Right).SetCounter(m.Counter).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("ent: insert: %w", err)
	}
	return 1, nil
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	// The bulk builder returns the affected count in one UPDATE; UpdateOneID fails on a missing id
	// and follows the update with a SELECT.
	n, err := a.client.Model.Update().Where(model.ID(id)).
		SetName(m.Name).SetTitle(m.Title).SetFax(m.Fax).SetWeb(m.Web).
		SetAge(m.Age).SetRight(m.Right).SetCounter(m.Counter).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("ent: update: %w", err)
	}
	return n, nil
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	n, err := a.client.Model.Delete().Where(model.ID(id)).Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("ent: delete: %w", err)
	}
	return n, nil
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

// Union is unsupported: ent's entity API has no way to combine two entity types.
func (a *Adapter) Union(context.Context, string) (int, error) { return 0, bench.ErrUnsupported }

func (a *Adapter) selectModel(ctx context.Context, id int) (*entgen.Model, error) {
	m, err := a.client.Model.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ent: select: %w", err)
	}
	return m, nil
}

func (a *Adapter) joinOrders(ctx context.Context, customerID int) ([]*entgen.Order, error) {
	orders, err := a.client.Order.Query().Where(order.CustomerID(customerID)).WithCustomer().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent: join: %w", err)
	}
	return orders, nil
}
