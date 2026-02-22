package scoring

import (
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

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
