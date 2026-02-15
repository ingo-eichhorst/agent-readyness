package scoring

import (
	"fmt"
	"sort"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

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
	extractC6Isolation(m, evidence)
	extractC6AssertionDensity(m, evidence)

	ensureEvidenceKeys(evidence, []string{
		"test_to_code_ratio", "coverage_percent", "test_isolation",
		"assertion_density_avg", "test_file_ratio",
	})

	return map[string]float64{
		"test_to_code_ratio":    m.TestToCodeRatio,
		"coverage_percent":      m.CoveragePercent,
		"test_isolation":        m.TestIsolation,
		"assertion_density_avg": m.AssertionDensity.Avg,
		"test_file_ratio":       testFileRatio,
	}, unavailable, evidence
}

// extractC6Isolation collects top test functions with external dependencies.
func extractC6Isolation(m *types.C6Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.TestFunctions) == 0 {
		return
	}
	var withExtDep []types.TestFunctionMetric
	for _, tf := range m.TestFunctions {
		if tf.HasExternalDep {
			withExtDep = append(withExtDep, tf)
		}
	}
	if len(withExtDep) == 0 {
		return
	}
	evidence["test_isolation"] = topNEvidence(len(withExtDep), func(i int) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: withExtDep[i].File, Line: withExtDep[i].Line, Value: 1,
			Description: fmt.Sprintf("%s has external dependency", withExtDep[i].Name),
		}
	})
}

// extractC6AssertionDensity collects tests with lowest assertion counts.
func extractC6AssertionDensity(m *types.C6Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.TestFunctions) == 0 {
		return
	}
	sorted := make([]types.TestFunctionMetric, len(m.TestFunctions))
	copy(sorted, m.TestFunctions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AssertionCount < sorted[j].AssertionCount
	})
	evidence["assertion_density_avg"] = topNEvidence(len(sorted), func(i int) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: sorted[i].File, Line: sorted[i].Line,
			Value:       float64(sorted[i].AssertionCount),
			Description: fmt.Sprintf("%s has %d assertions", sorted[i].Name, sorted[i].AssertionCount),
		}
	})
}
