package output

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/ingo/agent-readyness/internal/scoring"
	"github.com/ingo/agent-readyness/pkg/types"
)

// renderBreakpointTrace renders the full C1-C6 trace modal content showing
// the breakpoint scoring table with the current band highlighted, and
// top-5 worst offender evidence.
func renderBreakpointTrace(metricName string, rawValue float64, score float64, breakpoints []scoring.Breakpoint, evidence []types.EvidenceItem) string {
	if len(breakpoints) == 0 && len(evidence) == 0 {
		return ""
	}

	var b strings.Builder

	// Breakpoint table section
	if len(breakpoints) > 0 {
		currentBand := findCurrentBand(rawValue, breakpoints)

		b.WriteString(`<div class="trace-section"><h4>Scoring Scale</h4>`)
		b.WriteString(`<table class="trace-breakpoint-table"><thead><tr><th>Range</th><th>Score</th></tr></thead><tbody>`)

		for i, bp := range breakpoints {
			var rangeStr string
			if i == 0 {
				rangeStr = fmt.Sprintf("&le; %.4g", bp.Value)
			} else if i == len(breakpoints)-1 {
				rangeStr = fmt.Sprintf("&ge; %.4g", bp.Value)
			} else {
				rangeStr = fmt.Sprintf("%.4g &ndash; %.4g", breakpoints[i-1].Value, bp.Value)
			}

			rowClass := ""
			if i == currentBand {
				rowClass = ` class="trace-current-band"`
			}

			b.WriteString(fmt.Sprintf(`<tr%s><td>%s</td><td>%.0f</td></tr>`, rowClass, rangeStr, bp.Score))
		}

		b.WriteString(`</tbody></table>`)

		formattedVal := formatMetricValue(metricName, rawValue, true)
		b.WriteString(fmt.Sprintf(`<p class="trace-summary">Current value: <strong>%s</strong> &rarr; Score: <strong>%.1f</strong></p>`,
			template.HTMLEscapeString(formattedVal), score))
		b.WriteString(`</div>`)
	}

	// Evidence section (top offenders)
	if len(evidence) > 0 {
		b.WriteString(`<div class="trace-section"><h4>Top Offenders</h4>`)
		b.WriteString(`<table class="trace-evidence-table"><thead><tr><th>File</th><th>Line</th><th>Value</th><th>Description</th></tr></thead><tbody>`)

		for _, ev := range evidence {
			filePath := template.HTMLEscapeString(ev.FilePath)
			desc := template.HTMLEscapeString(ev.Description)
			b.WriteString(fmt.Sprintf(`<tr><td title="%s">%s</td><td>%d</td><td>%.4g</td><td>%s</td></tr>`,
				filePath, filePath, ev.Line, ev.Value, desc))
		}

		b.WriteString(`</tbody></table></div>`)
	}

	return b.String()
}

// findCurrentBand returns the index of the breakpoint row that should be
// highlighted for the given raw value. Returns -1 if breakpoints are empty.
//
// The logic handles both ascending-score (higher value = higher score, e.g. coverage)
// and descending-score (higher value = lower score, e.g. complexity) breakpoint tables.
func findCurrentBand(rawValue float64, breakpoints []scoring.Breakpoint) int {
	if len(breakpoints) == 0 {
		return -1
	}

	// Determine direction: ascending values with ascending scores vs descending scores
	ascending := breakpoints[0].Score < breakpoints[len(breakpoints)-1].Score

	if ascending {
		// Values go up, scores go up (e.g., coverage: 0->1, 30->4, 50->6, ...)
		// rawValue falls in the band at or before the first breakpoint whose Value >= rawValue
		for i := 0; i < len(breakpoints); i++ {
			if rawValue <= breakpoints[i].Value {
				return i
			}
		}
		return len(breakpoints) - 1
	}

	// Values go up, scores go down (e.g., complexity: 1->10, 5->8, 10->6, ...)
	// rawValue falls in the band at or before the first breakpoint whose Value >= rawValue
	for i := 0; i < len(breakpoints); i++ {
		if rawValue <= breakpoints[i].Value {
			return i
		}
	}
	return len(breakpoints) - 1
}

// renderC7Trace renders trace HTML for a C7 metric.
// Returns empty string if no matching metric or no DebugSamples.
func renderC7Trace(metricID string, metricResults []types.C7MetricResult) string {
	// Find the matching metric result
	var mr *types.C7MetricResult
	for i := range metricResults {
		if metricResults[i].MetricID == metricID {
			mr = &metricResults[i]
			break
		}
	}

	if mr == nil || len(mr.DebugSamples) == 0 {
		return ""
	}

	var b strings.Builder

	for i, ds := range mr.DebugSamples {
		if i > 0 {
			b.WriteString(`<hr style="margin: 1.5rem 0; border: none; border-top: 1px solid var(--color-border);">`)
		}

		b.WriteString(fmt.Sprintf(`<h4 style="margin-bottom: 0.75rem;">Sample %d: %s</h4>`,
			i+1, template.HTMLEscapeString(ds.Description)))

		// Score checklist section
		b.WriteString(`<div class="trace-checklist">`)
		for _, ind := range ds.ScoreTrace.Indicators {
			cssClass := "trace-indicator unmatched"
			mark := "&#10007;" // cross mark
			if ind.Matched {
				cssClass = "trace-indicator matched"
				mark = "&#10003;" // check mark
			}
			deltaStr := ""
			if ind.Delta > 0 {
				deltaStr = fmt.Sprintf(" (+%d)", ind.Delta)
			} else if ind.Delta < 0 {
				deltaStr = fmt.Sprintf(" (%d)", ind.Delta)
			}
			b.WriteString(fmt.Sprintf(`<div class="%s">%s %s%s</div>`,
				cssClass, mark, template.HTMLEscapeString(ind.Name), deltaStr))
		}
		b.WriteString(fmt.Sprintf(`<p class="trace-score-summary">Base: %d &#8594; Final: %d</p>`,
			ds.ScoreTrace.BaseScore, ds.ScoreTrace.FinalScore))
		b.WriteString(`</div>`)

		// Collapsible prompt section
		escapedFilePath := template.HTMLEscapeString(ds.FilePath)
		escapedPrompt := template.HTMLEscapeString(ds.Prompt)
		b.WriteString(fmt.Sprintf(`<details class="trace-collapsible"><summary>Prompt (sample: %s)</summary>`, escapedFilePath))
		b.WriteString(`<div class="trace-code-block">`)
		b.WriteString(`<button class="trace-copy-btn" onclick="navigator.clipboard.writeText(this.parentElement.querySelector('code').textContent).then(function(){var b=event.target;b.textContent='Copied!';setTimeout(function(){b.textContent='Copy'},1500)})">Copy</button>`)
		b.WriteString(fmt.Sprintf(`<pre><code>%s</code></pre>`, escapedPrompt))
		b.WriteString(`</div></details>`)

		// Collapsible response section
		escapedResponse := template.HTMLEscapeString(ds.Response)
		b.WriteString(`<details class="trace-collapsible"><summary>Response</summary>`)
		b.WriteString(`<div class="trace-code-block">`)
		b.WriteString(`<button class="trace-copy-btn" onclick="navigator.clipboard.writeText(this.parentElement.querySelector('code').textContent).then(function(){var b=event.target;b.textContent='Copied!';setTimeout(function(){b.textContent='Copy'},1500)})">Copy</button>`)
		b.WriteString(fmt.Sprintf(`<pre><code>%s</code></pre>`, escapedResponse))
		b.WriteString(`</div></details>`)

		// Error section if present
		if ds.Error != "" {
			b.WriteString(fmt.Sprintf(`<p style="color: var(--color-red); margin-top: 0.5rem;">Error: %s</p>`,
				template.HTMLEscapeString(ds.Error)))
		}
	}

	return b.String()
}
