package scoring

import (
	"fmt"
	"sort"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// extractC1 extracts C1 (Code Health) metrics from an AnalysisResult and collects evidence.
func extractC1(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c1"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C1Metrics)
	if !ok {
		return nil, nil, nil
	}

	evidence := make(map[string][]types.EvidenceItem)

	extractC1Complexity(m, evidence)
	extractC1FuncLength(m, evidence)
	extractC1FileSize(m, evidence)
	extractC1Coupling(m, evidence)
	extractC1Duplication(m, evidence)

	ensureEvidenceKeys(evidence, []string{
		"complexity_avg", "func_length_avg", "file_size_avg",
		"afferent_coupling_avg", "efferent_coupling_avg", "duplication_rate",
	})

	return map[string]float64{
		"complexity_avg":        m.CyclomaticComplexity.Avg,
		"func_length_avg":      m.FunctionLength.Avg,
		"file_size_avg":        m.FileSize.Avg,
		"afferent_coupling_avg": avgMapValues(m.AfferentCoupling),
		"efferent_coupling_avg": avgMapValues(m.EfferentCoupling),
		"duplication_rate":      m.DuplicationRate,
	}, nil, evidence
}

// extractC1Complexity collects top functions by cyclomatic complexity.
func extractC1Complexity(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.Functions) == 0 {
		return
	}
	sorted := sortedFunctions(m.Functions, func(a, b types.FunctionMetric) bool {
		return a.Complexity > b.Complexity
	})
	evidence["complexity_avg"] = topNEvidence(len(sorted), func(i int) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: sorted[i].File, Line: sorted[i].Line,
			Value:       float64(sorted[i].Complexity),
			Description: fmt.Sprintf("%s has complexity %d", sorted[i].Name, sorted[i].Complexity),
		}
	})
}

// extractC1FuncLength collects top functions by line count.
func extractC1FuncLength(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.Functions) == 0 {
		return
	}
	sorted := sortedFunctions(m.Functions, func(a, b types.FunctionMetric) bool {
		return a.LineCount > b.LineCount
	})
	evidence["func_length_avg"] = topNEvidence(len(sorted), func(i int) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: sorted[i].File, Line: sorted[i].Line,
			Value:       float64(sorted[i].LineCount),
			Description: fmt.Sprintf("%s is %d lines", sorted[i].Name, sorted[i].LineCount),
		}
	})
}

// extractC1FileSize collects the largest file evidence.
func extractC1FileSize(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if m.FileSize.MaxEntity == "" {
		return
	}
	evidence["file_size_avg"] = []types.EvidenceItem{{
		FilePath: m.FileSize.MaxEntity, Value: float64(m.FileSize.Max),
		Description: fmt.Sprintf("largest file: %d lines", m.FileSize.Max),
	}}
}

// extractC1Coupling collects top packages by coupling counts.
func extractC1Coupling(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.AfferentCoupling) > 0 {
		evidence["afferent_coupling_avg"] = couplingEvidence(m.AfferentCoupling, "imported by %d packages")
	}
	if len(m.EfferentCoupling) > 0 {
		evidence["efferent_coupling_avg"] = couplingEvidence(m.EfferentCoupling, "imports %d packages")
	}
}

// extractC1Duplication collects top duplicate blocks by line count.
func extractC1Duplication(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.DuplicatedBlocks) == 0 {
		return
	}
	sorted := make([]types.DuplicateBlock, len(m.DuplicatedBlocks))
	copy(sorted, m.DuplicatedBlocks)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LineCount > sorted[j].LineCount
	})
	evidence["duplication_rate"] = topNEvidence(len(sorted), func(i int) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: sorted[i].FileA, Line: sorted[i].StartA,
			Value:       float64(sorted[i].LineCount),
			Description: fmt.Sprintf("%d-line duplicate block", sorted[i].LineCount),
		}
	})
}

// sortedFunctions returns a sorted copy of functions using the given less function.
func sortedFunctions(funcs []types.FunctionMetric, less func(a, b types.FunctionMetric) bool) []types.FunctionMetric {
	sorted := make([]types.FunctionMetric, len(funcs))
	copy(sorted, funcs)
	sort.Slice(sorted, func(i, j int) bool { return less(sorted[i], sorted[j]) })
	return sorted
}

// pkgCount pairs a package path with a count for sorting.
type pkgCount struct {
	pkg   string
	count int
}

// couplingEvidence builds top-N evidence items from a coupling map.
func couplingEvidence(coupling map[string]int, descFmt string) []types.EvidenceItem {
	entries := make([]pkgCount, 0, len(coupling))
	for pkg, count := range coupling {
		entries = append(entries, pkgCount{pkg, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})
	return topNEvidence(len(entries), func(i int) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: entries[i].pkg, Value: float64(entries[i].count),
			Description: fmt.Sprintf(descFmt, entries[i].count),
		}
	})
}
