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
		return extractC5Unavailable()
	}

	evidence := make(map[string][]types.EvidenceItem)
	extractC5Churn(m, evidence)
	extractC5Coupling(m, evidence)
	extractC5Authors(m, evidence)
	extractC5Hotspots(m, evidence)

	ensureEvidenceKeys(evidence, c5MetricKeys)

	return map[string]float64{
		"churn_rate":            m.ChurnRate,
		"temporal_coupling_pct": m.TemporalCouplingPct,
		"author_fragmentation":  m.AuthorFragmentation,
		"commit_stability":      m.CommitStability,
		"hotspot_concentration": m.HotspotConcentration,
	}, nil, evidence
}

var c5MetricKeys = []string{
	"churn_rate", "temporal_coupling_pct", "author_fragmentation",
	"commit_stability", "hotspot_concentration",
}

func extractC5Unavailable() (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	unavailable := make(map[string]bool)
	emptyEvidence := make(map[string][]types.EvidenceItem)
	for _, k := range c5MetricKeys {
		unavailable[k] = true
		emptyEvidence[k] = []types.EvidenceItem{}
	}
	return map[string]float64{}, unavailable, emptyEvidence
}

func extractC5Churn(m *types.C5Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.TopHotspots) == 0 {
		return
	}
	evidence["churn_rate"] = topNEvidence(len(m.TopHotspots), func(i int) types.EvidenceItem {
		h := m.TopHotspots[i]
		return types.EvidenceItem{
			FilePath: h.Path, Value: float64(h.CommitCount),
			Description: fmt.Sprintf("%d commits", h.CommitCount),
		}
	})
}

func extractC5Coupling(m *types.C5Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.CoupledPairs) == 0 {
		return
	}
	evidence["temporal_coupling_pct"] = topNEvidence(len(m.CoupledPairs), func(i int) types.EvidenceItem {
		p := m.CoupledPairs[i]
		return types.EvidenceItem{
			FilePath: p.FileA, Value: p.Coupling,
			Description: fmt.Sprintf("coupled with %s (%.0f%%)", p.FileB, p.Coupling),
		}
	})
}

func extractC5Authors(m *types.C5Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.TopHotspots) == 0 {
		return
	}
	sorted := make([]types.FileChurn, len(m.TopHotspots))
	copy(sorted, m.TopHotspots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AuthorCount > sorted[j].AuthorCount
	})
	evidence["author_fragmentation"] = topNEvidence(len(sorted), func(i int) types.EvidenceItem {
		return types.EvidenceItem{
			FilePath: sorted[i].Path, Value: float64(sorted[i].AuthorCount),
			Description: fmt.Sprintf("%d distinct authors", sorted[i].AuthorCount),
		}
	})
}

func extractC5Hotspots(m *types.C5Metrics, evidence map[string][]types.EvidenceItem) {
	if len(m.TopHotspots) == 0 {
		return
	}
	evidence["hotspot_concentration"] = topNEvidence(len(m.TopHotspots), func(i int) types.EvidenceItem {
		h := m.TopHotspots[i]
		return types.EvidenceItem{
			FilePath: h.Path, Value: float64(h.TotalChanges),
			Description: fmt.Sprintf("hotspot: %d changes", h.TotalChanges),
		}
	})
}
