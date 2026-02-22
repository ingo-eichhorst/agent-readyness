package scoring

import (
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// extractC2 extracts C2 (Semantic Explicitness) metrics from an AnalysisResult.
func extractC2(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	return extractMetrics[types.C2Metrics, *types.C2Metrics](ar, "c2", func(m *types.C2Metrics) (map[string]float64, map[string][]types.EvidenceItem, []string) {
		if m.Aggregate == nil {
			return nil, nil, nil
		}

		keys := []string{"type_annotation_coverage", "naming_consistency", "magic_number_ratio", "type_strictness", "null_safety"}
		evidence := make(map[string][]types.EvidenceItem)

		return map[string]float64{
			"type_annotation_coverage": m.Aggregate.TypeAnnotationCoverage,
			"naming_consistency":       m.Aggregate.NamingConsistency,
			"magic_number_ratio":       m.Aggregate.MagicNumberRatio,
			"type_strictness":          m.Aggregate.TypeStrictness,
			"null_safety":              m.Aggregate.NullSafety,
		}, evidence, keys
	})
}
