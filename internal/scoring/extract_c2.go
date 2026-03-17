package scoring

import (
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// extractC2 extracts C2 (Semantic Explicitness) metrics from an AnalysisResult.
// Metrics marked as unavailable in the aggregate are passed through to the scorer
// so they're excluded from the weighted average instead of penalizing with a 0.
func extractC2(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	rawValues, _, evidence := extractMetrics[types.C2Metrics, *types.C2Metrics](ar, "c2", func(m *types.C2Metrics) (map[string]float64, map[string][]types.EvidenceItem, []string) {
		if m.Aggregate == nil {
			return nil, nil, nil
		}

		keys := []string{"type_annotation_coverage", "naming_consistency", "magic_number_ratio", "type_strictness", "null_safety"}
		ev := make(map[string][]types.EvidenceItem)

		return map[string]float64{
			"type_annotation_coverage": m.Aggregate.TypeAnnotationCoverage,
			"naming_consistency":       m.Aggregate.NamingConsistency,
			"magic_number_ratio":       m.Aggregate.MagicNumberRatio,
			"type_strictness":          m.Aggregate.TypeStrictness,
			"null_safety":              m.Aggregate.NullSafety,
		}, ev, keys
	})

	// Flow unavailability from the C2 aggregate to the scorer.
	var unavailable map[string]bool
	if rawValues != nil {
		raw, ok := ar.Metrics["c2"]
		if ok {
			if m, ok := raw.(*types.C2Metrics); ok && m.Aggregate != nil && len(m.Aggregate.Unavailable) > 0 {
				unavailable = m.Aggregate.Unavailable
			}
		}
	}

	return rawValues, unavailable, evidence
}
