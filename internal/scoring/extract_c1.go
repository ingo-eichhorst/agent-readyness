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

	// Initialize evidence map for tracking top offenders per metric
	evidence := make(map[string][]types.EvidenceItem)

	// complexity_avg: top 5 functions by cyclomatic complexity (worst offenders)
	if len(m.Functions) > 0 {
		sorted := make([]types.FunctionMetric, len(m.Functions))
		copy(sorted, m.Functions)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Complexity > sorted[j].Complexity
		})
		limit := evidenceTopN
		if len(sorted) < limit {
			limit = len(sorted)
		}
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

	// func_length_avg: top 5 functions by line count
	if len(m.Functions) > 0 {
		sorted := make([]types.FunctionMetric, len(m.Functions))
		copy(sorted, m.Functions)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].LineCount > sorted[j].LineCount
		})
		limit := evidenceTopN
		if len(sorted) < limit {
			limit = len(sorted)
		}
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

	// file_size_avg: single worst file
	if m.FileSize.MaxEntity != "" {
		evidence["file_size_avg"] = []types.EvidenceItem{{
			FilePath:    m.FileSize.MaxEntity,
			Line:        0,
			Value:       float64(m.FileSize.Max),
			Description: fmt.Sprintf("largest file: %d lines", m.FileSize.Max),
		}}
	}

	// afferent_coupling_avg: top 5 packages by incoming dependency count
	if len(m.AfferentCoupling) > 0 {
		type pkgCount struct {
			pkg   string
			count int
		}
		entries := make([]pkgCount, 0, len(m.AfferentCoupling))
		for pkg, count := range m.AfferentCoupling {
			entries = append(entries, pkgCount{pkg, count})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].count > entries[j].count
		})
		limit := evidenceTopN
		if len(entries) < limit {
			limit = len(entries)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			items[i] = types.EvidenceItem{
				FilePath:    entries[i].pkg,
				Line:        0,
				Value:       float64(entries[i].count),
				Description: fmt.Sprintf("imported by %d packages", entries[i].count),
			}
		}
		evidence["afferent_coupling_avg"] = items
	}

	// efferent_coupling_avg: top 5 packages by outgoing dependency count
	if len(m.EfferentCoupling) > 0 {
		type pkgCount struct {
			pkg   string
			count int
		}
		entries := make([]pkgCount, 0, len(m.EfferentCoupling))
		for pkg, count := range m.EfferentCoupling {
			entries = append(entries, pkgCount{pkg, count})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].count > entries[j].count
		})
		limit := evidenceTopN
		if len(entries) < limit {
			limit = len(entries)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			items[i] = types.EvidenceItem{
				FilePath:    entries[i].pkg,
				Line:        0,
				Value:       float64(entries[i].count),
				Description: fmt.Sprintf("imports %d packages", entries[i].count),
			}
		}
		evidence["efferent_coupling_avg"] = items
	}

	// duplication_rate: top 5 duplicate blocks by line count
	if len(m.DuplicatedBlocks) > 0 {
		sorted := make([]types.DuplicateBlock, len(m.DuplicatedBlocks))
		copy(sorted, m.DuplicatedBlocks)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].LineCount > sorted[j].LineCount
		})
		limit := evidenceTopN
		if len(sorted) < limit {
			limit = len(sorted)
		}
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

	// Ensure all 6 metric keys have at least empty arrays (never nil)
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
