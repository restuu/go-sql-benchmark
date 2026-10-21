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

// cellRank is a library's rank per metric in one (scenario, profile).
type cellRank struct {
	scenario, profile string
	ranks             [metricCount]int
	of                int // libraries ranked in this cell
}

// libScore is a library's overall standing; a lower score is better.
type libScore struct {
	library       string
	score         float64 // +Inf when no cell was ranked
	ranked, total int     // ranked cells out of cells present
	meanRank      [metricCount]float64
	cells         []cellRank
}

type ranking struct{ libs []libScore }

// rankCells ranks every library on p95, B/op and CPU ns/op in each (scenario, profile). Cells are
// visited in a fixed order so the floating-point sums do not depend on the order of run.Cells.
func rankCells(run result.Run, w Weights) ranking {
	idx := indexCells(run)
	names := libraryNames(run)
	scores := make([]libScore, len(names))
	for i, n := range names {
		scores[i].library = n
	}
	for _, p := range run.Config.Profiles {
		for _, s := range bench.AllScenarios {
			rankOneCell(idx, scores, string(s), p.Name)
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

// rankOneCell appends this (scenario, profile)'s ranks to each ranked library.
func rankOneCell(idx cellIndex, scores []libScore, scenario, profile string) {
	var inCell []int // indexes into scores of the ranked libraries
	for i := range scores {
		c, ok := idx[cellID{scores[i].library, scenario, profile}]
		if !ok {
			continue
		}
		scores[i].total++
		if isRanked(c) {
			inCell = append(inCell, i)
		}
	}
	for _, i := range inCell {
		cr := cellRank{scenario: scenario, profile: profile, of: len(inCell)}
		own := idx[cellID{scores[i].library, scenario, profile}].Median
		for m := range metricCount {
			cr.ranks[m] = 1
			for _, j := range inCell {
				other := idx[cellID{scores[j].library, scenario, profile}].Median
				if m.value(*other) < m.value(*own) {
					cr.ranks[m]++
				}
			}
		}
		scores[i].cells = append(scores[i].cells, cr)
		scores[i].ranked++
	}
}

func (s *libScore) finish(weights [metricCount]float64) {
	s.score = math.Inf(1)
	if len(s.cells) == 0 {
		return
	}
	var weighted, weightSum float64
	var sums [metricCount]float64
	for _, c := range s.cells {
		for m := range metricCount {
			r := float64(c.ranks[m])
			weighted += weights[m] * r
			weightSum += weights[m]
			sums[m] += r
		}
	}
	s.score = weighted / weightSum
	for m := range metricCount {
		s.meanRank[m] = sums[m] / float64(len(s.cells))
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

// ranked returns the libraries with at least one ranked cell, best first.
func (r ranking) ranked() []libScore {
	n := 0
	for n < len(r.libs) && !math.IsInf(r.libs[n].score, 1) {
		n++
	}
	return r.libs[:n]
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

// rest returns every library not in top(n), including unranked ones.
func (r ranking) rest(n int) []libScore { return r.libs[len(r.top(n)):] }
