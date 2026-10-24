// Command benchcmp compares two result files: per-cell p95 latency and the top-ranked libraries.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/report"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	exitOK         = 0
	exitExceeded   = 1
	exitUsage      = 2
	defaultTol     = 0.10
	defaultLowTol  = 0.60
	defaultGap     = 0.02
	defaultRubric  = "report/rubric.json"
	percentDisplay = 100
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run returns the process exit code: 0 within tolerance and the same top 5, 1 beyond tolerance, on a missing
// cell or when the top 5 differ (see sameTop), 2 on usage or load errors. The top 5 is scored without
// union; the per-case p95 comparison still covers every case.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("benchcmp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	tolerance := fs.Float64("tolerance", defaultTol, "maximum relative p95 difference per cell")
	toleranceLow := fs.Float64("tolerance-low", defaultLowTol, "same, for the low profile: its calls are paced and a few ms long, so its p95 is noisier")
	scoreGap := fs.Float64("score-gap", defaultGap, "scores closer than this are tied: their order may differ between the runs")
	rubricPath := fs.String("rubric", defaultRubric, "rubric file whose weights rank the libraries")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 2 || !(*tolerance >= 0) || !(*toleranceLow >= 0) || !(*scoreGap >= 0) { // also rejects NaN
		fmt.Fprintln(stderr, "usage: benchcmp [-tolerance 0.10] [-tolerance-low 0.60] [-score-gap 0.02] [-rubric report/rubric.json] a.json b.json (tolerances and score-gap must be >= 0)")
		return exitUsage
	}
	a, err := result.Load(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	b, err := result.Load(fs.Arg(1))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}

	rb, err := report.LoadRubric(*rubricPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}

	code := exitOK
	for _, d := range result.CompareP95(a, b, func(profile string) float64 {
		if profile == bench.ProfileLow {
			return *toleranceLow
		}
		return *tolerance
	}) {
		fmt.Fprintln(stdout, format(d))
		if d.Exceeds {
			code = exitExceeded
		}
	}

	topA, topB := report.TopScores(a, rb), report.TopScores(b, rb)
	switch {
	case len(topA) == 0 && len(topB) == 0: // an empty ranking proves nothing about stability
		fmt.Fprintln(stdout, "NO RANKED LIBRARIES in either run")
		return exitExceeded
	case sameTop(topA, topB, *scoreGap):
		fmt.Fprintf(stdout, "top ranking unchanged: %v\n", names(topA))
		return code
	default:
		fmt.Fprintf(stdout, "TOP RANKING DIFFERS: %v vs %v\n", names(topA), names(topB))
		return exitExceeded
	}
}

// sameTop reports whether two top lists hold the same libraries and agree on the order of every
// pair whose scores are more than gap apart in either run. Two libraries closer than that in both
// runs are tied, so swapping them is noise rather than a change.
func sameTop(a, b []report.ScoredLibrary, gap float64) bool {
	if len(a) != len(b) {
		return false
	}
	posB := make(map[string]int, len(b))
	for i, l := range b {
		posB[l.Name] = i
	}
	for i, x := range a {
		if _, ok := posB[x.Name]; !ok {
			return false
		}
		for _, y := range a[i+1:] {
			if _, ok := posB[y.Name]; !ok {
				return false
			}
			flipped := posB[y.Name] < posB[x.Name]
			outsideGap := y.Score-x.Score > gap || b[posB[x.Name]].Score-b[posB[y.Name]].Score > gap
			if flipped && outsideGap {
				return false
			}
		}
	}
	return true
}

func names(libs []report.ScoredLibrary) []string {
	out := make([]string, len(libs))
	for i, l := range libs {
		out[i] = l.Name
	}
	return out
}

func format(d result.Deviation) string {
	head := fmt.Sprintf("%-14s %-8s %-7s", d.Library, d.Scenario, d.Profile)
	if d.Note != "" {
		return fmt.Sprintf("%s %s  EXCEEDS", head, d.Note)
	}
	line := fmt.Sprintf("%s %12v %12v %7.1f%%", head, time.Duration(d.P95A), time.Duration(d.P95B), d.Delta*percentDisplay)
	if d.Exceeds {
		line += "  EXCEEDS"
	}
	return line
}
