# Phase 32: Call Trace Modals - Research

**Researched:** 2026-02-06
**Domain:** HTML report template engineering, Go template data flow, inline JavaScript
**Confidence:** HIGH

## Summary

This phase adds "View Trace" buttons to every metric row in the HTML report and populates modals with scoring derivation details. The work is entirely within the existing Go template/CSS/JS system -- no new libraries needed.

Two distinct modal content types exist: C7 trace modals (LLM-based, showing prompts/responses/score breakdowns from `C7DebugSample` data) and C1-C6 trace modals (rule-based, showing breakpoint tables and evidence items). The critical data flow gap is that C7 debug samples are currently only populated when `--debug-c7` is active, but the CONTEXT.md decision says "always embed all C7 trace data." This means the C7 analyzer must populate `DebugSamples` unconditionally when C7 is enabled. For C1-C6, all data already exists: `SubScore.Evidence` carries worst offenders and `scoring.ScoringConfig` carries breakpoints.

The modal infrastructure from Phase 31 (`openModal(title, bodyHTML)` / `closeModal()`, `<dialog>` element, `.ars-modal-trigger` CSS class) is already in place. This phase needs to: (1) thread additional data into HTML template rendering, (2) generate modal body HTML in Go template functions, (3) add View Trace buttons to metric rows, and (4) add CSS for syntax highlighting and breakpoint tables.

**Primary recommendation:** Build trace content as Go template helper functions that return `template.HTML`, embed all trace data as inline HTML hidden in the page, and wire buttons to `openModal()` with the pre-rendered content.

## Standard Stack

No new libraries needed. Everything uses the existing stack:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `html/template` | Go stdlib | Template rendering with auto-escaping | Already in use |
| `embed` | Go stdlib | Embed templates/CSS | Already in use |
| Native `<dialog>` | HTML5 | Modal infrastructure | Decided in Phase 31 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` | Go stdlib | Escape C7 prompt/response content for safe HTML embedding | C7 trace data with arbitrary content |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Inline HTML pre-rendering | `<script type="application/json">` + JS rendering | Would add JS complexity; Go templates are simpler and already established |
| CSS-only syntax highlighting | highlight.js / Prism.js | Would violate size budget spirit; 2-3 color CSS is sufficient per CONTEXT.md |

## Architecture Patterns

### Data Flow: Pipeline -> HTML Generator -> Template

The current data flow is:
```
Pipeline.generateHTMLReport()
  -> output.NewHTMLGenerator()
  -> gen.GenerateReport(writer, scored, recs, baseline)
    -> buildHTMLCategories(scored.Categories, citations)
      -> buildHTMLSubScores(cat.SubScores)
    -> template.Execute(data)
```

For trace modals, we need to add:
1. **Breakpoint data** from `scoring.ScoringConfig` into `HTMLSubScore`
2. **C7 trace data** from `C7MetricResult.DebugSamples` into `HTMLSubScore`
3. **Pre-rendered trace HTML** as `template.HTML` field on `HTMLSubScore`

### Recommended Approach: TraceHTML Field

Add a `TraceHTML template.HTML` field to `HTMLSubScore`. Compute it during `buildHTMLSubScores()` using helper functions that take the raw data and return safe HTML strings.

```
HTMLSubScore {
    ...existing fields...
    TraceHTML  template.HTML  // Pre-rendered modal body content
    HasTrace   bool           // Whether trace data is available
}
```

### Pattern 1: C1-C6 Breakpoint Table Rendering

**What:** Go function that takes a metric name, raw value, and scoring config, returns an HTML table showing breakpoint ranges with the current band highlighted.

**When to use:** All C1-C6 metrics.

**Data available:**
- `scoring.ScoringConfig.Categories[catName].Metrics[i].Breakpoints` -- all breakpoints for this metric
- `SubScore.RawValue` -- current raw value
- `SubScore.Score` -- interpolated score
- `SubScore.Evidence` -- top-5 worst offenders

**Rendering structure:**
```html
<div class="trace-section">
  <h4>Scoring Breakpoints</h4>
  <table class="trace-breakpoint-table">
    <tr><th>Raw Value</th><th>Score</th></tr>
    <tr class="trace-current-band"><td>...</td><td>...</td></tr>
    ...
  </table>
  <p>Your value: <strong>X</strong> -> Score: <strong>Y</strong></p>
</div>
<div class="trace-section">
  <h4>Top Offenders</h4>
  <table class="trace-evidence-table">
    <tr><th>File</th><th>Line</th><th>Value</th><th>Detail</th></tr>
    ...
  </table>
</div>
```

### Pattern 2: C7 Trace Rendering

**What:** Go function that takes `C7MetricResult` (with `DebugSamples`) and returns HTML with score breakdown (checklist) plus collapsible prompt/response sections.

**When to use:** All C7 metrics (when C7 is enabled).

**Data available (per sample):**
- `C7DebugSample.Prompt` -- full prompt text
- `C7DebugSample.Response` -- full response text
- `C7DebugSample.ScoreTrace.BaseScore` -- starting score
- `C7DebugSample.ScoreTrace.Indicators` -- matched/unmatched indicators with deltas
- `C7DebugSample.ScoreTrace.FinalScore` -- final clamped score

**Rendering structure:**
```html
<div class="trace-section">
  <h4>Score Breakdown</h4>
  <div class="trace-checklist">
    <div class="trace-indicator matched">checkmark Name (+1)</div>
    <div class="trace-indicator unmatched">cross Name (-1)</div>
  </div>
  <p>Base: X -> Final: Y</p>
</div>
<details class="trace-collapsible">
  <summary>Prompt</summary>
  <div class="trace-code-block">
    <button class="trace-copy-btn" onclick="...">Copy</button>
    <pre><code>...prompt text...</code></pre>
  </div>
</details>
<details class="trace-collapsible">
  <summary>Response</summary>
  <div class="trace-code-block">
    <button class="trace-copy-btn" onclick="...">Copy</button>
    <pre><code>...response text...</code></pre>
  </div>
</details>
```

### Pattern 3: View Trace Button in Template

**What:** Add a "View Trace" button to each metric row that opens the modal with pre-rendered content.

**Implementation:** Store trace HTML in a hidden element or data attribute, wire button to `openModal()`.

```html
<!-- In metric-row -->
<td>
  {{if .HasTrace}}
  <button class="ars-modal-trigger" onclick="openModal('{{.DisplayName}} Trace', document.getElementById('trace-{{.Key}}').innerHTML)">View Trace</button>
  {{end}}
</td>

<!-- Hidden content store (outside table) -->
<template id="trace-{{.Key}}">{{.TraceHTML}}</template>
```

Using `<template>` elements is cleaner than `<div style="display:none">` because `<template>` content is inert (scripts don't execute, images don't load).

### Anti-Patterns to Avoid
- **Building HTML strings in JavaScript:** All trace HTML should be pre-rendered in Go templates. JS should only call `openModal()` with existing content.
- **Embedding large JSON blobs for JS parsing:** This adds complexity. Pre-render HTML server-side.
- **Putting `<template>` elements inside `<table>` rows:** Invalid HTML. Place `<template>` elements outside the `<table>` in the category `<div>`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTML escaping of C7 prompts/responses | Manual string escaping | `html/template` auto-escaping + `template.HTML` for safe content | C7 prompts may contain arbitrary code, HTML entities, etc. |
| Syntax highlighting | Full token-based highlighter | CSS class-based coloring on `<pre>` blocks with 2-3 classes | CONTEXT.md says subtle/minimal; CSS is sufficient |
| Copy-to-clipboard | Custom clipboard API wrapper | `navigator.clipboard.writeText()` inline | Already used for badge copy; single line of JS |
| Collapsible sections | Custom JS accordion | Native `<details>`/`<summary>` HTML5 elements | Progressive enhancement, no JS needed |
| Modal focus trapping | Custom focus trap | Native `<dialog>` `showModal()` | Already handles focus trapping per Phase 31 |

## Common Pitfalls

### Pitfall 1: C7 DebugSamples Not Populated Outside Debug Mode
**What goes wrong:** Currently `C7DebugSample` data is only populated when `a.debug == true` in the C7 analyzer. Without this data, C7 trace modals have nothing to show.
**Why it happens:** Debug samples were designed for CLI debugging, not HTML reports.
**How to avoid:** Change the C7 analyzer to always populate `DebugSamples` when C7 is enabled (remove the `if a.debug` guard). The debug flag should only control terminal output, not data capture.
**Warning signs:** C7 trace modals showing empty content or "No trace data available."

### Pitfall 2: HTML Escaping of C7 Prompt/Response Content
**What goes wrong:** C7 prompts contain Go source code, shell commands, and arbitrary text that may include `<`, `>`, `&`, quotes. If not properly escaped, this breaks the HTML structure.
**Why it happens:** Using `template.HTML` (which bypasses escaping) on user-derived content.
**How to avoid:** Use `template.HTMLEscapeString()` on raw prompt/response text before embedding in pre-rendered HTML. Only mark the final assembled trace HTML as `template.HTML`.
**Warning signs:** Broken HTML rendering, XSS-like artifacts in modal content.

### Pitfall 3: `<template>` Elements Inside `<table>`
**What goes wrong:** Placing `<template>` elements inside `<tbody>` is invalid HTML. Browsers may move them outside the table or discard them.
**Why it happens:** Natural desire to keep trace content near its metric row.
**How to avoid:** Place all `<template id="trace-...">` elements in a `<div>` outside the `<table>`, grouped per category.
**Warning signs:** Trace content missing or appearing in wrong location.

### Pitfall 4: Breakpoint Config Not Available in HTML Generator
**What goes wrong:** `buildHTMLSubScores()` currently only receives `[]types.SubScore` which has raw values and scores but NOT the breakpoint definitions from `scoring.ScoringConfig`.
**Why it happens:** The scoring config is in the pipeline/scorer, not passed through to the HTML generator.
**How to avoid:** Thread `*scoring.ScoringConfig` (or just the breakpoints map) into `GenerateReport()` and through to `buildHTMLSubScores()`.
**Warning signs:** C1-C6 trace modals missing breakpoint tables.

### Pitfall 5: C7 Metric-to-DebugSample Mapping
**What goes wrong:** The HTML generator needs to match C7 metric names to their debug sample data, but `ScoredResult` only contains `SubScore` objects without C7-specific fields.
**Why it happens:** `ScoredResult` is a generic scoring type that doesn't carry analyzer-specific debug data.
**How to avoid:** Pass `[]types.AnalysisResult` (the raw analysis results) into the HTML generator alongside `ScoredResult`, or add C7-specific data to the HTML data model via a separate channel. The cleanest approach: pass `[]*types.AnalysisResult` to `GenerateReport()` and extract C7 data from `ar.Metrics["c7"].(*types.C7Metrics).MetricResults`.
**Warning signs:** C7 trace buttons appear but modal content is empty.

### Pitfall 6: Noscript Progressive Enhancement
**What goes wrong:** TR-08 requires `<details>` fallback without JS. The `noscript` style already hides `.ars-modal-trigger` buttons. But the trace content must also be accessible somehow.
**Why it happens:** Modal content is hidden in `<template>` elements which are inherently invisible.
**How to avoid:** For no-JS fallback, also render a `<details>` element with trace content directly in the metric details row. The `<details>` is hidden when JS is available (via a `.js-enabled` body class set by script), shown when not.
**Warning signs:** Trace data completely inaccessible without JavaScript.

## Code Examples

### Example 1: Threading Scoring Config to HTML Generator
```go
// Modified signature in html.go
func (g *HTMLGenerator) GenerateReport(
    w io.Writer,
    scored *types.ScoredResult,
    recs []recommend.Recommendation,
    baseline *types.ScoredResult,
    scoringCfg *scoring.ScoringConfig,    // NEW
    analysisResults []*types.AnalysisResult, // NEW: for C7 debug data
) error {
```

### Example 2: Breakpoint Table Helper
```go
// renderBreakpointTable generates HTML for a scoring breakpoint table
func renderBreakpointTable(metricName string, rawValue float64, score float64, breakpoints []scoring.Breakpoint) string {
    var b strings.Builder
    b.WriteString(`<table class="trace-breakpoint-table">`)
    b.WriteString(`<tr><th>Range</th><th>Score</th></tr>`)
    for i, bp := range breakpoints {
        isCurrentBand := false
        if i == 0 && rawValue <= bp.Value {
            isCurrentBand = true
        } else if i > 0 && rawValue > breakpoints[i-1].Value && rawValue <= bp.Value {
            isCurrentBand = true
        } else if i == len(breakpoints)-1 && rawValue >= bp.Value {
            isCurrentBand = true
        }
        cls := ""
        if isCurrentBand {
            cls = ` class="trace-current-band"`
        }
        b.WriteString(fmt.Sprintf(`<tr%s><td>%s</td><td>%.1f</td></tr>`, cls, formatRange(i, breakpoints), bp.Score))
    }
    b.WriteString(`</table>`)
    b.WriteString(fmt.Sprintf(`<p class="trace-summary">Current value: <strong>%s</strong> &rarr; Score: <strong>%.1f</strong></p>`, formatMetricValue(metricName, rawValue, true), score))
    return b.String()
}
```

### Example 3: C7 Score Checklist Helper
```go
// renderC7ScoreChecklist generates checklist HTML for C7 indicator matches
func renderC7ScoreChecklist(trace types.C7ScoreTrace) string {
    var b strings.Builder
    b.WriteString(`<div class="trace-checklist">`)
    for _, ind := range trace.Indicators {
        icon := "&#x2717;" // cross mark
        cls := "unmatched"
        if ind.Matched {
            icon = "&#x2713;" // check mark
            cls = "matched"
        }
        sign := "+"
        if ind.Delta < 0 {
            sign = ""
        }
        b.WriteString(fmt.Sprintf(
            `<div class="trace-indicator %s">%s %s (%s%d)</div>`,
            cls, icon, template.HTMLEscapeString(ind.Name), sign, ind.Delta,
        ))
    }
    b.WriteString(`</div>`)
    b.WriteString(fmt.Sprintf(
        `<p class="trace-score-summary">Base: %d &rarr; Final: %d</p>`,
        trace.BaseScore, trace.FinalScore,
    ))
    return b.String()
}
```

### Example 4: Copy Button Pattern
```go
// In template, for each code block:
`<div class="trace-code-block">
  <button class="trace-copy-btn ars-modal-trigger"
    onclick="navigator.clipboard.writeText(this.nextElementSibling.textContent)">Copy</button>
  <pre><code>` + template.HTMLEscapeString(content) + `</code></pre>
</div>`
```

### Example 5: View Trace Button Wiring
```html
<!-- In metric row, add a new column -->
<td class="trace-cell">
  {{if .HasTrace}}
  <button class="ars-modal-trigger"
    onclick="openModal('{{.DisplayName}}', document.getElementById('trace-{{.Key}}').innerHTML)">
    View Trace
  </button>
  {{end}}
</td>

<!-- After the metric table, inside the category div -->
{{range .SubScores}}
{{if .HasTrace}}
<template id="trace-{{.Key}}">{{.TraceHTML}}</template>
{{end}}
{{end}}
```

### Example 6: Minimal Syntax Highlighting CSS
```css
/* Syntax highlighting - subtle, 2-3 colors */
.trace-code-block pre {
  background: #f8f9fa;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  padding: 1rem;
  overflow-x: auto;
  font-size: 0.75rem;
  line-height: 1.5;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}

/* JSON key highlighting via JS after modal opens */
.trace-json-key { color: #0550ae; }    /* blue for keys */
.trace-json-string { color: #0a3069; } /* dark blue for string values */
.trace-json-number { color: #953800; } /* orange for numbers */
```

Note: For JSON syntax highlighting, a simple regex-based JS function can be called after `openModal()` sets innerHTML. The function matches JSON patterns (`"key":`, `"string value"`, numbers) and wraps them in `<span>` elements. This is simpler than a full tokenizer and meets the "2-3 colors" constraint.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hidden div for modal content | `<template>` element | HTML5 standard | Content is inert -- no side effects from hidden scripts/images |
| JS-built accordion | Native `<details>`/`<summary>` | HTML5 standard | Progressive enhancement, no JS needed for expand/collapse |
| Complex syntax highlighters | CSS class-based with regex post-processing | N/A | Minimal JS, meets "subtle" constraint |

## Open Questions

1. **C7 DebugSamples data availability**
   - What we know: Currently gated behind `a.debug` flag in C7 analyzer
   - What's unclear: Whether always populating DebugSamples has performance/memory concerns for large codebases
   - Recommendation: Always populate when C7 is enabled. The data is already computed; the flag only prevents storage. Memory impact is minimal (text strings).

2. **GenerateReport signature change**
   - What we know: Need to pass scoring config and analysis results to HTML generator
   - What's unclear: Whether to add parameters to existing function or create a new options struct
   - Recommendation: Add a `TraceData` struct parameter that bundles scoring config + analysis results. Keeps the signature change minimal.

3. **Multiple samples per C7 metric**
   - What we know: Each C7 metric may have multiple `DebugSample` entries (one per sample file evaluated)
   - What's unclear: How to present multiple samples -- tabs, sequential sections, or separate trace per sample
   - Recommendation: Sequential sections with sample file path as heading. Simple, no tab UI needed.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `internal/output/html.go`, `internal/output/templates/report.html`, `internal/output/templates/styles.css`
- Codebase analysis: `pkg/types/types.go`, `pkg/types/scoring.go` -- all relevant types
- Codebase analysis: `internal/scoring/config.go` -- breakpoint definitions for all 7 categories
- Codebase analysis: `internal/scoring/scorer.go` -- `Interpolate()` function, `scoreMetrics()` flow
- Codebase analysis: `internal/analyzer/c7_agent/agent.go` -- C7 debug sample population logic
- Codebase analysis: `internal/output/descriptions.go` -- existing metric description system

### Secondary (MEDIUM confidence)
- HTML5 `<template>` element behavior: standard spec, widely supported
- HTML5 `<details>`/`<summary>` behavior: standard spec, widely supported
- `navigator.clipboard.writeText()`: already used in existing badge copy button

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new libraries, all existing codebase patterns
- Architecture: HIGH - thorough codebase analysis, clear data flow mapping
- Pitfalls: HIGH - identified from actual code inspection, not speculation

**Research date:** 2026-02-06
**Valid until:** 2026-03-06 (stable domain, no external dependencies)
