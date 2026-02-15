package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Display thresholds for C7 score coloring (0-100 scale).
const (
	c7ScoreGreenMin  = 70
	c7ScoreYellowMin = 40
	c7ScoreScale     = 10 // Multiplier to convert 1-10 scores to 0-100 for color
)

// c7ScoreColor returns a color based on C7 score (0-100, higher is better).
func c7ScoreColor(score int) *color.Color {
	if score >= c7ScoreGreenMin {
		return color.New(color.FgGreen)
	}
	if score >= c7ScoreYellowMin {
		return color.New(color.FgYellow)
	}
	return color.New(color.FgRed)
}

func renderC7(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c7"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C7Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C7: Agent Evaluation")
	fmt.Fprintln(w, "────────────────────────────────────────")

	if !m.Available {
		fmt.Fprintln(w, "  Not available (LLM features disabled)")
		return
	}

	// New MECE metrics (1-10 scale)
	if m.TaskExecutionConsistency > 0 || m.CodeBehaviorComprehension > 0 ||
		m.CrossFileNavigation > 0 || m.IdentifierInterpretability > 0 ||
		m.DocumentationAccuracyDetection > 0 {
		// Show new MECE metrics
		m1c := c7ScoreColor(m.TaskExecutionConsistency * c7ScoreScale)
		m1c.Fprintf(w, "  M1 Exec Consistency:  %d/10\n", m.TaskExecutionConsistency)

		m2c := c7ScoreColor(m.CodeBehaviorComprehension * c7ScoreScale)
		m2c.Fprintf(w, "  M2 Comprehension:     %d/10\n", m.CodeBehaviorComprehension)

		m3c := c7ScoreColor(m.CrossFileNavigation * c7ScoreScale)
		m3c.Fprintf(w, "  M3 Navigation:        %d/10\n", m.CrossFileNavigation)

		m4c := c7ScoreColor(m.IdentifierInterpretability * c7ScoreScale)
		m4c.Fprintf(w, "  M4 Identifiers:       %d/10\n", m.IdentifierInterpretability)

		m5c := c7ScoreColor(m.DocumentationAccuracyDetection * c7ScoreScale)
		m5c.Fprintf(w, "  M5 Documentation:     %d/10\n", m.DocumentationAccuracyDetection)
	} else {
		// Fallback to legacy metrics for backward compatibility
		ic := c7ScoreColor(m.IntentClarity)
		ic.Fprintf(w, "  Intent clarity:       %d/100\n", m.IntentClarity)

		mc := c7ScoreColor(m.ModificationConfidence)
		mc.Fprintf(w, "  Modification conf:    %d/100\n", m.ModificationConfidence)

		cfc := c7ScoreColor(m.CrossFileCoherence)
		cfc.Fprintf(w, "  Cross-file coherence: %d/100\n", m.CrossFileCoherence)

		sc := c7ScoreColor(m.SemanticCompleteness)
		sc.Fprintf(w, "  Semantic complete:    %d/100\n", m.SemanticCompleteness)
	}

	// Summary metrics
	fmt.Fprintln(w, "  ─────────────────────────────────────")
	if m.MECEScore > 0 {
		// Show MECE score (weighted average of 5 metrics, 1-10 scale)
		os := c7ScoreColor(int(m.MECEScore * float64(c7ScoreScale)))
		os.Fprintf(w, "  MECE Score:           %.1f/10\n", m.MECEScore)
	} else {
		// Show legacy overall score (0-100 scale)
		os := c7ScoreColor(int(m.OverallScore))
		os.Fprintf(w, "  Overall score:        %.1f/100\n", m.OverallScore)
	}
	fmt.Fprintf(w, "  Duration:             %.1fs\n", m.TotalDuration)
	fmt.Fprintf(w, "  Estimated cost:       $%.4f\n", m.CostUSD)

	// Verbose: per-task breakdown
	if verbose && len(m.TaskResults) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Per-task results:")
		for _, tr := range m.TaskResults {
			fmt.Fprintf(w, "    %s: score=%d status=%s (%.1fs)\n", tr.TaskName, tr.Score, tr.Status, tr.Duration)
			if tr.Reasoning != "" {
				fmt.Fprintf(w, "      Reasoning: %s\n", tr.Reasoning)
			}
		}
	}
}

// RenderC7Debug renders detailed C7 debug data (prompts, responses, scores, traces)
// to the provided writer. This is called only when --debug-c7 is active.
func RenderC7Debug(w io.Writer, analysisResults []*types.AnalysisResult) {
	// Find the C7 result
	var c7Result *types.AnalysisResult
	for _, ar := range analysisResults {
		if ar.Category == "C7" {
			c7Result = ar
			break
		}
	}
	if c7Result == nil {
		return
	}

	raw, ok := c7Result.Metrics["c7"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C7Metrics)
	if !ok || !m.Available {
		return
	}
	if len(m.MetricResults) == 0 {
		return
	}

	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	red := color.New(color.FgRed)

	// Header
	fmt.Fprintln(w)
	bold.Fprintln(w, "C7 Debug: Agent Evaluation Details")
	fmt.Fprintln(w, strings.Repeat("=", separatorWide))

	for _, mr := range m.MetricResults {
		fmt.Fprintln(w)
		bold.Fprintf(w, "[%s] %s  score=%d/10  (%.1fs)\n", mr.MetricID, mr.MetricName, mr.Score, mr.Duration)
		fmt.Fprintln(w, strings.Repeat("-", separatorNarrow))

		if len(mr.DebugSamples) == 0 {
			dim.Fprintln(w, "  No debug samples captured")
			continue
		}

		for i, ds := range mr.DebugSamples {
			fmt.Fprintf(w, "  Sample %d: %s\n", i+1, ds.Description)
			fmt.Fprintf(w, "  File:     %s\n", ds.FilePath)
			fmt.Fprintf(w, "  Score:    %d/10  Duration: %.1fs\n", ds.Score, ds.Duration)

			// Prompt (truncated, dim)
			prompt := truncateString(ds.Prompt, truncateShort)
			dim.Fprintf(w, "  Prompt:   %s\n", prompt)

			// Response (truncated)
			response := truncateString(ds.Response, truncateLong)
			fmt.Fprintf(w, "  Response: %s\n", response)

			// Score trace
			renderScoreTrace(w, ds.ScoreTrace)

			// Error (red, if present)
			if ds.Error != "" {
				red.Fprintf(w, "  Error: %s\n", ds.Error)
			}

			// Blank line between samples (but not after the last)
			if i < len(mr.DebugSamples)-1 {
				fmt.Fprintln(w)
			}
		}
	}
}

// renderScoreTrace prints the score trace breakdown for a single debug sample.
func renderScoreTrace(w io.Writer, trace types.C7ScoreTrace) {
	var parts []string
	for _, ind := range trace.Indicators {
		if ind.Matched {
			sign := "+"
			if ind.Delta < 0 {
				sign = ""
			}
			parts = append(parts, fmt.Sprintf("%s(%s%d)", ind.Name, sign, ind.Delta))
		}
	}
	indicators := strings.Join(parts, " ")
	if indicators != "" {
		indicators = " " + indicators + " "
	} else {
		indicators = " "
	}
	fmt.Fprintf(w, "  Trace:    base=%d%s-> final=%d\n", trace.BaseScore, indicators, trace.FinalScore)
}
