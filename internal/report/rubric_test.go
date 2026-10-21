package report

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const validRubric = `{
  "weights": {"p95": 1, "bytes_per_op": 1, "cpu_ns_per_op": 2},
  "methodology": ["line one"],
  "libraries": [
    {"name": "database_sql", "display": "database/sql", "version_used": "stdlib", "pros": ["p"], "cons": ["c"]},
    {"name": "ent", "display": "ent", "module": "ent", "label": "l"}
  ],
  "cell_notes": [
    {"library": "ent", "scenario": "join", "note": "first"},
    {"library": "ent", "scenario": "join", "note": "second"},
    {"library": "ent", "scenario": "union", "note": "other"}
  ]
}`

func writeRubric(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rubric.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write rubric: %v", err)
	}
	return path
}

func TestLoadRubric(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"valid", validRubric, ""},
		{"unknown field", strings.Replace(validRubric, `"methodology"`, `"methodolgy"`, 1), "unknown field"},
		{"invalid json", `{`, "parse"},
		{"duplicate library", strings.Replace(validRubric, `"name": "ent"`, `"name": "database_sql"`, 1), "duplicate library"},
		{"empty name", strings.Replace(validRubric, `"name": "ent"`, `"name": ""`, 1), "empty name"},
		{"empty display", strings.Replace(validRubric, `"display": "ent"`, `"display": ""`, 1), "empty display"},
		{"negative weight", strings.Replace(validRubric, `"p95": 1`, `"p95": -1`, 1), "negative weight"},
		{"weight sum overflows", strings.Replace(strings.Replace(validRubric, `"p95": 1`, `"p95": 1e308`, 1), `"cpu_ns_per_op": 2`, `"cpu_ns_per_op": 1e308`, 1), "overflows"},
		{"all zero weights", strings.Replace(strings.Replace(strings.Replace(validRubric, `"p95": 1`, `"p95": 0`, 1), `"bytes_per_op": 1`, `"bytes_per_op": 0`, 1), `"cpu_ns_per_op": 2`, `"cpu_ns_per_op": 0`, 1), "all zero"},
		{"note unknown scenario", strings.Replace(validRubric, `"scenario": "join"`, `"scenario": "joinn"`, 1), "unknown scenario"},
		{"note unknown library", strings.Replace(validRubric, `"library": "ent", "scenario": "join", "note": "first"`, `"library": "nope", "scenario": "join", "note": "first"`, 1), "unknown library"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			path := writeRubric(t, tt.body)

			// Act
			rb, err := LoadRubric(path)

			// Assert
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("LoadRubric() error = %v", err)
				}
				if len(rb.Libraries) != 2 || rb.Weights.CPUNsPerOp != 2 {
					t.Fatalf("LoadRubric() = %+v", rb)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("LoadRubric() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadRubricMissingFile(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "none.json")

	// Act
	_, err := LoadRubric(path)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "report: read rubric") {
		t.Fatalf("LoadRubric() error = %v", err)
	}
}

func TestRubricLookups(t *testing.T) {
	// Arrange
	rb, err := LoadRubric(writeRubric(t, validRubric))
	if err != nil {
		t.Fatal(err)
	}

	// Act
	info, found := rb.library("ent")
	_, missing := rb.library("nope")
	joinNotes := rb.notesFor("ent", "join")
	noNotes := rb.notesFor("ent", "insert")

	// Assert
	if !found || info.Module != "ent" {
		t.Fatalf("library(ent) = %+v, %v", info, found)
	}
	if missing {
		t.Fatal("library(nope) found")
	}
	if want := []string{"first", "second"}; !reflect.DeepEqual(joinNotes, want) {
		t.Fatalf("notesFor(ent, join) = %v, want %v", joinNotes, want)
	}
	if len(noNotes) != 0 {
		t.Fatalf("notesFor(ent, insert) = %v, want none", noNotes)
	}
}
