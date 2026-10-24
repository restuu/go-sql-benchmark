package report

import (
	"slices"
	"strings"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/result"
)

// libCells builds one cell per scenario and profile for a library: n/a for the scenarios in na,
// measured at the given p95 (and the same bytes and CPU) for the rest.
func libCells(library string, p95 int64, na ...bench.Scenario) []result.Cell {
	var cells []result.Cell
	for _, profile := range []string{"low", "stress"} {
		for _, s := range bench.AllScenarios {
			if slices.Contains(na, s) {
				cells = append(cells, naCell(library, string(s), profile))
				continue
			}
			cells = append(cells, okCell(library, string(s), profile, p95, float64(p95), float64(p95)))
		}
	}
	return cells
}

func rubricOf(notes []CellNote, mentions map[string]string, libs ...string) Rubric {
	rb := Rubric{Weights: equalWeights, CellNotes: notes}
	for _, l := range libs {
		rb.Libraries = append(rb.Libraries, LibraryInfo{Name: l, Display: strings.ToUpper(l), HonorableMention: mentions[l]})
	}
	return rb
}

func TestWriteFeatureMatrix(t *testing.T) {
	tests := []struct {
		name  string
		cells []result.Cell
		notes []CellNote
		setup func(rb *Rubric)
		want  []string
		deny  []string
	}{
		{
			name:  "marks native and n/a per scenario",
			cells: append(append(libCells(ref, 10), libCells("a", 20, bench.ScenarioUnion)...), libCells("b", 30)...),
			want: []string{
				"## Feature matrix\n",
				"| Library | insert | update | delete | select | join | union |\n|---|---|---|---|---|---|---|\n",
				"| DATABASE_SQL | native | native | native | native | native | native |\n",
				"| A | native | native | native | native | native | n/a |\n",
			},
		},
		{
			name:  "footnotes follow row order and skip notes without the flag",
			cells: append(append(libCells(ref, 10), libCells("a", 20)...), libCells("b", 30)...),
			notes: []CellNote{
				{Library: "b", Scenario: "join", Note: "second row", Feature: true},
				{Library: "a", Scenario: "union", Note: "first row", Feature: true},
				{Library: "a", Scenario: "insert", Note: "not a feature note"},
			},
			want: []string{
				"| A | native | native | native | native | native | native [1] |\n",
				"| B | native | native | native | native | native [2] | native |\n",
				"1. first row\n2. second row\n",
			},
			deny: []string{"not a feature note"},
		},
		{
			name: "n/a only when both profiles are n/a",
			cells: append(libCells(ref, 10), append(
				without(libCells("a", 20), func(c result.Cell) bool { return c.Scenario == "union" }),
				naCell("a", "union", "low"), okCell("a", "union", "stress", 20, 20, 20))...),
			want: []string{"| A | native | native | native | native | native | native |\n"},
		},
		{
			name:  "display name is escaped",
			cells: append(libCells(ref, 10), libCells("a", 20)...),
			setup: func(rb *Rubric) { rb.Libraries[1].Display = "A|B" },
			want:  []string{"| A\\|B |"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			run := runOf(tt.cells...)
			rb := rubricOf(tt.notes, nil, ref, "a", "b")
			if tt.setup != nil {
				tt.setup(&rb)
			}
			var sb strings.Builder

			// Act
			writeFeatureMatrix(&sb, run, rb, rankCells(run, rb.Weights, nil))

			// Assert
			got := sb.String()
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("matrix does not contain %q\n%s", w, got)
				}
			}
			for _, d := range tt.deny {
				if strings.Contains(got, d) {
					t.Errorf("matrix must not contain %q", d)
				}
			}
		})
	}
}

func TestWriteUnion(t *testing.T) {
	tests := []struct {
		name     string
		cells    []result.Cell
		mentions map[string]string
		want     []string
	}{
		{
			name: "ranks supporters, lists the rest and shows mentions without a rank",
			cells: slices.Concat(
				libCells(ref, 10), libCells("a", 20), libCells("e", 30, bench.ScenarioUnion), libCells("g", 5),
			),
			mentions: map[string]string{"g": "Postgres only"},
			want: []string{
				"## Union\n",
				"not part of any score",
				"| Rank | Library | Score | Ranked cases | p95 | B/op | CPU ns/op |\n|---:|---|---:|---:|---:|---:|---:|\n",
				"| 1 | DATABASE_SQL | 1.00 | 2/2 | 1.00x | 1.00x | 1.00x |\n",
				"| 2 | A | 2.00 | 2/2 | 2.00x | 2.00x | 2.00x |\n",
				"| - | G | 0.50 | 2/2 | 0.50x | 0.50x | 0.50x |\n",
				"No ranked union result: E.\n",
			},
		},
		{
			name:  "without a baseline nothing can be ranked",
			cells: slices.Concat(libCells("a", 20), libCells("b", 30)),
			want:  []string{"## Union\n", "Nothing could be ranked on union"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			run := runOf(tt.cells...)
			rb := rubricOf(nil, tt.mentions, ref, "a", "b", "e", "g")
			union := rankScenarios(run, rb.Weights, rb.honorableMentions(), []bench.Scenario{unscoredScenario})
			var sb strings.Builder

			// Act
			writeUnion(&sb, rb, union)

			// Assert
			got := sb.String()
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("union section does not contain %q\n%s", w, got)
				}
			}
		})
	}
}
