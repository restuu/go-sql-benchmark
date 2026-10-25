package report

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestOrderCombined(t *testing.T) {
	tests := []struct {
		name string
		rows []combinedRow
		want []string // "rank library sum", best first
	}{
		{
			name: "orders by the sum of the two ranks",
			rows: []combinedRow{{"a", 1, 3}, {"b", 2, 1}, {"c", 3, 3}},
			want: []string{"1 b 3", "2 a 4", "3 c 6"},
		},
		{
			name: "equal sums share a rank, the next one skips ahead, ties are alphabetical",
			rows: []combinedRow{{"z", 1, 3}, {"y", 3, 1}, {"x", 2, 2}, {"w", 4, 4}},
			want: []string{"1 x 4", "1 y 4", "1 z 4", "4 w 8"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			rb := Rubric{}

			// Act
			ordered, ranks := orderCombined(rb, tt.rows)

			// Assert
			var got []string
			for i, row := range ordered {
				got = append(got, fmt.Sprintf("%d %s %d", ranks[i], row.library, row.sum()))
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("orderCombined() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderCombinedRanking(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)

	// Act
	doc, err := Render(run, rb, DefaultFiles)

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if strings.Contains(doc.Summary, "## Combined ranking") {
		t.Error("the combined ranking belongs in the detailed report, not the summary")
	}
	if strings.Index(doc.Details, "## Combined ranking") < strings.Index(doc.Details, "## Qualitative ranking") {
		t.Fatal("combined ranking must follow the qualitative ranking")
	}
	body := section(t, doc.Details, "## Combined ranking")
	rows := 0
	for _, line := range strings.Split(body, "\n") {
		cells := strings.Split(strings.Trim(line, "| "), " | ")
		if len(cells) != 5 || cells[0] == "Rank" || strings.HasPrefix(cells[0], "-") {
			continue
		}
		rows++
		quant, _ := strconv.Atoi(cells[2])
		qual, _ := strconv.Atoi(cells[3])
		if sum, _ := strconv.Atoi(cells[4]); sum != quant+qual || quant == 0 || qual == 0 {
			t.Errorf("row %q: sum is not quantitative + qualitative rank", line)
		}
	}
	if want := len(rankCells(run, rb.Weights, rb.honorableMentions()).ranked()); rows != want {
		t.Errorf("table has %d rows, want %d (one per ranked library)", rows, want)
	}
}
