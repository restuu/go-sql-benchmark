package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/report"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

const (
	fixtureRun    = "../../internal/report/testdata/fixture.json"
	fixtureRubric = "../../internal/report/testdata/rubric.json"
	realRubric    = "../../report/rubric.json"
	reportTitle   = "# Go SQL library benchmark"
)

// writeIncompleteRun writes a valid run holding one cell, so rendering it fails the completeness check.
func writeIncompleteRun(t *testing.T, dir string) string {
	t.Helper()
	m := result.Metrics{Ops: 10, ThroughputOpsPerSec: 1, LatencyP50Ns: 1, LatencyP95Ns: 2, LatencyP99Ns: 3, ElapsedSeconds: 1}
	run := result.Run{
		SchemaVersion: result.SchemaVersion, StartedAt: time.Now().UTC(),
		Config: result.Config{Seed: 1, Repeats: 1, Profiles: []result.ProfileConfig{{Name: "low", Workers: 1}}},
		Cells:  []result.Cell{{Library: "database_sql", Scenario: "select", Profile: "low", Status: result.StatusOK, Repeats: []result.Metrics{m}, Median: &m}},
	}
	path := filepath.Join(dir, "incomplete.json")
	if err := result.Write(path, run); err != nil {
		t.Fatalf("write run: %v", err)
	}
	return path
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	incomplete := writeIncompleteRun(t, dir)
	badRubric := filepath.Join(dir, "bad-rubric.json")
	if err := os.WriteFile(badRubric, []byte(`{"weights": {"p95": 1}, "typo": 1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	notJSON := filepath.Join(dir, "run.json")
	if err := os.WriteFile(notJSON, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
	}{
		{"renders to stdout", []string{"-in", fixtureRun, "-rubric", fixtureRubric}, exitOK, reportTitle},
		{"missing -in", []string{"-rubric", fixtureRubric}, exitUsage, ""},
		{"unknown flag", []string{"-nope"}, exitUsage, ""},
		{"stray argument", []string{"-in", fixtureRun, "-rubric", fixtureRubric, "extra"}, exitUsage, ""},
		{"missing run file", []string{"-in", filepath.Join(dir, "none.json"), "-rubric", fixtureRubric}, exitUsage, ""},
		{"run file is not json", []string{"-in", notJSON, "-rubric", fixtureRubric}, exitUsage, ""},
		{"missing rubric file", []string{"-in", fixtureRun, "-rubric", filepath.Join(dir, "none.json")}, exitUsage, ""},
		{"rubric with an unknown field", []string{"-in", fixtureRun, "-rubric", badRubric}, exitUsage, ""},
		{"-out without -details-out", []string{"-in", fixtureRun, "-rubric", fixtureRubric, "-out", filepath.Join(dir, "s.md")}, exitUsage, ""},
		{"-details-out without -out", []string{"-in", fixtureRun, "-rubric", fixtureRubric, "-details-out", filepath.Join(dir, "d.md")}, exitUsage, ""},
		{"-out and -details-out are the same file", []string{"-in", fixtureRun, "-rubric", fixtureRubric, "-out", filepath.Join(dir, "same.md"), "-details-out", filepath.Join(dir, ".", "same.md")}, exitUsage, ""},
		{"-out and -details-out are in different directories", []string{"-in", fixtureRun, "-rubric", fixtureRubric, "-out", filepath.Join(dir, "a", "s.md"), "-details-out", filepath.Join(dir, "b", "d.md")}, exitUsage, ""},
		{"incomplete run", []string{"-in", incomplete, "-rubric", fixtureRubric}, exitRender, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var out, errOut bytes.Buffer

			// Act
			code := run(tt.args, &out, &errOut)

			// Assert
			if code != tt.wantCode {
				t.Fatalf("run() = %d, want %d (stderr %q)", code, tt.wantCode, errOut.String())
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Fatalf("stdout does not contain %q", tt.wantOut)
			}
			if tt.wantCode != exitOK && errOut.Len() == 0 {
				t.Fatal("failure printed nothing to stderr")
			}
			if tt.wantCode != exitOK && out.Len() != 0 {
				t.Fatalf("failure wrote to stdout: %q", out.String())
			}
		})
	}
}

func TestRunWritesBothFilesAndCreatesParents(t *testing.T) {
	// Arrange
	dir := filepath.Join(t.TempDir(), "missing", "dir")
	summary, details := filepath.Join(dir, "sum.md"), filepath.Join(dir, "more.md")
	var out, errOut bytes.Buffer

	// Act
	code := run([]string{"-in", fixtureRun, "-rubric", fixtureRubric, "-out", summary, "-details-out", details}, &out, &errOut)

	// Assert
	if code != exitOK {
		t.Fatalf("run() = %d, stderr %q", code, errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout should be empty with -out, got %d bytes", out.Len())
	}
	sumData, sumErr := os.ReadFile(summary)
	detData, detErr := os.ReadFile(details)
	if sumErr != nil || detErr != nil {
		t.Fatalf("read outputs: %v, %v", sumErr, detErr)
	}
	if !strings.HasPrefix(string(sumData), reportTitle) || !strings.Contains(string(sumData), "(more.md)") {
		t.Fatalf("summary does not start with the title or link to more.md: %q", string(sumData)[:min(60, len(sumData))])
	}
	if !strings.Contains(string(detData), "(sum.md)") || strings.Contains(string(detData), "## Top 5") {
		t.Fatalf("details should link back to sum.md and hold no ranking")
	}
}

func TestRunOutIntoUnwritablePathFails(t *testing.T) {
	// Arrange
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer

	// Act
	code := run([]string{"-in", fixtureRun, "-rubric", fixtureRubric, "-out", filepath.Join(blocker, "sub", "r.md"), "-details-out", filepath.Join(blocker, "sub", "d.md")}, &out, &errOut)

	// Assert
	if code != exitUsage || errOut.Len() == 0 {
		t.Fatalf("run() = %d, stderr %q, want usage error", code, errOut.String())
	}
}

// TestRealRubric checks the committed rubric against the libraries this repo benchmarks. The list
// mirrors cmd/bench registry (package main of another command, so not importable): update both
// together when a library is added.
func TestRealRubric(t *testing.T) {
	registry := []string{"beego", "bun", "database_sql", "ent", "gorm", "gorm_gen", "jet", "pgx", "pgx_pool", "pop", "rel", "sqlboiler", "sqlc", "sqlx", "xorm"}

	// Arrange
	rb, err := report.LoadRubric(realRubric)
	if err != nil {
		t.Fatalf("LoadRubric(%s): %v", realRubric, err)
	}
	names := make([]string, len(rb.Libraries))
	labels := make(map[string]string, len(rb.Libraries))
	for i, l := range rb.Libraries {
		names[i] = l.Name
		labels[l.Name] = l.Label
	}
	hasNote := func(library, scenario, contains string) bool {
		return slices.ContainsFunc(rb.CellNotes, func(n report.CellNote) bool {
			return n.Library == library && n.Scenario == scenario && strings.Contains(n.Note, contains)
		})
	}

	// Act
	slices.Sort(names)

	// Assert
	if !slices.Equal(names, registry) {
		t.Fatalf("rubric libraries = %v, want the registry %v", names, registry)
	}
	if labels["sqlx"] != "no updates since 2024" {
		t.Errorf("sqlx label = %q", labels["sqlx"])
	}
	for _, want := range []struct{ library, scenario, contains string }{
		{"ent", "join", "eager"},
		{"ent", "union", "raw SQL"},
		{"gorm", "union", "raw SQL"},
		{"gorm", "insert", "SkipDefaultTransaction"},
		{"gorm", "update", "explicit column"},
	} {
		if !hasNote(want.library, want.scenario, want.contains) {
			t.Errorf("missing cell note %s/%s containing %q", want.library, want.scenario, want.contains)
		}
	}
	methodology := strings.Join(rb.Methodology, "\n")
	for _, want := range []string{"pgx", "allocs/op", "coordinated", "machine"} {
		if !strings.Contains(methodology, want) {
			t.Errorf("methodology does not mention %q", want)
		}
	}
}
