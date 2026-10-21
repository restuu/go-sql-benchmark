package bench

import "time"

// Profile describes one load shape. All values are overridable by flags and recorded in the result.
type Profile struct {
	Name       string        // "low" or "stress"
	Workers    int           // concurrent goroutines and pool size
	Interval   time.Duration // start-to-start spacing per worker; 0 = unpaced
	Warmup     time.Duration
	Duration   time.Duration // measured window
	DeletePool int           // ids pre-inserted for the DELETE scenario
}

// Profile names.
const (
	ProfileLow    = "low"
	ProfileStress = "stress"
)

// Low is the low-traffic preset: a few paced workers far below saturation.
func Low() Profile {
	return Profile{
		Name:       ProfileLow,
		Workers:    4,
		Interval:   10 * time.Millisecond,
		Warmup:     1 * time.Second,
		Duration:   5 * time.Second,
		DeletePool: 5_000,
	}
}

// Stress is the saturation preset: many unpaced workers. Final parameters are a Phase 5 decision.
func Stress() Profile {
	return Profile{
		Name:       ProfileStress,
		Workers:    100,
		Warmup:     2 * time.Second,
		Duration:   10 * time.Second,
		DeletePool: 400_000,
	}
}

// Pool returns the pool size matching the worker count.
func (p Profile) Pool() PoolConfig {
	return PoolConfig{MaxOpen: p.Workers, MaxIdle: p.Workers}
}
