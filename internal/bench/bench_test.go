package bench

import (
	"errors"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/seedcheck"
)

func TestScenarioExpectedRows(t *testing.T) {
	tests := []struct {
		scenario Scenario
		want     int
	}{
		{ScenarioInsert, 1},
		{ScenarioUpdate, 1},
		{ScenarioDelete, 1},
		{ScenarioSelect, 1},
		{ScenarioJoin, 10},
		{ScenarioUnion, 40},
	}
	for _, tt := range tests {
		t.Run(string(tt.scenario), func(t *testing.T) {
			// Arrange
			// (scenario from tt)

			// Act
			got := tt.scenario.ExpectedRows()

			// Assert
			if got != tt.want {
				t.Fatalf("ExpectedRows() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExpectedRowsMatchSeed(t *testing.T) {
	// Arrange
	rows := seedcheck.ExpectedRows

	// Act
	joinRows := int(rows["orders"] / rows["customers"])
	unionRows := 2 * int(rows["customers"]) / seedCities

	// Assert
	if got := ScenarioJoin.ExpectedRows(); got != joinRows {
		t.Errorf("join rows = %d, seed implies %d", got, joinRows)
	}
	if got := ScenarioUnion.ExpectedRows(); got != unionRows {
		t.Errorf("union rows = %d, seed implies %d", got, unionRows)
	}
	if rows["customers"] != rows["suppliers"] {
		t.Errorf("customers (%d) and suppliers (%d) must match for the union count", rows["customers"], rows["suppliers"])
	}
}

func TestParseScenario(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Scenario
		wantErr bool
	}{
		{"insert", "insert", ScenarioInsert, false},
		{"union", "union", ScenarioUnion, false},
		{"unknown", "merge", "", true},
		{"empty", "", "", true},
		{"wrong case", "INSERT", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (input from tt)

			// Act
			got, err := ParseScenario(tt.in)

			// Assert
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseScenario(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ParseScenario(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAllScenariosOrder(t *testing.T) {
	// Arrange
	want := []Scenario{ScenarioInsert, ScenarioUpdate, ScenarioDelete, ScenarioSelect, ScenarioJoin, ScenarioUnion}

	// Act
	got := AllScenarios

	// Assert
	if len(got) != len(want) {
		t.Fatalf("AllScenarios len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllScenarios[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestProfilePresets(t *testing.T) {
	tests := []struct {
		name        string
		profile     Profile
		wantName    string
		wantPaced   bool
		wantWorkers int
	}{
		{"low", Low(), "low", true, 4},
		{"stress", Stress(), "stress", false, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			p := tt.profile

			// Act
			pool := p.Pool()

			// Assert
			if p.Name != tt.wantName || p.Workers != tt.wantWorkers {
				t.Errorf("profile = %s/%d workers, want %s/%d", p.Name, p.Workers, tt.wantName, tt.wantWorkers)
			}
			if (p.Interval > 0) != tt.wantPaced {
				t.Errorf("paced = %v, want %v", p.Interval > 0, tt.wantPaced)
			}
			if pool.MaxOpen != p.Workers || pool.MaxIdle != p.Workers {
				t.Errorf("pool = %+v, want %d open and idle", pool, p.Workers)
			}
			if p.Warmup <= 0 || p.Duration <= 0 || p.DeletePool <= 0 {
				t.Errorf("profile has non-positive durations or delete pool: %+v", p)
			}
		})
	}
}

func TestNewModel(t *testing.T) {
	// Arrange
	// (no input)

	// Act
	m := NewModel()

	// Assert
	if m.Name == "" || m.Title == "" || m.Fax == "" || m.Web == "" || m.Age == 0 || m.Counter == 0 {
		t.Fatalf("NewModel() has empty fields: %+v", m)
	}
}

func TestErrUnsupportedIsSentinel(t *testing.T) {
	// Arrange
	wrapped := errors.Join(errors.New("x"), ErrUnsupported)

	// Act
	ok := errors.Is(wrapped, ErrUnsupported)

	// Assert
	if !ok {
		t.Fatal("errors.Is(wrapped, ErrUnsupported) = false")
	}
}

func TestCities(t *testing.T) {
	// Arrange
	// (no input)

	// Act
	cities := Cities()

	// Assert
	if len(cities) != seedCities || cities[0] != "city 0" || cities[seedCities-1] != "city 49" {
		t.Fatalf("Cities() = %v", cities)
	}
}
