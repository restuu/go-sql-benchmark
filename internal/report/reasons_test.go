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
			name:    "weight-0 metric is never called weakest",
			weights: Weights{P95: 1, BytesPerOp: 1},
			lib: libScore{
				library: "a", score: 1, ranked: 1, total: 1,
				meanRank: [metricCount]float64{1, 2, 5},
				cells:    []cellRank{{scenario: "select", profile: "low", ranks: [metricCount]int{1, 2, 5}, of: 5}},
			},
			want: []string{
				"fastest p95 in 1 of 1 ranked cells",
				"weakest on B/op: mean rank 2.0",
			},
		},
		{
			name:    "all three sentences",
			weights: equal,
			lib: libScore{
				library: "a", score: 2, ranked: 2, total: 2,
				meanRank: [metricCount]float64{2, 1.5, 2.5},
				cells: []cellRank{
					{scenario: "select", profile: "low", ranks: [metricCount]int{1, 2, 3}, of: 3},
					{scenario: "join", profile: "stress", ranks: [metricCount]int{3, 1, 2}, of: 4},
				},
			},
			want: []string{
				"fastest p95 in 1 of 2 ranked cells",
				"weakest on CPU ns/op: mean rank 2.5",
				"slowest cell: join/stress (p95 rank 3 of 4)",
			},
		},
		{
			name:    "best everywhere has no weakness to report",
			weights: equal,
			lib: libScore{
				library: "a", score: 1, ranked: 2, total: 2,
				meanRank: [metricCount]float64{1, 1, 1},
				cells: []cellRank{
					{scenario: "select", profile: "low", ranks: [metricCount]int{1, 1, 1}, of: 3},
					{scenario: "join", profile: "low", ranks: [metricCount]int{1, 1, 1}, of: 3},
				},
			},
			want: []string{"fastest p95 in 2 of 2 ranked cells"},
		},
		{
			name:    "never fastest skips the fastest sentence",
			weights: equal,
			lib: libScore{
				library: "a", score: 2, ranked: 1, total: 1,
				meanRank: [metricCount]float64{2, 2, 2},
				cells:    []cellRank{{scenario: "select", profile: "low", ranks: [metricCount]int{2, 2, 2}, of: 2}},
			},
			want: []string{
				"weakest on p95: mean rank 2.0",
				"slowest cell: select/low (p95 rank 2 of 2)",
			},
		},
		{
			name:    "fewer ranked than present cells is stated",
			weights: equal,
			lib: libScore{
				library: "a", score: 1, ranked: 1, total: 2,
				meanRank: [metricCount]float64{1, 1, 1},
				cells:    []cellRank{{scenario: "select", profile: "low", ranks: [metricCount]int{1, 1, 1}, of: 2}},
			},
			want: []string{
				"fastest p95 in 1 of 1 ranked cells",
				"judged on 1 of 2 cells",
			},
		},
		{
			name:    "no ranked cell gives no reasons",
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
