package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// C3 metric color thresholds.
const (
	c3DirDepthGreen     = 4
	c3DirDepthYellow    = 7
	c3FanoutGreen       = 5.0
	c3FanoutYellow      = 10.0
	c3CircularDepsMax   = 2
	c3DeadExportsGreen  = 5
	c3DeadExportsYellow = 20
)

func renderC3(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c3"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C3Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C3: Architecture")
	fmt.Fprintln(w, "────────────────────────────────────────")

	dd := colorForInt(m.MaxDirectoryDepth, c3DirDepthGreen, c3DirDepthYellow)
	dd.Fprintf(w, "  Max directory depth: %d\n", m.MaxDirectoryDepth)
	fmt.Fprintf(w, "  Avg directory depth: %.1f\n", m.AvgDirectoryDepth)

	fo := colorForFloat(m.ModuleFanout.Avg, c3FanoutGreen, c3FanoutYellow)
	fo.Fprintf(w, "  Avg module fanout:   %.1f\n", m.ModuleFanout.Avg)

	circCount := len(m.CircularDeps)
	cc := colorForInt(circCount, 0, c3CircularDepsMax)
	cc.Fprintf(w, "  Circular deps:       %d\n", circCount)

	deadCount := len(m.DeadExports)
	dc := colorForInt(deadCount, c3DeadExportsGreen, c3DeadExportsYellow)
	dc.Fprintf(w, "  Dead exports:        %d\n", deadCount)

	// Verbose: coupling details + dead exports
	if verbose {
		if circCount > 0 {
			fmt.Fprintln(w)
			bold.Fprintln(w, "  Circular dependencies:")
			for i, cycle := range m.CircularDeps {
				fmt.Fprintf(w, "    %d. %s\n", i+1, joinCycle(cycle))
			}
		}

		if deadCount > 0 {
			fmt.Fprintln(w)
			bold.Fprintln(w, "  Dead exports:")
			for _, de := range m.DeadExports {
				fmt.Fprintf(w, "    %s %s.%s  (%s:%d)\n", de.Kind, de.Package, de.Name, de.File, de.Line)
			}
		}
	}
}
