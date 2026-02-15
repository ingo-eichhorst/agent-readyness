package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// C4 metric color thresholds (inverse: higher is better).
const (
	c4CommentDensityRed    = 5.0
	c4CommentDensityYellow = 15.0
	c4APIDocRed            = 30.0
	c4APIDocYellow         = 60.0
	c4LLMScoreRed          = 4
	c4LLMScoreYellow       = 7
)

func renderC4(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	raw, ok := ar.Metrics["c4"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C4Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C4: Documentation Quality")
	fmt.Fprintln(w, "────────────────────────────────────────")

	if !m.Available {
		fmt.Fprintln(w, "  Not available")
		return
	}

	// README
	if m.ReadmePresent {
		green.Fprintf(w, "  README:              present (%d words)\n", m.ReadmeWordCount)
	} else {
		red.Fprintln(w, "  README:              absent")
	}

	// Comment density
	cd := colorForFloatInverse(m.CommentDensity, c4CommentDensityRed, c4CommentDensityYellow)
	cd.Fprintf(w, "  Comment density:     %.1f%%\n", m.CommentDensity)

	// API doc coverage
	ad := colorForFloatInverse(m.APIDocCoverage, c4APIDocRed, c4APIDocYellow)
	ad.Fprintf(w, "  API doc coverage:    %.1f%%\n", m.APIDocCoverage)

	// CHANGELOG
	if m.ChangelogPresent {
		green.Fprintln(w, "  CHANGELOG:           present")
	} else {
		red.Fprintln(w, "  CHANGELOG:           absent")
	}

	// Examples
	if m.ExamplesPresent {
		green.Fprintln(w, "  Examples:            present")
	} else {
		red.Fprintln(w, "  Examples:            absent")
	}

	// CONTRIBUTING
	if m.ContributingPresent {
		green.Fprintln(w, "  CONTRIBUTING:        present")
	} else {
		red.Fprintln(w, "  CONTRIBUTING:        absent")
	}

	// Diagrams
	if m.DiagramsPresent {
		green.Fprintln(w, "  Diagrams:            present")
	} else {
		color.New(color.FgYellow).Fprintln(w, "  Diagrams:            absent")
	}

	// LLM-based metrics (if enabled)
	fmt.Fprintln(w)
	bold.Fprintln(w, "  LLM Analysis:")
	if m.LLMEnabled {
		rc := colorForIntInverse(m.ReadmeClarity, c4LLMScoreRed, c4LLMScoreYellow)
		rc.Fprintf(w, "    README clarity:      %d/10\n", m.ReadmeClarity)
		eq := colorForIntInverse(m.ExampleQuality, c4LLMScoreRed, c4LLMScoreYellow)
		eq.Fprintf(w, "    Example quality:     %d/10\n", m.ExampleQuality)
		cp := colorForIntInverse(m.Completeness, c4LLMScoreRed, c4LLMScoreYellow)
		cp.Fprintf(w, "    Completeness:        %d/10\n", m.Completeness)
		cr := colorForIntInverse(m.CrossRefCoherence, c4LLMScoreRed, c4LLMScoreYellow)
		cr.Fprintf(w, "    Cross-ref coherence: %d/10\n", m.CrossRefCoherence)
		fmt.Fprintf(w, "    LLM cost:            $%.4f (%d tokens)\n", m.LLMCostUSD, m.LLMTokensUsed)
	} else {
		color.New(color.FgHiBlack).Fprintln(w, "    README clarity:      n/a (Claude CLI not detected)")
		color.New(color.FgHiBlack).Fprintln(w, "    Example quality:     n/a")
		color.New(color.FgHiBlack).Fprintln(w, "    Completeness:        n/a")
		color.New(color.FgHiBlack).Fprintln(w, "    Cross-ref coherence: n/a")
	}

	// Verbose: show counts
	if verbose {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Detailed metrics:")
		fmt.Fprintf(w, "    Total source lines:  %d\n", m.TotalSourceLines)
		fmt.Fprintf(w, "    Comment lines:       %d\n", m.CommentLines)
		fmt.Fprintf(w, "    Public APIs:         %d\n", m.PublicAPIs)
		fmt.Fprintf(w, "    Documented APIs:     %d\n", m.DocumentedAPIs)
	}
}
