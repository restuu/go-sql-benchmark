package runner

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

// errNoOps reports a measured window in which no operation succeeded.
var errNoOps = errors.New("runner: measured window recorded no successful ops")

// errProbeRows reports an adapter that returns the wrong row count on its first call.
var errProbeRows = errors.New("runner: probe returned an unexpected row count")

// runCell measures one library x scenario x profile. It reseeds the database, probes the
// adapter once (reseeding again afterwards because the probe may mutate state), then runs the
// warm-up and the measured window. It returns errUnsupported when the adapter cannot express
// the scenario.
func runCell(ctx context.Context, a bench.Adapter, s Store, sc bench.Scenario, p bench.Profile, dsn string, seed uint64) (result.Metrics, error) {
	if p.Workers < 1 || p.Duration <= 0 {
		return result.Metrics{}, fmt.Errorf("runner: profile %q needs workers >= 1 and a positive duration", p.Name)
	}
	if err := probe(ctx, a, s, sc, p, dsn); err != nil {
		return result.Metrics{}, err
	}
	if err := open(ctx, a, s, p, dsn); err != nil {
		return result.Metrics{}, err
	}
	defer func() { _ = a.Close() }()

	var ids *deleteIDs
	if sc == bench.ScenarioDelete {
		pool, err := s.InsertDeletable(ctx, p.DeletePool)
		if err != nil {
			return result.Metrics{}, err
		}
		ids = &deleteIDs{ids: pool}
	}
	spec := phaseSpec{adapter: a, scenario: sc, profile: p, seed: seed, ids: ids}

	if p.Warmup > 0 {
		warm := spec
		warm.duration = p.Warmup
		if err := newPhase(warm).run(ctx); err != nil {
			return result.Metrics{}, err
		}
	}
	spec.duration, spec.record = p.Duration, true
	m, err := measure(ctx, newPhase(spec))
	if err != nil {
		return result.Metrics{}, err
	}
	if m.Ops == 0 {
		return result.Metrics{}, fmt.Errorf("%w: %s %s %s (%d errors; for delete, the warm-up may have used the whole pool)",
			errNoOps, a.Name(), sc, p.Name, m.Errors)
	}
	return m, nil
}

// open reseeds the database and initialises the adapter with the profile's pool. Reseeding
// first matters: TRUNCATE needs no adapter connection to be open.
func open(ctx context.Context, a bench.Adapter, s Store, p bench.Profile, dsn string) error {
	if err := s.Reseed(ctx); err != nil {
		return err
	}
	if err := a.Init(ctx, dsn, p.Pool()); err != nil {
		return fmt.Errorf("runner: init %s: %w", a.Name(), err)
	}
	return nil
}

// probe runs the scenario once on a throwaway pool and closes it.
func probe(ctx context.Context, a bench.Adapter, s Store, sc bench.Scenario, p bench.Profile, dsn string) error {
	if err := open(ctx, a, s, p, dsn); err != nil {
		return err
	}
	defer func() { _ = a.Close() }()

	probeCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	rows, err := probeCall(probeCtx, a, s, sc)
	switch {
	case errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
		return fmt.Errorf("%w: %s %s %s probe exceeded %s", errPhaseStuck, a.Name(), sc, p.Name, opTimeout)
	case errors.Is(err, bench.ErrUnsupported):
		return errUnsupported
	case err != nil:
		return fmt.Errorf("runner: probe %s %s: %w", a.Name(), sc, err)
	case rows != sc.ExpectedRows():
		return fmt.Errorf("%w: %s %s returned %d rows, want %d", errProbeRows, a.Name(), sc, rows, sc.ExpectedRows())
	}
	return nil
}

func probeCall(ctx context.Context, a bench.Adapter, s Store, sc bench.Scenario) (int, error) {
	switch sc {
	case bench.ScenarioInsert:
		return a.Insert(ctx, bench.NewModel())
	case bench.ScenarioUpdate:
		return a.Update(ctx, firstSeededID, bench.NewModel())
	case bench.ScenarioDelete:
		ids, err := s.InsertDeletable(ctx, 1)
		if err != nil {
			return 0, err
		}
		if len(ids) != 1 {
			return 0, fmt.Errorf("store returned %d ids, want 1", len(ids))
		}
		return a.Delete(ctx, ids[0])
	case bench.ScenarioSelect:
		return a.Select(ctx, firstSeededID)
	case bench.ScenarioJoin:
		return a.Join(ctx, firstSeededID)
	case bench.ScenarioUnion:
		return a.Union(ctx, bench.Cities()[0])
	}
	return 0, fmt.Errorf("unknown scenario %q", sc)
}

// measure runs the window with GC, memory and CPU baselines taken immediately around it.
// ReadMemStats stops the world, so it runs only before and after the window.
func measure(ctx context.Context, ph *phase) (result.Metrics, error) {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	cpuBefore, err := cpuTime()
	if err != nil {
		return result.Metrics{}, err
	}

	start := time.Now()
	runErr := ph.run(ctx)
	elapsed := time.Since(start)

	cpuAfter, cpuErr := cpuTime()
	runtime.ReadMemStats(&after)
	if runErr != nil {
		return result.Metrics{}, runErr
	}
	if cpuErr != nil {
		return result.Metrics{}, cpuErr
	}

	latencies, errCount := ph.collect()
	return summarize(windowStats{
		Latencies:  latencies,
		Errors:     errCount,
		Elapsed:    elapsed,
		AllocBytes: after.TotalAlloc - before.TotalAlloc,
		Mallocs:    after.Mallocs - before.Mallocs,
		CPU:        cpuAfter - cpuBefore,
	}), nil
}

// cpuTime returns the process's user plus system CPU time. It includes GC and harness work.
func cpuTime() (time.Duration, error) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0, fmt.Errorf("runner: getrusage: %w", err)
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano()), nil
}
