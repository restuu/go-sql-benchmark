package report

import (
	"math"
	"reflect"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

const ref = baselineLibrary

func scoreOf(t *testing.T, r ranking, name string) libScore {
	t.Helper()
	s, ok := r.lookup(name)
	if !ok {
		t.Fatalf("library %s not ranked", name)
	}
	return s
}

func order(r ranking) []string { return libraryNamesOf(r.libs) }

func libraryNamesOf(libs []libScore) []string {
	names := make([]string, len(libs))
	for i, l := range libs {
		names[i] = l.library
	}
	return names
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestRankCells(t *testing.T) {
	tests := []struct {
		name      string
		cells     []result.Cell
		weights   Weights
		wantOrder []string
		check     func(t *testing.T, r ranking)
	}{
		{
			name: "score is the ratio to the baseline, so a small gap is a small score gap",
			cells: []result.Cell{
				okCell("x", "select", "low", 20, 20, 20),
				okCell(ref, "select", "low", 10, 10, 10),
				okCell("y", "select", "low", 11, 11, 11),
			},
			weights:   equalWeights,
			wantOrder: []string{ref, "y", "x"},
			check: func(t *testing.T, r ranking) {
				for lib, want := range map[string]float64{ref: 1, "y": 1.1, "x": 2} {
					if got := scoreOf(t, r, lib).score; !near(got, want) {
						t.Fatalf("score(%s) = %v, want %v", lib, got, want)
					}
				}
			},
		},
		{
			name: "equal to the baseline scores 1 and the name breaks the tie",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				okCell("b", "select", "low", 10, 10, 10),
				okCell("c", "select", "low", 10, 10, 10),
			},
			weights:   equalWeights,
			wantOrder: []string{"b", "c", ref}, // equal scores: name breaks the tie
			check: func(t *testing.T, r ranking) {
				if got := scoreOf(t, r, "c").score; got != 1 {
					t.Fatalf("score(c) = %v, want 1 (same as the baseline)", got)
				}
			},
		},
		{
			name: "cases combine as a geometric mean, so 2x and 0.5x cancel",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				okCell(ref, "join", "low", 10, 10, 10),
				okCell("x", "select", "low", 20, 20, 20),
				okCell("x", "join", "low", 5, 5, 5),
			},
			weights:   equalWeights,
			wantOrder: []string{ref, "x"},
			check: func(t *testing.T, r ranking) {
				if got := scoreOf(t, r, "x").score; !near(got, 1) {
					t.Fatalf("score(x) = %v, want 1", got)
				}
			},
		},
		{
			name: "weights decide how much each metric counts",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				okCell("x", "select", "low", 20, 40, 10),
			},
			weights:   Weights{P95: 1},
			wantOrder: []string{ref, "x"},
			check: func(t *testing.T, r ranking) {
				if got := scoreOf(t, r, "x").score; !near(got, 2) {
					t.Fatalf("score(x) = %v, want 2 (p95 only)", got)
				}
			},
		},
		{
			name: "equal weights favour the all-rounder",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				okCell("fast", "select", "low", 5, 40, 40),
				okCell("round", "select", "low", 11, 11, 11),
			},
			weights:   equalWeights,
			wantOrder: []string{ref, "round", "fast"},
		},
		{
			name: "p95-only weights follow latency",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				okCell("fast", "select", "low", 5, 40, 40),
				okCell("round", "select", "low", 11, 11, 11),
			},
			weights:   Weights{P95: 1},
			wantOrder: []string{"fast", ref, "round"},
		},
		{
			name: "na case counts in total but not in ranked",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				okCell("b", "select", "low", 20, 20, 20),
				okCell(ref, "union", "low", 10, 10, 10),
				naCell("b", "union", "low"),
			},
			weights:   equalWeights,
			wantOrder: []string{ref, "b"},
			check: func(t *testing.T, r ranking) {
				b := scoreOf(t, r, "b")
				if b.ranked != 1 || b.total != 2 {
					t.Fatalf("b ranked/total = %d/%d, want 1/2", b.ranked, b.total)
				}
			},
		},
		{
			name: "case above the error rate limit is not ranked",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				withErrorRate(okCell("b", "select", "low", 5, 5, 5), 0.02),
				okCell("c", "select", "low", 20, 20, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{ref, "c", "b"},
			check: func(t *testing.T, r ranking) {
				b := scoreOf(t, r, "b")
				if b.ranked != 0 || b.total != 1 || !math.IsInf(b.score, 1) {
					t.Fatalf("b = %+v, want unranked with +Inf score", b)
				}
			},
		},
		{
			name: "case exactly at the error rate limit is ranked",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				withErrorRate(okCell("a", "select", "low", 5, 5, 5), maxRankedErrorRate),
			},
			weights:   equalWeights,
			wantOrder: []string{"a", ref},
		},
		{
			name: "library with only na cases sorts last with an infinite score",
			cells: []result.Cell{
				naCell("z", "union", "low"),
				okCell(ref, "union", "low", 10, 10, 10),
			},
			weights:   equalWeights,
			wantOrder: []string{ref, "z"},
			check: func(t *testing.T, r ranking) {
				if z := scoreOf(t, r, "z"); !math.IsInf(z.score, 1) || z.ranked != 0 {
					t.Fatalf("z = %+v", z)
				}
			},
		},
		{
			name: "a case without a usable baseline is ranked for nobody",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				okCell("b", "select", "low", 20, 20, 20),
				naCell(ref, "union", "low"),
				okCell("b", "union", "low", 20, 20, 20),
				withErrorRate(okCell(ref, "join", "low", 10, 10, 10), 0.5),
				okCell("b", "join", "low", 20, 20, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{ref, "b"},
			check: func(t *testing.T, r ranking) {
				if b := scoreOf(t, r, "b"); b.ranked != 1 || b.total != 3 {
					t.Fatalf("b ranked/total = %d/%d, want 1/3", b.ranked, b.total)
				}
			},
		},
		{
			name: "a zero metric cannot form a ratio and leaves that case unranked",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				okCell("b", "select", "low", 20, 0, 20),
				okCell("c", "select", "low", 20, 20, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{ref, "c", "b"},
			check: func(t *testing.T, r ranking) {
				if b := scoreOf(t, r, "b"); b.ranked != 0 || b.total != 1 {
					t.Fatalf("b ranked/total = %d/%d, want 0/1", b.ranked, b.total)
				}
			},
		},
		{
			name: "no baseline in the run ranks nobody",
			cells: []result.Cell{
				okCell("a", "select", "low", 10, 10, 10),
				okCell("b", "select", "low", 20, 20, 20),
			},
			weights:   equalWeights,
			wantOrder: []string{"a", "b"},
			check: func(t *testing.T, r ranking) {
				if got := r.ranked(); len(got) != 0 {
					t.Fatalf("ranked = %+v, want none", got)
				}
			},
		},
		{
			name: "mean ratio per metric",
			cells: []result.Cell{
				okCell(ref, "select", "low", 10, 10, 10),
				okCell(ref, "join", "low", 10, 10, 10),
				okCell("a", "select", "low", 20, 20, 10),
				okCell("a", "join", "low", 20, 20, 10),
			},
			weights:   equalWeights,
			wantOrder: []string{ref, "a"},
			check: func(t *testing.T, r ranking) {
				got := scoreOf(t, r, "a").meanRatio
				for m, want := range [metricCount]float64{2, 2, 1} {
					if !near(got[m], want) {
						t.Fatalf("meanRatio(a) = %v, want [2 2 1]", got)
					}
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			run := runOf(tt.cells...)

			// Act
			r := rankCells(run, tt.weights, nil)

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
		okCell(ref, "select", "low", 10, 33.3, 7.7),
		okCell("b", "select", "low", 20, 11.1, 9.9),
		okCell(ref, "join", "stress", 15, 1.1, 2.2),
		okCell("b", "join", "stress", 14, 3.3, 4.4),
	}
	reversed := make([]result.Cell, len(cells))
	for i, c := range cells {
		reversed[len(cells)-1-i] = c
	}

	// Act
	forward := rankCells(runOf(cells...), equalWeights, nil)
	backward := rankCells(runOf(reversed...), equalWeights, nil)

	// Assert
	if !reflect.DeepEqual(forward, backward) {
		t.Fatalf("ranking depends on cell order:\n%+v\n%+v", forward, backward)
	}
}

func TestRankingTop(t *testing.T) {
	// Arrange
	run := runOf(
		okCell(ref, "select", "low", 10, 10, 10),
		okCell("b", "select", "low", 20, 20, 20),
		okCell("c", "select", "low", 30, 30, 30),
		naCell("d", "select", "low"),
	)
	r := rankCells(run, equalWeights, nil)

	// Act
	top2 := r.top(2)
	topAll := r.top(topCount)

	// Assert
	if got := libraryNamesOf(top2); !reflect.DeepEqual(got, []string{ref, "b"}) {
		t.Fatalf("top(2) = %v", got)
	}
	if len(topAll) != 3 {
		t.Fatalf("top(%d) has %d libraries, want 3 (d has no ranked case)", topCount, len(topAll))
	}
	if got := r.rest(topCount); len(got) != 1 || got[0].library != "d" {
		t.Fatalf("rest = %+v, want d", got)
	}
}

func TestRankCellsHonorableMention(t *testing.T) {
	// Arrange: g is the fastest library everywhere but sits outside the ranking
	cells := []result.Cell{
		okCell(ref, "select", "low", 10, 10, 10),
		okCell("b", "select", "low", 20, 20, 20),
	}
	withG := append([]result.Cell{okCell("g", "select", "low", 5, 5, 5)}, cells...)

	// Act
	r := rankCells(runOf(withG...), equalWeights, map[string]bool{"g": true})
	without := rankCells(runOf(cells...), equalWeights, nil)

	// Assert
	if g := scoreOf(t, r, "g"); !g.guest || !near(g.score, 0.5) {
		t.Fatalf("g = %+v, want a guest scored 0.5", g)
	}
	for _, lib := range []string{ref, "b"} {
		if got, want := scoreOf(t, r, lib).score, scoreOf(t, without, lib).score; got != want {
			t.Fatalf("score(%s) = %v with g, %v without", lib, got, want)
		}
	}
	if got := libraryNamesOf(r.top(topCount)); !reflect.DeepEqual(got, []string{ref, "b"}) {
		t.Fatalf("top = %v, want [database_sql b] without g", got)
	}
	if got := r.rest(topCount); len(got) != 0 {
		t.Fatalf("rest = %+v, want none (g is a mention, not a rest entry)", got)
	}
	if got := libraryNamesOf(r.mentions()); !reflect.DeepEqual(got, []string{"g"}) {
		t.Fatalf("mentions = %v, want [g]", got)
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

func TestTopLibraries(t *testing.T) {
	// seven libraries around the baseline: from g (fastest, most bytes) to a (slowest, fewest bytes)
	var ladder []result.Cell
	for i, name := range []string{"g", "f", "e", ref, "c", "b", "a"} {
		p := int64(10 * (i + 1))
		ladder = append(ladder, okCell(name, "select", "low", p, float64(100-p), float64(p)))
	}
	tests := []struct {
		name    string
		cells   []result.Cell
		weights Weights
		want    []string
	}{
		{"keeps the best five and skips an n/a library", append(append([]result.Cell{}, ladder...), naCell("z", "select", "low")), equalWeights, []string{"g", "f", "e", ref, "c"}},
		{"p95 only", ladder, Weights{P95: 1}, []string{"g", "f", "e", ref, "c"}},
		{"bytes only flips the order", ladder, Weights{BytesPerOp: 1}, []string{"a", "b", "c", ref, "e"}},
		{"fewer than five ranked", ladder[:4], equalWeights, []string{"g", "f", "e", ref}},
		{"nothing ranked", []result.Cell{naCell("z", "select", "low")}, equalWeights, []string{}},
		{"nothing is ranked without the baseline", append(append([]result.Cell{}, ladder[:3]...), ladder[4:]...), equalWeights, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			run := runOf(tt.cells...)

			// Act
			got := TopLibraries(run, Rubric{Weights: tt.weights})

			// Assert
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("TopLibraries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTopLibrariesSkipsHonorableMentions(t *testing.T) {
	// Arrange
	run := runOf(
		okCell("g", "select", "low", 5, 5, 5),
		okCell(ref, "select", "low", 10, 10, 10),
		okCell("b", "select", "low", 20, 20, 20),
	)
	rb := Rubric{Weights: equalWeights, Libraries: []LibraryInfo{{Name: "g", HonorableMention: "Postgres only"}}}

	// Act
	got := TopLibraries(run, rb)

	// Assert
	if want := []string{ref, "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("TopLibraries() = %v, want %v", got, want)
	}
}
