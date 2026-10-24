package report

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	supportNative = "native"
	supportNA     = "n/a"
)

// writeFeatureMatrix prints which scenarios each library expresses with its own query API, one row
// per library in score order, with the feature notes numbered under the table.
func writeFeatureMatrix(sb *strings.Builder, run result.Run, rb Rubric, r ranking) {
	idx := indexCells(run)
	sb.WriteString("## Feature matrix\n\n")
	sb.WriteString("Which scenarios each library expresses with its own query API. `n/a` means it needs hand-written SQL for that scenario.\n\n")
	header, divider := []string{"Library"}, []string{"---"}
	for _, s := range bench.AllScenarios {
		header, divider = append(header, string(s)), append(divider, "---")
	}
	sb.WriteString("| " + strings.Join(header, " | ") + " |\n|" + strings.Join(divider, "|") + "|\n")
	var notes []string
	for _, l := range r.libs {
		row := []string{escapeCell(displayName(rb, l.library))}
		for _, s := range bench.AllScenarios {
			label := supportOf(idx, run.Config.Profiles, l.library, string(s))
			for _, n := range rb.featureNotes(l.library, string(s)) {
				notes = append(notes, n)
				label += " [" + strconv.Itoa(len(notes)) + "]"
			}
			row = append(row, label)
		}
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	sb.WriteString("\n")
	for i, n := range notes {
		sb.WriteString(strconv.Itoa(i+1) + ". " + n + "\n")
	}
	if len(notes) > 0 {
		sb.WriteString("\n")
	}
}

// supportOf is n/a only when the library is n/a under every profile.
func supportOf(idx cellIndex, profiles []result.ProfileConfig, library, scenario string) string {
	for _, p := range profiles {
		if idx[cellID{library, scenario, p.Name}].Status != result.StatusNA {
			return supportNative
		}
	}
	return supportNA
}

// writeUnion prints the ranking of the unscored scenario; r must be scored on that scenario only.
func writeUnion(sb *strings.Builder, rb Rubric, r ranking) {
	sb.WriteString("## Union\n\n")
	sb.WriteString("UNION is measured for every library that can express it natively but is not part of any score above. " +
		"This ranking uses the same ratio to database/sql, over the union cases only (every load profile).\n\n")
	ranked := r.ranked()
	if len(ranked) == 0 {
		sb.WriteString("Nothing could be ranked on union: every score is a ratio to database/sql, so the run needs measured database/sql union cases.\n\n")
		return
	}
	sb.WriteString("| Rank | Library | Score | Ranked cases | p95 | B/op | CPU ns/op |\n|---:|---|---:|---:|---:|---:|---:|\n")
	for i, rank := range competitionRanks(ranked) {
		writeUnionRow(sb, rb, strconv.Itoa(rank), ranked[i])
	}
	var unranked []string
	for _, l := range r.libs {
		switch {
		case l.ranked == 0:
			unranked = append(unranked, escapeCell(displayName(rb, l.library)))
		case l.guest:
			writeUnionRow(sb, rb, "-", l)
		}
	}
	sb.WriteString("\n")
	if len(unranked) > 0 {
		fmt.Fprintf(sb, "No ranked union result: %s.\n\n", strings.Join(unranked, ", "))
	}
}

func writeUnionRow(sb *strings.Builder, rb Rubric, rank string, l libScore) {
	fmt.Fprintf(sb, "| %s | %s | %s | %d/%d | %s | %s | %s |\n", rank, escapeCell(displayName(rb, l.library)), scoreText(l), l.ranked, l.total,
		ratioText(l.meanRatio[metricP95]), ratioText(l.meanRatio[metricBytes]), ratioText(l.meanRatio[metricCPU]))
}
