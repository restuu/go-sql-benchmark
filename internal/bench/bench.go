// Package bench defines the contract between the runner and a library adapter.
package bench

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// Scenario names one benchmarked operation.
type Scenario string

const (
	ScenarioInsert Scenario = "insert"
	ScenarioUpdate Scenario = "update"
	ScenarioDelete Scenario = "delete"
	ScenarioSelect Scenario = "select"
	ScenarioJoin   Scenario = "join"
	ScenarioUnion  Scenario = "union"
)

// AllScenarios lists every scenario in report order.
var AllScenarios = []Scenario{
	ScenarioInsert, ScenarioUpdate, ScenarioDelete, ScenarioSelect, ScenarioJoin, ScenarioUnion,
}

// Row counts derived from db/seed.sql; internal tests fail if the seed drifts from them.
const (
	// ordersPerCustomer is 10000 orders / 1000 customers.
	ordersPerCustomer = 10
	// rowsPerCityPerTable is 1000 rows / 50 cities in each of customers and suppliers.
	rowsPerCityPerTable = 20
	// seedCities is the number of distinct 'city N' values in the seed.
	seedCities = 50
	// unionTables is the number of tables combined by the UNION scenario.
	unionTables = 2
)

// ExpectedRows returns the row count every adapter call for the scenario must report.
func (s Scenario) ExpectedRows() int {
	switch s {
	case ScenarioJoin:
		return ordersPerCustomer
	case ScenarioUnion:
		return unionTables * rowsPerCityPerTable
	default:
		return 1
	}
}

// Cities returns every seeded city name, "city 0" to "city 49", for the UNION scenario.
func Cities() []string {
	cities := make([]string, seedCities)
	for i := range cities {
		cities[i] = "city " + strconv.Itoa(i)
	}
	return cities
}

// ParseScenario converts a lower-case scenario name.
func ParseScenario(name string) (Scenario, error) {
	for _, s := range AllScenarios {
		if string(s) == name {
			return s, nil
		}
	}
	return "", fmt.Errorf("bench: unknown scenario %q", name)
}

// Model is the row shape of the models table, kept flat so every library can map it itself.
type Model struct {
	ID      int
	Name    string
	Title   string
	Fax     string
	Web     string
	Age     int
	Right   bool
	Counter int64
}

// NewModel returns the row inserted by the INSERT scenario (same values as db/seed.sql).
func NewModel() Model {
	return Model{
		Name:    "Orm Benchmark",
		Title:   "Just a Benchmark for fun",
		Fax:     "99909990",
		Web:     "http://blog.milkpod29.me",
		Age:     100,
		Right:   true,
		Counter: 1000,
	}
}

// PoolConfig is the connection pool every adapter must apply.
type PoolConfig struct {
	MaxOpen int
	MaxIdle int
}

// ErrUnsupported is returned by an adapter method the library cannot express natively.
var ErrUnsupported = errors.New("bench: scenario unsupported")

// Adapter is one library under test. Each method runs a single scenario call and returns the
// number of rows affected (writes) or scanned (reads). Reads must scan every selected column
// into Go values and must not use explicit Prepare.
type Adapter interface {
	Name() string
	Init(ctx context.Context, dsn string, pool PoolConfig) error
	Close() error
	Insert(ctx context.Context, m Model) (int, error)
	Update(ctx context.Context, id int, m Model) (int, error)
	Delete(ctx context.Context, id int) (int, error)
	Select(ctx context.Context, id int) (int, error)
	Join(ctx context.Context, customerID int) (int, error)
	Union(ctx context.Context, city string) (int, error)
}
