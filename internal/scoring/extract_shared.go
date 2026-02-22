package scoring

import (
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// categoryMetricsPtr is a constraint that ensures *M implements types.CategoryMetrics.
// This allows extractMetrics to perform a valid type assertion on the interface value.
type categoryMetricsPtr[M any] interface {
	types.CategoryMetrics
	*M
}

// extractMetrics is a generic helper that handles the boilerplate of type-asserting
// a metric value from ar.Metrics[key], calling a build function to produce scores
// and evidence, and ensuring all evidence keys are initialised.
func extractMetrics[M any, P categoryMetricsPtr[M]](
	ar *types.AnalysisResult,
	key string,
	build func(m P) (map[string]float64, map[string][]types.EvidenceItem, []string),
) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics[key]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(P)
	if !ok {
		return nil, nil, nil
	}
	scores, evidence, keys := build(m)
	ensureEvidenceKeys(evidence, keys...)
	return scores, nil, evidence
}
