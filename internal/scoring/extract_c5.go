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

	evidence := make(map[string][]types.EvidenceItem)

	// churn_rate: top 5 hotspots by commit count
	if len(m.TopHotspots) > 0 {
		limit := evidenceTopN
		if len(m.TopHotspots) < limit {
			limit = len(m.TopHotspots)
		}
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

	// temporal_coupling_pct: top 5 coupled pairs
	if len(m.CoupledPairs) > 0 {
		limit := evidenceTopN
		if len(m.CoupledPairs) < limit {
			limit = len(m.CoupledPairs)
		}
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

	// author_fragmentation: top 5 hotspots by author count
	if len(m.TopHotspots) > 0 {
		sorted := make([]types.FileChurn, len(m.TopHotspots))
		copy(sorted, m.TopHotspots)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].AuthorCount > sorted[j].AuthorCount
		})
		limit := evidenceTopN
		if len(sorted) < limit {
			limit = len(sorted)
		}
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

	// hotspot_concentration: top 5 hotspots by total changes
	if len(m.TopHotspots) > 0 {
		limit := evidenceTopN
		if len(m.TopHotspots) < limit {
			limit = len(m.TopHotspots)
		}
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

	// Ensure all 5 keys have at least empty arrays
	for _, key := range []string{"churn_rate", "temporal_coupling_pct", "author_fragmentation", "commit_stability", "hotspot_concentration"} {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}

	return map[string]float64{
		"churn_rate":            m.ChurnRate,
		"temporal_coupling_pct": m.TemporalCouplingPct,
		"author_fragmentation":  m.AuthorFragmentation,
		"commit_stability":      m.CommitStability,
		"hotspot_concentration": m.HotspotConcentration,
	}, nil, evidence
}
