// Package report renders one benchmark result file and a hand-written rubric into a deterministic Markdown report.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

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
	Name        string `json:"name"`
	Display     string `json:"display"`
	Module      string `json:"module"`
	URL         string `json:"url"` // home page, printed in the environment table and under the pros and cons
	Label       string `json:"label"`
	VersionUsed string `json:"version_used"`
	// HonorableMention, when set, is the reason this library is measured and shown but not ranked
	// against the others (for example a driver locked to one database).
	HonorableMention string        `json:"honorable_mention,omitempty"`
	Pros             []string      `json:"pros"`
	Cons             []string      `json:"cons"`
	Flexibility      Flexibility   `json:"flexibility"`
	EaseOfUse        EaseOfUse     `json:"ease_of_use"`
	LearningCurve    LearningCurve `json:"learning_curve"`
	Community        Community     `json:"community"`
}

// CellNote is a caveat printed next to one library's numbers for one scenario.
type CellNote struct {
	Library  string `json:"library"`
	Scenario string `json:"scenario"`
	Note     string `json:"note"`
	// Feature marks a note that qualifies how the library supports the scenario; it is also
	// printed under the feature matrix.
	Feature bool `json:"feature,omitempty"`
}

// Rubric holds every hand-written part of the report.
type Rubric struct {
	Weights     Weights  `json:"weights"`
	Methodology []string `json:"methodology"`
	// CommunityChecked is the YYYY-MM-DD date the Community facts were read.
	CommunityChecked string        `json:"community_checked"`
	Libraries        []LibraryInfo `json:"libraries"`
	CellNotes        []CellNote    `json:"cell_notes"`
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
	checked, err := time.Parse(dateLayout, rb.CommunityChecked)
	if err != nil {
		return fmt.Errorf("community_checked %q is not a YYYY-MM-DD date", rb.CommunityChecked)
	}
	libs := make(map[string]bool, len(rb.Libraries))
	for _, l := range rb.Libraries {
		switch {
		case l.Name == "":
			return fmt.Errorf("library with empty name")
		case l.Display == "":
			return fmt.Errorf("library %q has empty display", l.Name)
		case strings.ContainsAny(l.Display, "\r\n"):
			return fmt.Errorf("library %q: display contains a newline", l.Name)
		case !strings.HasPrefix(l.URL, "https://") || strings.ContainsAny(l.URL, " \r\n"):
			return fmt.Errorf("library %q: url %q is not an https URL", l.Name, l.URL)
		case libs[l.Name]:
			return fmt.Errorf("duplicate library %q", l.Name)
		case strings.ContainsAny(l.HonorableMention, "\r\n"):
			return fmt.Errorf("library %q: honorable_mention contains a newline", l.Name)
		}
		for _, check := range []func(string) error{l.Flexibility.validate, l.EaseOfUse.validate, l.LearningCurve.validate} {
			if err := check(l.Name); err != nil {
				return err
			}
		}
		if err := l.Community.validate(l.Name, checked); err != nil {
			return err
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

// honorableMentions returns the names of the libraries kept out of the ranking.
func (rb Rubric) honorableMentions() map[string]bool {
	mentions := make(map[string]bool)
	for _, l := range rb.Libraries {
		if l.HonorableMention != "" {
			mentions[l.Name] = true
		}
	}
	return mentions
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

// featureNotes returns the notes of one cell that qualify feature support, in rubric order.
func (rb Rubric) featureNotes(library, scenario string) []string {
	var notes []string
	for _, n := range rb.CellNotes {
		if n.Feature && n.Library == library && n.Scenario == scenario {
			notes = append(notes, n.Note)
		}
	}
	return notes
}
