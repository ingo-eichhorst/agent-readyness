package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// C5 metric color thresholds.
const (
	c5ChurnGreen            = 100.0
	c5ChurnYellow           = 300.0
	c5TemporalCouplingGreen = 10.0
	c5TemporalCouplingYellow = 30.0
	c5AuthorFragGreen       = 2.0
	c5AuthorFragYellow      = 4.0
	c5CommitStabilityRed    = 3.0
	c5CommitStabilityYellow = 7.0
	c5HotspotGreen          = 50.0
	c5HotspotYellow         = 75.0
)

func renderC5(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	m := extractC5Metrics(ar)
	if m == nil {
		return
	}

	renderC5Header(w, bold)

	if !m.Available {
		fmt.Fprintln(w, "  Not available (no .git directory)")
		return
	}

	renderC5CoreMetrics(w, m)
	renderC5VerboseDetails(w, m, verbose, bold)
}

func extractC5Metrics(ar *types.AnalysisResult) *types.C5Metrics {
	raw, ok := ar.Metrics["c5"]
	if !ok {
		return nil
	}
	m, ok := raw.(*types.C5Metrics)
	if !ok {
		return nil
	}
	return m
}

func renderC5Header(w io.Writer, bold *color.Color) {
	fmt.Fprintln(w)
	bold.Fprintln(w, "C5: Temporal Dynamics")
	fmt.Fprintln(w, "────────────────────────────────────────")
}

func renderC5CoreMetrics(w io.Writer, m *types.C5Metrics) {
	fmt.Fprintf(w, "  Total commits:       %d (%d-day window)\n", m.TotalCommits, m.TimeWindowDays)

	cr := colorForFloat(m.ChurnRate, c5ChurnGreen, c5ChurnYellow)
	cr.Fprintf(w, "  Churn rate:          %.1f lines/commit\n", m.ChurnRate)

	tc := colorForFloat(m.TemporalCouplingPct, c5TemporalCouplingGreen, c5TemporalCouplingYellow)
	tc.Fprintf(w, "  Temporal coupling:   %.1f%%\n", m.TemporalCouplingPct)

	af := colorForFloat(m.AuthorFragmentation, c5AuthorFragGreen, c5AuthorFragYellow)
	af.Fprintf(w, "  Author fragmentation: %.2f avg authors/file\n", m.AuthorFragmentation)

	cs := colorForFloatInverse(m.CommitStability, c5CommitStabilityRed, c5CommitStabilityYellow)
	cs.Fprintf(w, "  Commit stability:    %.1f days median\n", m.CommitStability)

	hc := colorForFloat(m.HotspotConcentration, c5HotspotGreen, c5HotspotYellow)
	hc.Fprintf(w, "  Hotspot concentration: %.1f%%\n", m.HotspotConcentration)
}

func renderC5VerboseDetails(w io.Writer, m *types.C5Metrics, verbose bool, bold *color.Color) {
	if !verbose {
		return
	}

	if len(m.TopHotspots) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Top hotspots:")
		for _, h := range m.TopHotspots {
			fmt.Fprintf(w, "    %s  changes=%d commits=%d authors=%d\n", h.Path, h.TotalChanges, h.CommitCount, h.AuthorCount)
		}
	}

	if len(m.CoupledPairs) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Coupled pairs (>70%% co-change):")
		limit := coupledPairsTopN
		if len(m.CoupledPairs) < limit {
			limit = len(m.CoupledPairs)
		}
		for _, cp := range m.CoupledPairs[:limit] {
			fmt.Fprintf(w, "    %s <-> %s  %.0f%% (%d shared commits)\n", cp.FileA, cp.FileB, cp.Coupling, cp.SharedCommits)
		}
	}
}
