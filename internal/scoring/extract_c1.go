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
	c1ComplexityEvidence(m, evidence)
	c1FuncLengthEvidence(m, evidence)
	c1FileSizeEvidence(m, evidence)
	c1CouplingEvidence(m, evidence)
	c1DuplicationEvidence(m, evidence)

	for _, key := range []string{"complexity_avg", "func_length_avg", "file_size_avg", "afferent_coupling_avg", "efferent_coupling_avg", "duplication_rate"} {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}

	return map[string]float64{
		"complexity_avg":        m.CyclomaticComplexity.Avg,
		"func_length_avg":      m.FunctionLength.Avg,
		"file_size_avg":        m.FileSize.Avg,
		"afferent_coupling_avg": avgMapValues(m.AfferentCoupling),
		"efferent_coupling_avg": avgMapValues(m.EfferentCoupling),
		"duplication_rate":      m.DuplicationRate,
	}, nil, evidence
}

func c1ComplexityEvidence(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.Functions) == 0 {
		return
	}
	sorted := make([]types.FunctionMetric, len(m.Functions))
	copy(sorted, m.Functions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Complexity > sorted[j].Complexity
	})
	limit := min(evidenceTopN, len(sorted))
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		items[i] = types.EvidenceItem{
			FilePath:    sorted[i].File,
			Line:        sorted[i].Line,
			Value:       float64(sorted[i].Complexity),
			Description: fmt.Sprintf("%s has complexity %d", sorted[i].Name, sorted[i].Complexity),
		}
	}
	evidence["complexity_avg"] = items
}

func c1FuncLengthEvidence(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.Functions) == 0 {
		return
	}
	sorted := make([]types.FunctionMetric, len(m.Functions))
	copy(sorted, m.Functions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LineCount > sorted[j].LineCount
	})
	limit := min(evidenceTopN, len(sorted))
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		items[i] = types.EvidenceItem{
			FilePath:    sorted[i].File,
			Line:        sorted[i].Line,
			Value:       float64(sorted[i].LineCount),
			Description: fmt.Sprintf("%s is %d lines", sorted[i].Name, sorted[i].LineCount),
		}
	}
	evidence["func_length_avg"] = items
}

func c1FileSizeEvidence(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if m.FileSize.MaxEntity == "" {
		return
	}
	evidence["file_size_avg"] = []types.EvidenceItem{{
		FilePath:    m.FileSize.MaxEntity,
		Line:        0,
		Value:       float64(m.FileSize.Max),
		Description: fmt.Sprintf("largest file: %d lines", m.FileSize.Max),
	}}
}

func c1CouplingEvidence(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	collectCouplingEvidence(m.AfferentCoupling, "imported by %d packages", "afferent_coupling_avg", evidence)
	collectCouplingEvidence(m.EfferentCoupling, "imports %d packages", "efferent_coupling_avg", evidence)
}

func collectCouplingEvidence(coupling map[string]int, descFmt, key string, evidence map[string][]types.EvidenceItem) {
	if len(coupling) == 0 {
		return
	}
	type pkgCount struct {
		pkg   string
		count int
	}
	entries := make([]pkgCount, 0, len(coupling))
	for pkg, count := range coupling {
		entries = append(entries, pkgCount{pkg, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})
	limit := min(evidenceTopN, len(entries))
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		items[i] = types.EvidenceItem{
			FilePath:    entries[i].pkg,
			Line:        0,
			Value:       float64(entries[i].count),
			Description: fmt.Sprintf(descFmt, entries[i].count),
		}
	}
	evidence[key] = items
}

func c1DuplicationEvidence(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.DuplicatedBlocks) == 0 {
		return
	}
	sorted := make([]types.DuplicateBlock, len(m.DuplicatedBlocks))
	copy(sorted, m.DuplicatedBlocks)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LineCount > sorted[j].LineCount
	})
	limit := min(evidenceTopN, len(sorted))
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		items[i] = types.EvidenceItem{
			FilePath:    sorted[i].FileA,
			Line:        sorted[i].StartA,
			Value:       float64(sorted[i].LineCount),
			Description: fmt.Sprintf("%d-line duplicate block", sorted[i].LineCount),
		}
	}
	evidence["duplication_rate"] = items
}
