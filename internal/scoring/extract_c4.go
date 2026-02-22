package scoring

import (
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

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
