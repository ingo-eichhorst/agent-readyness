# Architecture Research: Academic Citations in HTML Reports

**Domain:** Academic citation integration in technical HTML documentation
**Researched:** 2026-02-04
**Confidence:** HIGH

## Executive Summary

The ARS HTML report system already has a functional citation architecture. The milestone task is to **extend existing patterns** rather than build new infrastructure. The current system uses inline parenthetical citations with a per-category reference section—this is the correct approach for technical documentation and should be preserved.

**Key finding:** The existing architecture is sound. The work is content expansion (adding more citations to metric descriptions), not structural changes.

## Current Architecture Analysis

### Existing System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Template Layer                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ citations.go │  │descriptions│  │    html.go          │  │
│  │ Citation{}   │  │   .go      │  │ HTMLReportData{}    │  │
│  │ researchCi- │  │ metricDesc-│  │ buildHTMLCategories │  │
│  │ tations[]   │  │ riptions{} │  │ filterCitationsBy-  │  │
│  └──────┬──────┘  └──────┬─────┘  │ Category()          │  │
│         │                │        └──────────┬──────────┘  │
├─────────┴────────────────┴───────────────────┴──────────────┤
│                     HTML Template                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ templates/report.html                                    ││
│  │ - Inline: <span class="citation">                       ││
│  │ - Per-category: .category-citations                      ││
│  └─────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────┤
│                      CSS Styling                             │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ templates/styles.css                                     ││
│  │ - .citation { muted color, normal font-style }          ││
│  │ - .category-citations { border-top, smaller font }       ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | File | Current State | Modification Needed |
|-----------|------|---------------|---------------------|
| Citation struct | `citations.go` | Defines Category, Title, Authors, Year, URL, Description | **No change** - structure is complete |
| Citation data | `citations.go` | 13 citations across 6 categories | **Expand** - add metric-level citations |
| Metric descriptions | `descriptions.go` | 33 metrics with inline `<span class="citation">` | **Expand** - more inline citations per metric |
| HTML template | `report.html` | Renders per-category citations | **No change** - works correctly |
| CSS styling | `styles.css` | `.citation` and `.category-citations` classes | **No change** - styling is appropriate |

### Data Flow

```
researchCitations[]                 metricDescriptions{}
        │                                   │
        │                                   │
        ▼                                   ▼
buildHTMLCategories()              buildHTMLSubScores()
        │                                   │
        │                                   │
        ▼                                   ▼
HTMLCategory.Citations[]           HTMLSubScore.DetailedDescription
        │                                   │
        │                                   │
        ▼                                   ▼
┌────────────────────────────────────────────────────────────┐
│                    report.html                              │
├────────────────────────────────────────────────────────────┤
│  {{range .Categories}}                                      │
│    ...metric table with inline citations...                 │
│    {{if .Citations}}                                        │
│      <div class="category-citations">                       │
│        <h4>References</h4>                                  │
│        <ul>{{range .Citations}}<li>...</li>{{end}}</ul>     │
│      </div>                                                 │
│    {{end}}                                                  │
│  {{end}}                                                    │
└────────────────────────────────────────────────────────────┘
```

## Recommended Citation Patterns

### Pattern 1: Inline Parenthetical Citations (KEEP)

The existing approach is correct for technical documentation.

**What:** Citations appear inline as `(Author, Year)` within text
**Current implementation:**
```html
<span class="citation">(Borg et al., 2026)</span>
```
**Why this works:**
- Familiar academic format readers recognize
- Doesn't interrupt reading flow
- CSS-only rendering (no JavaScript required)
- Works within Go `template.HTML` content

**Best practice verified by research:**
- [W3C Scholarly HTML](https://w3c.github.io/scholarly-html/) recommends inline citations linked to reference sections
- [MDN `<cite>` element docs](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/cite) notes `<cite>` is for **titles** not author attributions
- The `<span class="citation">` approach is semantically correct for author-year references

### Pattern 2: Per-Category Reference Sections (KEEP)

**What:** Each category (C1-C6) has a "References" section at the bottom
**Current implementation:**
```html
{{if .Citations}}
<div class="category-citations">
    <h4>References</h4>
    <ul>
        {{range .Citations}}
        <li><a href="{{.URL}}" target="_blank" rel="noopener">{{.Title}}</a>
            ({{.Authors}}, {{.Year}}) - {{.Description}}</li>
        {{end}}
    </ul>
</div>
{{end}}
```
**Why this works:**
- Keeps references contextually relevant (same screen as metrics)
- Users can click through to verify claims
- Doesn't require scrolling to a distant global bibliography
- Aligns with [PubCSS](https://thomaspark.co/2015/01/pubcss-formatting-academic-publications-in-html-css/) approach for HTML academic papers

### Pattern 3: Linked Citations (CONSIDER FOR FUTURE)

**What:** Make inline citations clickable to jump to reference section
**Not currently implemented, would require:**
```html
<a href="#citation-borg-2026" class="citation">(Borg et al., 2026)</a>
...
<li id="citation-borg-2026">...</li>
```
**Trade-off:**
- Pro: Direct navigation to full reference
- Con: Additional complexity, CSS counters for numbered citations
- Recommendation: **Defer** - current parenthetical style is sufficient for this use case

## What NOT to Change

### 1. Do Not Add JavaScript-Based Features

The current architecture correctly uses CSS-only rendering. Avoid:
- Tooltip popovers for citation previews
- JavaScript-based citation numbering
- Dynamic reference loading

**Rationale:** CSP-safe, works offline, consistent rendering

### 2. Do Not Use Global Bibliography

A single "References" section at the document end would be worse:
- Forces users to scroll away from metric context
- Harder to find relevant citations
- Less scannable

**Rationale:** Per-category references keep context together

### 3. Do Not Use CSS Counters for Numbered Citations

[CSS counters](https://www.w3schools.com/css/css_counters.asp) could auto-number citations `[1]`, `[2]`, etc.:
```css
body { counter-reset: citations; }
.citation { counter-increment: citations; }
.citation::after { content: "[" counter(citations) "]"; }
```
**Why not use this:**
- Author-year format `(Borg, 2026)` is more informative than `[1]`
- Numbered format requires cross-referencing to understand
- Current format works well for 3-5 citations per description

## Implementation Architecture

### Component Changes Required

```
┌─────────────────────────────────────────────────────────────┐
│                   Files to Modify                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. internal/output/citations.go                             │
│     ├─ Add Metric field to Citation struct (OPTIONAL)        │
│     └─ Expand researchCitations with per-metric citations    │
│                                                              │
│  2. internal/output/descriptions.go                          │
│     └─ Add more inline <span class="citation"> in Detailed   │
│                                                              │
│  3. internal/output/html.go                                  │
│     └─ Possibly add filterCitationsByMetric() helper         │
│        (only if per-metric display desired)                  │
│                                                              │
│  No changes needed:                                          │
│  - templates/report.html (structure is correct)              │
│  - templates/styles.css (styling is appropriate)             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Option A: Expand Category-Level Citations (RECOMMENDED)

**Approach:** Keep `Citation.Category` field, add more citations per category

**Pros:**
- Minimal code changes
- Existing template works unchanged
- Simple data structure

**Cons:**
- Cannot show metric-specific references (but inline citations serve this purpose)

**Implementation:**
```go
// citations.go - just add more entries
var researchCitations = []Citation{
    // Existing C1 citations...
    {
        Category:    "C1",
        Title:       "New Research Paper",
        Authors:     "Smith et al.",
        Year:        2024,
        URL:         "https://...",
        Description: "Relevance to complexity metrics",
    },
}
```

### Option B: Add Metric-Level Citations (ALTERNATIVE)

**Approach:** Add `Metric` field to Citation struct

**Pros:**
- More granular citation-to-metric mapping
- Could show references per metric in future UI

**Cons:**
- Requires template changes for per-metric display
- More complex data structure
- May be over-engineering for current needs

**Implementation:**
```go
type Citation struct {
    Category    string // "C1", "C2", etc.
    Metric      string // "complexity_avg", "" for category-level
    Title       string
    Authors     string
    Year        int
    URL         string
    Description string
}
```

### Recommendation

**Use Option A (expand category-level)** because:
1. Existing architecture handles it correctly
2. Per-metric citations would clutter the UI
3. The inline `<span class="citation">` already links text to specific claims
4. Category-level references provide "further reading" without overwhelming

## Suggested Implementation Order

### Phase 1: Content Expansion (Primary Work)

Work through categories in dependency order:

| Order | Category | Metrics | Rationale |
|-------|----------|---------|-----------|
| 1 | C1: Code Health | 6 metrics | Most citations exist, foundational complexity research |
| 2 | C6: Testing | 5 metrics | Well-researched domain, clear academic sources |
| 3 | C2: Semantic Explicitness | 5 metrics | Type theory research readily available |
| 4 | C3: Architecture | 5 metrics | Classic software engineering references |
| 5 | C4: Documentation | 7 metrics | Mix of academic and industry sources |
| 6 | C5: Temporal Dynamics | 5 metrics | Tornhill's work covers most, fewer academic papers |

### Phase 2: Optional Enhancements (Defer)

These could improve citation UX but are not required:

1. **Citation linking** - Make inline citations jump to reference
2. **Citation tooltips** - CSS-only hover preview (requires `:hover` + sibling selectors)
3. **BibTeX export** - Generate downloadable bibliography

## Integration Points

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| citations.go <-> html.go | Direct import, buildHTMLCategories() | Well-defined, no changes needed |
| descriptions.go <-> html.go | Direct import, getMetricDescription() | Inline HTML content, no changes needed |
| html.go <-> report.html | template.Execute() | Data binding works correctly |

### External Dependencies

| Dependency | Status | Notes |
|------------|--------|-------|
| Go html/template | Stable | No changes to template syntax |
| CSS counters | Not used | Could add for numbered citations (not recommended) |
| JavaScript | Intentionally avoided | CSP-safe by design |

## Anti-Patterns to Avoid

### Anti-Pattern 1: Over-Citation

**What people do:** Add 5+ citations per paragraph to seem authoritative
**Why it's wrong:** Interrupts reading, diminishes each citation's impact
**Do this instead:** 1-3 citations per key claim, prefer seminal works over derivative papers

### Anti-Pattern 2: Citation Without Integration

**What people do:** Add citations at end of description without inline references
**Why it's wrong:** Reader doesn't know which claim each citation supports
**Do this instead:** Always have inline `<span class="citation">` matching reference section entries

### Anti-Pattern 3: Mixing Citation Styles

**What people do:** Some citations as `(Author, Year)`, others as `[1]`
**Why it's wrong:** Inconsistent, confusing
**Do this instead:** Use `(Author, Year)` throughout (already the standard in codebase)

## Confidence Assessment

| Aspect | Confidence | Reasoning |
|--------|------------|-----------|
| Keep inline parenthetical format | HIGH | [W3C Scholarly HTML](https://w3c.github.io/scholarly-html/), [MDN cite docs](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/cite), existing working implementation |
| Keep per-category references | HIGH | [PubCSS approach](https://thomaspark.co/2015/01/pubcss-formatting-academic-publications-in-html-css/), better UX than global bibliography |
| No JavaScript needed | HIGH | [Accessible footnotes guide](https://niquette.ca/articles/accessible-footnotes/) confirms CSS-only approach |
| Expand citations.go, descriptions.go | HIGH | Clear extension point, no architectural changes |

## Sources

### Official Documentation
- [W3C Scholarly HTML](https://w3c.github.io/scholarly-html/) - Reference section structure, semantic markup
- [MDN `<cite>` Element](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/cite) - Semantic HTML for titles vs attributions
- [W3C CSS Generated Content](https://www.w3.org/TR/CSS21/generate.html) - CSS counters specification

### Academic Citation Patterns
- [PubCSS: Formatting Academic Publications in HTML & CSS](https://thomaspark.co/2015/01/pubcss-formatting-academic-publications-in-html-css/) - HTML/CSS academic paper formatting
- [Chicago Style Footnotes](https://www.scribbr.com/chicago-style/footnotes/) - Academic footnote conventions

### Accessibility
- [Accessible Footnotes HTML Design](https://niquette.ca/articles/accessible-footnotes/) - ARIA attributes, CSS counters for footnotes
- [DubBot Accessible Footnotes](https://dubbot.com/dubblog/2024/a-footnote-on-footnotes-they-need-to-be-accessible.html) - Screen reader considerations

### CSS Techniques
- [CSS Counters - W3Schools](https://www.w3schools.com/css/css_counters.asp) - Counter-reset, counter-increment
- [CSS-Tricks counter()](https://css-tricks.com/almanac/functions/c/counter/) - Counter function reference

---
*Architecture research for: Academic citation integration in ARS HTML reports*
*Researched: 2026-02-04*
