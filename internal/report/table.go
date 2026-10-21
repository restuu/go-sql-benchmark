package report

import (
	"slices"
	"strconv"
	"strings"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	// baselineLibrary is the reference every p95 delta is relative to.
	baselineLibrary = "database_sql"
	// spreadWarn is the repeat spread above which a cell is starred; it is the stability criterion.
	spreadWarn = 0.10
	percent    = 100

	nsPerMicro = 1_000
	nsPerMilli = 1_000_000
	nsPerSec   = 1_000_000_000
	// twoDigitsFrom is the value from which one decimal would give three significant digits.
	twoDigitsFrom = 10

	tableHeader = "| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |\n" +
		"|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n"
	tableColumns = 11
)

// fixed formats with a fixed number of decimals so output never depends on %v float printing.
func fixed(v float64, decimals int) string { return strconv.FormatFloat(v, 'f', decimals, 64) }

// fmtDuration prints nanoseconds with two significant digits in the largest fitting unit.
func fmtDuration(ns int64) string {
	units := []struct {
		name string
		size int64
	}{{"s", nsPerSec}, {"ms", nsPerMilli}, {"µs", nsPerMicro}}
	for _, u := range units {
		if ns < u.size {
			continue
		}
		v := float64(ns) / float64(u.size)
		if v >= twoDigitsFrom {
			return fixed(v, 0) + " " + u.name
		}
		return fixed(v, 1) + " " + u.name
	}
	return strconv.FormatInt(ns, 10) + " ns"
}

// fmtDelta prints v relative to base as a signed percentage; a zero base has no delta.
func fmtDelta(base, v int64) string {
	if base == 0 {
		return "-"
	}
	s := fixed(float64(v-base)/float64(base)*percent, 1)
	if s == "-0.0" {
		s = "0.0"
	}
	if !strings.HasPrefix(s, "-") {
		s = "+" + s
	}
	return s + "%"
}

// fmtSpread prints (max-min)/median of the repeats' p95, starred above spreadWarn.
func fmtSpread(repeats []result.Metrics) string {
	if len(repeats) < 2 {
		return "-"
	}
	p95s := make([]float64, len(repeats))
	for i, m := range repeats {
		p95s[i] = float64(m.LatencyP95Ns)
	}
	slices.Sort(p95s)
	mid := len(p95s) / 2
	median := p95s[mid]
	if len(p95s)%2 == 0 {
		median = (p95s[mid-1] + p95s[mid]) / 2
	}
	if median == 0 {
		return "-"
	}
	spread := (p95s[len(p95s)-1] - p95s[0]) / median
	out := fixed(spread*percent, 1) + "%"
	if spread > spreadWarn {
		out += " *"
	}
	return out
}

func escapeCell(s string) string {
	return strings.NewReplacer("|", `\|`, "\r\n", " ", "\n", " ").Replace(s)
}

func displayName(rb Rubric, library string) string {
	if info, ok := rb.library(library); ok {
		return info.Display
	}
	return library
}

// scenarioTable renders one scenario under one profile, a row per library in the given order,
// followed by the numbered cell notes its markers point to.
func scenarioTable(idx cellIndex, libs []string, scenario, profile string, rb Rubric) string {
	var baseP95 int64
	var hasBase bool
	if b, ok := idx[cellID{baselineLibrary, scenario, profile}]; ok && b.Status == result.StatusOK && b.Median != nil {
		baseP95, hasBase = b.Median.LatencyP95Ns, true
	}
	var sb strings.Builder
	sb.WriteString(tableHeader)
	var notes []string
	for _, lib := range libs {
		c := idx[cellID{lib, scenario, profile}]
		label := escapeCell(displayName(rb, lib))
		if isExcluded(c) {
			label += " !"
		}
		for _, n := range rb.notesFor(lib, scenario) {
			notes = append(notes, n)
			label += " [" + strconv.Itoa(len(notes)) + "]"
		}
		cells := append([]string{label}, rowValues(lib, c, baseP95, hasBase)...)
		sb.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}
	if len(notes) > 0 {
		sb.WriteString("\n")
		for i, n := range notes {
			sb.WriteString(strconv.Itoa(i+1) + ". " + n + "\n")
		}
	}
	return sb.String()
}

// rowValues returns the ten value columns of a row; an unmeasured cell shows its reason first.
func rowValues(lib string, c result.Cell, baseP95 int64, hasBase bool) []string {
	vals := make([]string, tableColumns-1)
	for i := range vals {
		vals[i] = "-"
	}
	if c.Status != result.StatusOK || c.Median == nil {
		if c.Reason != "" {
			vals[0] = escapeCell(c.Reason)
		}
		return vals
	}
	m := c.Median
	copy(vals, []string{
		fmtDuration(m.LatencyP50Ns), fmtDuration(m.LatencyP95Ns), fmtDuration(m.LatencyP99Ns),
		fixed(m.ThroughputOpsPerSec, 0), fixed(m.BytesPerOp, 0), fixed(m.AllocsPerOp, 1), fixed(m.CPUNsPerOp, 0),
		fixed(m.ErrorRate*percent, 2),
	})
	switch {
	case lib == baselineLibrary:
		vals[8] = "baseline"
	case hasBase:
		vals[8] = fmtDelta(baseP95, m.LatencyP95Ns)
	}
	vals[9] = fmtSpread(c.Repeats)
	return vals
}
