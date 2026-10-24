package report

import (
	"fmt"
	"math"
	"strings"
)

const (
	ratioDecimals = 2
	// ratioScale turns a ratio into hundredths, the precision it is printed with.
	ratioScale = 100
)

// reasonsFor builds up to three fixed-template sentences from a library's ratios to the baseline:
// the mean ratio of each metric that counts in the score, the case with the highest p95 ratio when
// it is above the baseline, and how many cases the score is based on.
func reasonsFor(lib libScore, w Weights) []string {
	if len(lib.cells) == 0 {
		return nil
	}
	var out []string
	if lib.library == baselineLibrary {
		out = append(out, "the baseline: every ratio is 1.00 by definition")
	} else {
		out = append(out, meanRatioSentence(lib, w))
		if worst := slowestCase(lib.cells); worstAboveBaseline(worst.ratios[metricP95]) {
			out = append(out, fmt.Sprintf("slowest case: %s/%s (p95 %s database/sql)",
				worst.scenario, worst.profile, ratioText(worst.ratios[metricP95])))
		}
	}
	if lib.ranked < lib.total {
		out = append(out, fmt.Sprintf("judged on %d of %d cases", lib.ranked, lib.total))
	}
	return out
}

// meanRatioSentence lists the mean ratio of every metric with a non-zero weight, the ones that
// count in the score.
func meanRatioSentence(lib libScore, w Weights) string {
	weights := w.perMetric()
	var parts []string
	for m := range metricCount {
		if weights[m] > 0 {
			parts = append(parts, metricLabels[m]+" "+ratioText(lib.meanRatio[m]))
		}
	}
	return "ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): " + strings.Join(parts, ", ")
}

func slowestCase(cells []cellRatio) cellRatio {
	worst := cells[0]
	for _, c := range cells[1:] {
		if c.ratios[metricP95] > worst.ratios[metricP95] {
			worst = c
		}
	}
	return worst
}

// worstAboveBaseline is true when the ratio still reads above 1.00 at the printed precision.
func worstAboveBaseline(ratio float64) bool { return math.Round(ratio*ratioScale) > ratioScale }

func ratioText(ratio float64) string { return fixed(ratio, ratioDecimals) + "x" }
