package scoring

import (
	"fmt"
	"sort"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// extractC5 extracts C5 (Temporal Dynamics) metrics from an AnalysisResult and collects evidence.
func extractC5(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c5"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C5Metrics)
	if !ok {
		return nil, nil, nil
	}

	if !m.Available {
		return c5Unavailable()
	}

	evidence := make(map[string][]types.EvidenceItem)
	c5ChurnRateEvidence(evidence, m)
	c5TemporalCouplingEvidence(evidence, m)
	c5AuthorFragmentationEvidence(evidence, m)
	c5HotspotConcentrationEvidence(evidence, m)
	ensureEvidenceKeys(evidence, "churn_rate", "temporal_coupling_pct", "author_fragmentation", "commit_stability", "hotspot_concentration")

	return map[string]float64{
		"churn_rate":            m.ChurnRate,
		"temporal_coupling_pct": m.TemporalCouplingPct,
		"author_fragmentation":  m.AuthorFragmentation,
		"commit_stability":      m.CommitStability,
		"hotspot_concentration": m.HotspotConcentration,
	}, nil, evidence
}

// c5Unavailable returns empty results when C5 data is not available.
func c5Unavailable() (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	unavailable := map[string]bool{
		"churn_rate":            true,
		"temporal_coupling_pct": true,
		"author_fragmentation":  true,
		"commit_stability":      true,
		"hotspot_concentration": true,
	}
	emptyEvidence := make(map[string][]types.EvidenceItem)
	for k := range unavailable {
		emptyEvidence[k] = []types.EvidenceItem{}
	}
	return map[string]float64{}, unavailable, emptyEvidence
}

// c5ChurnRateEvidence collects top hotspots by commit count.
func c5ChurnRateEvidence(evidence map[string][]types.EvidenceItem, m *types.C5Metrics) {
	if len(m.TopHotspots) == 0 {
		return
	}
	limit := capLimit(len(m.TopHotspots), evidenceTopN)
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		h := m.TopHotspots[i]
		items[i] = types.EvidenceItem{
			FilePath:    h.Path,
			Line:        0,
			Value:       float64(h.CommitCount),
			Description: fmt.Sprintf("%d commits", h.CommitCount),
		}
	}
	evidence["churn_rate"] = items
}

// c5TemporalCouplingEvidence collects top coupled pairs.
func c5TemporalCouplingEvidence(evidence map[string][]types.EvidenceItem, m *types.C5Metrics) {
	if len(m.CoupledPairs) == 0 {
		return
	}
	limit := capLimit(len(m.CoupledPairs), evidenceTopN)
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		p := m.CoupledPairs[i]
		items[i] = types.EvidenceItem{
			FilePath:    p.FileA,
			Line:        0,
			Value:       p.Coupling,
			Description: fmt.Sprintf("coupled with %s (%.0f%%)", p.FileB, p.Coupling),
		}
	}
	evidence["temporal_coupling_pct"] = items
}

// c5AuthorFragmentationEvidence collects top hotspots by author count.
func c5AuthorFragmentationEvidence(evidence map[string][]types.EvidenceItem, m *types.C5Metrics) {
	if len(m.TopHotspots) == 0 {
		return
	}
	sorted := make([]types.FileChurn, len(m.TopHotspots))
	copy(sorted, m.TopHotspots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AuthorCount > sorted[j].AuthorCount
	})
	limit := capLimit(len(sorted), evidenceTopN)
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		h := sorted[i]
		items[i] = types.EvidenceItem{
			FilePath:    h.Path,
			Line:        0,
			Value:       float64(h.AuthorCount),
			Description: fmt.Sprintf("%d distinct authors", h.AuthorCount),
		}
	}
	evidence["author_fragmentation"] = items
}

// c5HotspotConcentrationEvidence collects top hotspots by total changes.
func c5HotspotConcentrationEvidence(evidence map[string][]types.EvidenceItem, m *types.C5Metrics) {
	if len(m.TopHotspots) == 0 {
		return
	}
	limit := capLimit(len(m.TopHotspots), evidenceTopN)
	items := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		h := m.TopHotspots[i]
		items[i] = types.EvidenceItem{
			FilePath:    h.Path,
			Line:        0,
			Value:       float64(h.TotalChanges),
			Description: fmt.Sprintf("hotspot: %d changes", h.TotalChanges),
		}
	}
	evidence["hotspot_concentration"] = items
}

// capLimit returns min(n, max).
func capLimit(n, max int) int {
	if n < max {
		return n
	}
	return max
}

// ensureEvidenceKeys ensures all given keys have at least empty arrays.
func ensureEvidenceKeys(evidence map[string][]types.EvidenceItem, keys ...string) {
	for _, key := range keys {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}
}
