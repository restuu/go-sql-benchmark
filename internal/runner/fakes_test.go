package runner

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/seedcheck"
)

var errInjected = errors.New("injected failure")

// fakeAdapter is a configurable bench.Adapter. Call number 1 is the probe, so failEvery and
// wrongRowsEvery of n spare the first n-1 calls.
type fakeAdapter struct {
	name           string
	latency        time.Duration
	unsupported    map[bench.Scenario]bool
	failEvery      int64
	wrongRowsEvery int64
	failFrom       int64              // every call numbered >= failFrom fails (0 = off)
	hangOn         func(n int64) bool // block this call until ctx is done, or until release is closed
	release        chan struct{}      // when set, hung calls ignore ctx and wait for it

	calls    atomic.Int64
	inits    atomic.Int64
	closes   atomic.Int64
	perCount [6]atomic.Int64
	badArgs  atomic.Int64

	mu         sync.Mutex
	deletedIDs []int
}

func (f *fakeAdapter) Name() string { return f.name }

func (f *fakeAdapter) Init(context.Context, string, bench.PoolConfig) error {
	f.inits.Add(1)
	return nil
}

func (f *fakeAdapter) Close() error {
	f.closes.Add(1)
	return nil
}

func (f *fakeAdapter) respond(ctx context.Context, sc bench.Scenario) (int, error) {
	n := f.calls.Add(1)
	f.perCount[slices.Index(bench.AllScenarios, sc)].Add(1)
	if f.hangOn != nil && f.hangOn(n) {
		if f.release != nil {
			<-f.release
		} else {
			<-ctx.Done()
		}
		return 0, ctx.Err()
	}
	if f.failFrom > 0 && n >= f.failFrom {
		return 0, errInjected
	}
	if f.unsupported[sc] {
		return 0, bench.ErrUnsupported
	}
	if f.latency > 0 {
		time.Sleep(f.latency)
	}
	if f.failEvery > 0 && n%f.failEvery == 0 {
		return 0, errInjected
	}
	rows := sc.ExpectedRows()
	if f.wrongRowsEvery > 0 && n%f.wrongRowsEvery == 0 {
		rows++
	}
	return rows, nil
}

func (f *fakeAdapter) flagIf(bad bool) {
	if bad {
		f.badArgs.Add(1)
	}
}

func (f *fakeAdapter) Insert(ctx context.Context, m bench.Model) (int, error) {
	f.flagIf(m != bench.NewModel())
	return f.respond(ctx, bench.ScenarioInsert)
}

func (f *fakeAdapter) Update(ctx context.Context, id int, _ bench.Model) (int, error) {
	f.flagIf(id < 1 || id > int(seedcheck.ExpectedRows["models"]))
	return f.respond(ctx, bench.ScenarioUpdate)
}

func (f *fakeAdapter) Delete(ctx context.Context, id int) (int, error) {
	f.mu.Lock()
	f.deletedIDs = append(f.deletedIDs, id)
	f.mu.Unlock()
	return f.respond(ctx, bench.ScenarioDelete)
}

func (f *fakeAdapter) Select(ctx context.Context, id int) (int, error) {
	f.flagIf(id < 1 || id > int(seedcheck.ExpectedRows["models"]))
	return f.respond(ctx, bench.ScenarioSelect)
}

func (f *fakeAdapter) Join(ctx context.Context, customerID int) (int, error) {
	f.flagIf(customerID < 1 || customerID > int(seedcheck.ExpectedRows["customers"]))
	return f.respond(ctx, bench.ScenarioJoin)
}

func (f *fakeAdapter) Union(ctx context.Context, city string) (int, error) {
	f.flagIf(!slices.Contains(bench.Cities(), city))
	return f.respond(ctx, bench.ScenarioUnion)
}

func (f *fakeAdapter) count(sc bench.Scenario) int64 {
	return f.perCount[slices.Index(bench.AllScenarios, sc)].Load()
}

// fakeStore counts reseeds and hands out sequential ids above the seeded range.
type fakeStore struct {
	reseeds   atomic.Int64
	reseedErr error
	nextID    atomic.Int64
}

func (s *fakeStore) Reseed(context.Context) error {
	s.reseeds.Add(1)
	return s.reseedErr
}

func (s *fakeStore) InsertDeletable(_ context.Context, n int) ([]int, error) {
	if n < 0 {
		return nil, fmt.Errorf("fakeStore: negative n %d", n)
	}
	base := int(s.nextID.Add(int64(n))) - n + int(seedcheck.ExpectedRows["models"]) + 1
	ids := make([]int, n)
	for i := range ids {
		ids[i] = base + i
	}
	return ids, nil
}
