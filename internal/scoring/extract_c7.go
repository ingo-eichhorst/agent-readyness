package scoring

import (
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

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
