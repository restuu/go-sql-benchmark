package report

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// combinedRow is one library's place in the measured ranking and in the qualitative ranking.
type combinedRow struct {
	library     string
	quant, qual int
}

func (c combinedRow) sum() int { return c.quant + c.qual }

// orderCombined sorts rows by the sum of the two ranks, lowest first, ties by display name, and
// numbers them: equal sums share a rank and the next distinct sum skips ahead (1, 1, 3).
func orderCombined(rb Rubric, rows []combinedRow) ([]combinedRow, []int) {
	ordered := slices.Clone(rows)
	slices.SortStableFunc(ordered, func(a, b combinedRow) int {
		if a.sum() != b.sum() {
			return a.sum() - b.sum()
		}
		return strings.Compare(displayName(rb, a.library), displayName(rb, b.library))
	})
	ranks := make([]int, len(ordered))
	for i := range ordered {
		ranks[i] = i + 1
		if i > 0 && ordered[i].sum() == ordered[i-1].sum() {
			ranks[i] = ranks[i-1]
		}
	}
	return ordered, ranks
}

// writeCombinedRanking prints the sum of each library's measured rank and qualitative rank. Only
// libraries that have both are listed; honorable mentions are left out, as in both rankings.
func writeCombinedRanking(sb *strings.Builder, rb Rubric, r ranking) error {
	checked, err := time.Parse(dateLayout, rb.CommunityChecked)
	if err != nil {
		return fmt.Errorf("report: community_checked %q: %w", rb.CommunityChecked, err)
	}
	measured := r.ranked()
	quals, err := qualScores(rb, r.pool(), checked) // the same pool as the printed qualitative ranking
	if err != nil {
		return err
	}
	qualRank := make(map[string]int, len(quals))
	for i, rank := range qualRanks(quals) {
		qualRank[quals[i].library] = rank
	}
	rows := make([]combinedRow, len(measured))
	for i, rank := range competitionRanks(measured) {
		qual, ok := qualRank[measured[i].library]
		if !ok {
			return fmt.Errorf("report: combined ranking: library %q has no qualitative rank", measured[i].library)
		}
		rows[i] = combinedRow{library: measured[i].library, quant: rank, qual: qual}
	}

	sb.WriteString("\n## Combined ranking (judgement)\n\n")
	sb.WriteString("A third ranking that adds each library's place in the measured ranking (Top 5 and \"Also measured\") " +
		"to its place in the qualitative ranking above; the lowest sum is first. Both rankings count equally. " +
		"Equal sums share a rank and are listed alphabetically. Only places are added, so how far apart two libraries are in score " +
		"is lost, and a sum of a few places is within noise. The qualitative half is the author's judgement, so this ranking is too.\n\n")
	sb.WriteString("| Rank | Library | Measured rank | Qualitative rank | Sum |\n|---|---|---|---|---|\n")
	ordered, ranks := orderCombined(rb, rows)
	for i, row := range ordered {
		fmt.Fprintf(sb, "| %d | %s | %d | %d | %d |\n", ranks[i], escapeCell(displayName(rb, row.library)), row.quant, row.qual, row.sum())
	}
	if len(r.mentions()) > 0 {
		sb.WriteString("\nHonorable mentions are not ranked here either.\n")
	}
	if unmeasured := r.pool()[len(measured):]; len(unmeasured) > 0 {
		names := make([]string, len(unmeasured))
		for i, l := range unmeasured {
			names[i] = displayName(rb, l.library)
		}
		fmt.Fprintf(sb, "\nNot listed (no ranked case): %s.\n", strings.Join(names, ", "))
	}
	return nil
}
