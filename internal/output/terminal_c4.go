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
	raw, ok := ar.Metrics["c4"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C4Metrics)
	if !ok {
		return
	}

	bold := color.New(color.Bold)
	fmt.Fprintln(w)
	bold.Fprintln(w, "C4: Documentation Quality")
	fmt.Fprintln(w, "────────────────────────────────────────")

	if !m.Available {
		fmt.Fprintln(w, "  Not available")
		return
	}

	renderC4Presence(w, m)
	renderC4LLM(w, m, bold)

	if verbose {
		renderC4Verbose(w, m, bold)
	}
}

// renderC4Presence renders presence/absence indicators for documentation artifacts.
func renderC4Presence(w io.Writer, m *types.C4Metrics) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	if m.ReadmePresent {
		green.Fprintf(w, "  README:              present (%d words)\n", m.ReadmeWordCount)
	} else {
		red.Fprintln(w, "  README:              absent")
	}

	colorForFloatInverse(m.CommentDensity, c4CommentDensityRed, c4CommentDensityYellow).Fprintf(w, "  Comment density:     %.1f%%\n", m.CommentDensity)
	colorForFloatInverse(m.APIDocCoverage, c4APIDocRed, c4APIDocYellow).Fprintf(w, "  API doc coverage:    %.1f%%\n", m.APIDocCoverage)

	renderPresenceLine(w, "CHANGELOG", m.ChangelogPresent, green, red)
	renderPresenceLine(w, "Examples", m.ExamplesPresent, green, red)
	renderPresenceLine(w, "CONTRIBUTING", m.ContributingPresent, green, red)

	if m.DiagramsPresent {
		green.Fprintln(w, "  Diagrams:            present")
	} else {
		color.New(color.FgYellow).Fprintln(w, "  Diagrams:            absent")
	}
}

func renderPresenceLine(w io.Writer, label string, present bool, green, red *color.Color) {
	if present {
		green.Fprintf(w, "  %-22spresent\n", label+":")
	} else {
		red.Fprintf(w, "  %-22sabsent\n", label+":")
	}
}

// renderC4LLM renders LLM-based analysis metrics.
func renderC4LLM(w io.Writer, m *types.C4Metrics, bold *color.Color) {
	fmt.Fprintln(w)
	bold.Fprintln(w, "  LLM Analysis:")
	if !m.LLMEnabled {
		dim := color.New(color.FgHiBlack)
		dim.Fprintln(w, "    README clarity:      n/a (Claude CLI not detected)")
		dim.Fprintln(w, "    Example quality:     n/a")
		dim.Fprintln(w, "    Completeness:        n/a")
		dim.Fprintln(w, "    Cross-ref coherence: n/a")
		return
	}
	colorForIntInverse(m.ReadmeClarity, c4LLMScoreRed, c4LLMScoreYellow).Fprintf(w, "    README clarity:      %d/10\n", m.ReadmeClarity)
	colorForIntInverse(m.ExampleQuality, c4LLMScoreRed, c4LLMScoreYellow).Fprintf(w, "    Example quality:     %d/10\n", m.ExampleQuality)
	colorForIntInverse(m.Completeness, c4LLMScoreRed, c4LLMScoreYellow).Fprintf(w, "    Completeness:        %d/10\n", m.Completeness)
	colorForIntInverse(m.CrossRefCoherence, c4LLMScoreRed, c4LLMScoreYellow).Fprintf(w, "    Cross-ref coherence: %d/10\n", m.CrossRefCoherence)
	fmt.Fprintf(w, "    LLM cost:            $%.4f (%d tokens)\n", m.LLMCostUSD, m.LLMTokensUsed)
}

// renderC4Verbose renders detailed counts.
func renderC4Verbose(w io.Writer, m *types.C4Metrics, bold *color.Color) {
	fmt.Fprintln(w)
	bold.Fprintln(w, "  Detailed metrics:")
	fmt.Fprintf(w, "    Total source lines:  %d\n", m.TotalSourceLines)
	fmt.Fprintf(w, "    Comment lines:       %d\n", m.CommentLines)
	fmt.Fprintf(w, "    Public APIs:         %d\n", m.PublicAPIs)
	fmt.Fprintf(w, "    Documented APIs:     %d\n", m.DocumentedAPIs)
}
