package report

import (
	"cmp"
	"math"
	"slices"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	// maxRankedErrorRate is the highest error rate a cell may have and still be ranked.
	maxRankedErrorRate = 0.01
	// topCount is the number of libraries the report ranks and explains.
	topCount = 5
)

// metric is one ranked measurement; lower is better for all of them.
type metric int

const (
	metricP95 metric = iota
	metricBytes
	metricCPU
	metricCount
)

var metricLabels = [metricCount]string{"p95", "B/op", "CPU ns/op"}

func (m metric) value(x result.Metrics) float64 {
	switch m {
	case metricP95:
		return float64(x.LatencyP95Ns)
	case metricBytes:
		return x.BytesPerOp
	default:
		return x.CPUNsPerOp
	}
}

// perMetric returns the weights indexed by metric.
func (w Weights) perMetric() [metricCount]float64 {
	return [metricCount]float64{w.P95, w.BytesPerOp, w.CPUNsPerOp}
}

// cellID addresses one cell of a run.
type cellID struct{ library, scenario, profile string }

type cellIndex map[cellID]result.Cell

func indexCells(run result.Run) cellIndex {
	idx := make(cellIndex, len(run.Cells))
	for _, c := range run.Cells {
		idx[cellID{c.Library, c.Scenario, c.Profile}] = c
	}
	return idx
}

// libraryNames returns the sorted unique library names of a run.
func libraryNames(run result.Run) []string {
	var names []string
	for _, c := range run.Cells {
		names = append(names, c.Library)
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// isRanked reports whether a cell takes part in the ranking.
func isRanked(c result.Cell) bool {
	return c.Status == result.StatusOK && c.Median != nil && c.Median.ErrorRate <= maxRankedErrorRate
}

// isExcluded reports a measured cell left out of the ranking because of its error rate.
func isExcluded(c result.Cell) bool {
	return c.Status == result.StatusOK && !isRanked(c)
}

// cellRatio is a library's cost in one (scenario, profile) relative to the baseline: each metric
// divided by database/sql's value in the same case. 1 is the same, 2 is twice as much, 0.5 half.
type cellRatio struct {
	scenario, profile string
	ratios            [metricCount]float64
}

// libScore is a library's overall standing; a lower score is better.
type libScore struct {
	library       string
	guest         bool    // honorable mention: scored like the others but kept out of the Top 5
	score         float64 // +Inf when no cell was ranked
	ranked, total int     // ranked cells out of cells present
	meanRatio     [metricCount]float64
	cells         []cellRatio
}

type ranking struct{ libs []libScore }

// unscoredScenario is measured and reported on its own but kept out of every score: not every
// library can express it natively, so a score that included it would cover different cases.
const unscoredScenario = bench.ScenarioUnion

// scoredScenarios returns bench.AllScenarios without the unscored one, in report order.
func scoredScenarios() []bench.Scenario {
	return slices.DeleteFunc(slices.Clone(bench.AllScenarios), func(s bench.Scenario) bool { return s == unscoredScenario })
}

// rankCells scores every library on the scored scenarios; see rankScenarios.
func rankCells(run result.Run, w Weights, guests map[string]bool) ranking {
	return rankScenarios(run, w, guests, scoredScenarios())
}

// rankScenarios scores every library on p95, B/op and CPU ns/op as the ratio to the baseline in each
// (scenario, profile) of the given scenarios, combined by a weighted geometric mean. A ratio keeps the size of a
// difference (1% slower is 1.01, not one place lower), and the score of a library does not depend
// on which other libraries are in the run. Cells are visited in a fixed order so the
// floating-point sums do not depend on the order of run.Cells. Libraries in guests are honorable
// mentions: scored the same way but left out of top and rest.
func rankScenarios(run result.Run, w Weights, guests map[string]bool, scenarios []bench.Scenario) ranking {
	idx := indexCells(run)
	names := libraryNames(run)
	scores := make([]libScore, len(names))
	for i, n := range names {
		scores[i].library = n
		scores[i].guest = guests[n]
	}
	for _, p := range run.Config.Profiles {
		for _, s := range scenarios {
			scoreOneCell(idx, scores, string(s), p.Name)
		}
	}
	weights := w.perMetric()
	for i := range scores {
		scores[i].finish(weights)
	}
	slices.SortFunc(scores, func(a, b libScore) int {
		return cmp.Or(cmp.Compare(a.score, b.score), cmp.Compare(a.library, b.library))
	})
	return ranking{libs: scores}
}

// scoreOneCell appends this (scenario, profile)'s ratios to each library that has a ranked cell
// there. Without a ranked baseline cell the case cannot be scored for anyone, and a library whose
// cell has a non-positive metric (or whose baseline has) has no ratio for it.
func scoreOneCell(idx cellIndex, scores []libScore, scenario, profile string) {
	var base *result.Metrics
	if b, ok := idx[cellID{baselineLibrary, scenario, profile}]; ok && isRanked(b) {
		base = b.Median
	}
	for i := range scores {
		c, ok := idx[cellID{scores[i].library, scenario, profile}]
		if !ok {
			continue
		}
		scores[i].total++
		if base == nil || !isRanked(c) {
			continue
		}
		ratios, ok := ratiosTo(*base, *c.Median)
		if !ok {
			continue
		}
		scores[i].cells = append(scores[i].cells, cellRatio{scenario: scenario, profile: profile, ratios: ratios})
		scores[i].ranked++
	}
}

// ratiosTo divides each metric of m by the baseline's; it reports false when a value is not positive.
func ratiosTo(base, m result.Metrics) ([metricCount]float64, bool) {
	var ratios [metricCount]float64
	for k := range metricCount {
		b, v := k.value(base), k.value(m)
		if !(b > 0 && v > 0) {
			return ratios, false
		}
		ratios[k] = v / b
	}
	return ratios, true
}

func (s *libScore) finish(weights [metricCount]float64) {
	s.score = math.Inf(1)
	if len(s.cells) == 0 {
		return
	}
	var weighted, weightSum float64
	var logSums [metricCount]float64
	for _, c := range s.cells {
		for m := range metricCount {
			l := math.Log(c.ratios[m])
			weighted += weights[m] * l
			weightSum += weights[m]
			logSums[m] += l
		}
	}
	s.score = math.Exp(weighted / weightSum)
	for m := range metricCount {
		s.meanRatio[m] = math.Exp(logSums[m] / float64(len(s.cells)))
	}
}

func (r ranking) lookup(name string) (libScore, bool) {
	for _, l := range r.libs {
		if l.library == name {
			return l, true
		}
	}
	return libScore{}, false
}

// pool returns the libraries that take part in the ranking (everything but the honorable
// mentions), best first.
func (r ranking) pool() []libScore {
	return slices.DeleteFunc(slices.Clone(r.libs), func(l libScore) bool { return l.guest })
}

// mentions returns the honorable mentions, best score first.
func (r ranking) mentions() []libScore {
	return slices.DeleteFunc(slices.Clone(r.libs), func(l libScore) bool { return !l.guest })
}

// ranked returns the pool libraries with at least one ranked cell, best first.
func (r ranking) ranked() []libScore {
	p := r.pool()
	n := 0
	for n < len(p) && !math.IsInf(p[n].score, 1) {
		n++
	}
	return p[:n]
}

// top returns up to n best libraries that were ranked at all.
func (r ranking) top(n int) []libScore { return r.ranked()[:min(n, len(r.ranked()))] }

// competitionRanks numbers libraries sorted best first: equal scores share a rank and the next
// distinct score skips ahead (1, 2, 2, 4).
func competitionRanks(libs []libScore) []int {
	ranks := make([]int, len(libs))
	for i := range libs {
		ranks[i] = i + 1
		if i > 0 && libs[i].score == libs[i-1].score {
			ranks[i] = ranks[i-1]
		}
	}
	return ranks
}

// rest returns every pool library not in top(n), including unranked ones.
func (r ranking) rest(n int) []libScore { return r.pool()[len(r.top(n)):] }

// TopLibraries returns the names of the top-ranked libraries of a run, best first, using the
// same ranking as the report; honorable mentions are left out.
func TopLibraries(run result.Run, rb Rubric) []string {
	top := rankCells(run, rb.Weights, rb.honorableMentions()).top(topCount)
	names := make([]string, len(top))
	for i, l := range top {
		names[i] = l.library
	}
	return names
}
