package scoring

import (
	"fmt"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// extractC5 extracts C5 (Temporal Dynamics) metrics from an AnalysisResult and collects evidence.
func extractC5(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	// Check availability before delegating to the generic helper so that the
	// non-nil unavailable map is returned correctly when git data is absent.
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

	return extractMetrics[types.C5Metrics, *types.C5Metrics](ar, "c5", func(m *types.C5Metrics) (map[string]float64, map[string][]types.EvidenceItem, []string) {
		evidence := make(map[string][]types.EvidenceItem)
		c5ChurnRateEvidence(evidence, m)
		c5TemporalCouplingEvidence(evidence, m)
		c5AuthorFragmentationEvidence(evidence, m)
		c5HotspotConcentrationEvidence(evidence, m)

		values := map[string]float64{
			"churn_rate":            m.ChurnRate,
			"temporal_coupling_pct": m.TemporalCouplingPct,
			"author_fragmentation":  m.AuthorFragmentation,
			"commit_stability":      m.CommitStability,
			"hotspot_concentration": m.HotspotConcentration,
			"repo_staleness_months": m.RepoStalenessMonths,
		}

		// Apply staleness as a category-level multiplier via the _multiplier convention.
		// Active repos (< 18 months) get factor 1.0 (no change).
		// Stale repos get progressively lower factors, reducing the C5 category score.
		values["_multiplier"] = c5StalenessFactor(m.RepoStalenessMonths)

		return values, evidence, []string{"churn_rate", "temporal_coupling_pct", "author_fragmentation", "commit_stability", "hotspot_concentration"}
	})
}

// c5StalenessBreakpoints maps repo staleness (months) to a score multiplier (0.0–1.0).
// Progressive penalty: no penalty under 18 months, then smooth degradation.
var c5StalenessBreakpoints = []Breakpoint{
	{Value: 0, Score: 1.0},   // just committed
	{Value: 18, Score: 1.0},  // threshold: no penalty
	{Value: 24, Score: 0.95}, // ~2 years: very mild
	{Value: 36, Score: 0.75}, // 3 years: noticeable
	{Value: 60, Score: 0.50}, // 5 years: significant
	{Value: 72, Score: 0.35}, // 6 years: heavy
	{Value: 120, Score: 0.10}, // 10+ years: near-zero
}

// c5StalenessFactor returns a multiplier (0.0–1.0) for the C5 category score
// based on how many months have passed since the repo's last commit.
func c5StalenessFactor(months float64) float64 {
	return Interpolate(c5StalenessBreakpoints, months)
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
	evidence["churn_rate"] = buildTopNEvidence(
		m.TopHotspots,
		nil,
		func(h types.FileChurn) float64 { return float64(h.CommitCount) },
		func(h types.FileChurn) string { return h.Path },
		nil,
		func(h types.FileChurn) string { return fmt.Sprintf("%d commits", h.CommitCount) },
	)
}

// c5TemporalCouplingEvidence collects top coupled pairs.
func c5TemporalCouplingEvidence(evidence map[string][]types.EvidenceItem, m *types.C5Metrics) {
	evidence["temporal_coupling_pct"] = buildTopNEvidence(
		m.CoupledPairs,
		nil,
		func(p types.CoupledPair) float64 { return p.Coupling },
		func(p types.CoupledPair) string { return p.FileA },
		nil,
		func(p types.CoupledPair) string {
			return fmt.Sprintf("coupled with %s (%.0f%%)", p.FileB, p.Coupling)
		},
	)
}

// c5AuthorFragmentationEvidence collects top hotspots by author count.
func c5AuthorFragmentationEvidence(evidence map[string][]types.EvidenceItem, m *types.C5Metrics) {
	evidence["author_fragmentation"] = buildTopNEvidence(
		m.TopHotspots,
		func(a, b types.FileChurn) bool { return a.AuthorCount > b.AuthorCount },
		func(h types.FileChurn) float64 { return float64(h.AuthorCount) },
		func(h types.FileChurn) string { return h.Path },
		nil,
		func(h types.FileChurn) string { return fmt.Sprintf("%d distinct authors", h.AuthorCount) },
	)
}

// c5HotspotConcentrationEvidence collects top hotspots by total changes.
func c5HotspotConcentrationEvidence(evidence map[string][]types.EvidenceItem, m *types.C5Metrics) {
	evidence["hotspot_concentration"] = buildTopNEvidence(
		m.TopHotspots,
		nil,
		func(h types.FileChurn) float64 { return float64(h.TotalChanges) },
		func(h types.FileChurn) string { return h.Path },
		nil,
		func(h types.FileChurn) string { return fmt.Sprintf("hotspot: %d changes", h.TotalChanges) },
	)
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
