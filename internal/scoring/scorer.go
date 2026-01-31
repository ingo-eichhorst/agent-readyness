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

// Score computes scored results from raw analysis metrics.
// It dispatches each AnalysisResult to the appropriate category scorer
// based on the Category field, computes a weighted composite, and classifies a tier.
func (s *Scorer) Score(results []*types.AnalysisResult) (*types.ScoredResult, error) {
	var categories []types.CategoryScore

	for _, ar := range results {
		switch ar.Category {
		case "C1":
			categories = append(categories, s.scoreC1(ar))
		case "C3":
			categories = append(categories, s.scoreC3(ar))
		case "C6":
			categories = append(categories, s.scoreC6(ar))
		default:
			// Unknown categories are silently skipped
			continue
		}
	}

	composite := s.computeComposite(categories)
	tier := s.classifyTier(composite)

	return &types.ScoredResult{
		Categories: categories,
		Composite:  composite,
		Tier:       tier,
	}, nil
}

// scoreC1 extracts C1 (Code Health) metrics and computes the category score.
func (s *Scorer) scoreC1(ar *types.AnalysisResult) types.CategoryScore {
	raw, ok := ar.Metrics["c1"]
	if !ok {
		return types.CategoryScore{Name: "C1", Weight: s.Config.C1.Weight}
	}
	m, ok := raw.(*types.C1Metrics)
	if !ok {
		return types.CategoryScore{Name: "C1", Weight: s.Config.C1.Weight}
	}

	rawValues := map[string]float64{
		"complexity_avg":        m.CyclomaticComplexity.Avg,
		"func_length_avg":      m.FunctionLength.Avg,
		"file_size_avg":        m.FileSize.Avg,
		"afferent_coupling_avg": avgMapValues(m.AfferentCoupling),
		"efferent_coupling_avg": avgMapValues(m.EfferentCoupling),
		"duplication_rate":      m.DuplicationRate,
	}

	subScores, score := scoreMetrics(s.Config.C1, rawValues, nil)

	return types.CategoryScore{
		Name:      "C1",
		Score:     score,
		Weight:    s.Config.C1.Weight,
		SubScores: subScores,
	}
}

// scoreC3 extracts C3 (Architecture) metrics and computes the category score.
func (s *Scorer) scoreC3(ar *types.AnalysisResult) types.CategoryScore {
	raw, ok := ar.Metrics["c3"]
	if !ok {
		return types.CategoryScore{Name: "C3", Weight: s.Config.C3.Weight}
	}
	m, ok := raw.(*types.C3Metrics)
	if !ok {
		return types.CategoryScore{Name: "C3", Weight: s.Config.C3.Weight}
	}

	rawValues := map[string]float64{
		"max_dir_depth":         float64(m.MaxDirectoryDepth),
		"module_fanout_avg":     m.ModuleFanout.Avg,
		"circular_deps":         float64(len(m.CircularDeps)),
		"import_complexity_avg": m.ImportComplexity.Avg,
		"dead_exports":          float64(len(m.DeadExports)),
	}

	subScores, score := scoreMetrics(s.Config.C3, rawValues, nil)

	return types.CategoryScore{
		Name:      "C3",
		Score:     score,
		Weight:    s.Config.C3.Weight,
		SubScores: subScores,
	}
}

// scoreC6 extracts C6 (Testing) metrics and computes the category score.
func (s *Scorer) scoreC6(ar *types.AnalysisResult) types.CategoryScore {
	raw, ok := ar.Metrics["c6"]
	if !ok {
		return types.CategoryScore{Name: "C6", Weight: s.Config.C6.Weight}
	}
	m, ok := raw.(*types.C6Metrics)
	if !ok {
		return types.CategoryScore{Name: "C6", Weight: s.Config.C6.Weight}
	}

	// Compute test_file_ratio with zero-division guard
	var testFileRatio float64
	if m.SourceFileCount > 0 {
		testFileRatio = float64(m.TestFileCount) / float64(m.SourceFileCount)
	}

	rawValues := map[string]float64{
		"test_to_code_ratio":  m.TestToCodeRatio,
		"coverage_percent":    m.CoveragePercent,
		"test_isolation":      m.TestIsolation,
		"assertion_density_avg": m.AssertionDensity.Avg,
		"test_file_ratio":     testFileRatio,
	}

	// Mark coverage as unavailable if == -1
	unavailable := map[string]bool{}
	if m.CoveragePercent == -1 {
		unavailable["coverage_percent"] = true
	}

	subScores, score := scoreMetrics(s.Config.C6, rawValues, unavailable)

	return types.CategoryScore{
		Name:      "C6",
		Score:     score,
		Weight:    s.Config.C6.Weight,
		SubScores: subScores,
	}
}

// scoreMetrics is a generic scoring helper for any category.
// It iterates over the category's metric configs, looks up raw values by name,
// interpolates scores, and computes the weighted average. Metrics in the
// unavailable set are marked Available=false and excluded from the average.
func scoreMetrics(catConfig CategoryConfig, rawValues map[string]float64, unavailable map[string]bool) ([]types.SubScore, float64) {
	var subScores []types.SubScore

	for _, mt := range catConfig.Metrics {
		rv := rawValues[mt.Name]
		ss := types.SubScore{
			MetricName: mt.Name,
			RawValue:   rv,
			Weight:     mt.Weight,
			Available:  true,
		}

		if unavailable[mt.Name] {
			ss.Available = false
			ss.Score = 0
		} else {
			ss.Score = Interpolate(mt.Breakpoints, rv)
		}

		subScores = append(subScores, ss)
	}

	score := categoryScore(subScores)
	return subScores, score
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
