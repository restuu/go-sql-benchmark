// Package result defines the benchmark JSON contract (schema_version 1), its validation and comparison.
package result

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

// SchemaVersion is the only result format version this package reads and writes.
const SchemaVersion = 1

// Cell statuses.
const (
	StatusOK = "ok"
	StatusNA = "na"
)

const (
	resultDirMode  = 0o755
	resultFileMode = 0o644
)

// Run is one complete benchmark execution.
type Run struct {
	SchemaVersion int       `json:"schema_version"`
	StartedAt     time.Time `json:"started_at"`
	Host          Host      `json:"host"`
	Config        Config    `json:"config"`
	Cells         []Cell    `json:"cells"`
}

// Host records the machine and versions the numbers were measured on.
type Host struct {
	GOOS            string            `json:"goos"`
	GOARCH          string            `json:"goarch"`
	GoVersion       string            `json:"go_version"`
	NumCPU          int               `json:"num_cpu"`
	GOMAXPROCS      int               `json:"gomaxprocs"`
	PostgresVersion string            `json:"postgres_version"`
	Libraries       map[string]string `json:"libraries"`
}

// Config records the effective run parameters (after flag overrides).
type Config struct {
	Seed     uint64          `json:"seed"`
	Repeats  int             `json:"repeats"`
	Profiles []ProfileConfig `json:"profiles"`
}

// ProfileConfig is the exact load shape used for a profile.
type ProfileConfig struct {
	Name       string `json:"name"`
	Workers    int    `json:"workers"`
	IntervalNs int64  `json:"interval_ns"`
	WarmupNs   int64  `json:"warmup_ns"`
	DurationNs int64  `json:"duration_ns"`
	DeletePool int    `json:"delete_pool"`
}

// Cell is one library x scenario x profile result.
type Cell struct {
	Library  string    `json:"library"`
	Scenario string    `json:"scenario"`
	Profile  string    `json:"profile"`
	Status   string    `json:"status"`
	Reason   string    `json:"reason,omitempty"`
	Repeats  []Metrics `json:"repeats,omitempty"`
	Median   *Metrics  `json:"median,omitempty"`
}

// Metrics are the measurements of one measured window (or the per-field median of several).
type Metrics struct {
	Ops                 int64   `json:"ops"`
	Errors              int64   `json:"errors"`
	ErrorRate           float64 `json:"error_rate"`
	ThroughputOpsPerSec float64 `json:"throughput_ops_per_sec"`
	LatencyP50Ns        int64   `json:"latency_p50_ns"`
	LatencyP95Ns        int64   `json:"latency_p95_ns"`
	LatencyP99Ns        int64   `json:"latency_p99_ns"`
	BytesPerOp          float64 `json:"bytes_per_op"`
	AllocsPerOp         float64 `json:"allocs_per_op"`
	CPUNsPerOp          float64 `json:"cpu_ns_per_op"`
	ElapsedSeconds      float64 `json:"elapsed_seconds"`
}

type cellKey struct{ library, scenario, profile string }

func (c Cell) key() cellKey { return cellKey{c.Library, c.Scenario, c.Profile} }

func (k cellKey) String() string { return k.library + "/" + k.scenario + "/" + k.profile }

// Validate checks every rule of the schema; Write and Load call it.
func (r Run) Validate() error {
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("result: schema_version %d, want %d", r.SchemaVersion, SchemaVersion)
	}
	if r.Config.Repeats < 1 {
		return fmt.Errorf("result: config repeats %d, want >= 1", r.Config.Repeats)
	}
	if len(r.Config.Profiles) == 0 {
		return fmt.Errorf("result: config has no profiles")
	}
	profiles := make(map[string]bool, len(r.Config.Profiles))
	for _, p := range r.Config.Profiles {
		profiles[p.Name] = true
	}
	seen := make(map[cellKey]bool, len(r.Cells))
	for _, c := range r.Cells {
		if err := c.validate(r.Config.Repeats, profiles); err != nil {
			return err
		}
		if seen[c.key()] {
			return fmt.Errorf("result: duplicate cell %s", c.key())
		}
		seen[c.key()] = true
	}
	return nil
}

func (c Cell) validate(repeats int, profiles map[string]bool) error {
	if c.Library == "" || c.Scenario == "" || c.Profile == "" {
		return fmt.Errorf("result: cell has an empty library, scenario or profile: %+v", c.key())
	}
	if !profiles[c.Profile] {
		return fmt.Errorf("result: cell %s: profile not in config", c.key())
	}
	switch c.Status {
	case StatusOK:
		return c.validateOK(repeats)
	case StatusNA:
		if c.Reason == "" || len(c.Repeats) > 0 || c.Median != nil {
			return fmt.Errorf("result: cell %s: status na needs a reason and no metrics", c.key())
		}
		return nil
	default:
		return fmt.Errorf("result: cell %s: unknown status %q", c.key(), c.Status)
	}
}

func (c Cell) validateOK(repeats int) error {
	if len(c.Repeats) != repeats {
		return fmt.Errorf("result: cell %s: %d repeats, want %d", c.key(), len(c.Repeats), repeats)
	}
	if c.Median == nil {
		return fmt.Errorf("result: cell %s: status ok needs a median", c.key())
	}
	for i, m := range c.Repeats {
		if err := m.validateOK(); err != nil {
			return fmt.Errorf("result: cell %s repeat %d: %w", c.key(), i+1, err)
		}
	}
	if err := c.Median.validateOK(); err != nil {
		return fmt.Errorf("result: cell %s median: %w", c.key(), err)
	}
	return nil
}

// validateOK adds the rules of a measured (status ok) window: it must have succeeded at least once.
func (m Metrics) validateOK() error {
	if m.Ops < 1 || m.ElapsedSeconds <= 0 {
		return fmt.Errorf("measured window needs ops >= 1 and elapsed_seconds > 0, got ops %d and elapsed %v", m.Ops, m.ElapsedSeconds)
	}
	return m.validate()
}

func (m Metrics) validate() error {
	floats := []float64{m.ErrorRate, m.ThroughputOpsPerSec, m.BytesPerOp, m.AllocsPerOp, m.CPUNsPerOp, m.ElapsedSeconds}
	for _, f := range floats {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("metric is not finite")
		}
		if f < 0 {
			return fmt.Errorf("negative metric %v", f)
		}
	}
	if m.Ops < 0 || m.Errors < 0 || m.LatencyP50Ns < 0 || m.LatencyP95Ns < 0 || m.LatencyP99Ns < 0 {
		return fmt.Errorf("negative metric")
	}
	if m.ErrorRate > 1 {
		return fmt.Errorf("error_rate %v above 1", m.ErrorRate)
	}
	if m.LatencyP50Ns > m.LatencyP95Ns || m.LatencyP95Ns > m.LatencyP99Ns {
		return fmt.Errorf("latency must satisfy p50 <= p95 <= p99, got %d, %d, %d", m.LatencyP50Ns, m.LatencyP95Ns, m.LatencyP99Ns)
	}
	return nil
}

// Write validates the run and writes it as indented JSON, creating parent directories.
func Write(path string, r Run) error {
	if err := r.Validate(); err != nil {
		return fmt.Errorf("result: write %s: %w", path, err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("result: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), resultDirMode); err != nil {
		return fmt.Errorf("result: create dir: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), resultFileMode); err != nil {
		return fmt.Errorf("result: write %s: %w", path, err)
	}
	return nil
}

// Load reads and validates a result file.
func Load(path string) (Run, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Run{}, fmt.Errorf("result: read %s: %w", path, err)
	}
	var r Run
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return Run{}, fmt.Errorf("result: parse %s: %w", path, err)
	}
	if err := r.Validate(); err != nil {
		return Run{}, fmt.Errorf("result: %s: %w", path, err)
	}
	return r, nil
}

// Deviation is the p95 comparison of one cell across two runs.
type Deviation struct {
	Library, Scenario, Profile string
	P95A, P95B                 int64
	Delta                      float64 // |a-b| / min(a,b); +Inf when only one side is zero
	Exceeds                    bool
	Note                       string // set when the cell could not be compared
}

// CompareP95 compares the median p95 of every cell present in either run. A cell that is na in
// both runs is skipped; a cell missing or differing in status is reported with Exceeds set.
func CompareP95(a, b Run, tolerance float64) []Deviation {
	bByKey := make(map[cellKey]Cell, len(b.Cells))
	for _, c := range b.Cells {
		bByKey[c.key()] = c
	}
	var devs []Deviation
	matched := make(map[cellKey]bool, len(a.Cells))
	for _, ca := range a.Cells {
		matched[ca.key()] = true
		cb, ok := bByKey[ca.key()]
		if d, report := compareCell(ca, cb, ok, tolerance); report {
			devs = append(devs, d)
		}
	}
	for _, cb := range b.Cells {
		if matched[cb.key()] {
			continue
		}
		devs = append(devs, Deviation{
			Library: cb.Library, Scenario: cb.Scenario, Profile: cb.Profile,
			Exceeds: true, Note: "missing in first run",
		})
	}
	return devs
}

func compareCell(ca, cb Cell, inB bool, tolerance float64) (Deviation, bool) {
	d := Deviation{Library: ca.Library, Scenario: ca.Scenario, Profile: ca.Profile}
	switch {
	case !inB:
		d.Exceeds, d.Note = true, "missing in second run"
	case ca.Status != cb.Status:
		d.Exceeds, d.Note = true, "status differs: "+ca.Status+" vs "+cb.Status
	case ca.Status == StatusNA:
		return d, false
	default:
		d.P95A, d.P95B = ca.Median.LatencyP95Ns, cb.Median.LatencyP95Ns
		d.Delta = relativeDiff(d.P95A, d.P95B)
		d.Exceeds = d.Delta > tolerance
	}
	return d, true
}

func relativeDiff(a, b int64) float64 {
	lo, hi := min(a, b), max(a, b)
	switch {
	case hi == lo:
		return 0
	case lo == 0:
		return math.Inf(1)
	default:
		return float64(hi-lo) / float64(lo)
	}
}
