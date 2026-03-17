package c2

import (
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// C2Analyzer implements the pipeline.Analyzer interface for C2: Semantic Explicitness.
// It dispatches to language-specific analyzers and aggregates results.
// It also implements GoAwareAnalyzer for Go-specific analysis via SetGoPackages.
type C2Analyzer struct {
	goAnalyzer *c2GoAnalyzer
	pyAnalyzer *c2PythonAnalyzer
	tsAnalyzer *c2TypeScriptAnalyzer
}

// NewC2Analyzer creates a C2Analyzer with Tree-sitter-based language analyzers.
// If tsParser is nil, only Go analysis (via SetGoPackages) is available.
func NewC2Analyzer(tsParser *parser.TreeSitterParser) *C2Analyzer {
	a := &C2Analyzer{}
	if tsParser != nil {
		a.pyAnalyzer = newC2PythonAnalyzer(tsParser)
		a.tsAnalyzer = newC2TypeScriptAnalyzer(tsParser)
	}
	return a
}

// Name returns the analyzer display name.
func (a *C2Analyzer) Name() string {
	return "C2: Semantic Explicitness"
}

// SetGoPackages stores Go-specific parsed packages for use during Analyze.
func (a *C2Analyzer) SetGoPackages(pkgs []*parser.ParsedPackage) {
	if a.goAnalyzer == nil {
		a.goAnalyzer = &c2GoAnalyzer{}
	}
	a.goAnalyzer.pkgs = pkgs
}

// Analyze runs C2 analysis on the given analysis targets.
// It dispatches to the appropriate language-specific analyzer for each target,
// then aggregates per-language results weighted by LOC.
func (a *C2Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	metrics := &types.C2Metrics{
		PerLanguage: make(map[types.Language]*types.C2LanguageMetrics),
	}

	for _, target := range targets {
		switch target.Language {
		case types.LangGo:
			if a.goAnalyzer == nil {
				continue
			}
			langMetrics, err := a.goAnalyzer.Analyze(target)
			if err != nil {
				continue
			}
			metrics.PerLanguage[types.LangGo] = langMetrics

		case types.LangPython:
			if a.pyAnalyzer == nil {
				continue
			}
			langMetrics, err := a.pyAnalyzer.Analyze(target)
			if err != nil {
				continue
			}
			metrics.PerLanguage[types.LangPython] = langMetrics

		case types.LangTypeScript:
			if a.tsAnalyzer == nil {
				continue
			}
			langMetrics, err := a.tsAnalyzer.Analyze(target)
			if err != nil {
				continue
			}
			metrics.PerLanguage[types.LangTypeScript] = langMetrics
		}
	}

	// Compute LOC-weighted aggregate across all languages
	metrics.Aggregate = aggregateC2Metrics(metrics.PerLanguage)

	return &types.AnalysisResult{
		Name:     "C2: Semantic Explicitness",
		Category: "C2",
		Metrics:  map[string]types.CategoryMetrics{"c2": metrics},
	}, nil
}

// aggregateC2Metrics computes a LOC-weighted aggregate of per-language C2 metrics.
// If there is only one language, it returns that language's metrics directly.
// Returns nil if no languages have metrics.
//
// Unavailable metrics are propagated: a metric is unavailable in the aggregate
// only if ALL languages mark it unavailable.
func aggregateC2Metrics(perLang map[types.Language]*types.C2LanguageMetrics) *types.C2LanguageMetrics {
	if len(perLang) == 0 {
		return nil
	}

	// If only one language, return it directly
	if len(perLang) == 1 {
		for _, m := range perLang {
			return m
		}
	}

	totalLOC := 0
	for _, m := range perLang {
		totalLOC += m.LOC
	}
	if totalLOC == 0 {
		return nil
	}

	agg := &types.C2LanguageMetrics{}
	for _, m := range perLang {
		w := float64(m.LOC) / float64(totalLOC)
		agg.TypeAnnotationCoverage += m.TypeAnnotationCoverage * w
		agg.NamingConsistency += m.NamingConsistency * w
		agg.MagicNumberRatio += m.MagicNumberRatio * w
		agg.TypeStrictness += m.TypeStrictness * w
		agg.NullSafety += m.NullSafety * w
		agg.TotalFunctions += m.TotalFunctions
		agg.TotalIdentifiers += m.TotalIdentifiers
		agg.MagicNumberCount += m.MagicNumberCount
		agg.LOC += m.LOC
	}

	// Merge unavailability: a metric is unavailable only if ALL languages
	// mark it unavailable. If any language computes the metric, it's available.
	agg.Unavailable = mergeUnavailable(perLang)

	return agg
}

// mergeUnavailable computes the intersection of unavailable metrics across languages.
// A metric is unavailable in the aggregate only if every language marks it unavailable.
func mergeUnavailable(perLang map[types.Language]*types.C2LanguageMetrics) map[string]bool {
	if len(perLang) == 0 {
		return nil
	}

	// Start with the unavailable set of the first language
	var intersection map[string]bool
	first := true
	for _, m := range perLang {
		if first {
			if len(m.Unavailable) > 0 {
				intersection = make(map[string]bool)
				for k, v := range m.Unavailable {
					if v {
						intersection[k] = true
					}
				}
			}
			first = false
			continue
		}

		if intersection == nil {
			// If any language has no unavailable metrics, intersection is empty
			return nil
		}

		// Keep only metrics that are also unavailable in this language
		for k := range intersection {
			if !m.Unavailable[k] {
				delete(intersection, k)
			}
		}

		if len(intersection) == 0 {
			return nil
		}
	}

	if len(intersection) == 0 {
		return nil
	}
	return intersection
}
