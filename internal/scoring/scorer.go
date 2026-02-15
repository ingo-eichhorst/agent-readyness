// Package scoring converts raw analysis metrics to normalized scores (1-10 scale)
// using piecewise linear interpolation over configurable breakpoints.
//
// The scoring system provides a consistent, predictable mapping from raw values
// (complexity counts, percentages, file sizes) to user-facing scores that directly
// correlate with agent-readiness tiers. All metrics flow through the Interpolate
// function, ensuring uniform scoring behavior across categories.
//
// Scoring philosophy: Breakpoints are empirically derived from agent success rates
// across diverse codebases. For example, complexity >15 correlates with 3x higher
// agent error rates, so the breakpoint mapping reflects this empirical relationship.
package scoring

import (
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// evidenceTopN is the maximum number of evidence items retained per metric.
const evidenceTopN = 5

// Scorer computes scores from raw analysis metrics using configurable breakpoints.
type Scorer struct {
	Config *ScoringConfig
}

// metricExtractor extracts raw metric values from an AnalysisResult.
// Returns raw values, a set of unavailable metrics, and per-metric evidence items.
type metricExtractor func(ar *types.AnalysisResult) (
	rawValues map[string]float64,
	unavailable map[string]bool,
	evidence map[string][]types.EvidenceItem,
)

// metricExtractors maps category name to a function that extracts raw metric values.
var metricExtractors = map[string]metricExtractor{
	"C1": extractC1,
	"C2": extractC2,
	"C3": extractC3,
	"C4": extractC4,
	"C5": extractC5,
	"C6": extractC6,
	"C7": extractC7,
}

// defaultInterpolateScore is the fallback score when no breakpoints are defined.
const defaultInterpolateScore = 5.0

// Interpolate computes the score for a given raw value using piecewise linear
// interpolation over the provided breakpoints. Values below the first breakpoint
// use the first score; values above the last use the last score.
func Interpolate(breakpoints []Breakpoint, rawValue float64) float64 {
	if len(breakpoints) == 0 {
		return defaultInterpolateScore
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
func (s *Scorer) classifyTier(score float64) string {
	for _, tier := range s.Config.Tiers {
		if score >= tier.MinScore {
			return tier.Name
		}
	}
	return "Agent-Hostile"
}

// CategoryScore computes the weighted average of sub-scores within a category.
//
// Returns -1.0 if all sub-scores are unavailable (Score < 0), indicating
// the category cannot be scored.
func CategoryScore(subScores []types.SubScore) float64 {
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
		return -1.0
	}
	return weightedSum / totalWeight
}

// Score computes scored results from raw analysis metrics.
func (s *Scorer) Score(results []*types.AnalysisResult) (*types.ScoredResult, error) {
	var categories []types.CategoryScore

	for _, ar := range results {
		catConfig, ok := s.Config.Categories[ar.Category]
		if !ok {
			continue
		}

		extractor, ok := metricExtractors[ar.Category]
		if !ok {
			continue
		}

		rawValues, unavailable, evidence := extractor(ar)
		if rawValues == nil {
			categories = append(categories, types.CategoryScore{
				Name:   ar.Category,
				Weight: catConfig.Weight,
			})
			continue
		}

		subScores, score := scoreMetrics(catConfig, rawValues, unavailable, evidence)
		categories = append(categories, types.CategoryScore{
			Name:      ar.Category,
			Score:     score,
			Weight:    catConfig.Weight,
			SubScores: subScores,
		})
	}

	composite := s.computeComposite(categories)
	tier := s.classifyTier(composite)

	return &types.ScoredResult{
		Categories: categories,
		Composite:  composite,
		Tier:       tier,
	}, nil
}

// scoreMetrics is a generic scoring helper for any category.
func scoreMetrics(catConfig CategoryConfig, rawValues map[string]float64, unavailable map[string]bool, evidence map[string][]types.EvidenceItem) ([]types.SubScore, float64) {
	var subScores []types.SubScore

	for _, mt := range catConfig.Metrics {
		rv := rawValues[mt.Name]
		ev := evidence[mt.Name]
		if ev == nil {
			ev = make([]types.EvidenceItem, 0)
		}
		ss := types.SubScore{
			MetricName: mt.Name,
			RawValue:   rv,
			Weight:     mt.Weight,
			Available:  true,
			Evidence:   ev,
		}

		if unavailable[mt.Name] {
			ss.Available = false
			ss.Score = 0
		} else {
			ss.Score = Interpolate(mt.Breakpoints, rv)
		}

		subScores = append(subScores, ss)
	}

	score := CategoryScore(subScores)
	return subScores, score
}

// topNEvidence builds up to evidenceTopN evidence items from a source slice using a converter function.
func topNEvidence(count int, convert func(i int) types.EvidenceItem) []types.EvidenceItem {
	limit := evidenceTopN
	if count < limit {
		limit = count
	}
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		items[i] = convert(i)
	}
	return items
}

// ensureEvidenceKeys ensures all given keys have at least empty arrays in the evidence map.
func ensureEvidenceKeys(evidence map[string][]types.EvidenceItem, keys []string) {
	for _, key := range keys {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}
}

// avgMapValues computes the average of all values in a map[string]int.
// Returns 0 for nil or empty maps.
func avgMapValues(m map[string]int) float64 {
	if len(m) == 0 {
		return 0
	}
	sum := 0
	for _, v := range m {
		sum += v
	}
	return float64(sum) / float64(len(m))
}

// findMetric finds a MetricThresholds by name in a slice.
// Returns nil if not found.
func findMetric(metrics []MetricThresholds, name string) *MetricThresholds {
	for i := range metrics {
		if metrics[i].Name == name {
			return &metrics[i]
		}
	}
	return nil
}
