package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// C2 metric color thresholds (inverse: higher is better for coverage/consistency).
const (
	c2TypeAnnotationRed    = 50.0
	c2TypeAnnotationYellow = 80.0
	c2NamingRed            = 70.0
	c2NamingYellow         = 90.0
	c2MagicNumGreen        = 5.0
	c2MagicNumYellow       = 15.0
)

func renderC2(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c2"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C2Metrics)
	if !ok || m.Aggregate == nil {
		return
	}

	agg := m.Aggregate

	fmt.Fprintln(w)
	bold.Fprintln(w, "C2: Semantic Explicitness")
	fmt.Fprintln(w, "────────────────────────────────────────")

	tc := colorForFloatInverse(agg.TypeAnnotationCoverage, c2TypeAnnotationRed, c2TypeAnnotationYellow)
	tc.Fprintf(w, "  Type annotation:     %.1f%%\n", agg.TypeAnnotationCoverage)

	nc := colorForFloatInverse(agg.NamingConsistency, c2NamingRed, c2NamingYellow)
	nc.Fprintf(w, "  Naming consistency:  %.1f%%\n", agg.NamingConsistency)

	mr := colorForFloat(agg.MagicNumberRatio, c2MagicNumGreen, c2MagicNumYellow)
	mr.Fprintf(w, "  Magic numbers:       %.1f per kLOC\n", agg.MagicNumberRatio)

	if agg.TypeStrictness >= 1 {
		color.New(color.FgGreen).Fprintf(w, "  Type strictness:     on\n")
	} else {
		color.New(color.FgYellow).Fprintf(w, "  Type strictness:     off\n")
	}

	ns := colorForFloatInverse(agg.NullSafety, 30, 60)
	ns.Fprintf(w, "  Null safety:         %.0f%%\n", agg.NullSafety)

	// Verbose: per-language C2 breakdown
	if verbose && len(m.PerLanguage) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Per-language C2 breakdown:")
		for lang, lm := range m.PerLanguage {
			strict := "off"
			if lm.TypeStrictness >= 1 {
				strict = "on"
			}
			fmt.Fprintf(w, "    %-12s type=%.0f%%  naming=%.0f%%  magic=%.1f/kLOC  strict=%s  null=%.0f%%  LOC=%d\n",
				string(lang)+":", lm.TypeAnnotationCoverage, lm.NamingConsistency,
				lm.MagicNumberRatio, strict, lm.NullSafety, lm.LOC)
		}
	}
}
