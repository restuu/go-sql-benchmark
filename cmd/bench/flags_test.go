package main

import (
	"strings"
	"testing"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

var testLibs = []string{"database_sql", "other"}

var testNow = time.Date(2026, 10, 21, 3, 4, 5, 0, time.UTC)

func TestParseFlagsDefaults(t *testing.T) {
	// Arrange
	// (no args)

	// Act
	cfg, err := parseFlags(nil, testLibs, testNow)

	// Assert
	if err != nil {
		t.Fatalf("parseFlags() = %v", err)
	}
	if len(cfg.libs) != 2 || len(cfg.scenarios) != len(bench.AllScenarios) || cfg.repeats != 3 {
		t.Errorf("defaults wrong: %+v", cfg)
	}
	if len(cfg.profiles) != 2 || cfg.profiles[0].Name != "low" || cfg.profiles[1].Name != "stress" {
		t.Errorf("profiles = %+v, want low then stress", cfg.profiles)
	}
	if cfg.seed != uint64(testNow.UnixNano()) {
		t.Errorf("seed = %d, want time-derived %d", cfg.seed, testNow.UnixNano())
	}
	if cfg.out != "results/run-20261021T030405Z.json" {
		t.Errorf("out = %q", cfg.out)
	}
}

func TestParseFlagsOverrides(t *testing.T) {
	// Arrange
	args := []string{"-libs", "database_sql", "-scenarios", "select,join", "-profiles", "stress", "-repeats", "1",
		"-seed", "9", "-workers", "8", "-stress-duration", "3s", "-low-duration", "2s", "-delete-pool", "77", "-out", "x.json"}

	// Act
	cfg, err := parseFlags(args, testLibs, testNow)

	// Assert
	if err != nil {
		t.Fatalf("parseFlags() = %v", err)
	}
	if len(cfg.libs) != 1 || len(cfg.scenarios) != 2 || cfg.scenarios[1] != bench.ScenarioJoin || cfg.repeats != 1 || cfg.seed != 9 || cfg.out != "x.json" {
		t.Errorf("overrides not applied: %+v", cfg)
	}
	p := cfg.profiles[0]
	if p.Name != "stress" || p.Workers != 8 || p.Duration != 3*time.Second || p.DeletePool != 77 || p.Pool().MaxOpen != 8 {
		t.Errorf("stress profile = %+v", p)
	}
}

func TestParseFlagsLowDurationOverride(t *testing.T) {
	// Arrange
	args := []string{"-profiles", "low", "-low-duration", "2s"}

	// Act
	cfg, err := parseFlags(args, testLibs, testNow)

	// Assert
	if err != nil || cfg.profiles[0].Duration != 2*time.Second {
		t.Fatalf("parseFlags() = %+v, %v", cfg.profiles, err)
	}
}

func TestParseFlagsErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown lib lists valid", []string{"-libs", "nope"}, "database_sql"},
		{"unknown scenario lists valid", []string{"-scenarios", "merge"}, "union"},
		{"unknown profile lists valid", []string{"-profiles", "turbo"}, "stress"},
		{"empty list", []string{"-libs", ""}, "-libs"},
		{"zero repeats", []string{"-repeats", "0"}, "-repeats"},
		{"zero workers", []string{"-workers", "0"}, "-workers"},
		{"zero duration", []string{"-stress-duration", "0s"}, "duration"},
		{"zero delete pool", []string{"-delete-pool", "0"}, "-delete-pool"},
		{"bad flag", []string{"-bogus"}, "bogus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (args from tt)

			// Act
			_, err := parseFlags(tt.args, testLibs, testNow)

			// Assert
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseFlags(%v) = %v, want error containing %q", tt.args, err, tt.want)
			}
		})
	}
}
