// Command report renders one benchmark result file and the rubric into a Markdown report.
package main

import (
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
	out := fs.String("out", "", "write the report to this file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *in == "" || fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: report -in results/run.json [-rubric report/rubric.json] [-out report.md]")
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
	md, err := report.Render(r, rb)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRender
	}
	if *out == "" {
		fmt.Fprint(stdout, md)
		return exitOK
	}
	if err := writeFile(*out, md); err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	return exitOK
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
