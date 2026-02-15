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
	"fmt"
	"sort"
	"strings"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// evidenceTopN is the maximum number of evidence items retained per metric.
const evidenceTopN = 5

// topNItems returns up to evidenceTopN items from a pre-sorted slice using the provided
// conversion function. The input should already be sorted by the desired criteria.
func topNItems[T any](sorted []T, toEvidence func(T) types.EvidenceItem) []types.EvidenceItem {
	limit := evidenceTopN
	if len(sorted) < limit {
		limit = len(sorted)
	}
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		items[i] = toEvidence(sorted[i])
	}
	return items
}

// couplingEvidence builds top-N evidence from a map[string]int (package->count).
func couplingEvidence(m map[string]int, descFmt string) []types.EvidenceItem {
	type pkgCount struct {
		pkg   string
		count int
	}
	entries := make([]pkgCount, 0, len(m))
	for pkg, count := range m {
		entries = append(entries, pkgCount{pkg, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})
	return topNItems(entries, func(e pkgCount) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath:    e.pkg,
			Value:       float64(e.count),
			Description: fmt.Sprintf(descFmt, e.count),
		}
	})
}

// ensureEvidenceKeys ensures all given keys have at least empty arrays (never nil).
func ensureEvidenceKeys(evidence map[string][]types.EvidenceItem, keys []string) {
	for _, key := range keys {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}
}

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

// Interpolate computes the score for a given raw value using piecewise linear
// interpolation over the provided breakpoints.
//
// Algorithm:
// 1. Find enclosing breakpoint segment [lo, hi] where lo.Value <= rawValue <= hi.Value
// 2. Linear interpolation: score = lo.Score + t*(hi.Score - lo.Score)
//    where t = (rawValue - lo.Value) / (hi.Value - lo.Value)
// 3. Clamp: values below first breakpoint use first score, above last use last score
//
// Breakpoints must be sorted by Value in ascending order.
// This produces smooth, predictable scoring curves for all metrics.
//
// CRITICAL: This is the CORE scoring function - ALL metrics (C1-C7) flow through
// this function. Any changes to interpolation logic affect every metric and tier
// classification. The algorithm is intentionally simple (linear segments) to keep
// scoring transparent and debuggable. Non-linear curves (exponential, logarithmic)
// would make score interpretation opaque to users.
//
// Edge case behavior:
// - Values below lowest breakpoint: use lowest breakpoint score (floor)
// - Values above highest breakpoint: use highest breakpoint score (ceiling)
// - Exact match to breakpoint: return that breakpoint's score (no interpolation)
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
//
// Weight normalization approach:
// - Only active categories (Score >= 0) contribute to composite
// - Weights are normalized by sum of active category weights
// - This ensures scoring 10/10 on all active categories yields composite=10
// - Unavailable categories (e.g., C5 without git, C7 without Claude CLI) don't penalize composite
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

// CategoryScore computes the weighted average of sub-scores within a category.
//
// Returns -1.0 if all sub-scores are unavailable (Score < 0), indicating
// the category cannot be scored (e.g., no git repo for C5, no tests for C6).
// When some sub-scores are unavailable, redistributes their weight to available metrics.
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
		return -1.0 // Mark category as unavailable (excluded from composite)
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

		rawValues, unavailable, evidence := extractor(ar)
		if rawValues == nil {
			// Extractor returned nil -- metrics not found
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

// extractC1 extracts C1 (Code Health) metrics from an AnalysisResult and collects evidence.
//
// Evidence selection approach:
// - Top 5 "worst offenders" per metric (highest complexity, longest functions, etc.)
// - Sorted by severity descending (most problematic first)
// - Empty arrays guaranteed (never nil) for JSON serialization
//
// Evidence helps developers prioritize improvements by identifying specific files/functions
// that drag down the score. For example, the top 5 most complex functions or longest files.
//
// Why evidence matters: Without concrete examples, users see abstract scores (e.g.,
// "complexity avg: 12.3") but don't know WHERE the problems are. Evidence transforms
// scores into actionable work items: "refactor parseConfig() at line 234 with complexity 47".
// The 5-item limit balances actionability (focused improvements) with comprehensiveness
// (enough examples to spot patterns).
func extractC1(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c1"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C1Metrics)
	if !ok {
		return nil, nil, nil
	}

	evidence := make(map[string][]types.EvidenceItem)
	evidence["complexity_avg"] = c1ComplexityEvidence(m.Functions)
	evidence["func_length_avg"] = c1FuncLengthEvidence(m.Functions)
	evidence["file_size_avg"] = metricSummaryEvidence(m.FileSize, "largest file: %d lines")
	evidence["afferent_coupling_avg"] = couplingEvidence(m.AfferentCoupling, "imported by %d packages")
	evidence["efferent_coupling_avg"] = couplingEvidence(m.EfferentCoupling, "imports %d packages")
	evidence["duplication_rate"] = c1DuplicationEvidence(m.DuplicatedBlocks)
	ensureEvidenceKeys(evidence, []string{"complexity_avg", "func_length_avg", "file_size_avg", "afferent_coupling_avg", "efferent_coupling_avg", "duplication_rate"})

	return map[string]float64{
		"complexity_avg":        m.CyclomaticComplexity.Avg,
		"func_length_avg":      m.FunctionLength.Avg,
		"file_size_avg":        m.FileSize.Avg,
		"afferent_coupling_avg": avgMapValues(m.AfferentCoupling),
		"efferent_coupling_avg": avgMapValues(m.EfferentCoupling),
		"duplication_rate":      m.DuplicationRate,
	}, nil, evidence
}

func c1ComplexityEvidence(funcs []types.FunctionMetric) []types.EvidenceItem {
	if len(funcs) == 0 {
		return nil
	}
	sorted := make([]types.FunctionMetric, len(funcs))
	copy(sorted, funcs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Complexity > sorted[j].Complexity })
	return topNItems(sorted, func(f types.FunctionMetric) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: f.File, Line: f.Line, Value: float64(f.Complexity),
			Description: fmt.Sprintf("%s has complexity %d", f.Name, f.Complexity),
		}
	})
}

func c1FuncLengthEvidence(funcs []types.FunctionMetric) []types.EvidenceItem {
	if len(funcs) == 0 {
		return nil
	}
	sorted := make([]types.FunctionMetric, len(funcs))
	copy(sorted, funcs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].LineCount > sorted[j].LineCount })
	return topNItems(sorted, func(f types.FunctionMetric) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: f.File, Line: f.Line, Value: float64(f.LineCount),
			Description: fmt.Sprintf("%s is %d lines", f.Name, f.LineCount),
		}
	})
}

func c1DuplicationEvidence(blocks []types.DuplicateBlock) []types.EvidenceItem {
	if len(blocks) == 0 {
		return nil
	}
	sorted := make([]types.DuplicateBlock, len(blocks))
	copy(sorted, blocks)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].LineCount > sorted[j].LineCount })
	return topNItems(sorted, func(b types.DuplicateBlock) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: b.FileA, Line: b.StartA, Value: float64(b.LineCount),
			Description: fmt.Sprintf("%d-line duplicate block", b.LineCount),
		}
	})
}

// metricSummaryEvidence returns a single evidence item for a MetricSummary's max entity.
func metricSummaryEvidence(ms types.MetricSummary, descFmt string) []types.EvidenceItem {
	if ms.MaxEntity == "" {
		return nil
	}
	return []types.EvidenceItem{{
		FilePath: ms.MaxEntity, Value: float64(ms.Max),
		Description: fmt.Sprintf(descFmt, ms.Max),
	}}
}

// extractC2 extracts C2 (Semantic Explicitness) metrics from an AnalysisResult.
func extractC2(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c2"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C2Metrics)
	if !ok {
		return nil, nil, nil
	}

	if m.Aggregate == nil {
		return nil, nil, nil
	}

	evidence := map[string][]types.EvidenceItem{
		"type_annotation_coverage": {},
		"naming_consistency":       {},
		"magic_number_ratio":       {},
		"type_strictness":          {},
		"null_safety":              {},
	}

	return map[string]float64{
		"type_annotation_coverage": m.Aggregate.TypeAnnotationCoverage,
		"naming_consistency":       m.Aggregate.NamingConsistency,
		"magic_number_ratio":       m.Aggregate.MagicNumberRatio,
		"type_strictness":          m.Aggregate.TypeStrictness,
		"null_safety":              m.Aggregate.NullSafety,
	}, nil, evidence
}

// extractC3 extracts C3 (Architecture) metrics from an AnalysisResult and collects evidence.
//
// Circular dependency evidence:
// - Reports each detected cycle as a chain of module dependencies (A -> B -> C -> A)
// - Note: Go code has zero cycles (compiler prevents import cycles)
//
// Dead code evidence:
// - Exported functions/types never referenced by other packages
// - Conservative (only flags clear dead exports, not vars/consts which may be config)
//
// Architectural evidence guides refactoring priorities. For example:
// - Cycles: "Break circular dependency between auth <-> db <-> models"
// - Dead exports: "Remove unused UserSerializer class (exported but never imported)"
// - High fanout: "Module X imports 27 packages - consider splitting responsibilities"
//
// These actionable insights help developers understand WHY architectural scores are low
// and provide specific files/modules to target for improvement.
func extractC3(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c3"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C3Metrics)
	if !ok {
		return nil, nil, nil
	}

	evidence := make(map[string][]types.EvidenceItem)
	evidence["module_fanout_avg"] = metricSummaryEvidence(m.ModuleFanout, "highest fanout: %d references")
	evidence["circular_deps"] = c3CircularDepsEvidence(m.CircularDeps)
	evidence["import_complexity_avg"] = metricSummaryEvidence(m.ImportComplexity, "most complex imports: %d segments")
	evidence["dead_exports"] = c3DeadExportsEvidence(m.DeadExports)
	ensureEvidenceKeys(evidence, []string{"max_dir_depth", "module_fanout_avg", "circular_deps", "import_complexity_avg", "dead_exports"})

	return map[string]float64{
		"max_dir_depth":        float64(m.MaxDirectoryDepth),
		"module_fanout_avg":    m.ModuleFanout.Avg,
		"circular_deps":        float64(len(m.CircularDeps)),
		"import_complexity_avg": m.ImportComplexity.Avg,
		"dead_exports":          float64(len(m.DeadExports)),
	}, nil, evidence
}

func c3CircularDepsEvidence(cycles [][]string) []types.EvidenceItem {
	if len(cycles) == 0 {
		return nil
	}
	return topNItems(cycles, func(cycle []string) types.EvidenceItem {
		filePath := ""
		if len(cycle) > 0 {
			filePath = cycle[0]
		}
		return types.EvidenceItem{
			FilePath: filePath, Value: float64(len(cycle)),
			Description: fmt.Sprintf("cycle: %s", strings.Join(cycle, " -> ")),
		}
	})
}

func c3DeadExportsEvidence(exports []types.DeadExport) []types.EvidenceItem {
	if len(exports) == 0 {
		return nil
	}
	return topNItems(exports, func(de types.DeadExport) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: de.File, Line: de.Line, Value: 1,
			Description: fmt.Sprintf("unused %s: %s", de.Kind, de.Name),
		}
	})
}

// extractC4 extracts C4 (Documentation Quality) metrics from an AnalysisResult.
func extractC4(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c4"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C4Metrics)
	if !ok {
		return nil, nil, nil
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

	evidence := map[string][]types.EvidenceItem{
		"readme_word_count":    {},
		"comment_density":      {},
		"api_doc_coverage":     {},
		"changelog_present":    {},
		"examples_present":     {},
		"contributing_present": {},
		"diagrams_present":     {},
	}

	return map[string]float64{
		"readme_word_count":     float64(m.ReadmeWordCount),
		"comment_density":       m.CommentDensity,
		"api_doc_coverage":      m.APIDocCoverage,
		"changelog_present":     changelogVal,
		"examples_present":      examplesVal,
		"contributing_present":  contributingVal,
		"diagrams_present":      diagramsVal,
	}, nil, evidence
}

// extractC6 extracts C6 (Testing) metrics from an AnalysisResult.
func extractC6(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c6"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C6Metrics)
	if !ok {
		return nil, nil, nil
	}

	var testFileRatio float64
	if m.SourceFileCount > 0 {
		testFileRatio = float64(m.TestFileCount) / float64(m.SourceFileCount)
	}

	unavailable := map[string]bool{}
	if m.CoveragePercent == -1 {
		unavailable["coverage_percent"] = true
	}

	evidence := make(map[string][]types.EvidenceItem)
	evidence["test_isolation"] = c6IsolationEvidence(m.TestFunctions)
	evidence["assertion_density_avg"] = c6AssertionEvidence(m.TestFunctions)
	c6Keys := []string{"test_to_code_ratio", "coverage_percent", "test_isolation", "assertion_density_avg", "test_file_ratio"}
	ensureEvidenceKeys(evidence, c6Keys)

	return map[string]float64{
		"test_to_code_ratio":    m.TestToCodeRatio,
		"coverage_percent":      m.CoveragePercent,
		"test_isolation":        m.TestIsolation,
		"assertion_density_avg": m.AssertionDensity.Avg,
		"test_file_ratio":       testFileRatio,
	}, unavailable, evidence
}

func c6IsolationEvidence(funcs []types.TestFunctionMetric) []types.EvidenceItem {
	var withExtDep []types.TestFunctionMetric
	for _, tf := range funcs {
		if tf.HasExternalDep {
			withExtDep = append(withExtDep, tf)
		}
	}
	if len(withExtDep) == 0 {
		return nil
	}
	return topNItems(withExtDep, func(tf types.TestFunctionMetric) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: tf.File, Line: tf.Line, Value: 1,
			Description: fmt.Sprintf("%s has external dependency", tf.Name),
		}
	})
}

func c6AssertionEvidence(funcs []types.TestFunctionMetric) []types.EvidenceItem {
	if len(funcs) == 0 {
		return nil
	}
	sorted := make([]types.TestFunctionMetric, len(funcs))
	copy(sorted, funcs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].AssertionCount < sorted[j].AssertionCount })
	return topNItems(sorted, func(tf types.TestFunctionMetric) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: tf.File, Line: tf.Line, Value: float64(tf.AssertionCount),
			Description: fmt.Sprintf("%s has %d assertions", tf.Name, tf.AssertionCount),
		}
	})
}

// extractC5 extracts C5 (Temporal Dynamics) metrics from an AnalysisResult and collects evidence.
//
// Evidence categories:
// - Hotspots: Files with highest churn rate (top 5 by lines changed/commit)
// - Temporal coupling: File pairs with >70% co-change rate (top 5 by strength)
// - Fragmentation: Files with most distinct authors (top 5)
//
// Temporal evidence reveals which files are most volatile and may need
// architectural attention (split, refactor, clarify ownership).
//
// Why temporal dynamics matter for agents: Git history reveals architectural problems
// that static analysis cannot detect. For example:
// - High churn = unstable interfaces that break agent assumptions between runs
// - Temporal coupling = hidden dependencies agents miss (no direct import, but change together)
// - Author fragmentation = conflicting styles/patterns that confuse agent comprehension
// - Hotspot concentration = code under heavy modification pressure, high merge conflict risk
//
// Agents operating on volatile code face a moving target - assumptions valid during
// initial analysis may be stale by the time changes are submitted. C5 evidence helps
// identify these high-risk areas where extra human oversight is warranted.
func extractC5(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c5"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C5Metrics)
	if !ok {
		return nil, nil, nil
	}

	c5Keys := []string{"churn_rate", "temporal_coupling_pct", "author_fragmentation", "commit_stability", "hotspot_concentration"}

	if !m.Available {
		return map[string]float64{}, markAllUnavailable(c5Keys), emptyEvidenceMap(c5Keys)
	}

	evidence := make(map[string][]types.EvidenceItem)
	evidence["churn_rate"] = c5HotspotEvidence(m.TopHotspots, func(h types.FileChurn) (float64, string) {
		return float64(h.CommitCount), fmt.Sprintf("%d commits", h.CommitCount)
	})
	evidence["temporal_coupling_pct"] = c5CouplingEvidence(m.CoupledPairs)
	evidence["author_fragmentation"] = c5AuthorFragmentationEvidence(m.TopHotspots)
	evidence["hotspot_concentration"] = c5HotspotEvidence(m.TopHotspots, func(h types.FileChurn) (float64, string) {
		return float64(h.TotalChanges), fmt.Sprintf("hotspot: %d changes", h.TotalChanges)
	})
	ensureEvidenceKeys(evidence, c5Keys)

	return map[string]float64{
		"churn_rate":            m.ChurnRate,
		"temporal_coupling_pct": m.TemporalCouplingPct,
		"author_fragmentation":  m.AuthorFragmentation,
		"commit_stability":      m.CommitStability,
		"hotspot_concentration": m.HotspotConcentration,
	}, nil, evidence
}

func c5HotspotEvidence(hotspots []types.FileChurn, valDesc func(types.FileChurn) (float64, string)) []types.EvidenceItem {
	if len(hotspots) == 0 {
		return nil
	}
	return topNItems(hotspots, func(h types.FileChurn) types.EvidenceItem {
		val, desc := valDesc(h)
		return types.EvidenceItem{FilePath: h.Path, Value: val, Description: desc}
	})
}

func c5CouplingEvidence(pairs []types.CoupledPair) []types.EvidenceItem {
	if len(pairs) == 0 {
		return nil
	}
	return topNItems(pairs, func(p types.CoupledPair) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: p.FileA, Value: p.Coupling,
			Description: fmt.Sprintf("coupled with %s (%.0f%%)", p.FileB, p.Coupling),
		}
	})
}

func c5AuthorFragmentationEvidence(hotspots []types.FileChurn) []types.EvidenceItem {
	if len(hotspots) == 0 {
		return nil
	}
	sorted := make([]types.FileChurn, len(hotspots))
	copy(sorted, hotspots)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].AuthorCount > sorted[j].AuthorCount })
	return topNItems(sorted, func(h types.FileChurn) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: h.Path, Value: float64(h.AuthorCount),
			Description: fmt.Sprintf("%d distinct authors", h.AuthorCount),
		}
	})
}

// markAllUnavailable returns a map marking all given keys as unavailable.
func markAllUnavailable(keys []string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

// emptyEvidenceMap returns an evidence map with empty slices for all keys.
func emptyEvidenceMap(keys []string) map[string][]types.EvidenceItem {
	m := make(map[string][]types.EvidenceItem, len(keys))
	for _, k := range keys {
		m[k] = []types.EvidenceItem{}
	}
	return m
}

// extractC7 extracts C7 (Agent Evaluation) metrics from an AnalysisResult.
func extractC7(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c7"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C7Metrics)
	if !ok {
		return nil, nil, nil
	}

	c7Keys := []string{"task_execution_consistency", "code_behavior_comprehension", "cross_file_navigation", "identifier_interpretability", "documentation_accuracy_detection"}

	if !m.Available {
		return map[string]float64{}, markAllUnavailable(c7Keys), emptyEvidenceMap(c7Keys)
	}

	return map[string]float64{
		"task_execution_consistency":       float64(m.TaskExecutionConsistency),
		"code_behavior_comprehension":      float64(m.CodeBehaviorComprehension),
		"cross_file_navigation":            float64(m.CrossFileNavigation),
		"identifier_interpretability":      float64(m.IdentifierInterpretability),
		"documentation_accuracy_detection": float64(m.DocumentationAccuracyDetection),
	}, nil, emptyEvidenceMap(c7Keys)
}

// scoreMetrics is a generic scoring helper for any category.
// It iterates over the category's metric configs, looks up raw values by name,
// interpolates scores, and computes the weighted average. Metrics in the
// unavailable set are marked Available=false and excluded from the average.
// Evidence items are attached to each SubScore, with nil maps treated as empty.
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
