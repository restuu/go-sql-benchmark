package report

import "fmt"

const rankDecimals = 1

// reasonsFor builds up to four fixed-template sentences from a library's ranks; each is left out
// when its input does not exist or says nothing (a library that never ranked below 1). The weakest
// metric is picked among the metrics with a non-zero weight, the ones that count in the score.
func reasonsFor(lib libScore, w Weights) []string {
	if len(lib.cells) == 0 {
		return nil
	}
	var out []string
	if fastest := countFastest(lib.cells); fastest > 0 {
		out = append(out, fmt.Sprintf("fastest p95 in %d of %d ranked cells", fastest, len(lib.cells)))
	}
	weights := w.perMetric()
	weakest, found := metricCount, false
	for m := range metricCount {
		if weights[m] > 0 && (!found || lib.meanRank[m] > lib.meanRank[weakest]) {
			weakest, found = m, true
		}
	}
	if found && lib.meanRank[weakest] > 1 {
		out = append(out, fmt.Sprintf("weakest on %s: mean rank %s", metricLabels[weakest],
			fixed(lib.meanRank[weakest], rankDecimals)))
	}
	worst := lib.cells[0]
	for _, c := range lib.cells[1:] {
		if c.ranks[metricP95] > worst.ranks[metricP95] {
			worst = c
		}
	}
	if worst.ranks[metricP95] > 1 {
		out = append(out, fmt.Sprintf("slowest cell: %s/%s (p95 rank %d of %d)",
			worst.scenario, worst.profile, worst.ranks[metricP95], worst.of))
	}
	if lib.ranked < lib.total {
		out = append(out, fmt.Sprintf("judged on %d of %d cells", lib.ranked, lib.total))
	}
	return out
}

func countFastest(cells []cellRank) int {
	n := 0
	for _, c := range cells {
		if c.ranks[metricP95] == 1 {
			n++
		}
	}
	return n
}
