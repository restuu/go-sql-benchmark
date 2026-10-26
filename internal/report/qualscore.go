package report

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

// Importance weights of the qualitative ranking, as given by the project owner on a 1 to 10 scale.
// Only their ratios matter; the report prints them normalised.
const (
	flexibilityWeight = 8
	easeOfUseWeight   = 7
	communityWeight   = 5
	typeSafetyWeight  = 9
)

const (
	// A last commit or release this many days before community_checked or less earns full marks;
	// the score falls linearly to zero at recencyZeroDays.
	recencyFullDays = 30
	recencyZeroDays = 730
	// issuesPer1000StarsCap is the open-issue density (issues per 1,000 stars) that scores zero.
	issuesPer1000StarsCap = 100.0
	hoursPerDay           = 24
	starsPerThousand      = 1000.0
	// totalPrecision rounds totals to 12 decimals so mathematically equal totals compare equal.
	totalPrecision = 1e12
)

// Linear scales from 1 (best) to 0 (worst).
var (
	flexLevelScores  = map[string]float64{"native": 1, "partial": 2.0 / 3, "raw_sql": 1.0 / 3, "static_only": 0}
	easeLevelScores  = map[string]float64{"good": 1, "fair": 0.5, "poor": 0}
	curveLevelScores = map[string]float64{"gentle": 1, "moderate": 0.5, "steep": 0}
)

// qualScore is the numeric reading of one library's hand-written qualitative entries, each part
// and the total on a 0 to 1 scale (higher is better).
type qualScore struct {
	library                                         string
	flexibility, ease, community, typeSafety, total float64
}

// scoreQualitative turns the rubric levels and community facts of one library into numbers.
// Flexibility is the mean over its capabilities. Ease of use is the mean over the ease criteria
// plus the learning curve as one more criterion. Type safety is its own rated level and the
// fourth weighted part. Community is the mean of last-commit recency,
// last-release recency and open-issue density (stars are left out because they reward age).
// checked is Rubric.CommunityChecked.
func scoreQualitative(info LibraryInfo, checked time.Time) (qualScore, error) {
	flexParts, err := levelScores(info.Flexibility.Capabilities, FlexCapabilities(), flexLevelScores)
	if err != nil {
		return qualScore{}, fmt.Errorf("library %q: flexibility: %w", info.Name, err)
	}
	easeParts, err := levelScores(info.EaseOfUse.Criteria, EaseCriteria(), easeLevelScores)
	if err != nil {
		return qualScore{}, fmt.Errorf("library %q: ease of use: %w", info.Name, err)
	}
	curve, ok := curveLevelScores[info.LearningCurve.Level]
	if !ok {
		return qualScore{}, fmt.Errorf("library %q: learning curve: unknown level %q", info.Name, info.LearningCurve.Level)
	}
	typeSafety, ok := easeLevelScores[info.TypeSafety]
	if !ok {
		return qualScore{}, fmt.Errorf("library %q: type safety: unknown level %q", info.Name, info.TypeSafety)
	}
	flex, ease := mean(flexParts), mean(append(easeParts, curve))

	commit, err := recencyScore(info.Community.LastCommit, checked)
	if err != nil {
		return qualScore{}, fmt.Errorf("library %q: community last_commit: %w", info.Name, err)
	}
	release, err := recencyScore(info.Community.LastRelease, checked)
	if err != nil {
		return qualScore{}, fmt.Errorf("library %q: community last_release: %w", info.Name, err)
	}
	community := mean([]float64{commit, release, issueScore(info.Community)})

	total := (flexibilityWeight*flex + easeOfUseWeight*ease + communityWeight*community + typeSafetyWeight*typeSafety) /
		float64(flexibilityWeight+easeOfUseWeight+communityWeight+typeSafetyWeight)
	total = math.Round(total*totalPrecision) / totalPrecision // equal ratings tie despite summation order
	return qualScore{library: info.Name, flexibility: flex, ease: ease, community: community, typeSafety: typeSafety, total: total}, nil
}

// levelScores maps the rating of each name in names to its score; an unknown or missing level is
// an error, so a new level in the vocabulary cannot silently score 0.
func levelScores(ratings map[string]string, names []string, scale map[string]float64) ([]float64, error) {
	out := make([]float64, len(names))
	for i, name := range names {
		score, ok := scale[ratings[name]]
		if !ok {
			return nil, fmt.Errorf("%s: unknown level %q", name, ratings[name])
		}
		out[i] = score
	}
	return out, nil
}

func mean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// recencyScore is 1 for a date within recencyFullDays of checked and 0 from recencyZeroDays on.
func recencyScore(date string, checked time.Time) (float64, error) {
	t, err := time.Parse(dateLayout, date)
	if err != nil {
		return 0, fmt.Errorf("%q is not a YYYY-MM-DD date: %w", date, err)
	}
	days := checked.Sub(t).Hours() / hoursPerDay
	switch {
	case days <= recencyFullDays:
		return 1, nil
	case days >= recencyZeroDays:
		return 0, nil
	}
	return 1 - (days-recencyFullDays)/(recencyZeroDays-recencyFullDays), nil
}

// issueScore is 1 with no open issues and falls linearly to 0 at issuesPer1000StarsCap issues per
// 1,000 stars. A repository with no stars scores 0 as soon as it has an open issue.
func issueScore(c Community) float64 {
	if c.OpenIssues == 0 {
		return 1
	}
	if c.Stars == 0 {
		return 0
	}
	density := float64(c.OpenIssues) / (float64(c.Stars) / starsPerThousand)
	return 1 - min(density, issuesPer1000StarsCap)/issuesPer1000StarsCap
}

// writeQualitativeRanking prints the ranking of the qualitative scores. It is separate from the
// measured score, Top 5 and benchcmp, and says so next to the table. Honorable mentions are scored
// but left unranked, as in the measured ranking.
func writeQualitativeRanking(sb *strings.Builder, rb Rubric, r ranking) error {
	checked, err := time.Parse(dateLayout, rb.CommunityChecked)
	if err != nil {
		return fmt.Errorf("report: community_checked %q: %w", rb.CommunityChecked, err)
	}
	pool, err := qualScores(rb, r.pool(), checked)
	if err != nil {
		return err
	}
	guests, err := qualScores(rb, r.mentions(), checked)
	if err != nil {
		return err
	}

	totalWeight := float64(flexibilityWeight + easeOfUseWeight + communityWeight + typeSafetyWeight)
	sb.WriteString("## Qualitative ranking (judgement)\n\n")
	fmt.Fprintf(sb, "A second, separate ranking of the four weighted parts of the qualitative sections above. It is not part of the measured score, the Top 5 or the stability check. "+
		"The levels behind it are the author's judgement and the score only turns them into numbers, so it looks more objective than it is.\n\n"+
		"Weights: flexibility %s, ease of use %s (the learning curve counts as one more criterion), community %s, type safety %s. "+
		"Levels count linearly from 1 (best) to 0 (worst). Community is the mean of last-commit recency and last-release recency "+
		"(1 within %d days of %s, 0 after %d days) and open issues per 1,000 stars (1 with none, 0 at %.0f or more); stars are left out because they reward age. "+
		"Equal totals share a rank and are listed alphabetically.\n\n",
		share(flexibilityWeight, totalWeight), share(easeOfUseWeight, totalWeight), share(communityWeight, totalWeight), share(typeSafetyWeight, totalWeight),
		recencyFullDays, rb.CommunityChecked, recencyZeroDays, issuesPer1000StarsCap)
	sb.WriteString("| Rank | Library | Total | Flexibility | Ease of use | Community | Type safety |\n|---|---|---|---|---|---|---|\n")
	for i, rank := range qualRanks(pool) {
		writeQualRow(sb, rb, fmt.Sprint(rank), pool[i])
	}
	for _, q := range guests {
		writeQualRow(sb, rb, "-", q)
	}
	if len(guests) > 0 {
		sb.WriteString("\nLibraries marked `-` are honorable mentions and are not ranked.\n")
	}
	return nil
}

// qualRanks numbers scores sorted best first: equal totals share a rank and the next distinct
// total skips ahead (1, 2, 2, 4), as in the measured ranking.
func qualRanks(scores []qualScore) []int {
	ranks := make([]int, len(scores))
	for i := range scores {
		ranks[i] = i + 1
		if i > 0 && scores[i].total == scores[i-1].total {
			ranks[i] = ranks[i-1]
		}
	}
	return ranks
}

func share(weight int, total float64) string {
	return fixed(float64(weight)/total, scoreDecimals)
}

func writeQualRow(sb *strings.Builder, rb Rubric, rank string, q qualScore) {
	fmt.Fprintf(sb, "| %s | %s | %s | %s | %s | %s | %s |\n", rank, escapeCell(displayName(rb, q.library)),
		fixed(q.total, scoreDecimals), fixed(q.flexibility, scoreDecimals), fixed(q.ease, scoreDecimals), fixed(q.community, scoreDecimals), fixed(q.typeSafety, scoreDecimals))
}

// qualScores scores the given libraries and sorts them by total, best first, ties by display name.
func qualScores(rb Rubric, libs []libScore, checked time.Time) ([]qualScore, error) {
	out := make([]qualScore, 0, len(libs))
	for _, l := range libs {
		info, _ := rb.library(l.library)
		q, err := scoreQualitative(info, checked)
		if err != nil {
			return nil, fmt.Errorf("report: %w", err)
		}
		out = append(out, q)
	}
	slices.SortStableFunc(out, func(a, b qualScore) int {
		switch {
		case a.total > b.total:
			return -1
		case a.total < b.total:
			return 1
		}
		return strings.Compare(displayName(rb, a.library), displayName(rb, b.library))
	})
	return out, nil
}
