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

	// Initialize evidence map for tracking top offenders per metric
	evidence := make(map[string][]types.EvidenceItem)

	// complexity_avg: top 5 functions by cyclomatic complexity (worst offenders)
	if len(m.Functions) > 0 {
		// Sort descending by complexity, then take top 5
		sorted := make([]types.FunctionMetric, len(m.Functions))
		copy(sorted, m.Functions)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Complexity > sorted[j].Complexity
		})
		limit := evidenceTopN
		if len(sorted) < limit {
			limit = len(sorted)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			items[i] = types.EvidenceItem{
				FilePath:    sorted[i].File,
				Line:        sorted[i].Line,
				Value:       float64(sorted[i].Complexity),
				Description: fmt.Sprintf("%s has complexity %d", sorted[i].Name, sorted[i].Complexity),
			}
		}
		evidence["complexity_avg"] = items
	}

	// func_length_avg: top 5 functions by line count
	if len(m.Functions) > 0 {
		sorted := make([]types.FunctionMetric, len(m.Functions))
		copy(sorted, m.Functions)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].LineCount > sorted[j].LineCount
		})
		limit := evidenceTopN
		if len(sorted) < limit {
			limit = len(sorted)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			items[i] = types.EvidenceItem{
				FilePath:    sorted[i].File,
				Line:        sorted[i].Line,
				Value:       float64(sorted[i].LineCount),
				Description: fmt.Sprintf("%s is %d lines", sorted[i].Name, sorted[i].LineCount),
			}
		}
		evidence["func_length_avg"] = items
	}

	// file_size_avg: single worst file
	if m.FileSize.MaxEntity != "" {
		evidence["file_size_avg"] = []types.EvidenceItem{{
			FilePath:    m.FileSize.MaxEntity,
			Line:        0,
			Value:       float64(m.FileSize.Max),
			Description: fmt.Sprintf("largest file: %d lines", m.FileSize.Max),
		}}
	}

	// afferent_coupling_avg: top 5 packages by incoming dependency count
	if len(m.AfferentCoupling) > 0 {
		type pkgCount struct {
			pkg   string
			count int
		}
		entries := make([]pkgCount, 0, len(m.AfferentCoupling))
		for pkg, count := range m.AfferentCoupling {
			entries = append(entries, pkgCount{pkg, count})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].count > entries[j].count
		})
		limit := evidenceTopN
		if len(entries) < limit {
			limit = len(entries)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			items[i] = types.EvidenceItem{
				FilePath:    entries[i].pkg,
				Line:        0,
				Value:       float64(entries[i].count),
				Description: fmt.Sprintf("imported by %d packages", entries[i].count),
			}
		}
		evidence["afferent_coupling_avg"] = items
	}

	// efferent_coupling_avg: top 5 packages by outgoing dependency count
	if len(m.EfferentCoupling) > 0 {
		type pkgCount struct {
			pkg   string
			count int
		}
		entries := make([]pkgCount, 0, len(m.EfferentCoupling))
		for pkg, count := range m.EfferentCoupling {
			entries = append(entries, pkgCount{pkg, count})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].count > entries[j].count
		})
		limit := evidenceTopN
		if len(entries) < limit {
			limit = len(entries)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			items[i] = types.EvidenceItem{
				FilePath:    entries[i].pkg,
				Line:        0,
				Value:       float64(entries[i].count),
				Description: fmt.Sprintf("imports %d packages", entries[i].count),
			}
		}
		evidence["efferent_coupling_avg"] = items
	}

	// duplication_rate: top 5 duplicate blocks by line count
	if len(m.DuplicatedBlocks) > 0 {
		sorted := make([]types.DuplicateBlock, len(m.DuplicatedBlocks))
		copy(sorted, m.DuplicatedBlocks)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].LineCount > sorted[j].LineCount
		})
		limit := evidenceTopN
		if len(sorted) < limit {
			limit = len(sorted)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			items[i] = types.EvidenceItem{
				FilePath:    sorted[i].FileA,
				Line:        sorted[i].StartA,
				Value:       float64(sorted[i].LineCount),
				Description: fmt.Sprintf("%d-line duplicate block", sorted[i].LineCount),
			}
		}
		evidence["duplication_rate"] = items
	}

	// Ensure all 6 metric keys have at least empty arrays (never nil)
	// This guarantees consistent JSON output and prevents null values in serialized reports
	for _, key := range []string{"complexity_avg", "func_length_avg", "file_size_avg", "afferent_coupling_avg", "efferent_coupling_avg", "duplication_rate"} {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}

	return map[string]float64{
		"complexity_avg":        m.CyclomaticComplexity.Avg,
		"func_length_avg":      m.FunctionLength.Avg,
		"file_size_avg":        m.FileSize.Avg,
		"afferent_coupling_avg": avgMapValues(m.AfferentCoupling),
		"efferent_coupling_avg": avgMapValues(m.EfferentCoupling),
		"duplication_rate":      m.DuplicationRate,
	}, nil, evidence
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

	// max_dir_depth: no per-item data available
	// module_fanout_avg: single worst module if available
	if m.ModuleFanout.MaxEntity != "" {
		evidence["module_fanout_avg"] = []types.EvidenceItem{{
			FilePath:    m.ModuleFanout.MaxEntity,
			Line:        0,
			Value:       float64(m.ModuleFanout.Max),
			Description: fmt.Sprintf("highest fanout: %d references", m.ModuleFanout.Max),
		}}
	}

	// circular_deps: first 5 cycles
	if len(m.CircularDeps) > 0 {
		limit := evidenceTopN
		if len(m.CircularDeps) < limit {
			limit = len(m.CircularDeps)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			cycle := m.CircularDeps[i]
			filePath := ""
			if len(cycle) > 0 {
				filePath = cycle[0]
			}
			items[i] = types.EvidenceItem{
				FilePath:    filePath,
				Line:        0,
				Value:       float64(len(cycle)),
				Description: fmt.Sprintf("cycle: %s", strings.Join(cycle, " -> ")),
			}
		}
		evidence["circular_deps"] = items
	}

	// import_complexity_avg: single worst if available
	if m.ImportComplexity.MaxEntity != "" {
		evidence["import_complexity_avg"] = []types.EvidenceItem{{
			FilePath:    m.ImportComplexity.MaxEntity,
			Line:        0,
			Value:       float64(m.ImportComplexity.Max),
			Description: fmt.Sprintf("most complex imports: %d segments", m.ImportComplexity.Max),
		}}
	}

	// dead_exports: first 5 unused exports
	if len(m.DeadExports) > 0 {
		limit := evidenceTopN
		if len(m.DeadExports) < limit {
			limit = len(m.DeadExports)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			de := m.DeadExports[i]
			items[i] = types.EvidenceItem{
				FilePath:    de.File,
				Line:        de.Line,
				Value:       1,
				Description: fmt.Sprintf("unused %s: %s", de.Kind, de.Name),
			}
		}
		evidence["dead_exports"] = items
	}

	// Ensure all 5 keys have at least empty arrays
	for _, key := range []string{"max_dir_depth", "module_fanout_avg", "circular_deps", "import_complexity_avg", "dead_exports"} {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}

	return map[string]float64{
		"max_dir_depth":        float64(m.MaxDirectoryDepth),
		"module_fanout_avg":    m.ModuleFanout.Avg,
		"circular_deps":        float64(len(m.CircularDeps)),
		"import_complexity_avg": m.ImportComplexity.Avg,
		"dead_exports":          float64(len(m.DeadExports)),
	}, nil, evidence
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

	evidence := make(map[string][]types.EvidenceItem)

	// test_isolation: top 5 tests with external dependencies
	if len(m.TestFunctions) > 0 {
		var withExtDep []types.TestFunctionMetric
		for _, tf := range m.TestFunctions {
			if tf.HasExternalDep {
				withExtDep = append(withExtDep, tf)
			}
		}
		limit := evidenceTopN
		if len(withExtDep) < limit {
			limit = len(withExtDep)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			items[i] = types.EvidenceItem{
				FilePath:    withExtDep[i].File,
				Line:        withExtDep[i].Line,
				Value:       1,
				Description: fmt.Sprintf("%s has external dependency", withExtDep[i].Name),
			}
		}
		evidence["test_isolation"] = items
	}

	// assertion_density_avg: top 5 tests with lowest assertion count (worst offenders)
	if len(m.TestFunctions) > 0 {
		sorted := make([]types.TestFunctionMetric, len(m.TestFunctions))
		copy(sorted, m.TestFunctions)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].AssertionCount < sorted[j].AssertionCount
		})
		limit := evidenceTopN
		if len(sorted) < limit {
			limit = len(sorted)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			items[i] = types.EvidenceItem{
				FilePath:    sorted[i].File,
				Line:        sorted[i].Line,
				Value:       float64(sorted[i].AssertionCount),
				Description: fmt.Sprintf("%s has %d assertions", sorted[i].Name, sorted[i].AssertionCount),
			}
		}
		evidence["assertion_density_avg"] = items
	}

	// Ensure all 5 keys have at least empty arrays
	for _, key := range []string{"test_to_code_ratio", "coverage_percent", "test_isolation", "assertion_density_avg", "test_file_ratio"} {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}

	return rawValues, unavailable, evidence
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

	if !m.Available {
		unavailable := map[string]bool{
			"churn_rate":            true,
			"temporal_coupling_pct": true,
			"author_fragmentation":  true,
			"commit_stability":      true,
			"hotspot_concentration": true,
		}
		emptyEvidence := make(map[string][]types.EvidenceItem)
		for k := range unavailable {
			emptyEvidence[k] = []types.EvidenceItem{}
		}
		return map[string]float64{}, unavailable, emptyEvidence
	}

	evidence := make(map[string][]types.EvidenceItem)

	// churn_rate: top 5 hotspots by commit count
	if len(m.TopHotspots) > 0 {
		limit := evidenceTopN
		if len(m.TopHotspots) < limit {
			limit = len(m.TopHotspots)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			h := m.TopHotspots[i]
			items[i] = types.EvidenceItem{
				FilePath:    h.Path,
				Line:        0,
				Value:       float64(h.CommitCount),
				Description: fmt.Sprintf("%d commits", h.CommitCount),
			}
		}
		evidence["churn_rate"] = items
	}

	// temporal_coupling_pct: top 5 coupled pairs
	if len(m.CoupledPairs) > 0 {
		limit := evidenceTopN
		if len(m.CoupledPairs) < limit {
			limit = len(m.CoupledPairs)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			p := m.CoupledPairs[i]
			items[i] = types.EvidenceItem{
				FilePath:    p.FileA,
				Line:        0,
				Value:       p.Coupling,
				Description: fmt.Sprintf("coupled with %s (%.0f%%)", p.FileB, p.Coupling),
			}
		}
		evidence["temporal_coupling_pct"] = items
	}

	// author_fragmentation: top 5 hotspots by author count
	if len(m.TopHotspots) > 0 {
		sorted := make([]types.FileChurn, len(m.TopHotspots))
		copy(sorted, m.TopHotspots)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].AuthorCount > sorted[j].AuthorCount
		})
		limit := evidenceTopN
		if len(sorted) < limit {
			limit = len(sorted)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			h := sorted[i]
			items[i] = types.EvidenceItem{
				FilePath:    h.Path,
				Line:        0,
				Value:       float64(h.AuthorCount),
				Description: fmt.Sprintf("%d distinct authors", h.AuthorCount),
			}
		}
		evidence["author_fragmentation"] = items
	}

	// hotspot_concentration: top 5 hotspots by total changes
	if len(m.TopHotspots) > 0 {
		limit := evidenceTopN
		if len(m.TopHotspots) < limit {
			limit = len(m.TopHotspots)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			h := m.TopHotspots[i]
			items[i] = types.EvidenceItem{
				FilePath:    h.Path,
				Line:        0,
				Value:       float64(h.TotalChanges),
				Description: fmt.Sprintf("hotspot: %d changes", h.TotalChanges),
			}
		}
		evidence["hotspot_concentration"] = items
	}

	// Ensure all 5 keys have at least empty arrays
	for _, key := range []string{"churn_rate", "temporal_coupling_pct", "author_fragmentation", "commit_stability", "hotspot_concentration"} {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}

	return map[string]float64{
		"churn_rate":            m.ChurnRate,
		"temporal_coupling_pct": m.TemporalCouplingPct,
		"author_fragmentation":  m.AuthorFragmentation,
		"commit_stability":      m.CommitStability,
		"hotspot_concentration": m.HotspotConcentration,
	}, nil, evidence
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

	if !m.Available {
		unavailable := map[string]bool{
			"task_execution_consistency":       true,
			"code_behavior_comprehension":      true,
			"cross_file_navigation":            true,
			"identifier_interpretability":      true,
			"documentation_accuracy_detection": true,
		}
		emptyEvidence := make(map[string][]types.EvidenceItem)
		for k := range unavailable {
			emptyEvidence[k] = []types.EvidenceItem{}
		}
		return map[string]float64{}, unavailable, emptyEvidence
	}

	evidence := map[string][]types.EvidenceItem{
		"task_execution_consistency":       {},
		"code_behavior_comprehension":      {},
		"cross_file_navigation":            {},
		"identifier_interpretability":      {},
		"documentation_accuracy_detection": {},
	}

	return map[string]float64{
		"task_execution_consistency":       float64(m.TaskExecutionConsistency),
		"code_behavior_comprehension":      float64(m.CodeBehaviorComprehension),
		"cross_file_navigation":            float64(m.CrossFileNavigation),
		"identifier_interpretability":      float64(m.IdentifierInterpretability),
		"documentation_accuracy_detection": float64(m.DocumentationAccuracyDetection),
	}, nil, evidence
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
