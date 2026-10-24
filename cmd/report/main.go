// Command report renders one benchmark result file and the rubric into a Markdown summary and a
// detailed report.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/restuu/go-sql-benchmark/internal/report"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	exitOK     = 0
	exitRender = 1
	exitUsage  = 2

	defaultRubric = "report/rubric.json"
	outDirMode    = 0o755
	outFileMode   = 0o644
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run returns the process exit code: 0 on success, 1 when the run cannot be rendered (an
// incomplete run or a library without a rubric entry), 2 on usage, load or write errors.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	in := fs.String("in", "", "result JSON file (required)")
	rubricPath := fs.String("rubric", defaultRubric, "rubric JSON file")
	out := fs.String("out", "", "write the summary to this file instead of stdout (requires -details-out)")
	detailsOut := fs.String("details-out", "", "write the detailed report to this file (requires -out)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *in == "" || fs.NArg() != 0 || (*out == "") != (*detailsOut == "") {
		fmt.Fprintln(stderr, "usage: report -in results/run.json [-rubric report/rubric.json] [-out README.md -details-out REPORT.md]")
		return exitUsage
	}
	r, err := result.Load(*in)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	rb, err := report.LoadRubric(*rubricPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	files := report.DefaultFiles
	if *out != "" {
		if err := checkOutputs(*out, *detailsOut); err != nil {
			fmt.Fprintln(stderr, err)
			return exitUsage
		}
		files = report.Files{Summary: filepath.Base(*out), Details: filepath.Base(*detailsOut)}
	}
	doc, err := report.Render(r, rb, files)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRender
	}
	if *out == "" {
		fmt.Fprint(stdout, doc.Summary+"\n"+doc.Details)
		return exitOK
	}
	if err := errors.Join(writeFile(*out, doc.Summary), writeFile(*detailsOut, doc.Details)); err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	return exitOK
}

// checkOutputs requires two different files in one directory, because the documents link to each
// other by file name.
func checkOutputs(summary, details string) error {
	summary, details = filepath.Clean(summary), filepath.Clean(details)
	switch {
	case summary == details:
		return fmt.Errorf("report: -out and -details-out are the same file %s", summary)
	case filepath.Dir(summary) != filepath.Dir(details):
		return fmt.Errorf("report: -out (%s) and -details-out (%s) must be in the same directory so their links work", summary, details)
	}
	return nil
}

// writeFile writes the report, creating parent directories like result.Write.
func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), outDirMode); err != nil {
		return fmt.Errorf("report: create dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), outFileMode); err != nil {
		return fmt.Errorf("report: write %s: %w", path, err)
	}
	return nil
}
