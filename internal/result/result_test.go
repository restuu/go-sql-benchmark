package result

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func validMetrics(p95 int64) Metrics {
	return Metrics{
		Ops: 1000, Errors: 0, ErrorRate: 0, ThroughputOpsPerSec: 200,
		LatencyP50Ns: p95 / 2, LatencyP95Ns: p95, LatencyP99Ns: p95 * 2,
		BytesPerOp: 512, AllocsPerOp: 9, CPUNsPerOp: 1500, ElapsedSeconds: 5,
	}
}

func validRun() Run {
	m := validMetrics(1_000_000)
	return Run{
		SchemaVersion: SchemaVersion,
		StartedAt:     time.Date(2026, 10, 21, 3, 0, 0, 0, time.UTC),
		Host:          Host{GOOS: "darwin", GOARCH: "arm64", GoVersion: "go1.27.3", NumCPU: 8, GOMAXPROCS: 8, PostgresVersion: "15.13", Libraries: map[string]string{"pgx": "v5.11.0"}},
		Config: Config{
			Seed: 42, Repeats: 2,
			Profiles: []ProfileConfig{{Name: "low", Workers: 4, IntervalNs: 10_000_000, WarmupNs: 1e9, DurationNs: 5e9, DeletePool: 5000}},
		},
		Cells: []Cell{
			{Library: "database_sql", Scenario: "insert", Profile: "low", Status: StatusOK, Repeats: []Metrics{m, m}, Median: &m},
			{Library: "database_sql", Scenario: "union", Profile: "low", Status: StatusNA, Reason: "n/a (unsupported)"},
		},
	}
}

func TestValidate(t *testing.T) {
	nan := math.NaN()
	tests := []struct {
		name    string
		mutate  func(*Run)
		wantErr string
	}{
		{"valid", func(*Run) {}, ""},
		{"schema version", func(r *Run) { r.SchemaVersion = 2 }, "schema_version"},
		{"zero repeats", func(r *Run) { r.Config.Repeats = 0 }, "repeats"},
		{"no profiles", func(r *Run) { r.Config.Profiles = nil }, "profiles"},
		{"empty library", func(r *Run) { r.Cells[0].Library = "" }, "empty"},
		{"duplicate cell", func(r *Run) { r.Cells = append(r.Cells, r.Cells[0]) }, "duplicate"},
		{"unknown profile", func(r *Run) { r.Cells[0].Profile = "nope" }, "profile"},
		{"bad status", func(r *Run) { r.Cells[0].Status = "maybe" }, "status"},
		{"ok wrong repeat count", func(r *Run) { r.Cells[0].Repeats = r.Cells[0].Repeats[:1] }, "repeats"},
		{"ok without median", func(r *Run) { r.Cells[0].Median = nil }, "median"},
		{"na without reason", func(r *Run) { r.Cells[1].Reason = "" }, "reason"},
		{"na with median", func(r *Run) { m := validMetrics(1); r.Cells[1].Median = &m }, "na"},
		{"ok cell with no ops", func(r *Run) { r.Cells[0].Repeats[0].Ops = 0 }, "ops"},
		{"ok cell with zero elapsed", func(r *Run) { r.Cells[0].Repeats[1].ElapsedSeconds = 0 }, "elapsed"},
		{"median with no ops", func(r *Run) { m := validMetrics(1); m.Ops = 0; r.Cells[0].Median = &m }, "ops"},
		{"nan metric", func(r *Run) { r.Cells[0].Repeats[0].BytesPerOp = nan }, "finite"},
		{"inf metric", func(r *Run) { r.Cells[0].Repeats[1].CPUNsPerOp = math.Inf(1) }, "finite"},
		{"error rate above 1", func(r *Run) { r.Cells[0].Repeats[0].ErrorRate = 1.5 }, "error_rate"},
		{"negative latency", func(r *Run) { r.Cells[0].Repeats[0].LatencyP50Ns = -1 }, "negative"},
		{"percentile order", func(r *Run) { r.Cells[0].Repeats[0].LatencyP95Ns = 1 }, "p50 <= p95 <= p99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			r := validRun()
			r.Cells[0].Repeats = append([]Metrics(nil), r.Cells[0].Repeats...)
			tt.mutate(&r)

			// Act
			err := r.Validate()

			// Assert
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestWriteLoadRoundTrip(t *testing.T) {
	// Arrange
	want := validRun()
	path := filepath.Join(t.TempDir(), "nested", "run.json")

	// Act
	writeErr := Write(path, want)
	got, loadErr := Load(path)

	// Assert
	if writeErr != nil || loadErr != nil {
		t.Fatalf("Write() = %v, Load() = %v", writeErr, loadErr)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestWriteRejectsInvalid(t *testing.T) {
	// Arrange
	r := validRun()
	r.SchemaVersion = 0
	path := filepath.Join(t.TempDir(), "run.json")

	// Act
	err := Write(path, r)

	// Assert
	if err == nil {
		t.Fatal("Write() of an invalid run = nil, want error")
	}
}

func TestWriteReplacesExistingFileWithoutLeftovers(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "run.json")
	writeFile(t, path, "stale content that is longer than nothing")
	want := validRun()

	// Act
	writeErr := Write(path, want)
	got, loadErr := Load(path)
	entries, readErr := os.ReadDir(dir)
	info, statErr := os.Stat(path)

	// Assert
	if writeErr != nil || loadErr != nil || readErr != nil || statErr != nil {
		t.Fatalf("Write() = %v, Load() = %v, ReadDir() = %v, Stat() = %v", writeErr, loadErr, readErr, statErr)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() after overwrite mismatch:\n got %+v\nwant %+v", got, want)
	}
	if len(entries) != 1 {
		t.Fatalf("directory holds %d entries after Write, want only run.json", len(entries))
	}
	if info.Mode().Perm() != resultFileMode {
		t.Fatalf("file mode = %v, want %v", info.Mode().Perm(), os.FileMode(resultFileMode))
	}
}

func TestWriteFailureLeavesNoTempFile(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "run.json")
	if err := os.Mkdir(path, resultDirMode); err != nil {
		t.Fatal(err)
	}

	// Act
	err := Write(path, validRun())
	entries, readErr := os.ReadDir(dir)

	// Assert
	if err == nil {
		t.Fatal("Write() onto a directory = nil, want error")
	}
	if readErr != nil || len(entries) != 1 {
		t.Fatalf("ReadDir() = %d entries, %v; want only the original directory", len(entries), readErr)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name string
		path func(t *testing.T) string
	}{
		{"missing file", func(t *testing.T) string { return filepath.Join(t.TempDir(), "none.json") }},
		{"not json", func(t *testing.T) string {
			p := filepath.Join(t.TempDir(), "bad.json")
			writeFile(t, p, "{")
			return p
		}},
		{"unknown field", func(t *testing.T) string {
			p := filepath.Join(t.TempDir(), "unknown.json")
			writeFile(t, p, `{"schema_version":1,"surprise":true}`)
			return p
		}},
		{"invalid run", func(t *testing.T) string {
			p := filepath.Join(t.TempDir(), "empty.json")
			writeFile(t, p, "{}")
			return p
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			path := tt.path(t)

			// Act
			_, err := Load(path)

			// Assert
			if err == nil {
				t.Fatal("Load() = nil, want error")
			}
		})
	}
}

func runWithP95(p95 int64) Run {
	r := validRun()
	m := validMetrics(p95)
	r.Cells[0].Repeats = []Metrics{m, m}
	r.Cells[0].Median = &m
	return r
}

func TestCompareP95(t *testing.T) {
	tests := []struct {
		name       string
		a, b       Run
		tolerance  float64
		wantExceed bool
		wantNote   string
	}{
		{"5 percent passes", runWithP95(1_000_000), runWithP95(1_050_000), 0.10, false, ""},
		{"15 percent flagged", runWithP95(1_000_000), runWithP95(1_150_000), 0.10, true, ""},
		{"identical", runWithP95(1_000_000), runWithP95(1_000_000), 0.10, false, ""},
		{"missing in b", validRun(), func() Run { r := validRun(); r.Cells = r.Cells[1:]; return r }(), 0.10, true, "missing"},
		{"status mismatch", validRun(), func() Run { r := validRun(); r.Cells[1] = r.Cells[0]; r.Cells[1].Scenario = "union"; return r }(), 0.10, true, "status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (runs from tt)

			// Act
			devs := CompareP95(tt.a, tt.b, func(string) float64 { return tt.tolerance })

			// Assert
			exceeded := false
			var notes []string
			for _, d := range devs {
				exceeded = exceeded || d.Exceeds
				notes = append(notes, d.Note)
			}
			if exceeded != tt.wantExceed {
				t.Fatalf("exceeds = %v, want %v (devs %+v)", exceeded, tt.wantExceed, devs)
			}
			if tt.wantNote != "" && !strings.Contains(strings.Join(notes, "|"), tt.wantNote) {
				t.Fatalf("notes %v do not contain %q", notes, tt.wantNote)
			}
		})
	}
}

func TestCompareP95Delta(t *testing.T) {
	// Arrange
	a, b := runWithP95(1_000_000), runWithP95(1_150_000)

	// Act
	devs := CompareP95(a, b, func(string) float64 { return 0.10 })

	// Assert
	if len(devs) != 1 {
		t.Fatalf("len(devs) = %d, want 1 (na cell in both runs is skipped)", len(devs))
	}
	if got, want := devs[0].Delta, 0.15; math.Abs(got-want) > 1e-9 {
		t.Fatalf("Delta = %v, want %v", got, want)
	}
}
