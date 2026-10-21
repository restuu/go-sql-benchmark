package runner

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

func matrixOptions(adapters ...*fakeAdapter) Options {
	factories := make([]func() bench.Adapter, len(adapters))
	for i, a := range adapters {
		factories[i] = func() bench.Adapter { return a }
	}
	return Options{
		DSN:       testDSN,
		Adapters:  factories,
		Scenarios: []bench.Scenario{bench.ScenarioSelect, bench.ScenarioUnion},
		Profiles:  []bench.Profile{{Name: "test", Workers: 2, Duration: 30 * time.Millisecond, DeletePool: 100}},
		Repeats:   2,
		Seed:      7,
	}
}

func TestRunMatrix(t *testing.T) {
	// Arrange
	alpha := &fakeAdapter{name: "alpha", latency: 100 * time.Microsecond}
	beta := &fakeAdapter{name: "beta", latency: 100 * time.Microsecond, unsupported: map[bench.Scenario]bool{bench.ScenarioUnion: true}}
	opts := matrixOptions(beta, alpha) // reversed: output must still be sorted by library name
	var progress bytes.Buffer
	opts.Progress = &progress

	// Act
	run, err := Run(context.Background(), &fakeStore{}, opts, result.Host{GOOS: "test"})

	// Assert
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if err := run.Validate(); err != nil {
		t.Fatalf("result does not validate: %v", err)
	}
	type cellWant struct{ library, scenario, status string }
	want := []cellWant{
		{"alpha", "select", result.StatusOK}, {"alpha", "union", result.StatusOK},
		{"beta", "select", result.StatusOK}, {"beta", "union", result.StatusNA},
	}
	if len(run.Cells) != len(want) {
		t.Fatalf("len(Cells) = %d, want %d", len(run.Cells), len(want))
	}
	for i, w := range want {
		c := run.Cells[i]
		if c.Library != w.library || c.Scenario != w.scenario || c.Status != w.status {
			t.Errorf("Cells[%d] = %s/%s/%s, want %s/%s/%s", i, c.Library, c.Scenario, c.Status, w.library, w.scenario, w.status)
		}
		if w.status == result.StatusOK && (len(c.Repeats) != opts.Repeats || c.Median == nil) {
			t.Errorf("Cells[%d] has %d repeats and median %v", i, len(c.Repeats), c.Median)
		}
	}
	if got := beta.count(bench.ScenarioUnion); got != 1 {
		t.Errorf("unsupported cell probed %d times, want once (skipped in later repeats)", got)
	}
	if run.Config.Seed != opts.Seed || run.Config.Repeats != opts.Repeats || len(run.Config.Profiles) != 1 || run.Config.Profiles[0].Workers != 2 {
		t.Errorf("config not recorded: %+v", run.Config)
	}
	if run.Host.GOOS != "test" || run.StartedAt.IsZero() {
		t.Errorf("host/start not recorded: %+v %v", run.Host, run.StartedAt)
	}
	// 2 libraries x 2 scenarios x 1 profile x 2 repeats = 8 cells; the repeat-2 na cell is skipped, not printed.
	if lines := strings.Count(progress.String(), "\n"); lines != 7 || !strings.HasPrefix(progress.String(), "[1/8] ") {
		t.Errorf("progress = %d lines, want 7 starting with [1/8]:\n%s", lines, progress.String())
	}
}

func TestRunReseedsAfterMatrix(t *testing.T) {
	// Arrange
	opts := matrixOptions(&fakeAdapter{name: "alpha"})
	opts.Scenarios = []bench.Scenario{bench.ScenarioSelect}
	opts.Repeats = 1
	store := &fakeStore{}

	// Act
	_, err := Run(context.Background(), store, opts, result.Host{})

	// Assert
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if got := store.reseeds.Load(); got != 3 {
		t.Errorf("reseeds = %d, want 3 (before probe, after probe, final)", got)
	}
}

func TestRunErrors(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name  string
		ctx   context.Context
		store *fakeStore
		tweak func(*Options)
		want  error // nil = any error
	}{
		{"reseed failure aborts", context.Background(), &fakeStore{reseedErr: errInjected}, func(*Options) {}, errInjected},
		{"cancelled context", cancelled, &fakeStore{}, func(*Options) {}, context.Canceled},
		{"zero repeats", context.Background(), &fakeStore{}, func(o *Options) { o.Repeats = 0 }, nil},
		{"no adapters", context.Background(), &fakeStore{}, func(o *Options) { o.Adapters = nil }, nil},
		{"no scenarios", context.Background(), &fakeStore{}, func(o *Options) { o.Scenarios = nil }, nil},
		{"no profiles", context.Background(), &fakeStore{}, func(o *Options) { o.Profiles = nil }, nil},
		{"duplicate adapter names", context.Background(), &fakeStore{}, func(o *Options) { o.Adapters = append(o.Adapters, o.Adapters[0]) }, nil},
		{"adapter hard failure", context.Background(), &fakeStore{}, func(o *Options) {
			o.Adapters = []func() bench.Adapter{func() bench.Adapter { return &fakeAdapter{name: "x", failEvery: 1} }}
		}, errInjected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			opts := matrixOptions(&fakeAdapter{name: "alpha"})
			tt.tweak(&opts)

			// Act
			_, err := Run(tt.ctx, tt.store, opts, result.Host{})

			// Assert
			if err == nil {
				t.Fatal("Run() = nil, want error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("Run() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestShuffleOrder(t *testing.T) {
	// Arrange
	const n = 12

	// Act
	a := shuffleOrder(n, 5, 1)
	b := shuffleOrder(n, 5, 1)
	otherRepeat := shuffleOrder(n, 5, 2)
	otherSeed := shuffleOrder(n, 6, 1)

	// Assert
	if !slices.Equal(a, b) {
		t.Errorf("same seed and repeat gave different orders: %v vs %v", a, b)
	}
	sorted := slices.Clone(a)
	slices.Sort(sorted)
	for i, v := range sorted {
		if v != i {
			t.Fatalf("not a permutation of 0..%d: %v", n-1, a)
		}
	}
	if slices.Equal(a, otherRepeat) && slices.Equal(a, otherSeed) {
		t.Errorf("order ignores repeat and seed: %v", a)
	}
}

func TestShuffleOrderEmpty(t *testing.T) {
	// Arrange
	// (no input)

	// Act
	got := shuffleOrder(0, 1, 1)

	// Assert
	if len(got) != 0 {
		t.Fatalf("shuffleOrder(0) = %v, want empty", got)
	}
}

func TestRunRetriesStuckCellOnce(t *testing.T) {
	tests := []struct {
		name      string
		hangOn    func(n int64) bool
		wantErr   bool
		wantRetry bool
	}{
		{"hang on first attempt only", func(n int64) bool { return n == 2 }, false, true},
		{"hang on every attempt fails the run", func(n int64) bool { return n >= 2 }, true, true},
		{"no hang means no retry", func(int64) bool { return false }, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			shortenTimeouts(t, 50*time.Millisecond, 50*time.Millisecond)
			opts := matrixOptions(&fakeAdapter{name: "alpha", latency: 100 * time.Microsecond, hangOn: tt.hangOn})
			opts.Scenarios = []bench.Scenario{bench.ScenarioSelect}
			opts.Repeats = 1
			opts.Profiles = []bench.Profile{{Name: "test", Workers: 2, Duration: 30 * time.Millisecond}}
			var progress bytes.Buffer
			opts.Progress = &progress

			// Act
			_, err := Run(context.Background(), &fakeStore{}, opts, result.Host{})

			// Assert
			if (err != nil) != tt.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, errPhaseStuck) {
				t.Errorf("Run() = %v, want errPhaseStuck", err)
			}
			if got := strings.Contains(progress.String(), "retrying"); got != tt.wantRetry {
				t.Errorf("progress mentions retrying = %v, want %v:\n%s", got, tt.wantRetry, progress.String())
			}
		})
	}
}
