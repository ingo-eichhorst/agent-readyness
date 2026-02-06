package output

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/ingo/agent-readyness/pkg/types"
)

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
