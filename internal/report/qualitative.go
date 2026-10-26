package report

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// EaseOfUse is the hand-written rating of how much work a library asks for day to day.
// Like Flexibility it is never ranked and never reaches the numeric score.
type EaseOfUse struct {
	// Criteria maps each criterion from EaseCriteria to one level from EaseLevels.
	Criteria map[string]string `json:"criteria"`
	Notes    []string          `json:"notes"`
}

// LearningCurve is one hand-written level plus what a newcomer must already know.
type LearningCurve struct {
	Level string   `json:"level"`
	Notes []string `json:"notes"`
}

// Community holds checkable facts about a library's repository; there is deliberately no rating.
// The check date is Rubric.CommunityChecked.
type Community struct {
	Repo        string `json:"repo"`
	Stars       int    `json:"stars"`
	OpenIssues  int    `json:"open_issues"`
	LastRelease string `json:"last_release"`
	LastCommit  string `json:"last_commit"`
	Note        string `json:"note,omitempty"`
}

const dateLayout = "2006-01-02"

var (
	easeCriteria = []string{"setup_effort", "query_writing", "debugging"}
	easeLevels   = []string{"good", "fair", "poor"}
	curveLevels  = []string{"gentle", "moderate", "steep"}

	easeCriterionLabels = map[string]string{
		"setup_effort":  "setup effort",
		"query_writing": "query writing",
		"debugging":     "debugging",
	}
)

// EaseCriteria returns the criteria a rubric must rate for ease of use, in report order.
func EaseCriteria() []string { return slices.Clone(easeCriteria) }

// EaseLevels returns the levels an ease-of-use criterion may take.
func EaseLevels() []string { return slices.Clone(easeLevels) }

// CurveLevels returns the levels a learning curve may take.
func CurveLevels() []string { return slices.Clone(curveLevels) }

// validateNotes requires at least one note and rejects blank ones and ones with a newline,
// which would break the Markdown bullet list.
func validateNotes(library, what string, notes []string) error {
	if len(notes) == 0 {
		return fmt.Errorf("library %q: no %s notes", library, what)
	}
	for _, n := range notes {
		switch {
		case strings.TrimSpace(n) == "":
			return fmt.Errorf("library %q: empty %s note", library, what)
		case strings.ContainsAny(n, "\r\n"):
			return fmt.Errorf("library %q: %s note contains a newline", library, what)
		}
	}
	return nil
}

// validateRatings requires every name in want, no others, and a level from levels for each.
// Names are checked in sorted order so the reported error does not depend on map order.
func validateRatings(library, what, noun string, got map[string]string, want, levels []string) error {
	names := make([]string, 0, len(got))
	for name := range got {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if !slices.Contains(want, name) {
			return fmt.Errorf("library %q: %s: unknown %s %q", library, what, noun, name)
		}
		if !slices.Contains(levels, got[name]) {
			return fmt.Errorf("library %q: %s: %s %q has unknown level %q", library, what, noun, name, got[name])
		}
	}
	for _, name := range want {
		if _, ok := got[name]; !ok {
			return fmt.Errorf("library %q: %s: missing %s %q", library, what, noun, name)
		}
	}
	return nil
}

func (e EaseOfUse) validate(library string) error {
	if err := validateRatings(library, "ease of use", "criterion", e.Criteria, easeCriteria, easeLevels); err != nil {
		return err
	}
	return validateNotes(library, "ease of use", e.Notes)
}

func (c LearningCurve) validate(library string) error {
	if !slices.Contains(curveLevels, c.Level) {
		return fmt.Errorf("library %q: learning curve has unknown level %q", library, c.Level)
	}
	return validateNotes(library, "learning curve", c.Notes)
}

// validate checks the counts and dates; checked is Rubric.CommunityChecked, and a fact cannot be
// dated after the day it was read.
func (c Community) validate(library string, checked time.Time) error {
	switch {
	case strings.TrimSpace(c.Repo) == "":
		return fmt.Errorf("library %q: community has empty repo", library)
	case c.Stars < 0 || c.OpenIssues < 0:
		return fmt.Errorf("library %q: community has a negative count", library)
	}
	if strings.ContainsAny(c.Note, "\r\n") {
		return fmt.Errorf("library %q: community note contains a newline", library)
	}
	dates := []struct{ field, value string }{{"last_release", c.LastRelease}, {"last_commit", c.LastCommit}}
	for _, d := range dates {
		date, err := time.Parse(dateLayout, d.value)
		if err != nil {
			return fmt.Errorf("library %q: community %s %q is not a YYYY-MM-DD date: %w", library, d.field, d.value, err)
		}
		if date.After(checked) {
			return fmt.Errorf("library %q: community %s %s is after community_checked %s", library, d.field, d.value, checked.Format(dateLayout))
		}
	}
	return nil
}

// inRunLibraries returns the rubric entries whose library is in the run, in rubric order so the
// qualitative sections carry no ranking signal.
func inRunLibraries(rb Rubric, r ranking) []LibraryInfo {
	inRun := make(map[string]bool, len(r.libs))
	for _, l := range r.libs {
		inRun[l.library] = true
	}
	var shown []LibraryInfo
	for _, info := range rb.Libraries {
		if inRun[info.Name] {
			shown = append(shown, info)
		}
	}
	return shown
}

func writeNotes(sb *strings.Builder, libs []LibraryInfo, notes func(LibraryInfo) []string) {
	for _, info := range libs {
		fmt.Fprintf(sb, "**%s**\n\n", info.Display)
		for _, n := range notes(info) {
			sb.WriteString("- " + n + "\n")
		}
		sb.WriteString("\n")
	}
}

func writeEaseOfUse(sb *strings.Builder, rb Rubric, r ranking) {
	libs := inRunLibraries(rb, r)
	sb.WriteString("## Ease of use (qualitative)\n\n")
	sb.WriteString("Hand-written judgement of the day-to-day work a library asks for. It is not part of the score, the Top 5 or the stability check. Levels for setup effort and query writing: `good` = little work, `fair` = some manual work, `poor` = mostly manual. Levels for debugging: `good` = the SQL that ran and its errors are visible with little setup, `fair` = visible after configuring logging or hooks, `poor` = hard to see what ran.\n\n")
	sb.WriteString("Criteria: setup effort = codegen, config and boilerplate before the first query; query writing = code a typical query takes; debugging = SQL visibility, error messages and logging hooks.\n\n")
	sb.WriteString("| Library |")
	for _, c := range easeCriteria {
		sb.WriteString(" " + easeCriterionLabels[c] + " |")
	}
	sb.WriteString("\n|---|" + strings.Repeat("---|", len(easeCriteria)) + "\n")
	for _, info := range libs {
		fmt.Fprintf(sb, "| %s |", escapeCell(info.Display))
		for _, c := range easeCriteria {
			sb.WriteString(" " + info.EaseOfUse.Criteria[c] + " |")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	writeNotes(sb, libs, func(i LibraryInfo) []string { return i.EaseOfUse.Notes })
}

func writeLearningCurve(sb *strings.Builder, rb Rubric, r ranking) {
	libs := inRunLibraries(rb, r)
	sb.WriteString("## Learning curve (qualitative)\n\n")
	sb.WriteString("Hand-written judgement of how long a Go developer who knows SQL needs before writing production queries. It is not part of the score. `gentle` = SQL and the Go standard library are enough, `moderate` = one new API or tool to learn, `steep` = a framework, DSL or several tools to learn.\n\n")
	sb.WriteString("| Library | Learning curve |\n|---|---|\n")
	for _, info := range libs {
		fmt.Fprintf(sb, "| %s | %s |\n", escapeCell(info.Display), info.LearningCurve.Level)
	}
	sb.WriteString("\n")
	writeNotes(sb, libs, func(i LibraryInfo) []string { return i.LearningCurve.Notes })
}

func writeTypeSafety(sb *strings.Builder, rb Rubric, r ranking) {
	libs := inRunLibraries(rb, r)
	sb.WriteString("## Type safety (qualitative)\n\n")
	sb.WriteString("Hand-written judgement of how many mistakes the Go compiler catches before the program runs, such as a misspelled column name or a wrong type. It is not part of the measured score, the Top 5 or the stability check, but it is one of the four weighted parts of the qualitative ranking. `good` = checked at compile time, `fair` = checked only partly, `poor` = mostly checked at run time.\n\n")
	sb.WriteString("| Library | Type safety |\n|---|---|\n")
	for _, info := range libs {
		fmt.Fprintf(sb, "| %s | %s |\n", escapeCell(info.Display), info.TypeSafety)
	}
	sb.WriteString("\n")
}

func writeCommunity(sb *strings.Builder, rb Rubric, r ranking) {
	libs := inRunLibraries(rb, r)
	sb.WriteString("## Community support (facts)\n\n")
	fmt.Fprintf(sb, "Repository facts read on %s, with no rating and no part in the score; they go stale, so re-check before relying on them. Open issues exclude pull requests.\n\n", rb.CommunityChecked)
	sb.WriteString("| Library | Repository | Stars | Last release | Last commit | Open issues |\n|---|---|---|---|---|---|\n")
	for _, info := range libs {
		c := info.Community
		fmt.Fprintf(sb, "| %s | %s | %d | %s | %s | %d |\n", escapeCell(info.Display), escapeCell(c.Repo), c.Stars, c.LastRelease, c.LastCommit, c.OpenIssues)
	}
	sb.WriteString("\n")
	for _, info := range libs {
		if info.Community.Note != "" {
			fmt.Fprintf(sb, "- **%s**: %s\n", info.Display, info.Community.Note)
		}
	}
}
