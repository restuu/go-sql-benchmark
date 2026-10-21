package report

import (
	"math"
	"reflect"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

func scoreOf(t *testing.T, r ranking, name string) libScore {
	t.Helper()
	s, ok := r.lookup(name)
	if !ok {
		t.Fatalf("library %s not ranked", name)
	}
	return s
}

func order(r ranking) []string {
	names := make([]string, len(r.libs))
	for i, l := range r.libs {
		names[i] = l.library
	}
	return names
}

func TestRankCells(t *testing.T) {
	tests := []struct {
		name      string
		cells     []result.Cell
		weights   Weights
		wantOrder []string
		check     func(t *testing.T, r ranking)
	}{
		{
			name: "strict order by score",
			cells: []result.Cell{
				okCell("c", "select", "low", 30, 30, 30),
				okCell("a", "select", "low", 10, 10, 10),
				okCell("b", "select", "low", 20, 20, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{"a", "b", "c"},
			check: func(t *testing.T, r ranking) {
				if got := scoreOf(t, r, "a").score; got != 1 {
					t.Fatalf("score(a) = %v, want 1", got)
				}
				if got := scoreOf(t, r, "c").score; got != 3 {
					t.Fatalf("score(c) = %v, want 3", got)
				}
			},
		},
		{
			name: "ties share a rank and the name breaks the score tie",
			cells: []result.Cell{
				okCell("b", "select", "low", 10, 10, 10),
				okCell("a", "select", "low", 10, 10, 10),
				okCell("c", "select", "low", 20, 20, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{"a", "b", "c"},
			check: func(t *testing.T, r ranking) {
				wantRanks := [metricCount]int{3, 3, 3}
				if got := scoreOf(t, r, "c").cells[0].ranks; got != wantRanks {
					t.Fatalf("ranks(c) = %v, want %v", got, wantRanks)
				}
				if got := scoreOf(t, r, "b").score; got != 1 {
					t.Fatalf("score(b) = %v, want 1 (tied for first)", got)
				}
			},
		},
		{
			name: "na cell counts in total but not in ranked",
			cells: []result.Cell{
				okCell("a", "select", "low", 10, 10, 10),
				okCell("b", "select", "low", 20, 20, 20),
				okCell("a", "union", "low", 10, 10, 10),
				naCell("b", "union", "low"),
			},
			weights:   equalWeights,
			wantOrder: []string{"a", "b"},
			check: func(t *testing.T, r ranking) {
				b := scoreOf(t, r, "b")
				if b.ranked != 1 || b.total != 2 {
					t.Fatalf("b ranked/total = %d/%d, want 1/2", b.ranked, b.total)
				}
				if a := scoreOf(t, r, "a"); a.cells[1].of != 1 {
					t.Fatalf("a is alone in union: of = %d, want 1", a.cells[1].of)
				}
			},
		},
		{
			name: "cell above the error rate limit is not ranked",
			cells: []result.Cell{
				okCell("a", "select", "low", 10, 10, 10),
				withErrorRate(okCell("b", "select", "low", 5, 5, 5), 0.02),
				okCell("c", "select", "low", 20, 20, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{"a", "c", "b"},
			check: func(t *testing.T, r ranking) {
				b := scoreOf(t, r, "b")
				if b.ranked != 0 || b.total != 1 || !math.IsInf(b.score, 1) {
					t.Fatalf("b = %+v, want unranked with +Inf score", b)
				}
				if got := scoreOf(t, r, "c").score; got != 2 {
					t.Fatalf("score(c) = %v, want 2 (b excluded)", got)
				}
			},
		},
		{
			name: "cell exactly at the error rate limit is ranked",
			cells: []result.Cell{
				withErrorRate(okCell("a", "select", "low", 5, 5, 5), maxRankedErrorRate),
				okCell("b", "select", "low", 20, 20, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{"a", "b"},
		},
		{
			name: "equal weights favour the all-rounder",
			cells: []result.Cell{
				okCell("fast", "select", "low", 10, 90, 90),
				okCell("round", "select", "low", 20, 20, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{"round", "fast"},
		},
		{
			name: "p95-only weights follow latency",
			cells: []result.Cell{
				okCell("fast", "select", "low", 10, 90, 90),
				okCell("round", "select", "low", 20, 20, 20),
			},
			weights:   Weights{P95: 1},
			wantOrder: []string{"fast", "round"},
		},
		{
			name: "library with only na cells sorts last with an infinite score",
			cells: []result.Cell{
				naCell("z", "union", "low"),
				okCell("a", "union", "low", 10, 10, 10),
			},
			weights:   equalWeights,
			wantOrder: []string{"a", "z"},
			check: func(t *testing.T, r ranking) {
				if z := scoreOf(t, r, "z"); !math.IsInf(z.score, 1) || z.ranked != 0 {
					t.Fatalf("z = %+v", z)
				}
			},
		},
		{
			name: "mean rank per metric",
			cells: []result.Cell{
				okCell("a", "select", "low", 10, 20, 10),
				okCell("b", "select", "low", 20, 10, 20),
				okCell("a", "join", "low", 10, 20, 10),
				okCell("b", "join", "low", 20, 10, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{"a", "b"},
			check: func(t *testing.T, r ranking) {
				want := [metricCount]float64{1, 2, 1}
				if got := scoreOf(t, r, "a").meanRank; got != want {
					t.Fatalf("meanRank(a) = %v, want %v", got, want)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			run := runOf(tt.cells...)

			// Act
			r := rankCells(run, tt.weights)

			// Assert
			if got := order(r); !reflect.DeepEqual(got, tt.wantOrder) {
				t.Fatalf("order = %v, want %v", got, tt.wantOrder)
			}
			if tt.check != nil {
				tt.check(t, r)
			}
		})
	}
}

func TestRankCellsIgnoresCellOrder(t *testing.T) {
	// Arrange
	cells := []result.Cell{
		okCell("a", "select", "low", 10, 33.3, 7.7),
		okCell("b", "select", "low", 20, 11.1, 9.9),
		okCell("a", "join", "stress", 15, 1.1, 2.2),
		okCell("b", "join", "stress", 14, 3.3, 4.4),
	}
	reversed := make([]result.Cell, len(cells))
	for i, c := range cells {
		reversed[len(cells)-1-i] = c
	}

	// Act
	forward := rankCells(runOf(cells...), equalWeights)
	backward := rankCells(runOf(reversed...), equalWeights)

	// Assert
	if !reflect.DeepEqual(forward, backward) {
		t.Fatalf("ranking depends on cell order:\n%+v\n%+v", forward, backward)
	}
}

func TestRankingTop(t *testing.T) {
	// Arrange
	run := runOf(
		okCell("a", "select", "low", 10, 10, 10),
		okCell("b", "select", "low", 20, 20, 20),
		okCell("c", "select", "low", 30, 30, 30),
		naCell("d", "select", "low"),
	)
	r := rankCells(run, equalWeights)

	// Act
	top2 := r.top(2)
	topAll := r.top(topCount)

	// Assert
	if len(top2) != 2 || top2[0].library != "a" || top2[1].library != "b" {
		t.Fatalf("top(2) = %+v", top2)
	}
	if len(topAll) != 3 {
		t.Fatalf("top(%d) has %d libraries, want 3 (d has no ranked cell)", topCount, len(topAll))
	}
	if got := r.rest(topCount); len(got) != 1 || got[0].library != "d" {
		t.Fatalf("rest = %+v, want d", got)
	}
}

func TestCompetitionRanks(t *testing.T) {
	tests := []struct {
		name string
		libs []libScore
		want []int
	}{
		{"distinct scores", []libScore{{score: 1}, {score: 2}, {score: 3}}, []int{1, 2, 3}},
		{"tie shares the rank and skips the next", []libScore{{score: 1}, {score: 2}, {score: 2}, {score: 3}}, []int{1, 2, 2, 4}},
		{"all tied", []libScore{{score: 2}, {score: 2}}, []int{1, 1}},
		{"empty", nil, []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			libs := tt.libs

			// Act
			got := competitionRanks(libs)

			// Assert
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("competitionRanks() = %v, want %v", got, tt.want)
			}
		})
	}
}
