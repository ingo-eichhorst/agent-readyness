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

	rawValues, unavailable := c6RawValues(m)
	evidence := make(map[string][]types.EvidenceItem)
	c6TestIsolationEvidence(evidence, m)
	c6AssertionDensityEvidence(evidence, m)
	ensureEvidenceKeys(evidence, "test_to_code_ratio", "coverage_percent", "test_isolation", "assertion_density_avg", "test_file_ratio")

	return rawValues, unavailable, evidence
}

// c6RawValues computes raw metric values and unavailability flags.
func c6RawValues(m *types.C6Metrics) (map[string]float64, map[string]bool) {
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

	unavailable := map[string]bool{}
	if m.CoveragePercent == -1 {
		unavailable["coverage_percent"] = true
	}

	return rawValues, unavailable
}

// c6TestIsolationEvidence collects top tests with external dependencies.
func c6TestIsolationEvidence(evidence map[string][]types.EvidenceItem, m *types.C6Metrics) {
	if len(m.TestFunctions) == 0 {
		return
	}
	var withExtDep []types.TestFunctionMetric
	for _, tf := range m.TestFunctions {
		if tf.HasExternalDep {
			withExtDep = append(withExtDep, tf)
		}
	}
	limit := capLimit(len(withExtDep), evidenceTopN)
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

// c6AssertionDensityEvidence collects tests with lowest assertion counts.
func c6AssertionDensityEvidence(evidence map[string][]types.EvidenceItem, m *types.C6Metrics) {
	if len(m.TestFunctions) == 0 {
		return
	}
	sorted := make([]types.TestFunctionMetric, len(m.TestFunctions))
	copy(sorted, m.TestFunctions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AssertionCount < sorted[j].AssertionCount
	})
	limit := capLimit(len(sorted), evidenceTopN)
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
