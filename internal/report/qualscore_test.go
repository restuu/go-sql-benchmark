package report

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"
)

const scoreEpsilon = 1e-9

var checkedAt = time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)

// uniformInfo rates every flexibility capability and every ease criterion at one level.
func uniformInfo(flex, ease, curve string, c Community) LibraryInfo {
	capabilities := map[string]string{}
	for _, name := range FlexCapabilities() {
		capabilities[name] = flex
	}
	criteria := map[string]string{}
	for _, name := range EaseCriteria() {
		criteria[name] = ease
	}
	return LibraryInfo{
		Name:          "lib",
		Flexibility:   Flexibility{Capabilities: capabilities},
		EaseOfUse:     EaseOfUse{Criteria: criteria},
		LearningCurve: LearningCurve{Level: curve},
		Community:     c,
	}
}

func TestScoreQualitative(t *testing.T) {
	tests := []struct {
		name                       string
		info                       LibraryInfo
		wantFlex, wantEase, wantCm float64
		wantTotal                  float64
	}{
		{
			"best everywhere",
			uniformInfo("native", "good", "gentle", Community{Stars: 1000, OpenIssues: 0, LastRelease: "2026-10-24", LastCommit: "2026-10-24"}),
			1, 1, 1, 1,
		},
		{
			"worst everywhere",
			uniformInfo("static_only", "poor", "steep", Community{Stars: 1000, OpenIssues: 100, LastRelease: "2020-01-01", LastCommit: "2020-01-01"}),
			0, 0, 0, 0,
		},
		{
			"middle levels",
			uniformInfo("partial", "fair", "moderate", Community{Stars: 1000, OpenIssues: 50, LastRelease: "2025-10-09", LastCommit: "2025-10-09"}),
			2.0 / 3, 0.5, 0.5, // 380 days old scores 0.5 on recency, 50 issues per 1000 stars scores 0.5
			(8*2.0/3 + 7*0.5 + 5*0.5) / 20,
		},
		{
			"mixed ease criteria and learning curve",
			func() LibraryInfo {
				i := uniformInfo("native", "good", "steep", Community{Stars: 1000, LastRelease: "2026-10-24", LastCommit: "2026-10-24"})
				i.EaseOfUse.Criteria["debugging"] = "poor"
				i.EaseOfUse.Criteria["type_safety"] = "fair"
				return i
			}(),
			1, (1 + 1 + 0.5 + 0 + 0) / 5.0, 1,
			(8*1 + 7*((1+1+0.5+0+0)/5.0) + 5*1) / 20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// (inputs are in the table)

			// Act
			got, err := scoreQualitative(tt.info, checkedAt)

			// Assert
			if err != nil {
				t.Fatalf("scoreQualitative() error = %v", err)
			}
			for field, pair := range map[string][2]float64{
				"flexibility": {got.flexibility, tt.wantFlex},
				"ease":        {got.ease, tt.wantEase},
				"community":   {got.community, tt.wantCm},
				"total":       {got.total, tt.wantTotal},
			} {
				if math.Abs(pair[0]-pair[1]) > scoreEpsilon {
					t.Errorf("%s = %v, want %v", field, pair[0], pair[1])
				}
			}
		})
	}
}

func TestRecencyScore(t *testing.T) {
	tests := []struct {
		name string
		days int
		want float64
	}{
		{"today", 0, 1},
		{"edge of full marks", 30, 1},
		{"halfway", 380, 0.5},
		{"edge of zero", 730, 0},
		{"older", 2000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			date := checkedAt.AddDate(0, 0, -tt.days).Format(dateLayout)

			// Act
			got, err := recencyScore(date, checkedAt)

			// Assert
			if err != nil || math.Abs(got-tt.want) > scoreEpsilon {
				t.Fatalf("recencyScore(%s) = %v, %v; want %v", date, got, err, tt.want)
			}
		})
	}
}

func TestIssueScoreHandlesZeroStars(t *testing.T) {
	// Arrange
	noStars := Community{Stars: 0, OpenIssues: 0}
	noStarsWithIssues := Community{Stars: 0, OpenIssues: 3}

	// Act
	clean := issueScore(noStars)
	busy := issueScore(noStarsWithIssues)

	// Assert
	if clean != 1 || busy != 0 {
		t.Fatalf("issueScore = %v and %v, want 1 and 0", clean, busy)
	}
}

func TestScoreQualitativeRejectsBadDate(t *testing.T) {
	// Arrange
	info := uniformInfo("native", "good", "gentle", Community{LastRelease: "soon", LastCommit: "2026-10-24"})

	// Act
	_, err := scoreQualitative(info, checkedAt)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "last_release") {
		t.Fatalf("scoreQualitative() error = %v, want one naming last_release", err)
	}
}

func TestRenderQualitativeRanking(t *testing.T) {
	const heading = "## Qualitative ranking (judgement)"
	tests := []struct {
		name     string
		mutate   func(rb *Rubric)
		wantRows []string
		wantNot  []string
	}{
		{
			"every library ranked by total",
			func(rb *Rubric) {},
			[]string{"| 1 | ", "| 2 | ", "| 3 | "},
			[]string{"| - |"},
		},
		{
			"honorable mention is scored but unranked",
			func(rb *Rubric) {
				for i := range rb.Libraries {
					if rb.Libraries[i].Name == "sqlx" {
						rb.Libraries[i].HonorableMention = "Fixture reason."
					}
				}
			},
			[]string{"| - | sqlx |", "Libraries marked `-` are honorable mentions"},
			[]string{"| 3 |"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			run, rb := loadFixture(t)
			rb.Libraries = slices.Clone(rb.Libraries)
			tt.mutate(&rb)

			// Act
			got, err := renderFull(run, rb)

			// Assert
			if err != nil {
				t.Fatalf("renderFull() error = %v", err)
			}
			body := section(t, got, heading)
			for _, want := range append(tt.wantRows, "not part of the measured score", "author's judgement", "Weights: flexibility 0.40, ease of use 0.35", "community 0.25", "| Rank | Library | Total | Flexibility | Ease of use | Community |") {
				if !strings.Contains(body, want) {
					t.Errorf("section missing %q:\n%s", want, body)
				}
			}
			for _, bad := range tt.wantNot {
				if strings.Contains(body, bad) {
					t.Errorf("section contains %q:\n%s", bad, body)
				}
			}
			if strings.Index(got, heading) < strings.Index(got, "## Community support (facts)") {
				t.Error("qualitative ranking must come after the Community support section")
			}
		})
	}
}

func TestRenderQualitativeRankingOrder(t *testing.T) {
	// Arrange: the best library on every rating must rank first whatever the rubric order is.
	run, rb := loadFixture(t)
	rb.Libraries = slices.Clone(rb.Libraries)
	for i := range rb.Libraries {
		if rb.Libraries[i].Name != "sqlx" {
			continue
		}
		best := uniformInfo("native", "good", "gentle", Community{Repo: "x/y", Stars: 1000, LastRelease: "2026-10-24", LastCommit: "2026-10-24"})
		rb.Libraries[i].Flexibility = best.Flexibility
		rb.Libraries[i].EaseOfUse.Criteria = best.EaseOfUse.Criteria
		rb.Libraries[i].LearningCurve.Level = best.LearningCurve.Level
		rb.Libraries[i].Community = best.Community
	}

	// Act
	got, err := renderFull(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("renderFull() error = %v", err)
	}
	body := section(t, got, "## Qualitative ranking (judgement)")
	if !strings.Contains(body, "| 1 | sqlx | 1.00 | 1.00 | 1.00 | 1.00 |") {
		t.Errorf("sqlx should rank first with 1.00 everywhere:\n%s", body)
	}
}

func TestRenderQualitativeRankingDoesNotChangeEarlierSections(t *testing.T) {
	// Arrange
	run, rb := loadFixture(t)
	changed := rb
	changed.Libraries = slices.Clone(rb.Libraries)
	for i := range changed.Libraries {
		changed.Libraries[i].Community.OpenIssues += 100000
	}

	// Act
	before, err := renderFull(run, rb)
	if err != nil {
		t.Fatal(err)
	}
	after, err := renderFull(run, changed)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	cut := func(s string) string { return s[:strings.Index(s, "## Community support (facts)")] }
	if cut(before) != cut(after) {
		t.Error("changing community facts changed the report before the Community support section")
	}
	rank := func(s string) string { return s[strings.Index(s, "## Qualitative ranking"):] }
	if rank(before) == rank(after) {
		t.Error("changing community facts did not change the qualitative ranking")
	}
}

func TestScoreQualitativeRejectsUnknownLevel(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(i *LibraryInfo)
		want   string
	}{
		{"flexibility", func(i *LibraryInfo) { i.Flexibility.Capabilities["optional_filters"] = "bogus" }, "bogus"},
		{"ease of use", func(i *LibraryInfo) { i.EaseOfUse.Criteria["debugging"] = "bogus" }, "bogus"},
		{"learning curve", func(i *LibraryInfo) { i.LearningCurve.Level = "bogus" }, "bogus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			info := uniformInfo("native", "good", "gentle", Community{Stars: 1000, LastRelease: "2026-10-24", LastCommit: "2026-10-24"})
			tt.mutate(&info)

			// Act
			_, err := scoreQualitative(info, checkedAt)

			// Assert
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("scoreQualitative() error = %v, want one containing %q", err, tt.want)
			}
		})
	}
}

func TestScaleCoversEveryLevel(t *testing.T) {
	// Arrange
	scales := []struct {
		name   string
		levels []string
		scale  map[string]float64
	}{
		{"flexibility", FlexLevels(), flexLevelScores},
		{"ease of use", EaseLevels(), easeLevelScores},
		{"learning curve", CurveLevels(), curveLevelScores},
	}
	for _, tt := range scales {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			missing := ""
			for _, level := range tt.levels {
				if _, ok := tt.scale[level]; !ok {
					missing = level
				}
			}

			// Assert
			if missing != "" || len(tt.scale) != len(tt.levels) {
				t.Fatalf("scale has %d entries for levels %v (missing %q)", len(tt.scale), tt.levels, missing)
			}
		})
	}
}

func TestRenderQualitativeRankingSharesRankOnEqualTotals(t *testing.T) {
	// Arrange: database/sql gets ent's ratings and facts, so the two tie exactly.
	run, rb := loadFixture(t)
	rb.Libraries = slices.Clone(rb.Libraries)
	var ent LibraryInfo
	for _, l := range rb.Libraries {
		if l.Name == "ent" {
			ent = l
		}
	}
	for i := range rb.Libraries {
		if rb.Libraries[i].Name == "database_sql" {
			rb.Libraries[i].Flexibility = ent.Flexibility
			rb.Libraries[i].EaseOfUse = ent.EaseOfUse
			rb.Libraries[i].LearningCurve = ent.LearningCurve
			rb.Libraries[i].Community = ent.Community
		}
	}

	// Act
	got, err := renderFull(run, rb)

	// Assert
	if err != nil {
		t.Fatalf("renderFull() error = %v", err)
	}
	body := section(t, got, "## Qualitative ranking (judgement)")
	for _, want := range []string{"| 1 | database/sql |", "| 1 | ent |", "| 3 | sqlx |"} {
		if !strings.Contains(body, want) {
			t.Errorf("section missing %q:\n%s", want, body)
		}
	}
}
