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

	renderC4StaticMetrics(w, m)
	renderC4LLMMetrics(w, m, bold)
	if verbose {
		renderC4VerboseMetrics(w, m, bold)
	}
}

// renderC4StaticMetrics renders README, comment density, API docs, and presence checks.
func renderC4StaticMetrics(w io.Writer, m *types.C4Metrics) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	if m.ReadmePresent {
		green.Fprintf(w, "  README:              present (%d words)\n", m.ReadmeWordCount)
	} else {
		red.Fprintln(w, "  README:              absent")
	}

	cd := colorForFloatInverse(m.CommentDensity, c4CommentDensityRed, c4CommentDensityYellow)
	cd.Fprintf(w, "  Comment density:     %.1f%%\n", m.CommentDensity)

	ad := colorForFloatInverse(m.APIDocCoverage, c4APIDocRed, c4APIDocYellow)
	ad.Fprintf(w, "  API doc coverage:    %.1f%%\n", m.APIDocCoverage)

	renderPresenceFlag(w, "CHANGELOG", m.ChangelogPresent, green, red)
	renderPresenceFlag(w, "Examples", m.ExamplesPresent, green, red)
	renderPresenceFlag(w, "CONTRIBUTING", m.ContributingPresent, green, red)
	renderPresenceFlagYellow(w, "Diagrams", m.DiagramsPresent, green)
}

// renderPresenceFlag renders a present/absent line with green/red coloring.
func renderPresenceFlag(w io.Writer, label string, present bool, green, red *color.Color) {
	padded := fmt.Sprintf("%-22s", label+":")
	if present {
		green.Fprintf(w, "  %s present\n", padded)
	} else {
		red.Fprintf(w, "  %s absent\n", padded)
	}
}

// renderPresenceFlagYellow renders a present/absent line with green/yellow coloring.
func renderPresenceFlagYellow(w io.Writer, label string, present bool, green *color.Color) {
	padded := fmt.Sprintf("%-22s", label+":")
	if present {
		green.Fprintf(w, "  %s present\n", padded)
	} else {
		color.New(color.FgYellow).Fprintf(w, "  %s absent\n", padded)
	}
}

// renderC4LLMMetrics renders LLM-based analysis scores.
func renderC4LLMMetrics(w io.Writer, m *types.C4Metrics, bold *color.Color) {
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
		dim := color.New(color.FgHiBlack)
		dim.Fprintln(w, "    README clarity:      n/a (Claude CLI not detected)")
		dim.Fprintln(w, "    Example quality:     n/a")
		dim.Fprintln(w, "    Completeness:        n/a")
		dim.Fprintln(w, "    Cross-ref coherence: n/a")
	}
}

// renderC4VerboseMetrics renders detailed count metrics.
func renderC4VerboseMetrics(w io.Writer, m *types.C4Metrics, bold *color.Color) {
	fmt.Fprintln(w)
	bold.Fprintln(w, "  Detailed metrics:")
	fmt.Fprintf(w, "    Total source lines:  %d\n", m.TotalSourceLines)
	fmt.Fprintf(w, "    Comment lines:       %d\n", m.CommentLines)
	fmt.Fprintf(w, "    Public APIs:         %d\n", m.PublicAPIs)
	fmt.Fprintf(w, "    Documented APIs:     %d\n", m.DocumentedAPIs)
}
