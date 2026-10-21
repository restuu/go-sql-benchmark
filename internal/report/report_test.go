package report

import (
	"flag"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

var update = flag.Bool("update", false, "rewrite testdata/report.golden.md")

const (
	fixturePath = "testdata/fixture.json"
	rubricPath  = "testdata/rubric.json"
	goldenPath  = "testdata/report.golden.md"
)

func loadFixture(t *testing.T) (result.Run, Rubric) {
	t.Helper()
	run, err := result.Load(fixturePath)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	rb, err := LoadRubric(rubricPath)
	if err != nil {
		t.Fatalf("load rubric: %v", err)
	}
	return run, rb
}

// withTwoRepeats gives every measured cell two repeats, with the database/sql insert/low cell
// spreading far enough to be starred. The result still passes result.Validate.
func withTwoRepeats(t *testing.T, run result.Run) result.Run {
	t.Helper()
	const spreadFactor = 1.3
	out := run
	out.Config.Repeats = 2
	out.Cells = slices.Clone(run.Cells)
	for i, c := range out.Cells {
		if c.Status != result.StatusOK {
			continue
		}
		second := *c.Median
		if c.Library == "database_sql" && c.Scenario == "insert" && c.Profile == "low" {
			second.LatencyP95Ns = int64(float64(second.LatencyP95Ns) * spreadFactor)
			second.LatencyP99Ns = max(second.LatencyP99Ns, second.LatencyP95Ns)
		}
		out.Cells[i].Repeats = []result.Metrics{*c.Median, second}
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("two-repeat fixture invalid: %v", err)
	}
	return out
}

func without(cells []result.Cell, drop func(result.Cell) bool) []result.Cell {
	var out []result.Cell
	for _, c := range cells {
		if !drop(c) {
			out = append(out, c)
		}
	}
	return out
}

func TestRenderGolden(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)
	run = withTwoRepeats(t, run)

	// Act
	got, err := Render(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if *update {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if got != string(want) {
		t.Fatalf("report differs from %s (run with -update and review the diff)", goldenPath)
	}
}

func TestRenderContent(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)

	// Act
	got, err := Render(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, want := range []string{
		"darwin/arm64", "go1.27.3", "15.19", "pgx v5.11.0", "v0.14.6",
		"no updates since 2024",
		"ent JOIN is edge eager-loading",
		"n/a (unsupported)",
		"Fixture methodology line two with a | pipe.",
		"baseline",
		"judged on 10 of 12 cells",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report does not contain %q", want)
		}
	}
	top, pros := strings.Index(got, "## Top 5"), strings.Index(got, "## Pros and cons")
	if top < 0 || pros < 0 || top > pros {
		t.Errorf("top 5 (%d) must come before pros and cons (%d)", top, pros)
	}
	for _, scenario := range []string{"insert", "update", "delete", "select", "join", "union"} {
		if strings.Count(got, "#### "+scenario+"\n") != 2 {
			t.Errorf("scenario %s should have one table per profile", scenario)
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)
	reversed := run
	reversed.Cells = slices.Clone(run.Cells)
	slices.Reverse(reversed.Cells)

	// Act
	first, err1 := Render(run, rb)
	second, err2 := Render(run, rb)
	fromReversed, err3 := Render(reversed, rb)

	// Assert
	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("Render errors: %v, %v, %v", err1, err2, err3)
	}
	if first != second {
		t.Fatal("two renders of the same run differ")
	}
	if first != fromReversed {
		t.Fatal("render depends on the order of cells")
	}
}

func TestRenderErrors(t *testing.T) {
	run, rb := loadFixture(t)
	tests := []struct {
		name    string
		mutate  func(run *result.Run, rb *Rubric)
		wantErr string
	}{
		{
			name: "missing cell names library, scenario and profile",
			mutate: func(run *result.Run, _ *Rubric) {
				run.Cells = without(run.Cells, func(c result.Cell) bool {
					return c.Library == "ent" && c.Scenario == "join" && c.Profile == "low"
				})
			},
			wantErr: "missing cell ent/join/low",
		},
		{
			name: "library without a rubric entry",
			mutate: func(_ *result.Run, rb *Rubric) {
				rb.Libraries = withoutLibrary(rb.Libraries, "sqlx")
			},
			wantErr: `library "sqlx" has no rubric entry`,
		},
		{
			name: "measured cell without a median",
			mutate: func(run *result.Run, _ *Rubric) {
				for i, c := range run.Cells {
					if c.Status == result.StatusOK {
						run.Cells[i].Median = nil
						return
					}
				}
			},
			wantErr: "has no median",
		},
		{
			name:    "run without cells",
			mutate:  func(run *result.Run, _ *Rubric) { run.Cells = nil },
			wantErr: "no cells",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			r, b := run, rb
			r.Cells = slices.Clone(run.Cells)
			b.Libraries = slices.Clone(rb.Libraries)
			tt.mutate(&r, &b)

			// Act
			_, err := Render(r, b)

			// Assert
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Render() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func withoutLibrary(libs []LibraryInfo, name string) []LibraryInfo {
	return slices.DeleteFunc(libs, func(l LibraryInfo) bool { return l.Name == name })
}

func TestRenderWithoutBaseline(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)
	run.Cells = without(run.Cells, func(c result.Cell) bool { return c.Library == baselineLibrary })

	// Act
	got, err := Render(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if strings.Contains(got, "| baseline |") {
		t.Error("report shows a baseline row although database_sql is not in the run")
	}
	if !strings.Contains(got, "| - | - |\n") {
		t.Error("deltas should be dashes without a baseline")
	}
}

func TestRenderLibraryWithOnlyNACells(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)
	for i, c := range run.Cells {
		if c.Library == "sqlx" {
			run.Cells[i] = naCell("sqlx", c.Scenario, c.Profile)
		}
	}

	// Act
	got, err := Render(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(got, "sqlx: score -") {
		t.Errorf("all-n/a library should print a dash score:\n%s", got)
	}
}

func TestRenderEmptyProsAndCons(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)

	// Act
	got, err := Render(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	const heading = "### ent\n"
	start := strings.Index(got, heading)
	if start < 0 {
		t.Fatalf("no %q section", heading)
	}
	section := got[start+len(heading):]
	if next := strings.Index(section, "### "); next >= 0 {
		section = section[:next]
	}
	if strings.Contains(section, "**Cons**") {
		t.Error("ent has no cons in the rubric; the heading should be omitted")
	}
	if !strings.Contains(section, "**Pros**") {
		t.Error("ent has a pro in the rubric; the heading should be present")
	}
}
