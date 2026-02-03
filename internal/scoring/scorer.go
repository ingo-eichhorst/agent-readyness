package scoring

import (
	"github.com/ingo/agent-readyness/pkg/types"
)

// Scorer computes scores from raw analysis metrics using configurable breakpoints.
type Scorer struct {
	Config *ScoringConfig
}

// MetricExtractor extracts raw metric values from an AnalysisResult.
// Returns raw values and a set of unavailable metrics.
type MetricExtractor func(ar *types.AnalysisResult) (rawValues map[string]float64, unavailable map[string]bool)

// metricExtractors maps category name to a function that extracts raw metric values.
var metricExtractors = map[string]MetricExtractor{
	"C1": extractC1,
	"C2": extractC2,
	"C3": extractC3,
	"C4": extractC4,
	"C5": extractC5,
	"C6": extractC6,
	"C7": extractC7,
}

// RegisterExtractor registers a metric extractor for a category.
// This allows new categories to be added without modifying the scorer.
func RegisterExtractor(category string, extractor MetricExtractor) {
	metricExtractors[category] = extractor
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
// redistributed among the remaining sub-scores. Returns 0.0 if no sub-scores
// are available (indicating a disabled/unavailable category).
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
		return 0.0 // Disabled capabilities show 0/10
	}
	return weightedSum / totalWeight
}

// Score computes scored results from raw analysis metrics.
// It dispatches each AnalysisResult to the appropriate category scorer
// based on the Category field, computes a weighted composite, and classifies a tier.
func (s *Scorer) Score(results []*types.AnalysisResult) (*types.ScoredResult, error) {
	var categories []types.CategoryScore

	for _, ar := range results {
		catConfig, ok := s.Config.Categories[ar.Category]
		if !ok {
			// Unknown categories are silently skipped
			continue
		}

		extractor, ok := metricExtractors[ar.Category]
		if !ok {
			// No extractor registered for this category
			continue
		}

		rawValues, unavailable := extractor(ar)
		if rawValues == nil {
			// Extractor returned nil -- metrics not found
			categories = append(categories, types.CategoryScore{
				Name:   ar.Category,
				Weight: catConfig.Weight,
			})
			continue
		}

		subScores, score := scoreMetrics(catConfig, rawValues, unavailable)
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

// extractC1 extracts C1 (Code Health) metrics from an AnalysisResult.
func extractC1(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
	raw, ok := ar.Metrics["c1"]
	if !ok {
		return nil, nil
	}
	m, ok := raw.(*types.C1Metrics)
	if !ok {
		return nil, nil
	}

	return map[string]float64{
		"complexity_avg":        m.CyclomaticComplexity.Avg,
		"func_length_avg":      m.FunctionLength.Avg,
		"file_size_avg":        m.FileSize.Avg,
		"afferent_coupling_avg": avgMapValues(m.AfferentCoupling),
		"efferent_coupling_avg": avgMapValues(m.EfferentCoupling),
		"duplication_rate":      m.DuplicationRate,
	}, nil
}

// extractC2 extracts C2 (Semantic Explicitness) metrics from an AnalysisResult.
func extractC2(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
	raw, ok := ar.Metrics["c2"]
	if !ok {
		return nil, nil
	}
	m, ok := raw.(*types.C2Metrics)
	if !ok {
		return nil, nil
	}

	if m.Aggregate == nil {
		return nil, nil
	}

	return map[string]float64{
		"type_annotation_coverage": m.Aggregate.TypeAnnotationCoverage,
		"naming_consistency":       m.Aggregate.NamingConsistency,
		"magic_number_ratio":       m.Aggregate.MagicNumberRatio,
		"type_strictness":          m.Aggregate.TypeStrictness,
		"null_safety":              m.Aggregate.NullSafety,
	}, nil
}

// extractC3 extracts C3 (Architecture) metrics from an AnalysisResult.
func extractC3(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
	raw, ok := ar.Metrics["c3"]
	if !ok {
		return nil, nil
	}
	m, ok := raw.(*types.C3Metrics)
	if !ok {
		return nil, nil
	}

	return map[string]float64{
		"max_dir_depth":        float64(m.MaxDirectoryDepth),
		"module_fanout_avg":    m.ModuleFanout.Avg,
		"circular_deps":        float64(len(m.CircularDeps)),
		"import_complexity_avg": m.ImportComplexity.Avg,
		"dead_exports":          float64(len(m.DeadExports)),
	}, nil
}

// extractC4 extracts C4 (Documentation Quality) metrics from an AnalysisResult.
func extractC4(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
	raw, ok := ar.Metrics["c4"]
	if !ok {
		return nil, nil
	}
	m, ok := raw.(*types.C4Metrics)
	if !ok {
		return nil, nil
	}

	// Convert boolean presence to 0/1 for scoring
	changelogVal := 0.0
	if m.ChangelogPresent {
		changelogVal = 1.0
	}
	examplesVal := 0.0
	if m.ExamplesPresent {
		examplesVal = 1.0
	}
	contributingVal := 0.0
	if m.ContributingPresent {
		contributingVal = 1.0
	}
	diagramsVal := 0.0
	if m.DiagramsPresent {
		diagramsVal = 1.0
	}

	return map[string]float64{
		"readme_word_count":     float64(m.ReadmeWordCount),
		"comment_density":       m.CommentDensity,
		"api_doc_coverage":      m.APIDocCoverage,
		"changelog_present":     changelogVal,
		"examples_present":      examplesVal,
		"contributing_present":  contributingVal,
		"diagrams_present":      diagramsVal,
	}, nil
}

// extractC6 extracts C6 (Testing) metrics from an AnalysisResult.
func extractC6(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
	raw, ok := ar.Metrics["c6"]
	if !ok {
		return nil, nil
	}
	m, ok := raw.(*types.C6Metrics)
	if !ok {
		return nil, nil
	}

	// Compute test_file_ratio with zero-division guard
	var testFileRatio float64
	if m.SourceFileCount > 0 {
		testFileRatio = float64(m.TestFileCount) / float64(m.SourceFileCount)
	}

	rawValues := map[string]float64{
		"test_to_code_ratio":    m.TestToCodeRatio,
		"coverage_percent":      m.CoveragePercent,
		"test_isolation":        m.TestIsolation,
		"assertion_density_avg": m.AssertionDensity.Avg,
		"test_file_ratio":       testFileRatio,
	}

	// Mark coverage as unavailable if == -1
	unavailable := map[string]bool{}
	if m.CoveragePercent == -1 {
		unavailable["coverage_percent"] = true
	}

	return rawValues, unavailable
}

// extractC5 extracts C5 (Temporal Dynamics) metrics from an AnalysisResult.
func extractC5(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
	raw, ok := ar.Metrics["c5"]
	if !ok {
		return nil, nil
	}
	m, ok := raw.(*types.C5Metrics)
	if !ok {
		return nil, nil
	}

	if !m.Available {
		unavailable := map[string]bool{
			"churn_rate":            true,
			"temporal_coupling_pct": true,
			"author_fragmentation":  true,
			"commit_stability":      true,
			"hotspot_concentration": true,
		}
		return map[string]float64{}, unavailable
	}

	return map[string]float64{
		"churn_rate":            m.ChurnRate,
		"temporal_coupling_pct": m.TemporalCouplingPct,
		"author_fragmentation":  m.AuthorFragmentation,
		"commit_stability":      m.CommitStability,
		"hotspot_concentration": m.HotspotConcentration,
	}, nil
}

// extractC7 extracts C7 (Agent Evaluation) metrics from an AnalysisResult.
func extractC7(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
	raw, ok := ar.Metrics["c7"]
	if !ok {
		return nil, nil
	}
	m, ok := raw.(*types.C7Metrics)
	if !ok {
		return nil, nil
	}

	if !m.Available {
		unavailable := map[string]bool{
			"overall_score": true,
		}
		return map[string]float64{}, unavailable
	}

	return map[string]float64{
		"overall_score": m.OverallScore,
	}, nil
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
