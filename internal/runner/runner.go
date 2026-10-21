package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

// naReason is recorded for cells whose adapter cannot express the scenario.
const naReason = "n/a (unsupported)"

// Options configures a matrix run.
type Options struct {
	DSN       string
	Adapters  []func() bench.Adapter
	Scenarios []bench.Scenario
	Profiles  []bench.Profile
	Repeats   int
	Seed      uint64
	Progress  io.Writer // optional; one line per cell
}

type cellID struct {
	library  string
	scenario bench.Scenario
	profile  string
}

type cellAcc struct {
	repeats []result.Metrics
	na      bool
}

func (o Options) validate() error {
	switch {
	case o.Repeats < 1:
		return fmt.Errorf("runner: repeats %d, want >= 1", o.Repeats)
	case len(o.Adapters) == 0:
		return errors.New("runner: no adapters")
	case len(o.Scenarios) == 0:
		return errors.New("runner: no scenarios")
	case len(o.Profiles) == 0:
		return errors.New("runner: no profiles")
	}
	return nil
}

// Run measures every adapter x scenario x profile Repeats times and returns the validated
// result. Library order is shuffled per repeat from Seed. A hard error (database down, reseed
// failure) aborts the run; only bench.ErrUnsupported produces an "na" cell. On success the
// database is reseeded once more so it is left in the seeded state.
func Run(ctx context.Context, store Store, opts Options, host result.Host) (result.Run, error) {
	if err := opts.validate(); err != nil {
		return result.Run{}, err
	}
	names, err := libraryNames(opts.Adapters)
	if err != nil {
		return result.Run{}, err
	}
	started := time.Now().UTC()
	acc := make(map[cellID]*cellAcc)
	total := opts.Repeats * len(opts.Adapters) * len(opts.Scenarios) * len(opts.Profiles)
	done := 0

	for repeat := 1; repeat <= opts.Repeats; repeat++ {
		for _, idx := range shuffleOrder(len(opts.Adapters), opts.Seed, repeat) {
			for _, p := range opts.Profiles {
				for _, sc := range opts.Scenarios {
					done++
					id := cellID{names[idx], sc, p.Name}
					if acc[id] != nil && acc[id].na {
						continue
					}
					if err := ctx.Err(); err != nil {
						return result.Run{}, fmt.Errorf("runner: %w", err)
					}
					progress(opts.Progress, done, total, id)
					metrics, err := runCellRetryingStuck(ctx, opts, idx, store, sc, p, opts.Seed+uint64(repeat))
					if err := record(acc, id, metrics, err); err != nil {
						return result.Run{}, fmt.Errorf("runner: %s %s %s: %w", id.library, sc, p.Name, err)
					}
				}
			}
		}
	}
	if err := store.Reseed(ctx); err != nil {
		return result.Run{}, err
	}
	run := assemble(started, host, opts, names, acc)
	if err := run.Validate(); err != nil {
		return result.Run{}, fmt.Errorf("runner: %w", err)
	}
	return run, nil
}

// runCellRetryingStuck runs one cell and, if a phase got stuck, retries it once with a fresh
// adapter (every cell reseeds, so a retry starts clean). A second stall fails the run.
func runCellRetryingStuck(ctx context.Context, opts Options, idx int, store Store, sc bench.Scenario, p bench.Profile, seed uint64) (result.Metrics, error) {
	m, err := runCell(ctx, opts.Adapters[idx](), store, sc, p, opts.DSN, seed)
	if !errors.Is(err, errPhaseStuck) {
		return m, err
	}
	if opts.Progress != nil {
		fmt.Fprintf(opts.Progress, "runner: %v\nrunner: retrying the cell once\n", err)
	}
	return runCell(ctx, opts.Adapters[idx](), store, sc, p, opts.DSN, seed)
}

// record stores a cell outcome; only errUnsupported is absorbed (as an na cell).
func record(acc map[cellID]*cellAcc, id cellID, m result.Metrics, err error) error {
	c := acc[id]
	if c == nil {
		c = &cellAcc{}
		acc[id] = c
	}
	switch {
	case errors.Is(err, errUnsupported):
		c.na = true
	case err != nil:
		return err
	default:
		c.repeats = append(c.repeats, m)
	}
	return nil
}

func libraryNames(factories []func() bench.Adapter) ([]string, error) {
	names := make([]string, len(factories))
	for i, f := range factories {
		names[i] = f().Name()
		if slices.Contains(names[:i], names[i]) {
			return nil, fmt.Errorf("runner: duplicate adapter name %q", names[i])
		}
	}
	return names, nil
}

// shuffleOrder returns a deterministic permutation of 0..n-1 for the given seed and repeat.
func shuffleOrder(n int, seed uint64, repeat int) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	rng := rand.New(rand.NewPCG(seed, uint64(repeat)))
	rng.Shuffle(n, func(i, j int) { order[i], order[j] = order[j], order[i] })
	return order
}

func progress(w io.Writer, done, total int, id cellID) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "[%d/%d] %s %s %s\n", done, total, id.library, id.scenario, id.profile)
}

// assemble builds the result in stable order: library name, scenario as given, profile as given.
func assemble(started time.Time, host result.Host, opts Options, names []string, acc map[cellID]*cellAcc) result.Run {
	profiles := make([]result.ProfileConfig, len(opts.Profiles))
	for i, p := range opts.Profiles {
		profiles[i] = result.ProfileConfig{
			Name: p.Name, Workers: p.Workers, IntervalNs: p.Interval.Nanoseconds(),
			WarmupNs: p.Warmup.Nanoseconds(), DurationNs: p.Duration.Nanoseconds(), DeletePool: p.DeletePool,
		}
	}
	sorted := slices.Sorted(slices.Values(names))
	var cells []result.Cell
	for _, lib := range sorted {
		for _, sc := range opts.Scenarios {
			for _, p := range opts.Profiles {
				cells = append(cells, buildCell(lib, sc, p.Name, acc[cellID{lib, sc, p.Name}]))
			}
		}
	}
	return result.Run{
		SchemaVersion: result.SchemaVersion,
		StartedAt:     started,
		Host:          host,
		Config:        result.Config{Seed: opts.Seed, Repeats: opts.Repeats, Profiles: profiles},
		Cells:         cells,
	}
}

func buildCell(lib string, sc bench.Scenario, profile string, c *cellAcc) result.Cell {
	cell := result.Cell{Library: lib, Scenario: string(sc), Profile: profile}
	if c.na {
		cell.Status, cell.Reason = result.StatusNA, naReason
		return cell
	}
	median := medianMetrics(c.repeats)
	cell.Status, cell.Repeats, cell.Median = result.StatusOK, c.repeats, &median
	return cell
}
