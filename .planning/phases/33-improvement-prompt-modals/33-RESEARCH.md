# Phase 33: Improvement Prompt Modals - Research

**Researched:** 2026-02-07
**Domain:** Go template-driven prompt generation, clipboard API fallback chains, HTML modal content rendering
**Confidence:** HIGH

## Summary

This phase adds per-metric "Improve" buttons to the HTML report that open modals containing research-backed, project-specific prompts users can copy and paste into an AI agent. The implementation builds entirely on existing infrastructure: the `<dialog>` modal from Phase 31, the `openModal()` JS function from Phase 32, the evidence data from Phase 30, and the metric descriptions from `descriptions.go`.

The core work is: (1) creating Go-side prompt template rendering that interpolates metric-specific data (current score, target score, evidence file paths) into structured prompts with the format Context / Build-Test / Task / Verification, (2) adding an "Improve" button alongside the existing "View Trace" button in each metric row, and (3) implementing a clipboard copy with fallback chain (Clipboard API, execCommand, visible pre block) since reports are often opened via `file://` protocol where `navigator.clipboard` is unavailable.

**Primary recommendation:** Render prompt HTML server-side in Go (like trace modals), store in `<template>` tags, and open via the existing `openModal()` function. Add a `renderImprovementPrompt()` function in a new `prompt.go` file alongside `trace.go`.

## Standard Stack

No new external libraries required. Everything uses existing Go templates and browser APIs.

### Core
| Technology | Version | Purpose | Why Standard |
|------------|---------|---------|--------------|
| Go `strings.Builder` | stdlib | Server-side prompt HTML rendering | Same pattern as `trace.go` |
| `<template>` elements | HTML5 | Store prompt HTML in DOM | Same pattern as trace templates |
| `openModal()` | Existing JS | Display prompt in modal dialog | Reuses Phase 31/32 infrastructure |
| `navigator.clipboard.writeText()` | Web API | Primary copy mechanism | Works on HTTPS origins |
| `document.execCommand('copy')` | Legacy Web API | Fallback for file:// protocol | Works where Clipboard API is blocked |

### Supporting
| Technology | Purpose | When to Use |
|------------|---------|-------------|
| `<details>` fallback | Progressive enhancement (PR-09) | No-JS fallback for prompt visibility |
| CSS `.prompt-*` classes | Prompt modal styling | Visual distinction from trace modals |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Server-side Go rendering | Client-side JS template literals | Go rendering is consistent with trace.go pattern; no new patterns |
| Per-metric prompts | Per-category prompts only | Per-metric is more actionable but more templates; requirements say per-metric |
| Hardcoded build commands | Detecting from project config | Config has no build/test commands today; use language-based defaults |

## Architecture Patterns

### Where New Code Lives

```
internal/output/
  prompt.go           # NEW: renderImprovementPrompt() + prompt templates
  prompt_test.go      # NEW: tests for prompt rendering
  html.go             # MODIFY: add PromptHTML/HasPrompt to HTMLSubScore, wire up in buildHTMLSubScores
  templates/
    report.html       # MODIFY: add Improve button + <template> tags + prompt fallback
    styles.css        # MODIFY: add prompt-specific CSS
```

### Pattern 1: Server-Side Prompt Rendering (Same as Trace)

**What:** Generate prompt HTML in Go, store in `HTMLSubScore.PromptHTML`, render into `<template>` elements in HTML.

**When to use:** All 32+ metrics across 7 categories.

**Structure:**
```go
// In prompt.go
func renderImprovementPrompt(categoryName string, metricName string, displayName string,
    rawValue float64, score float64, formattedValue string,
    evidence []types.EvidenceItem, breakpoints []scoring.Breakpoint) string {
    // Build structured prompt with sections:
    // 1. Context (what metric, current score, target)
    // 2. Build/Test Commands (language-appropriate)
    // 3. Task (specific improvement instructions with evidence)
    // 4. Verification (how to confirm improvement)
}
```

**In html.go HTMLSubScore:**
```go
type HTMLSubScore struct {
    // ... existing fields ...
    PromptHTML  template.HTML  // Pre-rendered prompt modal content
    HasPrompt   bool           // Whether prompt data is available
}
```

**In report.html:**
```html
<!-- Alongside existing View Trace button -->
{{if .HasPrompt}}
<button class="ars-modal-trigger" onclick="openModal('Improve {{.DisplayName}}', document.getElementById('prompt-{{.Key}}').innerHTML)">Improve</button>
{{end}}

<!-- Template storage -->
{{if .HasPrompt}}<template id="prompt-{{.Key}}">{{.PromptHTML}}</template>{{end}}

<!-- No-JS fallback (PR-09) -->
{{if .HasPrompt}}
<details class="prompt-fallback">
    <summary>Improvement Prompt</summary>
    <div class="prompt-fallback-content">{{.PromptHTML}}</div>
</details>
{{end}}
```

### Pattern 2: Prompt Template Structure (PR-08)

**What:** Every prompt follows the 4-section structure: Context, Build/Test Commands, Task, Verification.

**Template structure (rendered as copyable plain text inside a `<pre>` block):**
```
## Context

I'm working on improving the {metric_display_name} metric in this codebase.
Current score: {score}/10 (raw value: {formatted_value})
Target score: {target_score}/10 (target value: {target_value})

Category: {category_display_name}
Why it matters: {category_impact}

## Build & Test Commands

{language_appropriate_build_test_commands}

## Task

{metric_specific_task_instructions}

### Files to Focus On

{top_evidence_files_with_values}

## Verification

After making changes:
{metric_specific_verification_steps}
```

### Pattern 3: Clipboard Fallback Chain (PR-06)

**What:** Three-tier copy mechanism for maximum compatibility.

**Implementation:**
```javascript
function copyPromptText(buttonEl) {
    var text = buttonEl.closest('.prompt-copy-container').querySelector('code').textContent;

    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function() {
            showCopied(buttonEl);
        }).catch(function() {
            fallbackCopy(text, buttonEl);
        });
    } else {
        fallbackCopy(text, buttonEl);
    }
}

function fallbackCopy(text, buttonEl) {
    var textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    var ok = document.execCommand('copy');
    document.body.removeChild(textarea);
    if (ok) {
        showCopied(buttonEl);
    } else {
        // Show selectable pre block as last resort
        var pre = buttonEl.closest('.prompt-copy-container').querySelector('pre');
        pre.classList.add('prompt-select-fallback');
        buttonEl.textContent = 'Select All & Copy';
        buttonEl.onclick = function() {
            var range = document.createRange();
            range.selectNodeContents(pre);
            window.getSelection().removeAllRanges();
            window.getSelection().addRange(range);
        };
    }
}

function showCopied(btn) {
    btn.textContent = 'Copied!';
    setTimeout(function() { btn.textContent = 'Copy'; }, 1500);
}
```

### Pattern 4: Target Score Calculation

**What:** Determine the next achievable breakpoint target from the current score.

**Implementation:**
```go
func nextTarget(score float64, breakpoints []scoring.Breakpoint) (targetValue float64, targetScore float64) {
    // Find the next breakpoint that gives a better score
    // For descending breakpoints (complexity: lower value = higher score):
    //   find breakpoint with next higher score
    // For ascending breakpoints (coverage: higher value = higher score):
    //   find breakpoint with next higher score
    // If already at max, return current
}
```

### Anti-Patterns to Avoid

- **Client-side template rendering:** Don't build prompts in JavaScript. Go rendering keeps all logic server-side and is consistent with trace.go.
- **Generic prompts without evidence:** Prompts must include specific file paths and values from evidence data. A prompt saying "reduce complexity" without naming files is useless.
- **Hardcoded thresholds in prompts:** Use breakpoint data from ScoringConfig to derive targets, not hardcoded values.
- **Separate modal element:** Don't create a second `<dialog>`. Reuse the existing `ars-modal` via `openModal()`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Modal UI | New dialog element | Existing `<dialog id="ars-modal">` + `openModal()` | Phase 31/32 already built and tested this |
| Metric descriptions | New description text | `descriptions.go` "How to Improve" sections | Already have research-backed content per metric |
| Target calculations | Guessing improvement targets | `scoring.Breakpoint` data + `findCurrentBand()` | Exact breakpoint data already available in trace pipeline |
| Category context | New impact descriptions | `categoryImpact()` in html.go | Already maps C1-C7 to agent-readiness descriptions |
| Copy button styling | New button styles | `.trace-copy-btn` CSS class | Already styled and tested in Phase 32 |

## Common Pitfalls

### Pitfall 1: Clipboard API Fails on file:// Protocol
**What goes wrong:** `navigator.clipboard.writeText()` requires a secure context (HTTPS or localhost). Reports opened as local HTML files use `file://` which is NOT a secure context in most browsers.
**Why it happens:** Clipboard API is gated behind secure context for security.
**How to avoid:** Always implement the three-tier fallback chain (PR-06). Test with `file://` protocol specifically.
**Warning signs:** Copy button works in development (localhost) but fails when user opens the `.html` file directly.

### Pitfall 2: HTML Escaping in Prompt Text
**What goes wrong:** File paths containing special characters (`<`, `>`, `&`) break HTML rendering or create XSS vectors. The prompt text itself needs to be both displayed as HTML and copied as plain text.
**Why it happens:** The prompt is rendered as HTML but the copy function extracts `.textContent` for plain text.
**How to avoid:** Use `template.HTMLEscapeString()` for all interpolated values in HTML rendering. The `.textContent` property naturally strips HTML tags for the clipboard copy.
**Warning signs:** File paths with `&` in them render as `&amp;` in copied text.

### Pitfall 3: Empty Evidence Arrays
**What goes wrong:** Some metrics may have no evidence items (score 10, or C7 metrics, or unavailable metrics). Prompts with empty "Files to Focus On" sections look broken.
**Why it happens:** Evidence collection varies by metric and may be empty for high-scoring or unavailable metrics.
**How to avoid:** Check `len(evidence) > 0` before rendering the files section. For metrics without evidence, use the metric description's "How to Improve" guidance only. Consider not showing "Improve" button for score >= 9 or unavailable metrics.
**Warning signs:** Prompts with "Files to Focus On:" followed by nothing.

### Pitfall 4: Prompt Too Long for Context Window
**What goes wrong:** Including all evidence items (could be 20+) makes the prompt too verbose for practical use.
**Why it happens:** Evidence slices can be large for metrics like complexity or function length.
**How to avoid:** Limit to top 5 evidence items (worst offenders). This matches the trace modal pattern.
**Warning signs:** Prompts that are multiple pages long.

### Pitfall 5: Build/Test Commands Not Available
**What goes wrong:** The project config (`.arsrc.yml`) has no build/test command fields. Prompts need these per PR-08.
**Why it happens:** Config currently only has scoring overrides, not build/test metadata.
**How to avoid:** Use language-detected defaults. The pipeline already detects languages (Go/Python/TypeScript). Map to standard commands: Go -> `go build ./...` / `go test ./...`, Python -> `python -m pytest`, TypeScript -> `npm test`. Include a note like "(adjust commands for your project)".
**Warning signs:** Blank build/test sections.

### Pitfall 6: C7 Metrics Have No Breakpoints
**What goes wrong:** C7 metrics are scored by the agent evaluation system, not breakpoint interpolation. There are no `Breakpoint` entries for C7 metrics in `ScoringConfig`.
**Why it happens:** C7 scoring is fundamentally different from C1-C6.
**How to avoid:** For C7 metrics, use the description's "How to Improve" section as the task guidance. Set target score to a reasonable default (e.g., current + 2, max 10). Don't try to compute target raw values from breakpoints.
**Warning signs:** Crash or empty targets for C7 metrics.

## Code Examples

### Example 1: Prompt Renderer Function Signature

```go
// Source: derived from trace.go pattern
func renderImprovementPrompt(params PromptParams) string {
    var b strings.Builder
    // ... build prompt sections
    return b.String()
}

type PromptParams struct {
    CategoryName    string
    CategoryDisplay string
    CategoryImpact  string
    MetricName      string
    MetricDisplay   string
    RawValue        float64
    FormattedValue  string
    Score           float64
    TargetScore     float64
    TargetValue     float64
    Evidence        []types.EvidenceItem
    Breakpoints     []scoring.Breakpoint
    Language        string // detected project language for build/test commands
}
```

### Example 2: Wiring Into HTMLSubScore (in buildHTMLSubScores)

```go
// After existing trace population, add prompt:
if ss.Available && ss.Score < 9.0 {
    promptHTML := renderImprovementPrompt(PromptParams{
        CategoryName:    categoryName,
        CategoryDisplay: categoryDisplayName(categoryName),
        CategoryImpact:  categoryImpact(categoryName),
        MetricName:      ss.MetricName,
        MetricDisplay:   metricDisplayName(ss.MetricName),
        RawValue:        ss.RawValue,
        FormattedValue:  formatMetricValue(ss.MetricName, ss.RawValue, ss.Available),
        Score:           ss.Score,
        Evidence:        ss.Evidence,
        Breakpoints:     breakpoints,
        Language:        detectedLanguage,
    })
    if promptHTML != "" {
        hss.PromptHTML = template.HTML(promptHTML)
        hss.HasPrompt = true
    }
}
```

### Example 3: Copy Button with Fallback Chain

```html
<div class="prompt-copy-container">
    <button class="trace-copy-btn" onclick="copyPromptText(this)">Copy</button>
    <pre><code>{escaped prompt text}</code></pre>
</div>
```

### Example 4: Generated Prompt Output (complexity_avg, score 4.2)

```
## Context

I'm working on improving the Complexity avg metric in this codebase.
Current score: 4.2/10 (raw value: 18.3)
Target score: 6.0/10 (target value: 10.0)

Category: C1: Code Health
Why it matters: Lower complexity and smaller functions help agents reason about and modify code safely.

## Build & Test Commands

go build ./...
go test ./...

## Task

Reduce the average cyclomatic complexity from 18.3 to 10.0 or below.

Focus on these high-complexity functions:

1. internal/analyzer/c1_code_quality/go.go:142 - complexity 34 - analyzeGoFunction
2. internal/parser/treesitter.go:89 - complexity 28 - parseTypeScript
3. internal/output/html.go:208 - complexity 22 - buildHTMLSubScores
4. internal/pipeline/pipeline.go:156 - complexity 19 - runAnalyzers
5. cmd/scan.go:45 - complexity 18 - runScan

For each function:
- Replace nested conditionals with guard clauses (early returns)
- Extract conditional logic into well-named helper functions
- Use polymorphism or strategy pattern instead of switch statements
- Keep nesting depth at most 4 levels

## Verification

After making changes, run: go test ./...
Then re-scan: ars scan . --output-html /tmp/report.html
Check that the Complexity avg score has improved above 6.0.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `document.execCommand('copy')` | `navigator.clipboard.writeText()` | 2018+ | Must still support execCommand as fallback for file:// |
| Custom modal divs | Native `<dialog>` element | Baseline 2022 | Already implemented in Phase 31 |

**Deprecated/outdated:**
- `document.execCommand('copy')` is deprecated but still the only option for `file://` protocol. Must be kept as fallback.

## Open Questions

1. **Language detection for build/test commands**
   - What we know: Pipeline detects languages via `discovery.DetectProjectLanguages()`. The `HTMLReportData` does not currently include detected language.
   - What's unclear: Best way to pass language info through to prompt rendering. Could add to `TraceData`, `HTMLReportData`, or `HTMLCategory`.
   - Recommendation: Add a `Languages []string` field to `HTMLReportData` and pass through to prompt rendering. Use the first detected language for build/test command defaults. If multiple languages detected, include commands for all.

2. **Should C7 metrics get "Improve" buttons?**
   - What we know: C7 metrics are live agent evaluations. They have description "How to Improve" sections but no breakpoints.
   - What's unclear: Whether improvement prompts make sense for C7 (the metric is about how well agents perform, not a code property).
   - Recommendation: Yes, include them. The "How to Improve" sections for C7 metrics focus on code properties that improve agent performance (simplify code, better names, etc.), which are actionable.

3. **Maximum prompt length consideration**
   - What we know: Evidence is limited to top 5 already. Descriptions have "How to Improve" bullet lists.
   - What's unclear: Whether prompt length should be capped or left to natural content length.
   - Recommendation: Don't cap artificially. Natural content (5 evidence items + template text) should be 200-400 words, well within any AI context window.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `internal/output/html.go` - HTMLSubScore structure, trace wiring pattern
- Codebase analysis: `internal/output/trace.go` - renderBreakpointTrace() pattern for server-side HTML rendering
- Codebase analysis: `internal/output/descriptions.go` - "How to Improve" sections for all metrics
- Codebase analysis: `internal/output/templates/report.html` - existing modal, template elements, button patterns
- Codebase analysis: `internal/scoring/config.go` - Breakpoint structure, target calculation data
- Codebase analysis: `pkg/types/scoring.go` - EvidenceItem structure
- Phase 31 Research: `<dialog>` modal infrastructure, showModal(), backdrop click, scroll lock

### Secondary (MEDIUM confidence)
- MDN Web Docs: Clipboard API secure context requirement (well-documented browser behavior)
- MDN Web Docs: `document.execCommand('copy')` deprecated status but continued browser support

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - entirely based on existing codebase patterns (trace.go, html.go)
- Architecture: HIGH - follows exact same pattern as Phase 32 call trace modals
- Pitfalls: HIGH - clipboard fallback chain is well-documented; evidence edge cases observed in codebase
- Prompt templates: MEDIUM - the 4-section structure is defined by requirements, but the specific wording within templates is author discretion

**Research date:** 2026-02-07
**Valid until:** 2026-03-07 (stable - all patterns already established in codebase)
