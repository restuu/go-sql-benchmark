package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

const defaultRepeats = 3

// config is the validated command line.
type config struct {
	libs      []string
	scenarios []bench.Scenario
	profiles  []bench.Profile
	repeats   int
	seed      uint64
	out       string
}

// parseFlags parses and validates args. A zero -seed derives the seed from now. Flag overrides
// of the stress profile (-workers, -stress-duration, -delete-pool) and the low profile
// (-low-duration) are applied here so the result records the effective parameters.
func parseFlags(args []string, validLibs []string, now time.Time) (config, error) {
	low, stress := bench.Low(), bench.Stress()
	var libs, scenarios, profiles, out string
	var cfg config

	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&libs, "libs", strings.Join(validLibs, ","), "comma-separated libraries")
	fs.StringVar(&scenarios, "scenarios", names(bench.AllScenarios), "comma-separated scenarios")
	fs.StringVar(&profiles, "profiles", "low,stress", "comma-separated profiles")
	fs.IntVar(&cfg.repeats, "repeats", defaultRepeats, "repeats per cell; the median is reported")
	fs.Uint64Var(&cfg.seed, "seed", 0, "seed for library order and workload ids (0 = derive from the clock)")
	fs.IntVar(&stress.Workers, "workers", stress.Workers, "stress profile workers and pool size")
	fs.DurationVar(&stress.Duration, "stress-duration", stress.Duration, "stress profile measured window")
	fs.DurationVar(&low.Duration, "low-duration", low.Duration, "low profile measured window")
	fs.IntVar(&stress.DeletePool, "delete-pool", stress.DeletePool, "stress profile ids pre-inserted for DELETE")
	fs.StringVar(&out, "out", "results/run-"+now.UTC().Format("20060102T150405Z")+".json", "result file")
	if err := fs.Parse(args); err != nil {
		return config{}, fmt.Errorf("bench: %w\n%s", err, usage(fs))
	}

	var err error
	if cfg.libs, err = parseList("-libs", libs, validLibs); err != nil {
		return config{}, err
	}
	if cfg.scenarios, err = parseScenarios(scenarios); err != nil {
		return config{}, err
	}
	if cfg.profiles, err = parseProfiles(profiles, low, stress); err != nil {
		return config{}, err
	}
	if cfg.repeats < 1 {
		return config{}, errors.New("bench: -repeats must be >= 1")
	}
	if cfg.seed == 0 {
		cfg.seed = uint64(now.UnixNano())
	}
	cfg.out = out
	return cfg, nil
}

func usage(fs *flag.FlagSet) string {
	var b strings.Builder
	fs.SetOutput(&b)
	fs.PrintDefaults()
	return b.String()
}

func names(scenarios []bench.Scenario) string {
	parts := make([]string, len(scenarios))
	for i, s := range scenarios {
		parts[i] = string(s)
	}
	return strings.Join(parts, ",")
}

// parseList splits a comma-separated flag and checks every item against the valid values.
func parseList(flagName, value string, valid []string) ([]string, error) {
	if value == "" {
		return nil, fmt.Errorf("bench: %s must not be empty (valid: %s)", flagName, strings.Join(valid, ", "))
	}
	items := strings.Split(value, ",")
	for _, item := range items {
		if !slices.Contains(valid, item) {
			return nil, fmt.Errorf("bench: unknown %s value %q (valid: %s)", flagName, item, strings.Join(valid, ", "))
		}
	}
	return items, nil
}

func parseScenarios(value string) ([]bench.Scenario, error) {
	items, err := parseList("-scenarios", value, strings.Split(names(bench.AllScenarios), ","))
	if err != nil {
		return nil, err
	}
	out := make([]bench.Scenario, len(items))
	for i, item := range items {
		if out[i], err = bench.ParseScenario(item); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func parseProfiles(value string, low, stress bench.Profile) ([]bench.Profile, error) {
	items, err := parseList("-profiles", value, []string{bench.ProfileLow, bench.ProfileStress})
	if err != nil {
		return nil, err
	}
	out := make([]bench.Profile, len(items))
	for i, item := range items {
		out[i] = low
		if item == bench.ProfileStress {
			out[i] = stress
		}
		if err := checkProfile(out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func checkProfile(p bench.Profile) error {
	switch {
	case p.Workers < 1:
		return errors.New("bench: -workers must be >= 1")
	case p.Duration <= 0:
		return fmt.Errorf("bench: %s profile duration must be positive", p.Name)
	case p.DeletePool < 1:
		return errors.New("bench: -delete-pool must be >= 1")
	}
	return nil
}
