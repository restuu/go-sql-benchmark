package report

import (
	_ "embed"
	"errors"
	"fmt"
	"net/url"
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

// disclaimer and usage are hand-written Markdown printed verbatim in the summary.
//
//go:embed disclaimer.md
var disclaimer string

//go:embed usage.md
var usage string

//go:embed reading.md
var reading string

//go:embed glossary.md
var glossary string

// Files names the two generated documents so each can link to the other.
type Files struct{ Summary, Details string }

// DefaultFiles are the names the Makefile writes.
var DefaultFiles = Files{Summary: "README.md", Details: "REPORT.md"}

// Report is the rendered result: a short Summary (ranking, how to run it, environment,
// methodology) and the Details (per-case tables, pros and cons, qualitative sections).
type Report struct{ Summary, Details string }

// Render turns one run and the rubric into the summary and the detailed report. The output depends
// only on its inputs: no clock, no map order, and the order of run.Cells does not matter.
func Render(run result.Run, rb Rubric, files Files) (Report, error) {
	if err := checkComplete(run, rb); err != nil {
		return Report{}, err
	}
	r := rankCells(run, rb.Weights, rb.honorableMentions())
	unionRank := rankScenarios(run, rb.Weights, rb.honorableMentions(), []bench.Scenario{unscoredScenario})

	var summary strings.Builder
	writeHeader(&summary, run, files)
	summary.WriteString(strings.TrimSpace(reading) + "\n\n")
	writeTop(&summary, rb, r)
	writeHonorableMention(&summary, rb, r)
	writeFeatureMatrix(&summary, run, rb, r)
	writeUnion(&summary, rb, unionRank)
	summary.WriteString(strings.TrimSpace(usage) + "\n\n")
	writeLibraries(&summary, run, rb, r)
	writeConfig(&summary, run)
	writeMethodology(&summary, rb)
	summary.WriteString(strings.TrimSpace(glossary) + "\n")

	var details strings.Builder
	writeDetailsHeader(&details, files)
	writeReadingTables(&details)
	writeFeatureMatrix(&details, run, rb, r)
	writeTables(&details, run, rb, r)
	writeProsCons(&details, rb, r)
	writeFlexibility(&details, rb, r)
	writeEaseOfUse(&details, rb, r)
	writeLearningCurve(&details, rb, r)
	writeCommunity(&details, rb, r)
	if err := writeQualitativeRanking(&details, rb, r); err != nil {
		return Report{}, err
	}
	if err := writeCombinedRanking(&details, rb, r); err != nil {
		return Report{}, err
	}
	return Report{Summary: summary.String(), Details: details.String()}, nil
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

func writeHeader(sb *strings.Builder, run result.Run, files Files) {
	sb.WriteString("# Go SQL library benchmark\n\n")
	sb.WriteString(strings.TrimSpace(disclaimer) + "\n\n")
	fmt.Fprintf(sb, "Run started %s (UTC). Numbers are only comparable on the same machine.\n\n", run.StartedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(sb, "Compares %d libraries on three measures: **latency** (p95, the time within which 95%% of calls finished), "+
		"**memory** (B/op, bytes allocated per operation) and **CPU** (CPU ns/op, CPU time per operation). "+
		"Each library runs %d scenarios under %d load profiles; a **case** is one scenario under one load profile, "+
		"written scenario/profile, for example join/low. Union is measured but not scored (see Union below); "+
		"the other %d scenarios, %d cases, make up every score.\n\n",
		len(libraryNames(run)), len(bench.AllScenarios), len(run.Config.Profiles),
		len(scoredScenarios()), len(scoredScenarios())*len(run.Config.Profiles))
	fmt.Fprintf(sb, "This is the summary. The per-case tables, pros and cons, and the qualitative sections (flexibility, ease of use, learning curve, community, a separate qualitative ranking and a combined ranking) are in the [detailed report](%s).\n\n", url.PathEscape(files.Details))
}

func writeDetailsHeader(sb *strings.Builder, files Files) {
	sb.WriteString("# Go SQL library benchmark: detailed report\n\n")
	fmt.Fprintf(sb, "The ranking, how to run the benchmark, the environment and the methodology are in the [summary](%s); its [glossary](%s#glossary) explains the technical terms. Numbers are only comparable on the machine they were measured on, and they are not final.\n\n", url.PathEscape(files.Summary), url.PathEscape(files.Summary))
}

func writeLibraries(sb *strings.Builder, run result.Run, rb Rubric, r ranking) {
	h := run.Host
	sb.WriteString("## Environment\n\n")
	fmt.Fprintf(sb, "- OS/arch: %s/%s\n- Go: %s\n- CPUs: %d (GOMAXPROCS %d)\n- PostgreSQL: %s\n",
		h.GOOS, h.GOARCH, h.GoVersion, h.NumCPU, h.GOMAXPROCS, h.PostgresVersion)
	if v, ok := h.Libraries[driverModule]; ok {
		fmt.Fprintf(sb, "- Driver shared by every database/sql-based library: %s %s\n", driverModule, v)
	}
	sb.WriteString("\n| Library | Version | Label | Version used | Link |\n|---|---|---|---|---|\n")
	for _, l := range r.libs {
		info, _ := rb.library(l.library)
		fmt.Fprintf(sb, "| %s | %s | %s | %s | <%s> |\n", escapeCell(info.Display),
			dashIfEmpty(h.Libraries[info.Module]), dashIfEmpty(escapeCell(info.Label)), dashIfEmpty(escapeCell(info.VersionUsed)), info.URL)
	}
	sb.WriteString("\n")
}

func writeConfig(sb *strings.Builder, run result.Run) {
	c := run.Config
	sb.WriteString("## Run configuration\n\n")
	fmt.Fprintf(sb, "Repeats per case: %d (the median is reported). Seed: %d.\n\n", c.Repeats, c.Seed)
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
	sb.WriteString("\n")
}

func writeReadingTables(sb *strings.Builder) {
	sb.WriteString("## Reading the tables\n\n")
	sb.WriteString("Each table is one scenario under one load profile (a case); each row is one library. Terms such as p95 and B/op are also explained in the [glossary](README.md#glossary) of the summary.\n\n")
	sb.WriteString("- `Library`: the library measured.\n")
	sb.WriteString("- `p50`, `p95`, `p99`: how long one call took. Sort all the calls from fastest to slowest: p50 is the middle one, p95 is the time 95 out of 100 calls beat, p99 the time 99 out of 100 beat. Lower is better; p95 is the figure used for the score.\n")
	sb.WriteString("- `ops/s`: operations (calls) finished per second across all workers. Higher is better; not part of the score.\n")
	sb.WriteString("- `B/op`: bytes of memory allocated per call. Lower is better.\n")
	sb.WriteString("- `allocs/op`: how many separate memory allocations one call makes. Lower is usually better; not part of the score.\n")
	sb.WriteString("- `CPU ns/op`: processor time spent per call, in nanoseconds (billionths of a second). Lower is better.\n")
	sb.WriteString("- `err %`: the share of calls that failed, as a percentage.\n")
	sb.WriteString("- `p95 vs database/sql`: this library's p95 compared with database/sql's p95 in the same table, as a percentage; `+20.0%` is 20% slower, `-20.0%` is 20% faster.\n")
	fmt.Fprintf(sb, "- `p95 spread` is (max - min) / median of the repeats' p95; `*` marks a spread above %s%%, so read that row as approximate.\n", fixed(spreadWarn*percent, 0))
	fmt.Fprintf(sb, "- `!` after a library means an error rate above %s%%: the case is shown but not ranked.\n", fixed(maxRankedErrorRate*percent, 0))
	sb.WriteString("- `[n]` points to the numbered note under the table.\n")
	sb.WriteString("- `n/a (...)` means the library cannot express the scenario; the case is not ranked.\n\n")
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
	fmt.Fprintf(sb, "Score is the weighted geometric mean, over every ranked case, of each metric's ratio to database/sql (%s). 1.00 means the same as database/sql, 2.00 twice as much, 0.50 half; lower is better. Union is not scored, and a case needs a measured database/sql result to be scored.\n\n", weightsText(rb.Weights))
	sb.WriteString("Libraries with equal scores share a rank.\n\n")
	if len(r.mentions()) > 0 {
		sb.WriteString("Only libraries that are not tied to one database are ranked; the others are under Honorable mention.\n\n")
	}
	top := r.top(topCount)
	if len(top) == 0 {
		sb.WriteString("Nothing could be ranked: every score is a ratio to database/sql, so the run needs measured database/sql cases.\n\n")
	} else {
		writeTopTable(sb, rb, top)
	}
	if rest := r.rest(topCount); len(rest) > 0 {
		sb.WriteString("Also measured:\n\n")
		for _, l := range rest {
			fmt.Fprintf(sb, "- %s: score %s, ranked cases %d/%d\n", displayName(rb, l.library), scoreText(l), l.ranked, l.total)
		}
		sb.WriteString("\n")
	}
}

func writeTopTable(sb *strings.Builder, rb Rubric, top []libScore) {
	sb.WriteString("| Rank | Library | Score | Ranked cases |\n|---:|---|---:|---:|\n")
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
}

// writeHonorableMention lists the libraries measured but kept out of the Top 5. Their score is
// computed like every other score, so it is comparable to the Top 5 scores.
func writeHonorableMention(sb *strings.Builder, rb Rubric, r ranking) {
	mentions := r.mentions()
	if len(mentions) == 0 {
		return
	}
	sb.WriteString("## Honorable mention\n\n")
	sb.WriteString("Measured and shown in every table of the detailed report, but kept out of the ranking above. Score is computed the same way as in the Top 5, so it is comparable; lower is better.\n\n")
	sb.WriteString("| Library | Score | Ranked cases | Why not ranked |\n|---|---:|---:|---|\n")
	for _, l := range mentions {
		info, _ := rb.library(l.library)
		fmt.Fprintf(sb, "| %s | %s | %d/%d | %s |\n", escapeCell(info.Display), scoreText(l), l.ranked, l.total, escapeCell(info.HonorableMention))
	}
	sb.WriteString("\n")
}

func writeProsCons(sb *strings.Builder, rb Rubric, r ranking) {
	sb.WriteString("## Pros and cons\n\n")
	for _, l := range r.libs {
		info, _ := rb.library(l.library)
		fmt.Fprintf(sb, "### %s\n\n<%s>\n\n", info.Display, info.URL)
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
