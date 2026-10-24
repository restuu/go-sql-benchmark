package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

// writeRun writes a run with one "select"/"low" cell per library; libraries[i] gets p95 = p95 * (i+1)
// (p95 alone when no library is named, using "database_sql").
func writeRun(t *testing.T, dir, name string, p95 int64, libraries ...string) string {
	t.Helper()
	if len(libraries) == 0 {
		libraries = []string{"database_sql"}
	}
	var cells []result.Cell
	for i, lib := range libraries {
		m := result.Metrics{Ops: 10, ThroughputOpsPerSec: 1, LatencyP50Ns: p95 / 2, LatencyP95Ns: p95 * int64(i+1), LatencyP99Ns: p95 * int64(i+1), ElapsedSeconds: 1}
		cells = append(cells, result.Cell{Library: lib, Scenario: "select", Profile: "low", Status: result.StatusOK, Repeats: []result.Metrics{m}, Median: &m})
	}
	run := result.Run{
		SchemaVersion: result.SchemaVersion, StartedAt: time.Now().UTC(),
		Config: result.Config{Seed: 1, Repeats: 1, Profiles: []result.ProfileConfig{{Name: "low", Workers: 1}}},
		Cells:  cells,
	}
	path := filepath.Join(dir, name)
	if err := result.Write(path, run); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func writeRubric(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "rubric.json")
	if err := os.WriteFile(path, []byte(`{"weights":{"p95":1,"bytes_per_op":1,"cpu_ns_per_op":1},"community_checked":"2026-10-24"}`), 0o600); err != nil {
		t.Fatalf("write rubric: %v", err)
	}
	return path
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	rubric := writeRubric(t, dir)
	base := writeRun(t, dir, "a.json", 1_000_000)
	near := writeRun(t, dir, "near.json", 1_050_000)
	far := writeRun(t, dir, "far.json", 1_500_000)
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
	}{
		{"within default tolerance", []string{"-rubric", rubric, base, near}, exitOK, "5.0%"},
		{"beyond default tolerance", []string{"-rubric", rubric, base, far}, exitExceeded, "EXCEEDS"},
		{"custom tolerance allows it", []string{"-rubric", rubric, "-tolerance", "0.6", base, far}, exitOK, "50.0%"},
		{"one argument", []string{"-rubric", rubric, base}, exitUsage, ""},
		{"missing file", []string{"-rubric", rubric, base, filepath.Join(dir, "none.json")}, exitUsage, ""},
		{"nan tolerance", []string{"-rubric", rubric, "-tolerance", "NaN", base, near}, exitUsage, ""},
		{"missing rubric", []string{"-rubric", filepath.Join(dir, "none.json"), base, near}, exitUsage, ""},
		{"malformed rubric", []string{"-rubric", writeFile(t, dir, "bad.json", "{"), base, near}, exitUsage, ""},
		{"negative tolerance", []string{"-rubric", rubric, "-tolerance", "-1", base, near}, exitUsage, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var out, errOut bytes.Buffer

			// Act
			code := run(tt.args, &out, &errOut)

			// Assert
			if code != tt.wantCode {
				t.Fatalf("run() = %d, want %d (stdout %q, stderr %q)", code, tt.wantCode, out.String(), errOut.String())
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Fatalf("stdout %q does not contain %q", out.String(), tt.wantOut)
			}
			if tt.wantCode == exitUsage && errOut.Len() == 0 {
				t.Fatal("usage error printed nothing to stderr")
			}
		})
	}
}

func TestRunTopFive(t *testing.T) {
	dir := t.TempDir()
	rubric := writeRubric(t, dir)
	seven := []string{"a", "b", "c", "d", "e", "f", "g"}
	reorderedTail := []string{"a", "b", "c", "d", "e", "g", "f"} // only ranks 6 and 7 swap
	swappedTop := []string{"b", "a", "c", "d", "e", "f", "g"}    // ranks 1 and 2 swap
	newTop := []string{"a", "b", "c", "d", "f", "e", "g"}        // rank 5 changes
	tests := []struct {
		name     string
		other    []string
		wantCode int
		wantOut  string
	}{
		{"same order", seven, exitOK, "top ranking unchanged"},
		{"ranks beyond five may differ", reorderedTail, exitOK, "top ranking unchanged"},
		{"order within top five differs", swappedTop, exitExceeded, "TOP RANKING DIFFERS"},
		{"membership of top five differs", newTop, exitExceeded, "TOP RANKING DIFFERS"},
		{"fewer libraries ranked in the second run", seven[:4], exitExceeded, "TOP RANKING DIFFERS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			a := writeRun(t, dir, "a-"+tt.name+".json", 1_000_000, seven...)
			b := writeRun(t, dir, "b-"+tt.name+".json", 1_000_000, tt.other...)
			var out, errOut bytes.Buffer

			// Act
			code := run([]string{"-rubric", rubric, "-tolerance", "1000", a, b}, &out, &errOut)

			// Assert
			if code != tt.wantCode {
				t.Fatalf("run() = %d, want %d (stdout %q, stderr %q)", code, tt.wantCode, out.String(), errOut.String())
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Fatalf("stdout %q does not contain %q", out.String(), tt.wantOut)
			}
		})
	}
}

func TestRunNothingRanked(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.json")
	naRun := result.Run{
		SchemaVersion: result.SchemaVersion, StartedAt: time.Now().UTC(),
		Config: result.Config{Seed: 1, Repeats: 1, Profiles: []result.ProfileConfig{{Name: "low", Workers: 1}}},
		Cells:  []result.Cell{{Library: "database_sql", Scenario: "select", Profile: "low", Status: result.StatusNA, Reason: "n/a"}},
	}
	if err := result.Write(empty, naRun); err != nil {
		t.Fatalf("write: %v", err)
	}
	var out, errOut bytes.Buffer

	// Act
	code := run([]string{"-rubric", writeRubric(t, dir), empty, empty}, &out, &errOut)

	// Assert
	if code != exitExceeded || !strings.Contains(out.String(), "NO RANKED LIBRARIES") {
		t.Fatalf("run() = %d, stdout %q; want exit %d and NO RANKED LIBRARIES", code, out.String(), exitExceeded)
	}
}
