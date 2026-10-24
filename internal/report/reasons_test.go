package report

import (
	"reflect"
	"testing"
)

func TestReasonsFor(t *testing.T) {
	equal := Weights{P95: 1, BytesPerOp: 1, CPUNsPerOp: 1}
	tests := []struct {
		name    string
		lib     libScore
		weights Weights
		want    []string
	}{
		{
			name:    "weight-0 metric is left out of the ratios",
			weights: Weights{P95: 1, BytesPerOp: 1},
			lib: libScore{
				library: "a", ranked: 1, total: 1,
				meanRatio: [metricCount]float64{1.04, 1.43, 1.03},
				cells:     []cellRatio{{scenario: "select", profile: "low", ratios: [metricCount]float64{1.04, 1.43, 1.03}}},
			},
			want: []string{
				"ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 1.04x, B/op 1.43x",
				"slowest case: select/low (p95 1.04x database/sql)",
			},
		},
		{
			name:    "all sentences",
			weights: equal,
			lib: libScore{
				library: "a", ranked: 2, total: 3,
				meanRatio: [metricCount]float64{0.98, 1.5, 0.99},
				cells: []cellRatio{
					{scenario: "select", profile: "low", ratios: [metricCount]float64{0.9, 1.5, 1}},
					{scenario: "join", profile: "stress", ratios: [metricCount]float64{1.34, 1.5, 0.98}},
				},
			},
			want: []string{
				"ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 0.98x, B/op 1.50x, CPU ns/op 0.99x",
				"slowest case: join/stress (p95 1.34x database/sql)",
				"judged on 2 of 3 cases",
			},
		},
		{
			name:    "never slower than the baseline skips the slowest sentence",
			weights: equal,
			lib: libScore{
				library: "a", ranked: 1, total: 1,
				meanRatio: [metricCount]float64{0.8, 0.9, 0.7},
				cells:     []cellRatio{{scenario: "select", profile: "low", ratios: [metricCount]float64{0.8, 0.9, 0.7}}},
			},
			want: []string{
				"ratio to database/sql (geometric mean over ranked cases, 1.00 = same, lower is better): p95 0.80x, B/op 0.90x, CPU ns/op 0.70x",
			},
		},
		{
			name:    "the baseline is 1.00 by definition",
			weights: equal,
			lib: libScore{
				library: baselineLibrary, ranked: 1, total: 2,
				meanRatio: [metricCount]float64{1, 1, 1},
				cells:     []cellRatio{{scenario: "select", profile: "low", ratios: [metricCount]float64{1, 1, 1}}},
			},
			want: []string{
				"the baseline: every ratio is 1.00 by definition",
				"judged on 1 of 2 cases",
			},
		},
		{
			name:    "no ranked case gives no reasons",
			weights: equal,
			lib:     libScore{library: "a", ranked: 0, total: 2},
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			lib := tt.lib

			// Act
			got := reasonsFor(lib, tt.weights)

			// Assert
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("reasonsFor() = %q, want %q", got, tt.want)
			}
		})
	}
}
