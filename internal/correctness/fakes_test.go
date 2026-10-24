package correctness

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

const (
	fakeModels    = 10000
	fakeCustomers = 1000
	fakeCities    = 50
)

var fakeEpoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// world is an in-memory stand-in for the seeded database, shared by a fake store and fake adapters.
type world struct {
	models map[int]bench.Model
	nextID int
}

func newWorld() *world {
	w := &world{}
	w.reseed()
	return w
}

func (w *world) reseed() {
	w.models = make(map[int]bench.Model, fakeModels)
	for id := 1; id <= fakeModels; id++ {
		m := bench.NewModel()
		m.ID = id
		w.models[id] = m
	}
	w.nextID = fakeModels + 1
}

func (w *world) add(m bench.Model) int {
	m.ID = w.nextID
	w.models[m.ID] = m
	w.nextID++
	return m.ID
}

// seedJoin mirrors the db/seed.sql formulas for the orders of one customer.
func seedJoin(customer int) []bench.JoinRow {
	var rows []bench.JoinRow
	statuses := []string{"new", "paid", "shipped", "done"}
	for i := 1; i <= fakeModels; i++ {
		if i%fakeCustomers+1 != customer {
			continue
		}
		rows = append(rows, bench.JoinRow{
			OrderID: i, CustomerID: customer, Status: statuses[i%len(statuses)], Total: i%500 + 1,
			CreatedAt: fakeEpoch.Add(time.Duration(i) * time.Minute),
			Name:      fmt.Sprintf("customer %d", customer),
			Email:     fmt.Sprintf("customer%d@example.com", customer),
			City:      fmt.Sprintf("city %d", customer%fakeCities),
		})
	}
	return rows
}

// seedUnion mirrors the db/seed.sql formulas for customers and suppliers of one city.
func seedUnion(city string) []bench.UnionRow {
	var rows []bench.UnionRow
	for i := 1; i <= fakeCustomers; i++ {
		if fmt.Sprintf("city %d", i%fakeCities) != city {
			continue
		}
		rows = append(rows,
			bench.UnionRow{Name: fmt.Sprintf("customer %d", i), Email: fmt.Sprintf("customer%d@example.com", i), City: city},
			bench.UnionRow{Name: fmt.Sprintf("supplier %d", i), Email: fmt.Sprintf("supplier%d@example.com", i), City: city},
		)
	}
	return rows
}

type fakeStore struct {
	w          *world
	reseedErr  error
	reseedCall int
}

func (s *fakeStore) Reseed(context.Context) error {
	s.reseedCall++
	if s.reseedErr != nil {
		return s.reseedErr
	}
	s.w.reseed()
	return nil
}

func (s *fakeStore) InsertDeletable(_ context.Context, n int) ([]int, error) {
	ids := make([]int, n)
	for i := range ids {
		ids[i] = s.w.add(bench.NewModel())
	}
	return ids, nil
}

// fakeLib is an adapter over the shared world with switchable bugs.
type fakeLib struct {
	w    *world
	name string

	swapTitleFax     bool // writes store Title in Fax and vice versa
	dropFalseOnWrite bool // an update keeps the old Right when the new one is false (gorm zero-value skip)
	wrongRead        bool // ReadModel(5000) returns a wrong Age
	wrongFirst       bool // ReadModel(1) returns a wrong Age (a reference that misreads the seed)
	noUnion          bool // Union and ReadUnion both report ErrUnsupported
	verifierOnlyNoUn bool // ReadUnion reports ErrUnsupported but Union works
	keepOnDelete     bool // Delete reports 1 but leaves the row
	upsertOnUpdate   bool // Update of a missing id inserts it (gorm Save)
}

func (f *fakeLib) Name() string                                         { return f.name }
func (f *fakeLib) Init(context.Context, string, bench.PoolConfig) error { return nil }
func (f *fakeLib) Close() error                                         { return nil }

func (f *fakeLib) stored(m bench.Model) bench.Model {
	if f.swapTitleFax {
		m.Title, m.Fax = m.Fax, m.Title
	}
	return m
}

func (f *fakeLib) Insert(_ context.Context, m bench.Model) (int, error) {
	f.w.add(f.stored(m))
	return 1, nil
}

func (f *fakeLib) Update(_ context.Context, id int, m bench.Model) (int, error) {
	old, ok := f.w.models[id]
	if !ok && f.upsertOnUpdate {
		f.w.models[id] = f.stored(m)
		return 1, nil
	}
	if !ok {
		return 0, nil
	}
	next := f.stored(m)
	next.ID = id
	if f.dropFalseOnWrite && !m.Right {
		next.Right = old.Right
	}
	f.w.models[id] = next
	return 1, nil
}

func (f *fakeLib) Delete(_ context.Context, id int) (int, error) {
	if _, ok := f.w.models[id]; !ok {
		return 0, nil
	}
	if !f.keepOnDelete {
		delete(f.w.models, id)
	}
	return 1, nil
}

func (f *fakeLib) Select(ctx context.Context, id int) (int, error) {
	_, err := f.ReadModel(ctx, id)
	if err != nil {
		return 0, err
	}
	return 1, nil
}

func (f *fakeLib) Join(_ context.Context, customer int) (int, error) {
	return len(seedJoin(customer)), nil
}

func (f *fakeLib) Union(_ context.Context, city string) (int, error) {
	if f.noUnion {
		return 0, bench.ErrUnsupported
	}
	return len(seedUnion(city)), nil
}

func (f *fakeLib) ReadModel(_ context.Context, id int) (bench.Model, error) {
	m, ok := f.w.models[id]
	if !ok {
		return bench.Model{}, fmt.Errorf("fake: no such model: %w", sql.ErrNoRows)
	}
	if (f.wrongRead && id == 5000) || (f.wrongFirst && id == 1) {
		m.Age++
	}
	return m, nil
}

func (f *fakeLib) ReadJoin(_ context.Context, customer int) ([]bench.JoinRow, error) {
	return seedJoin(customer), nil
}

func (f *fakeLib) ReadUnion(_ context.Context, city string) ([]bench.UnionRow, error) {
	if f.noUnion || f.verifierOnlyNoUn {
		return nil, bench.ErrUnsupported
	}
	return seedUnion(city), nil
}

// plainAdapter hides the Verifier methods of the adapter it wraps.
type plainAdapter struct{ bench.Adapter }

// alwaysOneDelete reports one row for every delete, as an adapter whose library cannot count would.
type alwaysOneDelete struct{ *fakeLib }

func (a alwaysOneDelete) Delete(ctx context.Context, id int) (int, error) {
	_, _ = a.fakeLib.Delete(ctx, id)
	return 1, nil
}

// countlessDelete is alwaysOneDelete that declares the limitation, like pop.
type countlessDelete struct{ alwaysOneDelete }

func (countlessDelete) DeleteCountUnreported() {}
