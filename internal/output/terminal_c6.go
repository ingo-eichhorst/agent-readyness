package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// C6 metric color thresholds.
const (
	c6TestRatioRed        = 0.2
	c6TestRatioYellow     = 0.5
	c6CoverageRed         = 40.0
	c6CoverageYellow      = 70.0
	c6IsolationRed        = 50.0
	c6IsolationYellow     = 80.0
	c6AssertionNoThreshold = 999.0 // Sentinel: no meaningful yellow/red threshold
)

func renderC6(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c6"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C6Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C6: Testing")
	fmt.Fprintln(w, "────────────────────────────────────────")

	tr := colorForFloatInverse(m.TestToCodeRatio, c6TestRatioRed, c6TestRatioYellow)
	tr.Fprintf(w, "  Test-to-code ratio:  %.2f\n", m.TestToCodeRatio)

	if m.CoveragePercent >= 0 {
		cov := colorForFloatInverse(m.CoveragePercent, c6CoverageRed, c6CoverageYellow)
		cov.Fprintf(w, "  Coverage:            %.1f%% (%s)\n", m.CoveragePercent, m.CoverageSource)
	} else {
		fmt.Fprintf(w, "  Coverage:            n/a (no coverage data found)\n")
	}

	iso := colorForFloatInverse(m.TestIsolation, c6IsolationRed, c6IsolationYellow)
	iso.Fprintf(w, "  Test isolation:      %.0f%%\n", m.TestIsolation)

	ad := colorForFloat(m.AssertionDensity.Avg, c6AssertionNoThreshold, c6AssertionNoThreshold)
	ad.Fprintf(w, "  Assertion density:   %.1f avg\n", m.AssertionDensity.Avg)

	// Verbose: per-test function details
	if verbose && len(m.TestFunctions) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Test functions:")
		for _, tf := range m.TestFunctions {
			extDep := ""
			if tf.HasExternalDep {
				extDep = " [external-dep]"
			}
			fmt.Fprintf(w, "    %s.%s  assertions=%d%s  (%s:%d)\n", tf.Package, tf.Name, tf.AssertionCount, extDep, tf.File, tf.Line)
		}
	}
}
