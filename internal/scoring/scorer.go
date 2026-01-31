package scoring

import (
	"github.com/ingo/agent-readyness/pkg/types"
)

// Scorer computes scores from raw analysis metrics using configurable breakpoints.
type Scorer struct {
	Config *ScoringConfig
}

// Interpolate computes the score for a given raw value using piecewise linear
// interpolation over the provided breakpoints. Breakpoints must be sorted by
// Value in ascending order.
//
// Behavior:
//   - Empty breakpoints: returns rawValue as-is (neutral passthrough, capped at 5.0)
//   - Below first breakpoint: clamps to first breakpoint's score
//   - Above last breakpoint: clamps to last breakpoint's score
//   - Between breakpoints: linear interpolation
func Interpolate(breakpoints []Breakpoint, rawValue float64) float64 {
	if len(breakpoints) == 0 {
		return 5.0
	}

	// Clamp below first breakpoint
	if rawValue <= breakpoints[0].Value {
		return breakpoints[0].Score
	}

	// Clamp above last breakpoint
	last := breakpoints[len(breakpoints)-1]
	if rawValue >= last.Value {
		return last.Score
	}

	// Find enclosing segment and interpolate
	for i := 1; i < len(breakpoints); i++ {
		if rawValue <= breakpoints[i].Value {
			lo := breakpoints[i-1]
			hi := breakpoints[i]
			t := (rawValue - lo.Value) / (hi.Value - lo.Value)
			return lo.Score + t*(hi.Score-lo.Score)
		}
	}

	return last.Score
}

// computeComposite calculates the weighted composite score from category scores.
// It normalizes by the sum of active category weights (not 1.0), so that
// a project scoring 10/10 on all active categories gets a composite of 10.
// Categories with Score < 0 are skipped (unavailable).
func (s *Scorer) computeComposite(categories []types.CategoryScore) float64 {
	totalWeight := 0.0
	weightedSum := 0.0

	for _, cat := range categories {
		if cat.Score < 0 {
			continue
		}
		weightedSum += cat.Score * cat.Weight
		totalWeight += cat.Weight
	}

	if totalWeight == 0 {
		return 0
	}
	return weightedSum / totalWeight
}

// classifyTier returns the tier name for a given composite score.
// Tiers are checked in order (must be sorted by MinScore descending).
// Boundary semantics: score >= MinScore (inclusive lower bound).
func (s *Scorer) classifyTier(score float64) string {
	for _, tier := range s.Config.Tiers {
		if score >= tier.MinScore {
			return tier.Name
		}
	}
	return "Agent-Hostile"
}

// categoryScore computes the weighted average of sub-scores within a category.
// Sub-scores where Available == false are excluded, and their weight is
// redistributed among the remaining sub-scores. Returns 5.0 (neutral) if
// no sub-scores are available.
func categoryScore(subScores []types.SubScore) float64 {
	totalWeight := 0.0
	weightedSum := 0.0

	for _, ss := range subScores {
		if !ss.Available {
			continue
		}
		weightedSum += ss.Score * ss.Weight
		totalWeight += ss.Weight
	}

	if totalWeight == 0 {
		return 5.0
	}
	return weightedSum / totalWeight
}
