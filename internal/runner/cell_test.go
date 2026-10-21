package runner

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

const testDSN = "fake"

func testProfile() bench.Profile {
	return bench.Profile{Name: "test", Workers: 4, Warmup: 20 * time.Millisecond, Duration: 200 * time.Millisecond, DeletePool: 100_000}
}

func TestRunCellAllScenarios(t *testing.T) {
	for _, sc := range bench.AllScenarios {
		t.Run(string(sc), func(t *testing.T) {
			// Arrange
			a := &fakeAdapter{name: "fake", latency: 200 * time.Microsecond}
			s := &fakeStore{}

			// Act
			m, err := runCell(context.Background(), a, s, sc, testProfile(), testDSN, 1)

			// Assert
			if err != nil {
				t.Fatalf("runCell() = %v", err)
			}
			if m.Ops == 0 || m.Errors != 0 {
				t.Fatalf("ops = %d, errors = %d; want ops > 0 and no errors", m.Ops, m.Errors)
			}
			if a.count(sc) < m.Ops {
				t.Errorf("adapter saw %d %s calls, fewer than %d measured ops", a.count(sc), sc, m.Ops)
			}
			if a.badArgs.Load() != 0 {
				t.Errorf("%d calls had out-of-range arguments", a.badArgs.Load())
			}
			if a.inits.Load() != 2 || a.closes.Load() != 2 {
				t.Errorf("inits = %d, closes = %d; want probe pool plus measured pool (2, 2)", a.inits.Load(), a.closes.Load())
			}
			if s.reseeds.Load() != 2 {
				t.Errorf("reseeds = %d, want 2 (before probe and after probe)", s.reseeds.Load())
			}
		})
	}
}

func TestRunCellLatency(t *testing.T) {
	// Arrange
	const opLatency = time.Millisecond
	a := &fakeAdapter{name: "fake", latency: opLatency}

	// Act
	m, err := runCell(context.Background(), a, &fakeStore{}, bench.ScenarioSelect, testProfile(), testDSN, 1)

	// Assert
	if err != nil {
		t.Fatalf("runCell() = %v", err)
	}
	p50 := time.Duration(m.LatencyP50Ns)
	if p50 < opLatency || p50 > 5*opLatency {
		t.Errorf("p50 = %v, want about %v", p50, opLatency)
	}
	if m.LatencyP50Ns > m.LatencyP95Ns || m.LatencyP95Ns > m.LatencyP99Ns {
		t.Errorf("percentiles out of order: %d %d %d", m.LatencyP50Ns, m.LatencyP95Ns, m.LatencyP99Ns)
	}
	if m.ThroughputOpsPerSec <= 0 || m.ElapsedSeconds <= 0 {
		t.Errorf("throughput = %v, elapsed = %v", m.ThroughputOpsPerSec, m.ElapsedSeconds)
	}
}

func TestRunCellErrorCounting(t *testing.T) {
	tests := []struct {
		name string
		a    *fakeAdapter
	}{
		{"op errors", &fakeAdapter{name: "fake", latency: 100 * time.Microsecond, failEvery: 4}},
		{"wrong row counts", &fakeAdapter{name: "fake", latency: 100 * time.Microsecond, wrongRowsEvery: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			const wantRate = 0.25

			// Act
			m, err := runCell(context.Background(), tt.a, &fakeStore{}, bench.ScenarioJoin, testProfile(), testDSN, 1)

			// Assert
			if err != nil {
				t.Fatalf("runCell() = %v", err)
			}
			if math.Abs(m.ErrorRate-wantRate) > 0.03 {
				t.Fatalf("ErrorRate = %v (ops %d, errors %d), want about %v", m.ErrorRate, m.Ops, m.Errors, wantRate)
			}
		})
	}
}

func TestRunCellProbeFailures(t *testing.T) {
	tests := []struct {
		name string
		a    *fakeAdapter
		s    *fakeStore
		want error
	}{
		{"unsupported", &fakeAdapter{name: "fake", unsupported: map[bench.Scenario]bool{bench.ScenarioUnion: true}}, &fakeStore{}, errUnsupported},
		{"probe error", &fakeAdapter{name: "fake", failEvery: 1}, &fakeStore{}, errInjected},
		{"probe wrong rows", &fakeAdapter{name: "fake", wrongRowsEvery: 1}, &fakeStore{}, errProbeRows},
		{"reseed error", &fakeAdapter{name: "fake"}, &fakeStore{reseedErr: errInjected}, errInjected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (fakes from tt)

			// Act
			_, err := runCell(context.Background(), tt.a, tt.s, bench.ScenarioUnion, testProfile(), testDSN, 1)

			// Assert
			if !errors.Is(err, tt.want) {
				t.Fatalf("runCell() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRunCellDeletePoolExhausted(t *testing.T) {
	// Arrange
	const poolSize = 50
	a := &fakeAdapter{name: "fake"}
	p := bench.Profile{Name: "test", Workers: 2, Duration: 5 * time.Second, DeletePool: poolSize}

	// Act
	m, err := runCell(context.Background(), a, &fakeStore{}, bench.ScenarioDelete, p, testDSN, 1)

	// Assert
	if err != nil {
		t.Fatalf("runCell() = %v", err)
	}
	if m.Ops != poolSize {
		t.Errorf("Ops = %d, want %d (every pooled id used once)", m.Ops, poolSize)
	}
	if m.ElapsedSeconds >= p.Duration.Seconds()/2 {
		t.Errorf("ElapsedSeconds = %v, want the window to end early", m.ElapsedSeconds)
	}
	ids := slices.Clone(a.deletedIDs[1:]) // [0] is the probe's id
	slices.Sort(ids)
	if len(slices.Compact(ids)) != poolSize {
		t.Errorf("deleted ids are not distinct: %d unique of %d", len(slices.Compact(ids)), len(ids))
	}
}

func TestRunCellPacing(t *testing.T) {
	// Arrange
	p := bench.Profile{Name: "test", Workers: 4, Interval: 10 * time.Millisecond, Duration: 500 * time.Millisecond}
	want := float64(p.Workers) * float64(p.Duration) / float64(p.Interval)

	// Act
	m, err := runCell(context.Background(), &fakeAdapter{name: "fake"}, &fakeStore{}, bench.ScenarioSelect, p, testDSN, 1)

	// Assert
	if err != nil {
		t.Fatalf("runCell() = %v", err)
	}
	if math.Abs(float64(m.Ops)-want) > 0.2*want {
		t.Fatalf("Ops = %d, want %v within 20%%", m.Ops, want)
	}
}

func TestRunCellHarnessAllocationFloor(t *testing.T) {
	// Arrange
	a := &fakeAdapter{name: "fake"}
	p := bench.Profile{Name: "test", Workers: 2, Warmup: 10 * time.Millisecond, Duration: 100 * time.Millisecond}

	// Act
	m, err := runCell(context.Background(), a, &fakeStore{}, bench.ScenarioSelect, p, testDSN, 1)

	// Assert
	if err != nil {
		t.Fatalf("runCell() = %v", err)
	}
	t.Logf("no-op adapter: %d ops, %.2f B/op, %.3f allocs/op, %.0f cpu ns/op", m.Ops, m.BytesPerOp, m.AllocsPerOp, m.CPUNsPerOp)
	if m.AllocsPerOp > 0.5 || m.BytesPerOp > 128 {
		t.Fatalf("no-op adapter costs %v B/op and %v allocs/op; harness floor should be tiny", m.BytesPerOp, m.AllocsPerOp)
	}
}

func TestRunCellInvalidProfile(t *testing.T) {
	tests := []struct {
		name string
		p    bench.Profile
	}{
		{"no workers", bench.Profile{Duration: time.Second}},
		{"no duration", bench.Profile{Workers: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (profile from tt)

			// Act
			_, err := runCell(context.Background(), &fakeAdapter{name: "fake"}, &fakeStore{}, bench.ScenarioSelect, tt.p, testDSN, 1)

			// Assert
			if err == nil {
				t.Fatal("runCell() = nil, want error")
			}
		})
	}
}

func TestRunCellCancelled(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := &fakeAdapter{name: "fake", latency: time.Millisecond}

	// Act
	_, err := runCell(ctx, a, &fakeStore{}, bench.ScenarioSelect, testProfile(), testDSN, 1)

	// Assert
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runCell() = %v, want context.Canceled", err)
	}
}

// shortenTimeouts makes the stuck-phase paths testable and restores the defaults afterwards.
func shortenTimeouts(t *testing.T, op, grace time.Duration) {
	t.Helper()
	oldOp, oldGrace := opTimeout, abandonGrace
	opTimeout, abandonGrace = op, grace
	t.Cleanup(func() { opTimeout, abandonGrace = oldOp, oldGrace })
}

func TestRunCellStuckPhase(t *testing.T) {
	tests := []struct {
		name       string
		warmup     time.Duration
		ignoreCtx  bool
		wantPhase  string
		wantAbandn bool
	}{
		{"hung op in warm-up honours ctx", 20 * time.Millisecond, false, "warm-up", false},
		{"hung op in measured window honours ctx", 0, false, "measured window", false},
		{"hung op ignoring ctx is abandoned", 0, true, "measured window", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			shortenTimeouts(t, 50*time.Millisecond, 50*time.Millisecond)
			a := &fakeAdapter{name: "fake", latency: 100 * time.Microsecond, hangOn: func(n int64) bool { return n == 3 }}
			if tt.ignoreCtx {
				a.release = make(chan struct{})
				t.Cleanup(func() { close(a.release) })
			}
			p := bench.Profile{Name: "test", Workers: 2, Warmup: tt.warmup, Duration: 30 * time.Millisecond, DeletePool: 100}

			// Act
			start := time.Now()
			_, err := runCell(context.Background(), a, &fakeStore{}, bench.ScenarioSelect, p, testDSN, 1)

			// Assert
			if !errors.Is(err, errPhaseStuck) {
				t.Fatalf("runCell() = %v, want errPhaseStuck", err)
			}
			msg := err.Error()
			for _, want := range []string{tt.wantPhase, "select", "test", "goroutine profile"} {
				if !strings.Contains(msg, want) {
					t.Errorf("error does not mention %q: %.300s", want, msg)
				}
			}
			if got := strings.Contains(msg, "abandoned"); got != tt.wantAbandn {
				t.Errorf("mentions abandoned = %v, want %v", got, tt.wantAbandn)
			}
			if elapsed := time.Since(start); elapsed > 2*time.Second {
				t.Errorf("returned after %v, want it bounded by the timeouts", elapsed)
			}
		})
	}
}

func TestRunCellNoSuccessfulOps(t *testing.T) {
	tests := []struct {
		name string
		a    *fakeAdapter
		sc   bench.Scenario
		p    bench.Profile
	}{
		{
			"every op fails after the probe",
			&fakeAdapter{name: "fake", failFrom: 2},
			bench.ScenarioSelect,
			bench.Profile{Name: "test", Workers: 2, Duration: 30 * time.Millisecond},
		},
		{
			"delete pool exhausted by warm-up",
			&fakeAdapter{name: "fake"},
			bench.ScenarioDelete,
			bench.Profile{Name: "test", Workers: 2, Warmup: 50 * time.Millisecond, Duration: 30 * time.Millisecond, DeletePool: 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (fake and profile from tt)

			// Act
			_, err := runCell(context.Background(), tt.a, &fakeStore{}, tt.sc, tt.p, testDSN, 1)

			// Assert
			if !errors.Is(err, errNoOps) {
				t.Fatalf("runCell() = %v, want errNoOps", err)
			}
		})
	}
}
