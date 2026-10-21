// Package report renders one benchmark result file and a hand-written rubric into a deterministic Markdown report.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// Weights scale each ranked metric in a library's score; zero drops the metric.
type Weights struct {
	P95        float64 `json:"p95"`
	BytesPerOp float64 `json:"bytes_per_op"`
	CPUNsPerOp float64 `json:"cpu_ns_per_op"`
}

// LibraryInfo is the hand-written description of one library.
type LibraryInfo struct {
	Name        string   `json:"name"`
	Display     string   `json:"display"`
	Module      string   `json:"module"`
	Label       string   `json:"label"`
	VersionUsed string   `json:"version_used"`
	Pros        []string `json:"pros"`
	Cons        []string `json:"cons"`
}

// CellNote is a caveat printed next to one library's numbers for one scenario.
type CellNote struct {
	Library  string `json:"library"`
	Scenario string `json:"scenario"`
	Note     string `json:"note"`
}

// Rubric holds every hand-written part of the report.
type Rubric struct {
	Weights     Weights       `json:"weights"`
	Methodology []string      `json:"methodology"`
	Libraries   []LibraryInfo `json:"libraries"`
	CellNotes   []CellNote    `json:"cell_notes"`
}

// LoadRubric reads and validates a rubric file; unknown fields are an error so a typo cannot drop a note.
func LoadRubric(path string) (Rubric, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Rubric{}, fmt.Errorf("report: read rubric %s: %w", path, err)
	}
	var rb Rubric
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rb); err != nil {
		return Rubric{}, fmt.Errorf("report: parse rubric %s: %w", path, err)
	}
	if err := rb.validate(); err != nil {
		return Rubric{}, fmt.Errorf("report: rubric %s: %w", path, err)
	}
	return rb, nil
}

func (rb Rubric) validate() error {
	w := []float64{rb.Weights.P95, rb.Weights.BytesPerOp, rb.Weights.CPUNsPerOp}
	sum := 0.0
	for _, v := range w {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return fmt.Errorf("negative weight or not finite: %v", v)
		}
		sum += v
	}
	if math.IsInf(sum, 0) {
		return fmt.Errorf("weight sum overflows")
	}
	if sum == 0 {
		return fmt.Errorf("weights are all zero")
	}
	libs := make(map[string]bool, len(rb.Libraries))
	for _, l := range rb.Libraries {
		switch {
		case l.Name == "":
			return fmt.Errorf("library with empty name")
		case l.Display == "":
			return fmt.Errorf("library %q has empty display", l.Name)
		case libs[l.Name]:
			return fmt.Errorf("duplicate library %q", l.Name)
		}
		libs[l.Name] = true
	}
	scenarios := make(map[string]bool, len(bench.AllScenarios))
	for _, s := range bench.AllScenarios {
		scenarios[string(s)] = true
	}
	for _, n := range rb.CellNotes {
		if !libs[n.Library] {
			return fmt.Errorf("cell note for unknown library %q", n.Library)
		}
		if !scenarios[n.Scenario] {
			return fmt.Errorf("cell note for unknown scenario %q", n.Scenario)
		}
	}
	return nil
}

func (rb Rubric) library(name string) (LibraryInfo, bool) {
	for _, l := range rb.Libraries {
		if l.Name == name {
			return l, true
		}
	}
	return LibraryInfo{}, false
}

// notesFor returns the notes of one cell in rubric order.
func (rb Rubric) notesFor(library, scenario string) []string {
	var notes []string
	for _, n := range rb.CellNotes {
		if n.Library == library && n.Scenario == scenario {
			notes = append(notes, n.Note)
		}
	}
	return notes
}
