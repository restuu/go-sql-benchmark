// Command benchcmp compares the per-cell p95 latency of two result files.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	exitOK         = 0
	exitExceeded   = 1
	exitUsage      = 2
	defaultTol     = 0.10
	percentDisplay = 100
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run returns the process exit code: 0 within tolerance, 1 beyond it or on a missing cell, 2 on usage or load errors.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("benchcmp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	tolerance := fs.Float64("tolerance", defaultTol, "maximum relative p95 difference per cell")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 2 || !(*tolerance >= 0) { // also rejects NaN
		fmt.Fprintln(stderr, "usage: benchcmp [-tolerance 0.10] a.json b.json (tolerance must be >= 0)")
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

	code := exitOK
	for _, d := range result.CompareP95(a, b, *tolerance) {
		fmt.Fprintln(stdout, format(d))
		if d.Exceeds {
			code = exitExceeded
		}
	}
	return code
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
