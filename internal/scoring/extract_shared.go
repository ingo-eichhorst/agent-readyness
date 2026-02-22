package scoring

import (
	"sort"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// buildTopNEvidence is a generic helper that copies items, optionally sorts them,
// caps to evidenceTopN, and builds a []types.EvidenceItem slice.
func buildTopNEvidence[T any](
	items []T,
	less func(a, b T) bool, // nil = skip sort
	getValue func(T) float64,
	getPath func(T) string,
	getLine func(T) int, // nil = always 0
	getDesc func(T) string,
) []types.EvidenceItem {
	if len(items) == 0 {
		return []types.EvidenceItem{}
	}
	working := make([]T, len(items))
	copy(working, items)
	if less != nil {
		sort.Slice(working, func(i, j int) bool { return less(working[i], working[j]) })
	}
	limit := min(evidenceTopN, len(working))
	result := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		line := 0
		if getLine != nil {
			line = getLine(working[i])
		}
		result[i] = types.EvidenceItem{
			FilePath:    getPath(working[i]),
			Line:        line,
			Value:       getValue(working[i]),
			Description: getDesc(working[i]),
		}
	}
	return result
}

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
