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
		"judged on 10 of 12 cases",
		"**latency** (p95",
		"**memory** (B/op",
		"**CPU** (CPU ns/op",
		"a **case** is one scenario under one load profile, written scenario/profile, for example join/low",
		"3 libraries",
		"weighted geometric mean, over every ranked case, of each metric's ratio to database/sql",
		"ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report does not contain %q", want)
		}
	}
	if strings.Contains(strings.ToLower(got), " cell") {
		t.Errorf("report text must say case, not cell")
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
	if !strings.Contains(got, "Nothing could be ranked") {
		t.Error("a run without database_sql has no ratios, the Top 5 should say so")
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

func TestRenderFlexibility(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)

	// Act
	got, err := Render(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	start := strings.Index(got, "## Flexibility")
	if start < 0 {
		t.Fatalf("no Flexibility section:\n%s", got)
	}
	section := got[start:]
	for _, want := range []string{
		"qualitative",
		"not part of the score",
		"| Library | optional filters | dynamic ordering | conditional join | dynamic projection | in-list / batch | dynamic pagination |",
		"| ent | native | native | native | partial | native | native |",
		"| sqlx | raw SQL | raw SQL | raw SQL | raw SQL | native | static only |",
		"Dynamic SQL is built by hand with a | pipe.",
		"unverified: named queries only.",
	} {
		if !strings.Contains(section, want) {
			t.Errorf("Flexibility section missing %q:\n%s", want, section)
		}
	}
	if strings.Contains(got[:start], "unverified: named queries only.") {
		t.Error("flexibility notes leaked outside the Flexibility section")
	}
}

func TestRenderFlexibilityDoesNotChangeRanking(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)
	flipped := rb
	flipped.Libraries = slices.Clone(rb.Libraries)
	for i, l := range flipped.Libraries {
		caps := make(map[string]string, len(l.Flexibility.Capabilities))
		for name := range l.Flexibility.Capabilities {
			caps[name] = "static_only"
		}
		flipped.Libraries[i].Flexibility = Flexibility{Capabilities: caps, Notes: l.Flexibility.Notes}
	}

	// Act
	before, err := Render(run, rb)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Render(run, flipped)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	cut := func(s string) string { return s[:strings.Index(s, "## Flexibility")] }
	if !strings.Contains(before, "## Flexibility") || !strings.Contains(after, "## Flexibility") {
		t.Fatal("report has no Flexibility section")
	}
	if cut(before) != cut(after) {
		t.Error("changing flexibility levels changed the report outside the Flexibility section")
	}
}

// section returns the text from heading to the next "## " heading or the end.
func section(t *testing.T, doc, heading string) string {
	t.Helper()
	start := strings.Index(doc, heading)
	if start < 0 {
		t.Fatalf("no %q section:\n%s", heading, doc)
	}
	body := doc[start+len(heading):]
	if next := strings.Index(body, "\n## "); next >= 0 {
		body = body[:next]
	}
	return body
}

func TestRenderQualitativeSections(t *testing.T) {
	tests := []struct {
		name    string
		heading string
		want    []string
		leakKey string
	}{
		{"ease of use", "## Ease of use (qualitative)", []string{
			"not part of the score",
			"Levels for debugging: `good` = the SQL that ran",
			"| Library | setup effort | query writing | type safety | debugging |",
			"| ent | poor | good | good | fair |",
			"Ease note for sqlx with a | pipe.",
		}, "Ease note for"},
		{"learning curve", "## Learning curve (qualitative)", []string{
			"| Library | Learning curve |",
			"| ent | steep |",
			"| database/sql | gentle |",
			"Curve note for ent.",
		}, "Curve note for"},
		{"community", "## Community support (facts)", []string{
			"read on 2026-10-24",
			"| Library | Repository | Stars | Last release | Last commit | Open issues |",
			"| ent | ent/ent | 17000 | 2026-08-01 | 2026-10-20 | 400 |",
			"- **database/sql**: Counts cover the whole Go project.",
		}, "ent/ent"},
	}
	run, rb := loadFixture(t)
	got, err := Render(run, rb)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (got was rendered once above)

			// Act
			body := section(t, got, tt.heading)
			start := strings.Index(got, tt.heading)

			// Assert
			for _, want := range tt.want {
				if !strings.Contains(body, want) {
					t.Errorf("%s section missing %q:\n%s", tt.name, want, body)
				}
			}
			if strings.Contains(got[:start], tt.leakKey) {
				t.Errorf("%s content leaked before its section", tt.name)
			}
		})
	}
}

func TestRenderQualitativeDoesNotChangeRanking(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)
	changed := rb
	changed.Libraries = slices.Clone(rb.Libraries)
	for i, l := range changed.Libraries {
		criteria := make(map[string]string, len(l.EaseOfUse.Criteria))
		for name := range l.EaseOfUse.Criteria {
			criteria[name] = "poor"
		}
		changed.Libraries[i].EaseOfUse = EaseOfUse{Criteria: criteria, Notes: l.EaseOfUse.Notes}
		changed.Libraries[i].LearningCurve.Level = "steep"
		changed.Libraries[i].Community.Stars = 1
	}

	// Act
	before, err := Render(run, rb)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Render(run, changed)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(before, "## Ease of use") || !strings.Contains(after, "## Ease of use") {
		t.Fatal("report has no Ease of use section")
	}
	cut := func(s string) string { return s[:strings.Index(s, "## Ease of use")] }
	if cut(before) != cut(after) {
		t.Error("changing qualitative ratings changed the report before the Ease of use section")
	}
}

func TestRenderQualitativeSubsetRun(t *testing.T) {
	// Arrange: a rubric entry whose library is absent from the run must not be printed.
	run, rb := loadFixture(t)
	var cells []result.Cell
	for _, c := range run.Cells {
		if c.Library != "sqlx" {
			cells = append(cells, c)
		}
	}
	run.Cells = cells

	// Act
	got, err := Render(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, heading := range []string{"## Ease of use (qualitative)", "## Learning curve (qualitative)", "## Community support (facts)"} {
		body := section(t, got, heading)
		if strings.Contains(body, "sqlx") {
			t.Errorf("%s lists sqlx although it is not in the run", heading)
		}
		if !strings.Contains(body, "ent") {
			t.Errorf("%s lost ent, which is in the run", heading)
		}
	}
}

func TestRenderHonorableMention(t *testing.T) {
	// Arrange: sqlx moves out of the ranking pool
	run, rb := loadFixture(t)
	rb.Libraries = slices.Clone(rb.Libraries)
	for i := range rb.Libraries {
		if rb.Libraries[i].Name == "sqlx" {
			rb.Libraries[i].HonorableMention = "Fixture reason: single-database driver."
		}
	}

	// Act
	got, err := Render(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	top, mention, pros := strings.Index(got, "## Top 5"), strings.Index(got, "## Honorable mention"), strings.Index(got, "## Pros and cons")
	if top < 0 || mention < top || pros < mention {
		t.Fatalf("want Top 5 (%d) < Honorable mention (%d) < Pros and cons (%d)", top, mention, pros)
	}
	topSection, mentionSection := got[top:mention], got[mention:pros]
	if strings.Contains(topSection, "| sqlx |") || strings.Contains(topSection, "**sqlx**") {
		t.Errorf("sqlx must not appear in the Top 5 section:\n%s", topSection)
	}
	if !strings.Contains(mentionSection, "| sqlx |") || !strings.Contains(mentionSection, "Fixture reason: single-database driver.") {
		t.Errorf("honorable mention section lacks sqlx or its reason:\n%s", mentionSection)
	}
	if !strings.Contains(topSection, "| 1 | database/sql |") || !strings.Contains(topSection, "| 2 | ent |") {
		t.Errorf("remaining libraries should rank 1 and 2:\n%s", topSection)
	}
	if strings.Count(got, "| sqlx |") < 2 { // environment table and result tables still list it
		t.Errorf("sqlx must stay in the measured tables")
	}
}

func TestRenderWithoutHonorableMention(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)

	// Act
	got, err := Render(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if strings.Contains(got, "Honorable mention") {
		t.Errorf("no library is a mention, the section must not render")
	}
}
