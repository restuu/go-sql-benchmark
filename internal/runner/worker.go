package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/seedcheck"
)

// Stuck-phase bounds. Variables so tests can shorten them.
var (
	// opTimeout is how long in-flight ops may outlive the window end before the phase is cancelled.
	opTimeout = 10 * time.Second
	// abandonGrace is how long cancelled workers get to return before they are abandoned
	// (an adapter that ignores ctx would otherwise block the run forever).
	abandonGrace = 5 * time.Second
)

const (
	// pacedCapacitySlack is extra latency slots beyond duration/interval, for jitter.
	pacedCapacitySlack = 64
	// unpacedCapacityHint preallocates latency slots per worker so the window rarely grows a slice.
	unpacedCapacityHint = 1 << 17
	firstSeededID       = 1
)

// errUnsupported marks a cell whose adapter reported bench.ErrUnsupported.
var errUnsupported = errors.New("runner: scenario unsupported by adapter")

// errPhaseStuck marks a phase whose workers were still inside adapter calls long after the
// window end. The error carries a goroutine profile; Run retries the cell once.
var errPhaseStuck = errors.New("runner: ops still in flight after the window end")

// deleteIDs hands out pre-inserted ids to DELETE workers, each id exactly once.
type deleteIDs struct {
	ids  []int
	next atomic.Int64
}

func (d *deleteIDs) take() (int, bool) {
	i := int(d.next.Add(1)) - 1
	if i >= len(d.ids) {
		return 0, false
	}
	return d.ids[i], true
}

// phaseSpec describes one warm-up or measured window.
type phaseSpec struct {
	adapter  bench.Adapter
	scenario bench.Scenario
	profile  bench.Profile
	duration time.Duration
	seed     uint64
	ids      *deleteIDs // nil unless scenario is delete
	record   bool       // keep latencies (measured window) or drop them (warm-up)
}

// phase owns the workers of one window. Build it with newPhase before taking the memory
// baseline so that preallocation is not charged to the window.
type phase struct {
	spec        phaseSpec
	workers     []*worker
	abort       context.CancelFunc
	unsupported atomic.Bool
}

func (ph *phase) label() string {
	if ph.spec.record {
		return "measured window"
	}
	return "warm-up"
}

func newPhase(spec phaseSpec) *phase {
	ph := &phase{spec: spec, workers: make([]*worker, spec.profile.Workers)}
	cities := bench.Cities()
	capacity := 0
	if spec.record {
		capacity = latencyCapacity(spec.profile, spec.duration)
	}
	for i := range ph.workers {
		ph.workers[i] = &worker{
			ph:        ph,
			id:        i,
			rng:       rand.New(rand.NewPCG(spec.seed, uint64(i))),
			model:     bench.NewModel(),
			cities:    cities,
			models:    int(seedcheck.ExpectedRows["models"]),
			customers: int(seedcheck.ExpectedRows["customers"]),
			expected:  spec.scenario.ExpectedRows(),
			latencies: make([]int64, 0, capacity),
		}
	}
	return ph
}

func latencyCapacity(p bench.Profile, d time.Duration) int {
	if p.Interval > 0 {
		return int(d/p.Interval) + pacedCapacitySlack
	}
	return unpacedCapacityHint
}

// run starts every worker and waits for them. It returns errUnsupported if an adapter call
// reported bench.ErrUnsupported, an error if the parent context was cancelled, and an error
// wrapping errPhaseStuck if workers are still inside adapter calls opTimeout after the window
// end. In that case workers get abandonGrace to honour the cancellation and are abandoned
// (leaked) if they do not.
func (ph *phase) run(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	ph.abort = cancel

	start := time.Now()
	deadline := start.Add(ph.spec.duration)
	var wg sync.WaitGroup
	for _, w := range ph.workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.loop(ctx, start, deadline)
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	timer := time.NewTimer(ph.spec.duration + opTimeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		return ph.stuckError(cancel, done)
	}

	if ph.unsupported.Load() {
		return errUnsupported
	}
	if err := parent.Err(); err != nil {
		return fmt.Errorf("runner: phase aborted: %w", err)
	}
	return nil
}

// stuckError captures a goroutine profile, cancels the phase and reports which workers did not return.
func (ph *phase) stuckError(cancel context.CancelFunc, done <-chan struct{}) error {
	var profile bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&profile, 1) // best effort: a failed write only shortens the diagnosis
	cancel()
	abandoned := ""
	select {
	case <-done:
	case <-time.After(abandonGrace):
		abandoned = fmt.Sprintf(" (workers ignored cancellation for %s and were abandoned)", abandonGrace)
	}
	return fmt.Errorf("%w: %s %s %s after %s%s\ngoroutine profile at timeout:\n%s",
		errPhaseStuck, ph.spec.adapter.Name(), ph.spec.scenario, ph.spec.profile.Name+" "+ph.label(), opTimeout, abandoned, profile.String())
}

// collect merges the per-worker results; call it after run, outside the measured window.
func (ph *phase) collect() (latencies []int64, errCount int64) {
	total := 0
	for _, w := range ph.workers {
		total += len(w.latencies)
	}
	latencies = make([]int64, 0, total)
	for _, w := range ph.workers {
		latencies = append(latencies, w.latencies...)
		errCount += w.errors
	}
	return latencies, errCount
}

// worker is one load-generating goroutine. Its fields are touched only by that goroutine
// until run returns, so recording needs no lock.
type worker struct {
	ph        *phase
	id        int
	rng       *rand.Rand
	model     bench.Model
	cities    []string
	models    int
	customers int
	expected  int
	latencies []int64
	errors    int64
	targetID  int
	city      string
}

func (w *worker) loop(ctx context.Context, start, deadline time.Time) {
	spec := &w.ph.spec
	interval := spec.profile.Interval
	next := start
	if interval > 0 { // stagger workers so paced load is not one synchronized burst
		next = start.Add(interval * time.Duration(w.id) / time.Duration(spec.profile.Workers))
	}
	for {
		if interval > 0 {
			time.Sleep(time.Until(next))
			next = next.Add(interval)
		}
		if ctx.Err() != nil || !time.Now().Before(deadline) || !w.prepare() {
			return
		}
		t0 := time.Now()
		rows, err := w.call(ctx)
		elapsed := time.Since(t0)
		if !w.account(ctx, rows, err, elapsed) {
			return
		}
	}
}

// prepare chooses the arguments of the next call; false means the delete pool is exhausted.
func (w *worker) prepare() bool {
	switch w.ph.spec.scenario {
	case bench.ScenarioUpdate, bench.ScenarioSelect:
		w.targetID = firstSeededID + w.rng.IntN(w.models)
	case bench.ScenarioDelete:
		id, ok := w.ph.spec.ids.take()
		w.targetID = id
		return ok
	case bench.ScenarioJoin:
		w.targetID = firstSeededID + w.rng.IntN(w.customers)
	case bench.ScenarioUnion:
		w.city = w.cities[w.rng.IntN(len(w.cities))]
	}
	return true
}

func (w *worker) call(ctx context.Context) (int, error) {
	a := w.ph.spec.adapter
	switch w.ph.spec.scenario {
	case bench.ScenarioInsert:
		return a.Insert(ctx, w.model)
	case bench.ScenarioUpdate:
		return a.Update(ctx, w.targetID, w.model)
	case bench.ScenarioDelete:
		return a.Delete(ctx, w.targetID)
	case bench.ScenarioSelect:
		return a.Select(ctx, w.targetID)
	case bench.ScenarioJoin:
		return a.Join(ctx, w.targetID)
	default:
		return a.Union(ctx, w.city)
	}
}

// account classifies one call and reports whether the worker should continue.
func (w *worker) account(ctx context.Context, rows int, err error, elapsed time.Duration) bool {
	switch {
	case err == nil && rows == w.expected:
		if w.ph.spec.record {
			w.latencies = append(w.latencies, elapsed.Nanoseconds())
		}
	case errors.Is(err, bench.ErrUnsupported):
		w.ph.unsupported.Store(true)
		w.ph.abort()
		return false
	case ctx.Err() != nil:
		return false // aborted or timed out: not an adapter failure
	default:
		w.errors++
	}
	return true
}
