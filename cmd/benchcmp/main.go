// Command benchcmp compares two result files: per-cell p95 latency and the top-ranked libraries.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/report"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	exitOK         = 0
	exitExceeded   = 1
	exitUsage      = 2
	defaultTol     = 0.10
	defaultRubric  = "report/rubric.json"
	percentDisplay = 100
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run returns the process exit code: 0 within tolerance and the same top 5, 1 beyond tolerance, on a missing
// cell or when the top 5 (membership or order) differ, 2 on usage or load errors. The top 5 is
// scored without union; the per-case p95 comparison still covers every case.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("benchcmp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	tolerance := fs.Float64("tolerance", defaultTol, "maximum relative p95 difference per cell")
	rubricPath := fs.String("rubric", defaultRubric, "rubric file whose weights rank the libraries")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 2 || !(*tolerance >= 0) { // also rejects NaN
		fmt.Fprintln(stderr, "usage: benchcmp [-tolerance 0.10] [-rubric report/rubric.json] a.json b.json (tolerance must be >= 0)")
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
	for _, d := range result.CompareP95(a, b, *tolerance) {
		fmt.Fprintln(stdout, format(d))
		if d.Exceeds {
			code = exitExceeded
		}
	}

	topA, topB := report.TopLibraries(a, rb), report.TopLibraries(b, rb)
	switch {
	case len(topA) == 0 && len(topB) == 0: // an empty ranking proves nothing about stability
		fmt.Fprintln(stdout, "NO RANKED LIBRARIES in either run")
		return exitExceeded
	case slices.Equal(topA, topB):
		fmt.Fprintf(stdout, "top ranking unchanged: %v\n", topA)
		return code
	default:
		fmt.Fprintf(stdout, "TOP RANKING DIFFERS: %v vs %v\n", topA, topB)
		return exitExceeded
	}
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
