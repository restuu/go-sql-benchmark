package report

import (
	"time"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

// okCell builds a measured cell whose p50 and p99 bracket the given p95.
func okCell(library, scenario, profile string, p95 int64, bytesPerOp, cpuNsPerOp float64) result.Cell {
	m := result.Metrics{
		Ops: 100, ThroughputOpsPerSec: 100,
		LatencyP50Ns: p95 / 2, LatencyP95Ns: p95, LatencyP99Ns: p95 * 2,
		BytesPerOp: bytesPerOp, AllocsPerOp: 1, CPUNsPerOp: cpuNsPerOp, ElapsedSeconds: 1,
	}
	return result.Cell{
		Library: library, Scenario: scenario, Profile: profile, Status: result.StatusOK,
		Repeats: []result.Metrics{m}, Median: &m,
	}
}

func withErrorRate(c result.Cell, rate float64) result.Cell {
	m := *c.Median
	m.ErrorRate = rate
	c.Median = &m
	c.Repeats = []result.Metrics{m}
	return c
}

func naCell(library, scenario, profile string) result.Cell {
	return result.Cell{Library: library, Scenario: scenario, Profile: profile, Status: result.StatusNA, Reason: "n/a (unsupported)"}
}

// runOf wraps cells in a run with the low and stress profiles and one repeat.
func runOf(cells ...result.Cell) result.Run {
	return result.Run{
		SchemaVersion: result.SchemaVersion,
		StartedAt:     time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC),
		Config: result.Config{Seed: 7, Repeats: 1, Profiles: []result.ProfileConfig{
			{Name: "low", Workers: 4, IntervalNs: int64(10 * time.Millisecond), WarmupNs: int64(time.Second), DurationNs: int64(5 * time.Second), DeletePool: 5000},
			{Name: "stress", Workers: 100, WarmupNs: int64(2 * time.Second), DurationNs: int64(10 * time.Second), DeletePool: 400000},
		}},
		Cells: cells,
	}
}

var equalWeights = Weights{P95: 1, BytesPerOp: 1, CPUNsPerOp: 1}

// renderFull renders both documents with the default file names and joins them, so a test can look
// for a section without caring which document holds it.
func renderFull(run result.Run, rb Rubric) (string, error) {
	doc, err := Render(run, rb, DefaultFiles)
	if err != nil {
		return "", err
	}
	return doc.Summary + "\n" + doc.Details, nil
}
