package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

func writeRun(t *testing.T, dir, name string, p95 int64) string {
	t.Helper()
	m := result.Metrics{Ops: 10, ThroughputOpsPerSec: 1, LatencyP50Ns: p95 / 2, LatencyP95Ns: p95, LatencyP99Ns: p95, ElapsedSeconds: 1}
	run := result.Run{
		SchemaVersion: result.SchemaVersion, StartedAt: time.Now().UTC(),
		Config: result.Config{Seed: 1, Repeats: 1, Profiles: []result.ProfileConfig{{Name: "low", Workers: 1}}},
		Cells:  []result.Cell{{Library: "database_sql", Scenario: "select", Profile: "low", Status: result.StatusOK, Repeats: []result.Metrics{m}, Median: &m}},
	}
	path := filepath.Join(dir, name)
	if err := result.Write(path, run); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	base := writeRun(t, dir, "a.json", 1_000_000)
	near := writeRun(t, dir, "near.json", 1_050_000)
	far := writeRun(t, dir, "far.json", 1_500_000)
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
	}{
		{"within default tolerance", []string{base, near}, exitOK, "5.0%"},
		{"beyond default tolerance", []string{base, far}, exitExceeded, "EXCEEDS"},
		{"custom tolerance allows it", []string{"-tolerance", "0.6", base, far}, exitOK, "50.0%"},
		{"one argument", []string{base}, exitUsage, ""},
		{"missing file", []string{base, filepath.Join(dir, "none.json")}, exitUsage, ""},
		{"nan tolerance", []string{"-tolerance", "NaN", base, near}, exitUsage, ""},
		{"negative tolerance", []string{"-tolerance", "-1", base, near}, exitUsage, ""},
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
