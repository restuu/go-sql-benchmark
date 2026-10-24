package report

import (
	"fmt"
	"slices"
	"strings"
)

// Flexibility is the hand-written, qualitative note on how a library composes dynamic queries.
// It is never ranked and never reaches the numeric score.
type Flexibility struct {
	// Capabilities maps each capability from FlexCapabilities to one level from FlexLevels.
	Capabilities map[string]string `json:"capabilities"`
	Notes        []string          `json:"notes"`
}

// flexCapabilities and flexLevels are the fixed vocabulary; the report prints them in this order.
var (
	flexCapabilities = []string{"optional_filters", "dynamic_ordering", "conditional_join", "dynamic_projection", "in_list_batch", "dynamic_pagination"}
	flexLevels       = []string{"native", "partial", "raw_sql", "static_only"}

	flexCapabilityLabels = map[string]string{
		"optional_filters":   "optional filters",
		"dynamic_ordering":   "dynamic ordering",
		"conditional_join":   "conditional join",
		"dynamic_projection": "dynamic projection",
		"in_list_batch":      "in-list / batch",
		"dynamic_pagination": "dynamic pagination",
	}
	flexLevelLabels = map[string]string{
		"native":      "native",
		"partial":     "partial",
		"raw_sql":     "raw SQL",
		"static_only": "static only",
	}
)

// FlexCapabilities returns the capability names a rubric must rate, in report order.
func FlexCapabilities() []string { return slices.Clone(flexCapabilities) }

// FlexLevels returns the levels a capability may take.
func FlexLevels() []string { return slices.Clone(flexLevels) }

func (f Flexibility) validate(library string) error {
	if err := validateRatings(library, "flexibility", "capability", f.Capabilities, flexCapabilities, flexLevels); err != nil {
		return err
	}
	return validateNotes(library, "flexibility", f.Notes)
}

// writeFlexibility prints the capability table and notes for the libraries in the run, in rubric
// order so the section carries no ranking signal.
func writeFlexibility(sb *strings.Builder, rb Rubric, r ranking) {
	shown := inRunLibraries(rb, r)
	sb.WriteString("## Flexibility (qualitative)\n\n")
	sb.WriteString("Hand-written assessment of how each library handles dynamic queries, for example a search endpoint that adds conditions, joins or ordering only when a parameter is present. It is not part of the score, the Top 5 or the stability check, and no query composition was benchmarked.\n\n")
	sb.WriteString("Levels: `native` = the library's own API composes it; `partial` = works with caveats (see notes); `raw SQL` = needs string SQL or an escape hatch; `static only` = the query is fixed at build time, so use several queries.\n\n")
	sb.WriteString("| Library |")
	for _, c := range flexCapabilities {
		sb.WriteString(" " + flexCapabilityLabels[c] + " |")
	}
	sb.WriteString("\n|---|" + strings.Repeat("---|", len(flexCapabilities)) + "\n")
	for _, info := range shown {
		fmt.Fprintf(sb, "| %s |", escapeCell(info.Display))
		for _, c := range flexCapabilities {
			sb.WriteString(" " + flexLevelLabels[info.Flexibility.Capabilities[c]] + " |")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	writeNotes(sb, shown, func(i LibraryInfo) []string { return i.Flexibility.Notes })
}
