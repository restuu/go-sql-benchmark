package report

import (
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/result"
)

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		ns   int64
		want string
	}{
		{0, "0 ns"},
		{850, "850 ns"},
		{1_500, "1.5 µs"},
		{24_000, "24 µs"},
		{2_845_500, "2.8 ms"},
		{9_920_167, "9.9 ms"},
		{24_342_458, "24 ms"},
		{2_500_000_000, "2.5 s"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			// Arrange
			ns := tt.ns

			// Act
			got := fmtDuration(ns)

			// Assert
			if got != tt.want {
				t.Fatalf("fmtDuration(%d) = %q, want %q", ns, got, tt.want)
			}
		})
	}
}

func TestFmtDelta(t *testing.T) {
	tests := []struct {
		name       string
		base, node int64
		want       string
	}{
		{"slower", 1_000_000, 1_500_000, "+50.0%"},
		{"faster", 1_000_000, 800_000, "-20.0%"},
		{"equal", 1_000_000, 1_000_000, "+0.0%"},
		{"tiny negative rounds to zero", 1_000_000, 999_999, "+0.0%"},
		{"zero baseline", 0, 5, "-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			base, v := tt.base, tt.node

			// Act
			got := fmtDelta(base, v)

			// Assert
			if got != tt.want {
				t.Fatalf("fmtDelta(%d, %d) = %q, want %q", base, v, got, tt.want)
			}
		})
	}
}

func repeatsWithP95(p95s ...int64) []result.Metrics {
	out := make([]result.Metrics, len(p95s))
	for i, p := range p95s {
		out[i] = result.Metrics{LatencyP95Ns: p}
	}
	return out
}

func TestFmtSpread(t *testing.T) {
	tests := []struct {
		name    string
		repeats []result.Metrics
		want    string
	}{
		{"single repeat", repeatsWithP95(1_000_000), "-"},
		{"above the warning level", repeatsWithP95(1_000_000, 1_200_000, 1_100_000), "18.2% *"},
		{"exactly the warning level", repeatsWithP95(1_000, 1_100, 1_000), "10.0%"},
		{"two repeats use the mean as median", repeatsWithP95(1_000_000, 1_050_000), "4.9%"},
		{"zero median", repeatsWithP95(0, 0, 0), "-"},
		{"input order does not matter", repeatsWithP95(1_200_000, 1_100_000, 1_000_000), "18.2% *"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repeats := tt.repeats

			// Act
			got := fmtSpread(repeats)

			// Assert
			if got != tt.want {
				t.Fatalf("fmtSpread() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEscapeCell(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain", "plain"},
		{"a|b", `a\|b`},
		{"two\nlines", "two lines"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			// Arrange
			in := tt.in

			// Act
			got := escapeCell(in)

			// Assert
			if got != tt.want {
				t.Fatalf("escapeCell(%q) = %q, want %q", in, got, tt.want)
			}
		})
	}
}

func tableRubric() Rubric {
	return Rubric{
		Libraries: []LibraryInfo{
			{Name: "database_sql", Display: "database/sql"}, {Name: "ent", Display: "ent"},
			{Name: "sqlx", Display: "sqlx"}, {Name: "gorm", Display: "gorm"},
		},
		CellNotes: []CellNote{{Library: "ent", Scenario: "join", Note: "eager loading"}},
	}
}

func TestScenarioTable(t *testing.T) {
	const header = "| Library | p50 | p95 | p99 | ops/s | B/op | allocs/op | CPU ns/op | err % | p95 vs database/sql | p95 spread |\n" +
		"|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n"
	gormNA := naCell("gorm", "join", "low")
	gormNA.Reason = "n/a (needs | raw SQL)"
	tests := []struct {
		name  string
		cells []result.Cell
		libs  []string
		want  string
	}{
		{
			name: "baseline, footnote, excluded and na rows",
			cells: []result.Cell{
				okCell("database_sql", "join", "low", 1_000_000, 1000, 5000),
				okCell("ent", "join", "low", 1_500_000, 2000.6, 7500),
				withErrorRate(okCell("sqlx", "join", "low", 800_000, 1000, 5000), 0.02),
				gormNA,
			},
			libs: []string{"database_sql", "ent", "sqlx", "gorm"},
			want: header +
				"| database/sql | 500 µs | 1.0 ms | 2.0 ms | 100 | 1000 | 1.0 | 5000 | 0.00 | baseline | - |\n" +
				"| ent [1] | 750 µs | 1.5 ms | 3.0 ms | 100 | 2001 | 1.0 | 7500 | 0.00 | +50.0% | - |\n" +
				"| sqlx ! | 400 µs | 800 µs | 1.6 ms | 100 | 1000 | 1.0 | 5000 | 2.00 | -20.0% | - |\n" +
				"| gorm | n/a (needs \\| raw SQL) | - | - | - | - | - | - | - | - | - |\n" +
				"\n1. eager loading\n",
		},
		{
			name: "without a baseline the delta is a dash",
			cells: []result.Cell{
				okCell("ent", "select", "low", 1_500_000, 1000, 5000),
			},
			libs: []string{"ent"},
			want: header +
				"| ent | 750 µs | 1.5 ms | 3.0 ms | 100 | 1000 | 1.0 | 5000 | 0.00 | - | - |\n",
		},
		{
			name: "a baseline that is na leaves deltas as a dash",
			cells: []result.Cell{
				naCell("database_sql", "union", "low"),
				okCell("ent", "union", "low", 1_500_000, 1000, 5000),
			},
			libs: []string{"database_sql", "ent"},
			want: header +
				"| database/sql | n/a (unsupported) | - | - | - | - | - | - | - | - | - |\n" +
				"| ent | 750 µs | 1.5 ms | 3.0 ms | 100 | 1000 | 1.0 | 5000 | 0.00 | - | - |\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			idx := indexCells(runOf(tt.cells...))
			scenario := tt.cells[0].Scenario

			// Act
			got := scenarioTable(idx, tt.libs, scenario, "low", tableRubric())

			// Assert
			if got != tt.want {
				t.Fatalf("scenarioTable() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}
