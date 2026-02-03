# Phase 14: HTML Enhancements - Research

**Researched:** 2026-02-03
**Domain:** HTML/CSS expandable UI components, metric descriptions with research citations
**Confidence:** HIGH

## Summary

This phase enhances the existing HTML report with expandable metric descriptions. The core technology is the native HTML `<details>/<summary>` element, which provides CSS-only toggle functionality without JavaScript. Each metric will show a brief description (always visible) and an expandable detailed section with research citations.

The HTML `<details>` element has excellent browser support (baseline since January 2020) and provides built-in keyboard accessibility. The `open` boolean attribute controls default expansion state, which is perfect for the requirement to auto-expand low-scoring metrics. Styling is achieved through CSS using `details[open]` selectors and custom marker indicators.

**Critical limitation discovered:** A CSS-only "Expand All / Collapse All" control is NOT possible. CSS cannot programmatically set HTML attributes across multiple elements. The CONTEXT.md specifies this feature, but it requires minimal JavaScript (a single line to toggle `open` attributes). This is flagged as an open question requiring user decision.

**Primary recommendation:** Use native `<details>/<summary>` elements with the `open` attribute for low-scoring metrics. For "Expand All/Collapse All", either accept minimal JavaScript or remove this feature.

## Standard Stack

This phase requires no external libraries - only native HTML5 and CSS.

### Core
| Element | Support | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `<details>` | Baseline 2020 | Disclosure widget container | Native HTML5, no JS needed, built-in a11y |
| `<summary>` | Baseline 2020 | Clickable toggle label | Required child of details, keyboard accessible |
| `open` attribute | Baseline 2020 | Default expanded state | Boolean attribute controls initial state |

### CSS Selectors
| Selector | Purpose | Browser Support |
|----------|---------|-----------------|
| `details[open]` | Style expanded state | Universal |
| `summary::marker` | Custom triangle icon | Good (Safari needs webkit prefix) |
| `::-webkit-details-marker` | Safari marker support | Safari/WebKit only |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `<details>/<summary>` | checkbox hack | More CSS complexity, less semantic |
| Native marker | `::before` pseudo-element | More styling control, but removes list-item behavior |
| `open` attribute | JavaScript state | Unnecessary complexity for simple toggle |

**Installation:** None required - native HTML/CSS only.

## Architecture Patterns

### Recommended Data Structure Addition

```go
// Add to HTMLSubScore struct in html.go
type HTMLSubScore struct {
    // ... existing fields ...
    BriefDescription    string  // Always visible, 1-2 sentences
    DetailedDescription template.HTML // Expandable content with sections
    ShouldExpand        bool    // true if score is below threshold
}
```

### Recommended Template Structure

```html
<!-- Per-metric row with expandable description -->
<tr>
    <td>
        <details {{if .ShouldExpand}}open{{end}}>
            <summary>
                <span class="metric-name">{{.DisplayName}}</span>
                <span class="chevron"></span>
            </summary>
            <div class="metric-brief">{{.BriefDescription}}</div>
            <div class="metric-details">{{.DetailedDescription}}</div>
        </details>
    </td>
    <td>{{.FormattedValue}}</td>
    <td class="score-cell score-{{.ScoreClass}}">{{printf "%.1f" .Score}}</td>
    <td>{{printf "%.0f" .WeightPct}}%</td>
</tr>
```

### Recommended CSS Pattern

```css
/* Source: MDN Web Docs - details element styling */

/* Base styling */
details {
    border-left: 3px solid var(--color-border);
    padding-left: 0.75rem;
    margin: 0.5rem 0;
}

details[open] {
    border-left-color: var(--color-muted);
}

/* Summary (clickable header) */
summary {
    cursor: pointer;
    list-style: none; /* Remove default marker */
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

summary::-webkit-details-marker {
    display: none; /* Safari compatibility */
}

/* Custom chevron indicator */
.chevron::before {
    content: "\25B6"; /* Right-pointing triangle */
    font-size: 0.75rem;
    transition: transform 0.2s;
}

details[open] .chevron::before {
    content: "\25BC"; /* Down-pointing triangle */
}

/* Brief description (always visible under metric name) */
.metric-brief {
    font-size: 0.8rem;
    color: var(--color-muted);
    margin-top: 0.25rem;
    font-style: italic;
}

/* Detailed content (only visible when expanded) */
.metric-details {
    margin-top: 0.75rem;
    padding: 0.75rem;
    background: var(--color-surface);
    border-radius: 0.25rem;
    font-size: 0.85rem;
    line-height: 1.6;
}

/* Section headers within expanded content */
.metric-details h4 {
    font-size: 0.85rem;
    font-weight: 600;
    margin: 0.75rem 0 0.25rem 0;
    color: var(--color-text);
}

.metric-details h4:first-child {
    margin-top: 0;
}

/* Citations styling */
.citation {
    color: var(--color-muted);
    font-style: normal;
}
```

### Pattern: Content Structure for Expanded Sections

Per CONTEXT.md, expanded sections should follow this structure:

```html
<div class="metric-details">
    <h4>Definition</h4>
    <p>What this metric measures and how it's calculated.</p>

    <h4>Why It Matters for AI Agents</h4>
    <p>Impact on agent code comprehension and task completion.</p>

    <h4>Research Evidence</h4>
    <p>Studies supporting this metric. <span class="citation">(Author et al., Year)</span></p>

    <h4>Recommended Thresholds</h4>
    <p>Target values and what they mean for agent readiness.</p>

    <h4>How to Improve</h4>
    <p>Actionable steps to improve this metric.</p>
</div>
```

### Anti-Patterns to Avoid

- **Headings inside `<summary>`:** Screen readers (JAWS) ignore headings in summary elements, breaking document flow. Use spans with styling instead.
- **`display: flex` on `<summary>`:** This removes the list-item marker behavior. Use `list-style: none` and custom markers instead.
- **`open="false"` to close:** Boolean attributes are present/absent, not true/false. Use `removeAttribute("open")` or simply omit.
- **Nested `<details>` without child selectors:** Use `details[open] > summary` not `details[open] summary` to avoid styling nested details incorrectly.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Collapsible sections | JavaScript accordion | `<details>/<summary>` | Native a11y, keyboard support, no JS |
| Expand indicator | Custom icon management | CSS `::before` with Unicode | Automatic state updates via CSS |
| Initial expanded state | JavaScript onload | `open` attribute | Works without JS, SSR-friendly |
| Focus management | Custom keyboard handlers | Native `<summary>` | Built-in spacebar/enter support |

**Key insight:** The HTML5 `<details>` element handles all toggle logic, keyboard navigation, and ARIA semantics automatically. Custom JavaScript solutions add complexity without benefit for this use case.

## Common Pitfalls

### Pitfall 1: Missing Safari Marker Support
**What goes wrong:** Custom marker styling doesn't work in Safari
**Why it happens:** Safari uses `::-webkit-details-marker` instead of `::marker`
**How to avoid:** Include both selectors:
```css
summary::marker,
summary::-webkit-details-marker {
    display: none;
}
```
**Warning signs:** Triangle visible in Safari despite `list-style: none`

### Pitfall 2: Block Elements in Summary
**What goes wrong:** Heading or block element pushes content below the marker
**Why it happens:** Block elements create their own line, wrapping occurs
**How to avoid:** Use `display: inline` on elements inside summary:
```css
details summary > * {
    display: inline;
}
```
**Warning signs:** Arrow/chevron appears above text instead of beside it

### Pitfall 3: Cursor Not Pointer
**What goes wrong:** Users don't realize summary is clickable
**Why it happens:** Default cursor is text selection, not pointer
**How to avoid:** Add to CSS reset:
```css
details summary {
    cursor: pointer;
}
```
**Warning signs:** Text cursor when hovering over summary

### Pitfall 4: Auto-Expand Threshold Logic in Template
**What goes wrong:** Complex threshold logic clutters Go template
**Why it happens:** Each metric has different thresholds
**How to avoid:** Compute `ShouldExpand` boolean in Go code, pass to template
**Warning signs:** Template contains score comparisons and conditionals

### Pitfall 5: Content Descriptions as Inline Strings
**What goes wrong:** HTML content mixed with Go code becomes unmaintainable
**Why it happens:** Descriptions have structure (headers, citations, paragraphs)
**How to avoid:** Create a `descriptions.go` file with structured data or embed markdown
**Warning signs:** Long multi-line strings with HTML in Go code

## Code Examples

### Basic Details/Summary (MDN Reference)

```html
<!-- Source: MDN Web Docs - https://developer.mozilla.org/en-US/docs/Web/HTML/Element/details -->
<details>
    <summary>Click to expand</summary>
    <p>This content is hidden until the user clicks.</p>
</details>

<!-- Pre-expanded with open attribute -->
<details open>
    <summary>Already expanded</summary>
    <p>This content is visible by default.</p>
</details>
```

### Cross-Browser Marker Styling (CSS-Tricks)

```css
/* Source: CSS-Tricks - https://css-tricks.com/two-issues-styling-the-details-element-and-how-to-solve-them/ */

/* Reset for consistent cross-browser behavior */
details summary {
    cursor: pointer;
}

details summary > * {
    display: inline;
}

/* Remove default marker */
summary {
    list-style: none;
}

summary::-webkit-details-marker {
    display: none;
}

/* Custom marker with ::before */
summary::before {
    content: "\25B6 "; /* Right triangle */
    font-size: 0.7em;
    margin-right: 0.5em;
}

details[open] > summary::before {
    content: "\25BC "; /* Down triangle */
}
```

### Go Template Integration Pattern

```go
// descriptions.go - metric description data
type MetricDescription struct {
    Brief    string // 1-2 sentences, always visible
    Detailed string // Full HTML content for expanded section
    Threshold float64 // Score below which to auto-expand
}

var metricDescriptions = map[string]MetricDescription{
    "complexity_avg": {
        Brief: "Measures average cyclomatic complexity per function. Keep under 10 for optimal agent comprehension.",
        Detailed: `<h4>Definition</h4>
<p>Cyclomatic complexity counts the number of independent paths through a function's control flow graph...</p>
<h4>Research Evidence</h4>
<p>McCabe (1976) established complexity thresholds. <span class="citation">(McCabe, 1976)</span></p>`,
        Threshold: 6.0,
    },
    // ... more metrics
}
```

### Template Conditional Open Attribute

```html
{{/* Go template - conditional open attribute */}}
{{range .SubScores}}
<details {{if .ShouldExpand}}open{{end}}>
    <summary>{{.DisplayName}}</summary>
    <div class="metric-brief">{{.BriefDescription}}</div>
    <div class="metric-details">{{.DetailedDescription}}</div>
</details>
{{end}}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| JavaScript accordion | Native `<details>` | HTML5 (2014), baseline 2020 | No JS required, better a11y |
| Custom ARIA roles | Implicit `group` role | Built into `<details>` | Less manual a11y work |
| CSS checkbox hack | `<details>/<summary>` | 2020+ broad support | Cleaner markup |
| `::marker` only | `::marker` + `::-webkit-details-marker` | Safari lag | Cross-browser support |

**Deprecated/outdated:**
- **JavaScript toggle libraries:** Unnecessary for simple disclosure patterns
- **Hidden checkbox pattern:** More complex than native solution
- **Pre-Chromium Edge support:** Not needed (baseline 2020)

## Open Questions

### 1. Expand All / Collapse All Without JavaScript

**What we know:**
- CONTEXT.md specifies "Add Expand all / Collapse all control"
- CSS cannot modify HTML attributes programmatically
- CSS `:has()` selector can react to state but cannot set state
- Every solution found requires JavaScript

**What's unclear:**
- Whether minimal JS (single onclick handler) is acceptable
- If this feature should be deferred

**Recommendation:**
Either accept minimal JavaScript:
```html
<button onclick="document.querySelectorAll('details').forEach(d => d.open = true)">Expand All</button>
<button onclick="document.querySelectorAll('details').forEach(d => d.open = false)">Collapse All</button>
```
Or remove the feature entirely. A CSS-only solution does not exist.

### 2. Research Citations Data Source

**What we know:**
- Existing `citations.go` has per-category citations
- CONTEXT.md wants metric-level citations with full academic format
- Borg et al. (2026) is mentioned but not found in searches

**What's unclear:**
- Whether to create fictional research citations or use real ones
- How specific research maps to specific metrics

**Recommendation:**
Use existing real research (McCabe, Fowler, etc.) from `citations.go` and software engineering literature. Map research to specific metrics where empirical evidence exists. For metrics without direct research, cite the foundational papers that establish the concept.

### 3. Brief Description Position

**What we know:**
- CONTEXT.md says "Position: Below the metric value"
- Current template uses table rows per metric
- Adding description under value in table requires colspan or restructure

**What's unclear:**
- Whether to break table structure or use nested elements within cells

**Recommendation:**
Keep table structure for values/scores/weights but embed `<details>` within the Metric Name cell. Brief description appears under the metric name, not under the value column.

## Sources

### Primary (HIGH confidence)
- [MDN Web Docs - details element](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/details) - Complete API reference, browser support, accessibility
- [MDN Web Docs - summary element](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/summary) - Required structure, ARIA roles
- [CSS-Tricks - Styling Details Element](https://css-tricks.com/two-issues-styling-the-details-element-and-how-to-solve-them/) - Cross-browser CSS solutions

### Secondary (MEDIUM confidence)
- [web.dev - Details and Summary](https://web.dev/learn/html/details) - Accessibility patterns, content structure
- [SitePoint - Styling Details Element](https://www.sitepoint.com/style-html-details-element/) - 20 styling approaches
- Multiple academic sources for metric research citations (McCabe 1976, Fowler 1999, etc.)

### Tertiary (LOW confidence)
- WebSearch results for CSS-only expand-all (confirmed NOT possible)
- Anthropic 2026 Agentic Coding Trends Report (PDF not extractable)
- arXiv papers on LLM code comprehension (tangential relevance)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Native HTML5, MDN-documented, baseline 2020
- Architecture: HIGH - Patterns verified with MDN and CSS-Tricks
- Pitfalls: HIGH - Common issues well-documented across multiple sources
- Expand All limitation: HIGH - Confirmed across multiple searches, CSS cannot set attributes
- Research citations content: MEDIUM - Real research exists but mapping to metrics requires judgment

**Research date:** 2026-02-03
**Valid until:** Stable technology - 90+ days validity (HTML5 details element is mature)
