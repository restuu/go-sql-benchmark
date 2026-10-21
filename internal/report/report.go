package report

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	// driverModule is the key in Host.Libraries of the driver shared by every database/sql library.
	driverModule  = "pgx"
	scoreDecimals = 2
)

// Render turns one run and the rubric into Markdown. The output depends only on its inputs: no
// clock, no map order, and the order of run.Cells does not matter.
func Render(run result.Run, rb Rubric) (string, error) {
	if err := checkComplete(run, rb); err != nil {
		return "", err
	}
	r := rankCells(run, rb.Weights)
	var sb strings.Builder
	writeHeader(&sb, run)
	writeLibraries(&sb, run, rb, r)
	writeConfig(&sb, run)
	writeMethodology(&sb, rb)
	writeTables(&sb, run, rb, r)
	writeTop(&sb, rb, r)
	writeProsCons(&sb, rb, r)
	return sb.String(), nil
}

// checkComplete requires a cell (ok or na) for every library x scenario x profile and a rubric
// entry for every library, so a gap is an error rather than a silently shorter table.
func checkComplete(run result.Run, rb Rubric) error {
	libs := libraryNames(run)
	if len(libs) == 0 {
		return errors.New("report: run has no cells")
	}
	idx := indexCells(run)
	for _, lib := range libs {
		if _, ok := rb.library(lib); !ok {
			return fmt.Errorf("report: library %q has no rubric entry", lib)
		}
		for _, p := range run.Config.Profiles {
			for _, s := range bench.AllScenarios {
				c, ok := idx[cellID{lib, string(s), p.Name}]
				switch {
				case !ok:
					return fmt.Errorf("report: missing cell %s/%s/%s", lib, s, p.Name)
				case c.Status == result.StatusOK && c.Median == nil:
					return fmt.Errorf("report: cell %s/%s/%s has no median", lib, s, p.Name)
				}
			}
		}
	}
	return nil
}

func writeHeader(sb *strings.Builder, run result.Run) {
	sb.WriteString("# Go SQL library benchmark\n\n")
	fmt.Fprintf(sb, "Run started %s (UTC). Numbers are only comparable on the same machine.\n\n", run.StartedAt.UTC().Format(time.RFC3339))
}

func writeLibraries(sb *strings.Builder, run result.Run, rb Rubric, r ranking) {
	h := run.Host
	sb.WriteString("## Environment\n\n")
	fmt.Fprintf(sb, "- OS/arch: %s/%s\n- Go: %s\n- CPUs: %d (GOMAXPROCS %d)\n- PostgreSQL: %s\n",
		h.GOOS, h.GOARCH, h.GoVersion, h.NumCPU, h.GOMAXPROCS, h.PostgresVersion)
	if v, ok := h.Libraries[driverModule]; ok {
		fmt.Fprintf(sb, "- Driver shared by every database/sql-based library: %s %s\n", driverModule, v)
	}
	sb.WriteString("\n| Library | Version | Label | Version used |\n|---|---|---|---|\n")
	for _, l := range r.libs {
		info, _ := rb.library(l.library)
		fmt.Fprintf(sb, "| %s | %s | %s | %s |\n", escapeCell(info.Display),
			dashIfEmpty(h.Libraries[info.Module]), dashIfEmpty(escapeCell(info.Label)), dashIfEmpty(escapeCell(info.VersionUsed)))
	}
	sb.WriteString("\n")
}

func writeConfig(sb *strings.Builder, run result.Run) {
	c := run.Config
	sb.WriteString("## Run configuration\n\n")
	fmt.Fprintf(sb, "Repeats per cell: %d (the median is reported). Seed: %d.\n\n", c.Repeats, c.Seed)
	sb.WriteString("| Profile | Workers | Pacing | Warm-up | Window | DELETE ids |\n|---|---:|---:|---:|---:|---:|\n")
	for _, p := range c.Profiles {
		pacing := "unpaced"
		if p.IntervalNs > 0 {
			pacing = time.Duration(p.IntervalNs).String()
		}
		fmt.Fprintf(sb, "| %s | %d | %s | %s | %s | %d |\n", escapeCell(p.Name), p.Workers, pacing,
			time.Duration(p.WarmupNs), time.Duration(p.DurationNs), p.DeletePool)
	}
	sb.WriteString("\n")
}

func writeMethodology(sb *strings.Builder, rb Rubric) {
	sb.WriteString("## Methodology\n\n")
	for _, line := range rb.Methodology {
		sb.WriteString("- " + line + "\n")
	}
	sb.WriteString("\nReading the tables:\n\n")
	fmt.Fprintf(sb, "- `p95 vs database/sql` is relative to the baseline's p95 in the same table; negative is faster.\n")
	fmt.Fprintf(sb, "- `p95 spread` is (max - min) / median of the repeats' p95; `*` marks a spread above %s%%.\n", fixed(spreadWarn*percent, 0))
	fmt.Fprintf(sb, "- `!` after a library means an error rate above %s%%: the cell is shown but not ranked.\n", fixed(maxRankedErrorRate*percent, 0))
	sb.WriteString("- `[n]` points to the numbered note under the table.\n")
	sb.WriteString("- `n/a (...)` means the library cannot express the scenario; the cell is not ranked.\n\n")
}

func writeTables(sb *strings.Builder, run result.Run, rb Rubric, r ranking) {
	idx := indexCells(run)
	order := make([]string, len(r.libs))
	for i, l := range r.libs {
		order[i] = l.library
	}
	sb.WriteString("## Results\n\n")
	for _, p := range run.Config.Profiles {
		fmt.Fprintf(sb, "### Profile: %s\n\n", p.Name)
		for _, s := range bench.AllScenarios {
			fmt.Fprintf(sb, "#### %s\n\n%s\n", s, scenarioTable(idx, order, string(s), p.Name, rb))
		}
	}
}

func writeTop(sb *strings.Builder, rb Rubric, r ranking) {
	fmt.Fprintf(sb, "## Top %d\n\n", topCount)
	fmt.Fprintf(sb, "Score is the mean rank over p95, B/op and CPU ns/op in every ranked cell, weighted (%s); lower is better.\n\n", weightsText(rb.Weights))
	sb.WriteString("Libraries with equal scores share a rank.\n\n")
	sb.WriteString("| Rank | Library | Score | Ranked cells |\n|---:|---|---:|---:|\n")
	top := r.top(topCount)
	for i, rank := range competitionRanks(top) {
		l := top[i]
		fmt.Fprintf(sb, "| %d | %s | %s | %d/%d |\n", rank, escapeCell(displayName(rb, l.library)), scoreText(l), l.ranked, l.total)
	}
	sb.WriteString("\n")
	for _, l := range top {
		fmt.Fprintf(sb, "**%s**\n\n", displayName(rb, l.library))
		for _, reason := range reasonsFor(l, rb.Weights) {
			sb.WriteString("- " + reason + "\n")
		}
		sb.WriteString("\n")
	}
	if rest := r.rest(topCount); len(rest) > 0 {
		sb.WriteString("Also measured:\n\n")
		for _, l := range rest {
			fmt.Fprintf(sb, "- %s: score %s, ranked cells %d/%d\n", displayName(rb, l.library), scoreText(l), l.ranked, l.total)
		}
		sb.WriteString("\n")
	}
}

func writeProsCons(sb *strings.Builder, rb Rubric, r ranking) {
	sb.WriteString("## Pros and cons\n\n")
	for _, l := range r.libs {
		info, _ := rb.library(l.library)
		fmt.Fprintf(sb, "### %s\n\n", info.Display)
		writeList(sb, "Pros", info.Pros)
		writeList(sb, "Cons", info.Cons)
	}
}

func writeList(sb *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(sb, "**%s**\n\n", title)
	for _, item := range items {
		sb.WriteString("- " + item + "\n")
	}
	sb.WriteString("\n")
}

func weightsText(w Weights) string {
	parts := make([]string, metricCount)
	for m, v := range w.perMetric() {
		parts[m] = metricLabels[m] + " " + strconv.FormatFloat(v, 'f', -1, 64)
	}
	return strings.Join(parts, ", ")
}

func scoreText(l libScore) string {
	if l.ranked == 0 {
		return "-"
	}
	return fixed(l.score, scoreDecimals)
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
