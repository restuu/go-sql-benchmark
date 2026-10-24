package report

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const validFlex = `"flexibility": {"capabilities": {"optional_filters": "native", "dynamic_ordering": "native", "conditional_join": "partial", "dynamic_projection": "raw_sql", "in_list_batch": "native", "dynamic_pagination": "static_only"}, "notes": ["note"]}`

const validQual = `"ease_of_use": {"criteria": {"setup_effort": "good", "query_writing": "fair", "type_safety": "poor", "debugging": "good"}, "notes": ["ease note"]}, "learning_curve": {"level": "moderate", "notes": ["curve note"]}, "community": {"repo": "owner/name", "stars": 10, "open_issues": 2, "last_release": "2026-01-02", "last_commit": "2026-02-03"}`

var validRubric = strings.ReplaceAll(validRubricTemplate, "FLEX", validFlex+", "+validQual)

const validRubricTemplate = `{
  "weights": {"p95": 1, "bytes_per_op": 1, "cpu_ns_per_op": 2},
  "methodology": ["line one"],
  "community_checked": "2026-10-24",
  "libraries": [
    {"name": "database_sql", "display": "database/sql", "version_used": "stdlib", "pros": ["p"], "cons": ["c"], FLEX},
    {"name": "ent", "display": "ent", "module": "ent", "label": "l", FLEX}
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
		{"honorable mention with a reason", strings.Replace(validRubric, `"module": "ent"`, `"module": "ent", "honorable_mention": "Postgres only"`, 1), ""},
		{"honorable mention reason with a newline", strings.Replace(validRubric, `"module": "ent"`, `"module": "ent", "honorable_mention": "a\nb"`, 1), "honorable_mention"},
		{"unknown field", strings.Replace(validRubric, `"methodology"`, `"methodolgy"`, 1), "unknown field"},
		{"invalid json", `{`, "parse"},
		{"duplicate library", strings.Replace(validRubric, `"name": "ent"`, `"name": "database_sql"`, 1), "duplicate library"},
		{"empty name", strings.Replace(validRubric, `"name": "ent"`, `"name": ""`, 1), "empty name"},
		{"display with a newline", strings.Replace(validRubric, `"display": "ent"`, `"display": "e\nnt"`, 1), "display contains a newline"},
		{"empty display", strings.Replace(validRubric, `"display": "ent"`, `"display": ""`, 1), "empty display"},
		{"negative weight", strings.Replace(validRubric, `"p95": 1`, `"p95": -1`, 1), "negative weight"},
		{"weight sum overflows", strings.Replace(strings.Replace(validRubric, `"p95": 1`, `"p95": 1e308`, 1), `"cpu_ns_per_op": 2`, `"cpu_ns_per_op": 1e308`, 1), "overflows"},
		{"all zero weights", strings.Replace(strings.Replace(strings.Replace(validRubric, `"p95": 1`, `"p95": 0`, 1), `"bytes_per_op": 1`, `"bytes_per_op": 0`, 1), `"cpu_ns_per_op": 2`, `"cpu_ns_per_op": 0`, 1), "all zero"},
		{"note unknown scenario", strings.Replace(validRubric, `"scenario": "join"`, `"scenario": "joinn"`, 1), "unknown scenario"},
		{"flexibility missing", strings.Replace(validRubric, validFlex, `"flexibility": {"capabilities": {}, "notes": ["n"]}`, 1), "missing capability"},
		{"flexibility unknown capability", strings.Replace(validRubric, `"optional_filters"`, `"optional_filterz"`, 1), "unknown capability"},
		{"flexibility unknown level", strings.Replace(validRubric, `"native"`, `"nativ"`, 1), "unknown level"},
		{"flexibility no notes", strings.Replace(validRubric, `"notes": ["note"]`, `"notes": []`, 1), "no flexibility notes"},
		{"flexibility empty note", strings.Replace(validRubric, `"notes": ["note"]`, `"notes": [""]`, 1), "empty flexibility note"},
		{"ease missing criterion", strings.Replace(validRubric, `, "debugging": "good"}, "notes": ["ease note"]`, `}, "notes": ["ease note"]`, 1), "missing criterion"},
		{"ease unknown level", strings.Replace(validRubric, `"debugging": "good"`, `"debugging": ""`, 1), "ease of use: criterion \"debugging\" has unknown level"},
		{"ease blank note", strings.Replace(validRubric, `"ease note"`, `"  "`, 1), "empty ease of use note"},
		{"learning empty level", strings.Replace(validRubric, `"level": "moderate"`, `"level": ""`, 1), "learning curve has unknown level"},
		{"community negative issues", strings.Replace(validRubric, `"open_issues": 2`, `"open_issues": -2`, 1), "negative"},
		{"community impossible date", strings.Replace(validRubric, `"2026-02-03"`, `"2026-02-30"`, 1), "last_commit"},
		{"community newline note", strings.Replace(validRubric, `"last_commit": "2026-02-03"`, `"last_commit": "2026-02-03", "note": "a\nb"`, 1), "community note contains a newline"},
		{"ease unknown criterion", strings.Replace(validRubric, `"setup_effort"`, `"setup_effortz"`, 1), "unknown criterion"},
		{"ease no notes", strings.Replace(validRubric, `"notes": ["ease note"]`, `"notes": []`, 1), "no ease of use notes"},
		{"ease newline note", strings.Replace(validRubric, `"ease note"`, `"a\nb"`, 1), "newline"},
		{"learning unknown level", strings.Replace(validRubric, `"moderate"`, `"flat"`, 1), "learning curve"},
		{"learning no notes", strings.Replace(validRubric, `"notes": ["curve note"]`, `"notes": []`, 1), "no learning curve notes"},
		{"flexibility newline note", strings.Replace(validRubric, `"notes": ["note"]`, `"notes": ["a\nb"]`, 1), "newline"},
		{"community empty repo", strings.Replace(validRubric, `"repo": "owner/name"`, `"repo": ""`, 1), "empty repo"},
		{"community negative stars", strings.Replace(validRubric, `"stars": 10`, `"stars": -1`, 1), "negative"},
		{"community bad release date", strings.Replace(validRubric, `"2026-01-02"`, `"Jan 2"`, 1), "last_release"},
		{"community bad commit date", strings.Replace(validRubric, `"2026-02-03"`, `"soon"`, 1), "last_commit"},
		{"community release after the check date", strings.Replace(validRubric, `"2026-01-02"`, `"2026-10-25"`, 1), "last_release"},
		{"community commit after the check date", strings.Replace(validRubric, `"2026-02-03"`, `"2026-11-01"`, 1), "last_commit"},
		{"community dates on the check date", strings.Replace(strings.Replace(validRubric, `"2026-01-02"`, `"2026-10-24"`, 1), `"2026-02-03"`, `"2026-10-24"`, 1), ""},
		{"community checked missing", strings.Replace(validRubric, `"community_checked": "2026-10-24"`, `"community_checked": ""`, 1), "community_checked"},
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

func TestFlexibilityVocabulary(t *testing.T) {
	// Arrange
	wantCaps := []string{"optional_filters", "dynamic_ordering", "conditional_join", "dynamic_projection", "in_list_batch", "dynamic_pagination"}
	wantLevels := []string{"native", "partial", "raw_sql", "static_only"}

	// Act
	gotCaps := FlexCapabilities()
	gotLevels := FlexLevels()

	// Assert
	if !reflect.DeepEqual(gotCaps, wantCaps) {
		t.Fatalf("FlexCapabilities() = %v, want %v", gotCaps, wantCaps)
	}
	if !reflect.DeepEqual(gotLevels, wantLevels) {
		t.Fatalf("FlexLevels() = %v, want %v", gotLevels, wantLevels)
	}
}

func TestQualitativeVocabulary(t *testing.T) {
	// Arrange
	wantCriteria := []string{"setup_effort", "query_writing", "type_safety", "debugging"}
	wantEase := []string{"good", "fair", "poor"}
	wantCurve := []string{"gentle", "moderate", "steep"}

	// Act
	gotCriteria := EaseCriteria()
	gotEase := EaseLevels()
	gotCurve := CurveLevels()

	// Assert
	if !reflect.DeepEqual(gotCriteria, wantCriteria) {
		t.Fatalf("EaseCriteria() = %v, want %v", gotCriteria, wantCriteria)
	}
	if !reflect.DeepEqual(gotEase, wantEase) {
		t.Fatalf("EaseLevels() = %v, want %v", gotEase, wantEase)
	}
	if !reflect.DeepEqual(gotCurve, wantCurve) {
		t.Fatalf("CurveLevels() = %v, want %v", gotCurve, wantCurve)
	}
}
