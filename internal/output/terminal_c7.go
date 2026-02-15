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
	raw, ok := ar.Metrics["c7"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C7Metrics)
	if !ok {
		return
	}

	bold := color.New(color.Bold)
	fmt.Fprintln(w)
	bold.Fprintln(w, "C7: Agent Evaluation")
	fmt.Fprintln(w, "────────────────────────────────────────")

	if !m.Available {
		fmt.Fprintln(w, "  Not available (LLM features disabled)")
		return
	}

	renderC7Metrics(w, m)
	renderC7Summary(w, m)

	if verbose && len(m.TaskResults) > 0 {
		renderC7Tasks(w, m, bold)
	}
}

// renderC7Metrics renders the MECE or legacy metric scores.
func renderC7Metrics(w io.Writer, m *types.C7Metrics) {
	hasMECE := m.TaskExecutionConsistency > 0 || m.CodeBehaviorComprehension > 0 ||
		m.CrossFileNavigation > 0 || m.IdentifierInterpretability > 0 ||
		m.DocumentationAccuracyDetection > 0

	if hasMECE {
		renderC7MetricLine(w, "M1 Exec Consistency", m.TaskExecutionConsistency)
		renderC7MetricLine(w, "M2 Comprehension", m.CodeBehaviorComprehension)
		renderC7MetricLine(w, "M3 Navigation", m.CrossFileNavigation)
		renderC7MetricLine(w, "M4 Identifiers", m.IdentifierInterpretability)
		renderC7MetricLine(w, "M5 Documentation", m.DocumentationAccuracyDetection)
	} else {
		renderC7LegacyLine(w, "Intent clarity", m.IntentClarity)
		renderC7LegacyLine(w, "Modification conf", m.ModificationConfidence)
		renderC7LegacyLine(w, "Cross-file coherence", m.CrossFileCoherence)
		renderC7LegacyLine(w, "Semantic complete", m.SemanticCompleteness)
	}
}

func renderC7MetricLine(w io.Writer, label string, score int) {
	c := c7ScoreColor(score * c7ScoreScale)
	c.Fprintf(w, "  %-24s%d/10\n", label+":", score)
}

func renderC7LegacyLine(w io.Writer, label string, score int) {
	c := c7ScoreColor(score)
	c.Fprintf(w, "  %-24s%d/100\n", label+":", score)
}

// renderC7Summary renders overall score, duration, and cost.
func renderC7Summary(w io.Writer, m *types.C7Metrics) {
	fmt.Fprintln(w, "  ─────────────────────────────────────")
	if m.MECEScore > 0 {
		c := c7ScoreColor(int(m.MECEScore * float64(c7ScoreScale)))
		c.Fprintf(w, "  MECE Score:           %.1f/10\n", m.MECEScore)
	} else {
		c := c7ScoreColor(int(m.OverallScore))
		c.Fprintf(w, "  Overall score:        %.1f/100\n", m.OverallScore)
	}
	fmt.Fprintf(w, "  Duration:             %.1fs\n", m.TotalDuration)
	fmt.Fprintf(w, "  Estimated cost:       $%.4f\n", m.CostUSD)
}

// renderC7Tasks renders verbose per-task breakdown.
func renderC7Tasks(w io.Writer, m *types.C7Metrics, bold *color.Color) {
	fmt.Fprintln(w)
	bold.Fprintln(w, "  Per-task results:")
	for _, tr := range m.TaskResults {
		fmt.Fprintf(w, "    %s: score=%d status=%s (%.1fs)\n", tr.TaskName, tr.Score, tr.Status, tr.Duration)
		if tr.Reasoning != "" {
			fmt.Fprintf(w, "      Reasoning: %s\n", tr.Reasoning)
		}
	}
}

// RenderC7Debug renders detailed C7 debug data (prompts, responses, scores, traces)
// to the provided writer. This is called only when --debug-c7 is active.
func RenderC7Debug(w io.Writer, analysisResults []*types.AnalysisResult) {
	m := findC7Metrics(analysisResults)
	if m == nil || len(m.MetricResults) == 0 {
		return
	}

	bold := color.New(color.Bold)
	fmt.Fprintln(w)
	bold.Fprintln(w, "C7 Debug: Agent Evaluation Details")
	fmt.Fprintln(w, strings.Repeat("=", separatorWide))

	for _, mr := range m.MetricResults {
		renderC7DebugMetric(w, mr, bold)
	}
}

// findC7Metrics locates and returns C7 metrics from analysis results.
func findC7Metrics(results []*types.AnalysisResult) *types.C7Metrics {
	for _, ar := range results {
		if ar.Category != "C7" {
			continue
		}
		raw, ok := ar.Metrics["c7"]
		if !ok {
			return nil
		}
		m, ok := raw.(*types.C7Metrics)
		if !ok || !m.Available {
			return nil
		}
		return m
	}
	return nil
}

// renderC7DebugMetric renders debug output for a single metric result.
func renderC7DebugMetric(w io.Writer, mr types.C7MetricResult, bold *color.Color) {
	dim := color.New(color.FgHiBlack)
	red := color.New(color.FgRed)

	fmt.Fprintln(w)
	bold.Fprintf(w, "[%s] %s  score=%d/10  (%.1fs)\n", mr.MetricID, mr.MetricName, mr.Score, mr.Duration)
	fmt.Fprintln(w, strings.Repeat("-", separatorNarrow))

	if len(mr.DebugSamples) == 0 {
		dim.Fprintln(w, "  No debug samples captured")
		return
	}

	for i, ds := range mr.DebugSamples {
		renderC7DebugSample(w, i, ds, dim, red)
	}
}

// renderC7DebugSample renders debug output for a single sample.
func renderC7DebugSample(w io.Writer, idx int, ds types.C7DebugSample, dim, red *color.Color) {
	fmt.Fprintf(w, "  Sample %d: %s\n", idx+1, ds.Description)
	fmt.Fprintf(w, "  File:     %s\n", ds.FilePath)
	fmt.Fprintf(w, "  Score:    %d/10  Duration: %.1fs\n", ds.Score, ds.Duration)
	dim.Fprintf(w, "  Prompt:   %s\n", truncateString(ds.Prompt, truncateShort))
	fmt.Fprintf(w, "  Response: %s\n", truncateString(ds.Response, truncateLong))
	renderScoreTrace(w, ds.ScoreTrace)
	if ds.Error != "" {
		red.Fprintf(w, "  Error: %s\n", ds.Error)
	}
	if idx < len(ds.Response)-1 {
		fmt.Fprintln(w)
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
