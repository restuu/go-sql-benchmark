// Package beego is the Beego ORM adapter, running on database/sql over the pgx stdlib driver.
// UNION is reported as unsupported: Beego's QuerySeter and QueryBuilder have no UNION, only raw SQL.
// JOIN is RelatedSel on a foreign-key field, which loads the customer in the same query.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package beego

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// dialect is the name of Beego's built-in PostgreSQL driver type; the *sql.DB passed in still
// speaks pgx.
const dialect = "postgres"

type modelRow struct {
	ID      int    `orm:"column(id);pk;auto"`
	Name    string `orm:"column(name)"`
	Title   string `orm:"column(title)"`
	Fax     string `orm:"column(fax)"`
	Web     string `orm:"column(web)"`
	Age     int    `orm:"column(age)"`
	Right   bool   `orm:"column(right)"`
	Counter int64  `orm:"column(counter)"`
}

func (*modelRow) TableName() string { return "models" }

type customerRow struct {
	ID    int    `orm:"column(id);pk;auto"`
	Name  string `orm:"column(name)"`
	Email string `orm:"column(email)"`
	City  string `orm:"column(city)"`
}

func (*customerRow) TableName() string { return "customers" }

type orderRow struct {
	ID        int          `orm:"column(id);pk;auto"`
	Customer  *customerRow `orm:"column(customer_id);rel(fk)"`
	Status    string       `orm:"column(status)"`
	Total     int          `orm:"column(total)"`
	CreatedAt time.Time    `orm:"column(created_at)"`
}

func (*orderRow) TableName() string { return "orders" }

// updateColumns lists every mutable column so a zero value (a false Right) is still written.
var updateColumns = []string{"name", "title", "fax", "web", "age", "right", "counter"}

// aliasName is registered once per process: Beego rejects a reused alias name, has no way to
// remove one, and only needs the name to exist when it bootstraps the models. Each Init builds its
// Ormer from its own *sql.DB through NewOrmWithDB, which does not look the alias up again.
const aliasName = "bench_beego"

var (
	// registryMu guards the two flags. Beego's model registry is global and closes at the first NewOrm.
	registryMu                        sync.Mutex
	aliasRegistered, modelsRegistered bool
)

// ensureRegistered registers the alias, then the models, each once per process. Beego's
// registrations are not idempotent, so each step has its own flag and a failed step is the only
// one retried. A Beego panic or error is returned instead of escaping; the caller closes its *sql.DB.
func ensureRegistered(sqlDB *sql.DB) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	if !aliasRegistered {
		if err := recoverRegistration(func() error { return orm.AddAliasWthDB(aliasName, dialect, sqlDB) }); err != nil {
			return err
		}
		aliasRegistered = true
	}
	if !modelsRegistered {
		err := recoverRegistration(func() error {
			orm.RegisterModel(new(modelRow), new(customerRow), new(orderRow))
			return nil
		})
		if err != nil {
			return err
		}
		modelsRegistered = true
	}
	return nil
}

// recoverRegistration runs fn and turns a panic into an error.
func recoverRegistration(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn()
}

// Adapter implements bench.Adapter and bench.Verifier with Beego ORM. It is not safe to call Init concurrently.
// A second Init closes the previous pool first.
type Adapter struct {
	sqlDB *sql.DB
	orm   orm.Ormer
}

var _ bench.Verifier = (*Adapter)(nil)

// New returns an uninitialised adapter.
func New() bench.Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return "beego" }

func (a *Adapter) Init(ctx context.Context, dsn string, pool bench.PoolConfig) error {
	_ = a.Close() // a repeated Init must not leak the previous pool; its close error is not actionable
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("beego: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(pool.MaxOpen)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("beego: ping: %w", err)
	}
	if err := ensureRegistered(sqlDB); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("beego: register: %w", err)
	}
	o, err := orm.NewOrmWithDB(dialect, aliasName, sqlDB)
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("beego: init: %w", err)
	}
	a.sqlDB, a.orm = sqlDB, o
	return nil
}

func (a *Adapter) Close() error {
	if a.sqlDB == nil {
		return nil
	}
	err := a.sqlDB.Close()
	a.sqlDB, a.orm = nil, nil
	if err != nil {
		return fmt.Errorf("beego: close: %w", err)
	}
	return nil
}

func (a *Adapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	row := toRow(0, m)
	if _, err := a.orm.InsertWithCtx(ctx, &row); err != nil {
		return 0, fmt.Errorf("beego: insert: %w", err)
	}
	return 1, nil // Insert returns the new id, not a count; an error-free insert is one row
}

func (a *Adapter) Update(ctx context.Context, id int, m bench.Model) (int, error) {
	row := toRow(id, m)
	n, err := a.orm.UpdateWithCtx(ctx, &row, updateColumns...)
	if err != nil {
		return 0, fmt.Errorf("beego: update: %w", err)
	}
	return int(n), nil
}

func (a *Adapter) Delete(ctx context.Context, id int) (int, error) {
	n, err := a.orm.DeleteWithCtx(ctx, &modelRow{ID: id})
	if err != nil {
		return 0, fmt.Errorf("beego: delete: %w", err)
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
	orders, err := a.joinOrders(ctx, customerID)
	return len(orders), err
}

// Union is unsupported: Beego only offers UNION through raw SQL.
func (a *Adapter) Union(context.Context, string) (int, error) { return 0, bench.ErrUnsupported }

func (a *Adapter) selectModel(ctx context.Context, id int) (modelRow, error) {
	row := modelRow{ID: id}
	if err := a.orm.ReadWithCtx(ctx, &row); err != nil {
		return modelRow{}, fmt.Errorf("beego: select: %w", err)
	}
	return row, nil
}

func (a *Adapter) joinOrders(ctx context.Context, customerID int) ([]*orderRow, error) {
	var orders []*orderRow
	_, err := a.orm.QueryTable(new(orderRow)).Filter("Customer", customerID).RelatedSel().AllWithCtx(ctx, &orders)
	if err != nil {
		return nil, fmt.Errorf("beego: join: %w", err)
	}
	return orders, nil
}

func toRow(id int, m bench.Model) modelRow {
	return modelRow{ID: id, Name: m.Name, Title: m.Title, Fax: m.Fax, Web: m.Web, Age: m.Age, Right: m.Right, Counter: m.Counter}
}
