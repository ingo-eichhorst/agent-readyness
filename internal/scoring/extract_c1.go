package scoring

import (
	"fmt"
	"sort"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)


// extractC1 extracts C1 (Code Health) metrics from an AnalysisResult and collects evidence.
func extractC1(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	return extractMetrics[types.C1Metrics, *types.C1Metrics](ar, "c1", func(m *types.C1Metrics) (map[string]float64, map[string][]types.EvidenceItem, []string) {
		evidence := make(map[string][]types.EvidenceItem)
		c1ComplexityEvidence(m, evidence)
		c1FuncLengthEvidence(m, evidence)
		c1FileSizeEvidence(m, evidence)
		c1MaxFileSizeEvidence(m, evidence)
		c1LargeFilePctEvidence(m, evidence)
		c1CouplingEvidence(m, evidence)
		c1DuplicationEvidence(m, evidence)

		return map[string]float64{
			"complexity_avg":        m.CyclomaticComplexity.Avg,
			"func_length_avg":       m.FunctionLength.Avg,
			"file_size_avg":         m.FileSize.Avg,
			"max_file_size":         float64(m.FileSize.Max),
			"large_file_pct":        m.LargeFilePct,
			"afferent_coupling_avg": avgMapValues(m.AfferentCoupling),
			"efferent_coupling_avg": avgMapValues(m.EfferentCoupling),
			"duplication_rate":      m.DuplicationRate,
		}, evidence, []string{"complexity_avg", "func_length_avg", "file_size_avg", "max_file_size", "large_file_pct", "afferent_coupling_avg", "efferent_coupling_avg", "duplication_rate"}
	})
}

func c1ComplexityEvidence(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	evidence["complexity_avg"] = buildTopNEvidence(
		m.Functions,
		func(a, b types.FunctionMetric) bool { return a.Complexity > b.Complexity },
		func(f types.FunctionMetric) float64 { return float64(f.Complexity) },
		func(f types.FunctionMetric) string { return f.File },
		func(f types.FunctionMetric) int { return f.Line },
		func(f types.FunctionMetric) string {
			return fmt.Sprintf("%s has complexity %d", f.Name, f.Complexity)
		},
	)
}

func c1FuncLengthEvidence(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	evidence["func_length_avg"] = buildTopNEvidence(
		m.Functions,
		func(a, b types.FunctionMetric) bool { return a.LineCount > b.LineCount },
		func(f types.FunctionMetric) float64 { return float64(f.LineCount) },
		func(f types.FunctionMetric) string { return f.File },
		func(f types.FunctionMetric) int { return f.Line },
		func(f types.FunctionMetric) string {
			return fmt.Sprintf("%s is %d lines", f.Name, f.LineCount)
		},
	)
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

func c1MaxFileSizeEvidence(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if m.FileSize.MaxEntity == "" {
		return
	}
	evidence["max_file_size"] = []types.EvidenceItem{{
		FilePath:    m.FileSize.MaxEntity,
		Line:        0,
		Value:       float64(m.FileSize.Max),
		Description: fmt.Sprintf("largest file: %d lines", m.FileSize.Max),
	}}
}

func c1LargeFilePctEvidence(m *types.C1Metrics, evidence map[string][]types.EvidenceItem) {
	if m.LargeFilePct > 0 && m.FileSize.MaxEntity != "" {
		evidence["large_file_pct"] = []types.EvidenceItem{{
			FilePath:    m.FileSize.MaxEntity,
			Value:       m.LargeFilePct,
			Description: fmt.Sprintf("%.1f%% of files exceed 500 LOC", m.LargeFilePct),
		}}
	}
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
	evidence["duplication_rate"] = buildTopNEvidence(
		m.DuplicatedBlocks,
		func(a, b types.DuplicateBlock) bool { return a.LineCount > b.LineCount },
		func(d types.DuplicateBlock) float64 { return float64(d.LineCount) },
		func(d types.DuplicateBlock) string { return d.FileA },
		func(d types.DuplicateBlock) int { return d.StartA },
		func(d types.DuplicateBlock) string {
			return fmt.Sprintf("%d-line duplicate block", d.LineCount)
		},
	)
}
