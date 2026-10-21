package runner

import (
	"math"
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

func oneToHundred() []int64 {
	s := make([]int64, 100)
	for i := range s {
		s[i] = int64(i + 1)
	}
	return s
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		sorted []int64
		p      float64
		want   int64
	}{
		{"empty", nil, 50, 0},
		{"single p50", []int64{7}, 50, 7},
		{"single p99", []int64{7}, 99, 7},
		{"1..100 p50", oneToHundred(), 50, 50},
		{"1..100 p95", oneToHundred(), 95, 95},
		{"1..100 p99", oneToHundred(), 99, 99},
		{"p0 clamps to first", oneToHundred(), 0, 1},
		{"p100 is last", oneToHundred(), 100, 100},
		{"two samples p50", []int64{10, 20}, 50, 10},
		{"two samples p51", []int64{10, 20}, 51, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (inputs from tt)

			// Act
			got := percentile(tt.sorted, tt.p)

			// Assert
			if got != tt.want {
				t.Fatalf("percentile(p=%v) = %d, want %d", tt.p, got, tt.want)
			}
		})
	}
}

func TestMedian(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{"empty", nil, 0},
		{"single", []float64{4}, 4},
		{"odd", []float64{9, 1, 5}, 5},
		{"even", []float64{4, 1, 3, 2}, 2.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			in := append([]float64(nil), tt.values...)

			// Act
			got := median(in)

			// Assert
			if got != tt.want {
				t.Fatalf("median(%v) = %v, want %v", tt.values, got, tt.want)
			}
			for i := range tt.values {
				if in[i] != tt.values[i] {
					t.Fatalf("median mutated its input: %v", in)
				}
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name string
		in   windowStats
		want result.Metrics
	}{
		{
			name: "normal",
			in: windowStats{
				Latencies: shuffled(oneToHundred()), Errors: 0, Elapsed: 2 * time.Second,
				AllocBytes: 10_000, Mallocs: 500, CPU: 200 * time.Microsecond * 100,
			},
			want: result.Metrics{
				Ops: 100, ErrorRate: 0, ThroughputOpsPerSec: 50,
				LatencyP50Ns: 50, LatencyP95Ns: 95, LatencyP99Ns: 99,
				BytesPerOp: 100, AllocsPerOp: 5, CPUNsPerOp: 200_000, ElapsedSeconds: 2,
			},
		},
		{
			name: "all errors",
			in:   windowStats{Errors: 5, Elapsed: time.Second, AllocBytes: 99, Mallocs: 9, CPU: time.Millisecond},
			want: result.Metrics{Errors: 5, ErrorRate: 1, ElapsedSeconds: 1},
		},
		{
			name: "some errors",
			in:   windowStats{Latencies: []int64{1, 2, 3}, Errors: 1, Elapsed: time.Second},
			want: result.Metrics{
				Ops: 3, Errors: 1, ErrorRate: 0.25, ThroughputOpsPerSec: 3,
				LatencyP50Ns: 2, LatencyP95Ns: 3, LatencyP99Ns: 3, ElapsedSeconds: 1,
			},
		},
		{
			name: "zero elapsed",
			in:   windowStats{Latencies: []int64{5}},
			want: result.Metrics{Ops: 1, LatencyP50Ns: 5, LatencyP95Ns: 5, LatencyP99Ns: 5},
		},
		{name: "nothing", in: windowStats{}, want: result.Metrics{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (input from tt)

			// Act
			got := summarize(tt.in)

			// Assert
			if got != tt.want {
				t.Fatalf("summarize() = %+v, want %+v", got, tt.want)
			}
			assertFinite(t, got)
		})
	}
}

func shuffled(s []int64) []int64 {
	out := make([]int64, len(s))
	for i, v := range s {
		out[(i*37)%len(s)] = v // 37 is coprime with 100, so this is a permutation
	}
	return out
}

func assertFinite(t *testing.T, m result.Metrics) {
	t.Helper()
	for _, f := range []float64{m.ErrorRate, m.ThroughputOpsPerSec, m.BytesPerOp, m.AllocsPerOp, m.CPUNsPerOp, m.ElapsedSeconds} {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Fatalf("non-finite metric in %+v", m)
		}
	}
}

func TestMedianMetrics(t *testing.T) {
	// Arrange
	mk := func(p95 int64, tput float64) result.Metrics {
		return result.Metrics{Ops: p95, LatencyP50Ns: p95 / 2, LatencyP95Ns: p95, LatencyP99Ns: p95 + 1, ThroughputOpsPerSec: tput, ErrorRate: 0.1}
	}
	repeats := []result.Metrics{mk(300, 30), mk(100, 10), mk(200, 20)}

	// Act
	got := medianMetrics(repeats)

	// Assert
	want := mk(200, 20)
	if got != want {
		t.Fatalf("medianMetrics() = %+v, want %+v", got, want)
	}
}
