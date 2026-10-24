package report

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

const repoRoot = "../.."

func TestRenderEscapesFileNamesInLinks(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)

	// Act
	doc, err := Render(run, rb, Files{Summary: "my summary.md", Details: "more (1).md"})

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(doc.Summary, "(more%20%281%29.md)") || !strings.Contains(doc.Details, "(my%20summary.md)") {
		t.Error("links must percent-escape spaces and parentheses in file names")
	}
}

func TestRenderSummaryAndDetailsSplit(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)

	// Act
	doc, err := Render(run, rb, Files{Summary: "S.md", Details: "D.md"})

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	s := doc.Summary
	order := []string{"# Go SQL library benchmark\n", "These results are not final", "Run started", "## How to read the numbers", "## Top 5", "## Run it yourself", "### Add your own library", "### Using an AI coding assistant", "## Environment", "## Run configuration", "## Methodology", "## Glossary"}
	last := -1
	for _, marker := range order {
		at := strings.Index(s, marker)
		if at <= last {
			t.Fatalf("summary: %q at %d does not follow the previous marker (%d)", marker, at, last)
		}
		last = at
	}
	if !strings.Contains(s, "(D.md)") || !strings.Contains(doc.Details, "(S.md)") {
		t.Error("summary and details must link to each other by the given file names")
	}
	for _, heading := range []string{"## Results", "## Pros and cons", "## Flexibility", "## Ease of use", "## Learning curve", "## Community support", "## Qualitative ranking", "## Reading the tables"} {
		if strings.Contains(s, heading) {
			t.Errorf("summary holds %q, which belongs in the details", heading)
		}
		if !strings.Contains(doc.Details, heading) {
			t.Errorf("details lack %q", heading)
		}
	}
	for _, heading := range []string{"## Top 5", "## Methodology", "## Run it yourself", "## Environment"} {
		if strings.Contains(doc.Details, heading) {
			t.Errorf("details hold %q, which belongs in the summary", heading)
		}
	}
	if !strings.HasPrefix(doc.Details, "# Go SQL library benchmark: detailed report") {
		t.Error("details should start with their own title")
	}
}

// TestUsageNamesRealMakeTargetsAndPaths keeps the instructions from drifting: every `make` target
// and every repository path they mention must exist.
func TestUsageNamesRealMakeTargetsAndPaths(t *testing.T) {
	// Arrange
	makefile, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	makeTarget := regexp.MustCompile("(?m)(?:^|`)make ([a-z][a-z-]*)")
	repoPath := regexp.MustCompile("`((?:internal|cmd|report|db)/[A-Za-z0-9_./<>-]+|Makefile|LICENSE)`")

	// Act
	targets := makeTarget.FindAllStringSubmatch(usage, -1)
	paths := repoPath.FindAllStringSubmatch(usage, -1)

	// Assert
	if len(targets) == 0 || len(paths) == 0 {
		t.Fatalf("found %d make targets and %d paths in the usage text, want some of each", len(targets), len(paths))
	}
	for _, m := range targets {
		if !regexp.MustCompile("(?m)^" + regexp.QuoteMeta(m[1]) + ":").Match(makefile) {
			t.Errorf("usage mentions `make %s`, which the Makefile does not define", m[1])
		}
	}
	for _, m := range paths {
		path := m[1]
		if strings.ContainsAny(path, "<>") { // a placeholder such as internal/adapters/<name>/
			path = path[:strings.IndexAny(path, "<>")]
		}
		if _, err := os.Stat(filepath.Join(repoRoot, path)); err != nil {
			t.Errorf("usage mentions %q, which does not exist: %v", m[1], err)
		}
	}
}

func TestUsageNamesRealBenchFlags(t *testing.T) {
	// Arrange
	flagsSource, err := os.ReadFile(filepath.Join(repoRoot, "cmd/bench/flags.go"))
	if err != nil {
		t.Fatal(err)
	}
	flagName := regexp.MustCompile(`BENCH_ARGS="([^"]*)"`)

	// Act
	var used []string
	for _, m := range flagName.FindAllStringSubmatch(usage, -1) {
		for _, field := range strings.Fields(m[1]) {
			if strings.HasPrefix(field, "-") {
				used = append(used, strings.TrimPrefix(field, "-"))
			}
		}
	}

	// Assert
	if len(used) == 0 {
		t.Fatal("found no bench flags in the usage text")
	}
	for _, name := range used {
		if !strings.Contains(string(flagsSource), `"`+name+`"`) {
			t.Errorf("usage passes -%s, which cmd/bench/flags.go does not define", name)
		}
	}
}

func TestGlossaryExplainsTheTechnicalTerms(t *testing.T) {
	// Arrange
	terms := []string{
		"p50", "p95", "p99", "ops/s", "err %", "p95 vs database/sql", "p95 spread", "percentile", "latency", "throughput", "B/op", "allocs/op", "CPU ns/op", "garbage collector",
		"ratio", "geometric mean", "score", "median", "spread", "error rate", "baseline", "n/a",
		"scenario", "load profile", "low", "stress", "worker", "warm-up", "measured window", "case", "connection pool",
		"database/sql", "driver", "ORM", "query builder", "SQL-first", "code generation", "honorable mention", "PostgreSQL",
		"dynamic query", "raw SQL", "type safety",
	}
	for _, scenario := range bench.AllScenarios {
		terms = append(terms, string(scenario))
	}
	run, rb := loadFixture(t)

	// Act
	doc, err := Render(run, rb, DefaultFiles)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	glossary := doc.Summary[strings.Index(doc.Summary, "## Glossary"):]
	for _, term := range terms {
		if !strings.Contains(glossary, "**"+term+"**") {
			t.Errorf("glossary lacks an entry for %q", term)
		}
	}
	reading := section(t, doc.Summary, "## How to read the numbers")
	for _, want := range []string{"p95", "B/op", "CPU ns/op", "1.04x", "4% longer", "43% more memory", "(#glossary)"} {
		if !strings.Contains(reading, want) {
			t.Errorf("How to read the numbers lacks %q", want)
		}
	}
	if !strings.Contains(doc.Details, "(README.md#glossary)") {
		t.Error("details should link to the summary's glossary")
	}
}

func TestReadingTheTablesExplainsEveryColumn(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)
	header := strings.Split(strings.SplitN(tableHeader, "\n", 2)[0], "|")

	// Act
	doc, err := Render(run, rb, DefaultFiles)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	body := section(t, doc.Details, "## Reading the tables")
	for _, cell := range header {
		column := strings.TrimSpace(cell)
		if column == "" {
			continue
		}
		if !strings.Contains(body, "`"+column+"`") {
			t.Errorf("Reading the tables does not explain the column %q", column)
		}
	}
}
