// Package runner executes benchmark scenarios against adapters and assembles the result.
package runner

import (
	"math"
	"slices"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	percentileP50 = 50
	percentileP95 = 95
	percentileP99 = 99
	percentScale  = 100
)

// windowStats is everything measured over one window, before reduction to result.Metrics.
type windowStats struct {
	Latencies  []int64 // successful-op service times in ns; summarize sorts it in place
	Errors     int64
	Elapsed    time.Duration
	AllocBytes uint64 // TotalAlloc delta over the window
	Mallocs    uint64 // Mallocs delta over the window
	CPU        time.Duration
}

// percentile returns the nearest-rank percentile (p in 0..100) of a sorted slice; 0 for empty input.
func percentile(sorted []int64, p float64) int64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	rank := int(math.Ceil(p / percentScale * float64(n)))
	return sorted[min(max(rank-1, 0), n-1)]
}

// median returns the median of values without modifying them; 0 for empty input.
func median(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// summarize reduces a window to Metrics. Every ratio is guarded so the result is always finite.
func summarize(in windowStats) result.Metrics {
	slices.Sort(in.Latencies)
	ops := int64(len(in.Latencies))
	m := result.Metrics{
		Ops:            ops,
		Errors:         in.Errors,
		LatencyP50Ns:   percentile(in.Latencies, percentileP50),
		LatencyP95Ns:   percentile(in.Latencies, percentileP95),
		LatencyP99Ns:   percentile(in.Latencies, percentileP99),
		ElapsedSeconds: in.Elapsed.Seconds(),
	}
	if attempts := ops + in.Errors; attempts > 0 {
		m.ErrorRate = float64(in.Errors) / float64(attempts)
	}
	if secs := in.Elapsed.Seconds(); secs > 0 {
		m.ThroughputOpsPerSec = float64(ops) / secs
	}
	if ops > 0 {
		m.BytesPerOp = float64(in.AllocBytes) / float64(ops)
		m.AllocsPerOp = float64(in.Mallocs) / float64(ops)
		m.CPUNsPerOp = float64(in.CPU.Nanoseconds()) / float64(ops)
	}
	return m
}

// medianMetrics takes the per-field median over repeats. Percentiles stay ordered because the
// median is monotone per field.
func medianMetrics(repeats []result.Metrics) result.Metrics {
	field := func(get func(result.Metrics) float64) float64 {
		values := make([]float64, len(repeats))
		for i, r := range repeats {
			values[i] = get(r)
		}
		return median(values)
	}
	rounded := func(get func(result.Metrics) int64) int64 {
		return int64(math.Round(field(func(m result.Metrics) float64 { return float64(get(m)) })))
	}
	return result.Metrics{
		Ops:                 rounded(func(m result.Metrics) int64 { return m.Ops }),
		Errors:              rounded(func(m result.Metrics) int64 { return m.Errors }),
		ErrorRate:           field(func(m result.Metrics) float64 { return m.ErrorRate }),
		ThroughputOpsPerSec: field(func(m result.Metrics) float64 { return m.ThroughputOpsPerSec }),
		LatencyP50Ns:        rounded(func(m result.Metrics) int64 { return m.LatencyP50Ns }),
		LatencyP95Ns:        rounded(func(m result.Metrics) int64 { return m.LatencyP95Ns }),
		LatencyP99Ns:        rounded(func(m result.Metrics) int64 { return m.LatencyP99Ns }),
		BytesPerOp:          field(func(m result.Metrics) float64 { return m.BytesPerOp }),
		AllocsPerOp:         field(func(m result.Metrics) float64 { return m.AllocsPerOp }),
		CPUNsPerOp:          field(func(m result.Metrics) float64 { return m.CPUNsPerOp }),
		ElapsedSeconds:      field(func(m result.Metrics) float64 { return m.ElapsedSeconds }),
	}
}
