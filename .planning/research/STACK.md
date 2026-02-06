# Stack Research

**Domain:** Go CLI static analysis tool (codebase quality scorer) -- v0.0.3 features
**Researched:** 2026-02-03
**Confidence:** HIGH (verified against official documentation)

## Context: Current Stack (v0.0.2, Validated)

The following technologies are already in use and validated. Listed for reference only.

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.24+ | Runtime |
| `go/ast` + `go/parser` + `go/token` | stdlib | Go source AST parsing |
| `go/types` | stdlib | Go type checking |
| `golang.org/x/tools/go/packages` | v0.41.0 | Go package loading |
| `spf13/cobra` | v1.10.2 | CLI framework |
| `fzipp/gocyclo` | v0.6.0 | Cyclomatic complexity |
| `fatih/color` | v1.18.0 | Terminal color output |
| `sabhiram/go-gitignore` | latest | Gitignore pattern matching |
| `encoding/json` | stdlib | JSON output |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML config parsing |
| `tree-sitter/go-tree-sitter` | v0.25.0 | Multi-language parsing |
| `github.com/anthropics/anthropic-sdk-go` | v1.20.0 | **REMOVING in v0.0.3** |
| `html/template` | stdlib | HTML report generation |
| `embed` | stdlib | Embedded templates |
| `os/exec` | stdlib | Claude CLI execution |

---

## v0.0.3 Stack Changes

### Executive Summary

v0.0.3 requires **zero new Go dependencies**. The key changes are:

1. **REMOVE** `github.com/anthropics/anthropic-sdk-go` -- Claude Code headless replaces direct API
2. **ENHANCE** existing `os/exec` + `claude` CLI integration for C4 (was API-only) and C7
3. **ADD** badge URL generation (pure string formatting, no deps)
4. **ENHANCE** existing `html/template` with `<details>/<summary>` elements

**Net effect:** One dependency removed, zero added.

---

## 1. Claude Code Headless Integration (Replacing Anthropic SDK)

### Current State

C4 (Documentation Quality) uses `internal/llm/client.go` with the Anthropic SDK for LLM-based quality assessment. C7 (Agent Evaluation) uses `internal/agent/executor.go` with Claude CLI headless mode.

### v0.0.3 Change

**Unify both C4 and C7 under Claude Code headless mode.** Remove the Anthropic SDK entirely.

### Why Remove Anthropic SDK

| Factor | SDK Approach | Claude Code Headless |
|--------|--------------|---------------------|
| Auth | ANTHROPIC_API_KEY env var | Claude Code manages auth |
| Cost control | Manual token estimation | Built into Claude Code |
| Retry logic | Manual implementation | Built into Claude Code |
| Caching | None | Prompt caching handled |
| Agent capabilities | API only | Full agent loop for C7 |
| Maintenance | Two auth paths | Single tool |

**Decision:** Remove SDK, use `claude -p` for all LLM interactions.

### Implementation: Claude Code for C4

Replace `internal/llm/client.go` with a thin wrapper around Claude Code CLI:

```go
// internal/llm/claude_code.go
package llm

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
)

type Evaluation struct {
    Score     int    `json:"score"`
    Reasoning string `json:"reason"`
}

type claudeCodeResponse struct {
    Result          string `json:"result"`
    StructuredOutput *Evaluation `json:"structured_output,omitempty"`
}

func EvaluateWithClaudeCode(ctx context.Context, systemPrompt, content string) (Evaluation, error) {
    schemaJSON := `{"type":"object","properties":{"score":{"type":"integer","minimum":1,"maximum":10},"reason":{"type":"string"}},"required":["score","reason"]}`

    args := []string{
        "-p", content,
        "--append-system-prompt", systemPrompt,
        "--output-format", "json",
        "--json-schema", schemaJSON,
        "--allowedTools", "",  // No tools needed for evaluation
    }

    cmd := exec.CommandContext(ctx, "claude", args...)
    output, err := cmd.Output()
    if err != nil {
        return Evaluation{}, fmt.Errorf("claude CLI failed: %w", err)
    }

    var resp claudeCodeResponse
    if err := json.Unmarshal(output, &resp); err != nil {
        return Evaluation{}, fmt.Errorf("invalid JSON: %w", err)
    }

    if resp.StructuredOutput != nil {
        return *resp.StructuredOutput, nil
    }

    // Fallback: parse result field if structured_output not present
    var eval Evaluation
    if err := json.Unmarshal([]byte(resp.Result), &eval); err != nil {
        return Evaluation{}, fmt.Errorf("could not parse result: %w", err)
    }
    return eval, nil
}
```

### CLI Response Structure (Verified)

From [Claude Code Headless Documentation](https://code.claude.com/docs/en/headless):

```json
{
    "type": "result",
    "session_id": "abc123",
    "result": "The text response",
    "structured_output": { ... }  // When --json-schema provided
}
```

**Source:** [Claude Code CLI Reference](https://code.claude.com/docs/en/cli-reference)

### Key CLI Flags

| Flag | Purpose | C4 Usage | C7 Usage |
|------|---------|----------|----------|
| `-p "prompt"` | Non-interactive mode | Doc evaluation prompt | Task prompt |
| `--output-format json` | Structured output | Parse score | Parse task result |
| `--json-schema` | Enforce response format | Score extraction | Task scoring |
| `--append-system-prompt` | Add evaluation rubric | Quality rubric | Task instructions |
| `--allowedTools` | Permission scope | `""` (none) | `"Read,Grep,Glob"` |

### Dependency Removal

```bash
# Remove from go.mod
go mod edit -droprequire github.com/anthropics/anthropic-sdk-go
go mod tidy
```

Delete `internal/llm/client.go`, create `internal/llm/claude_code.go`.

---

## 2. Badge Generation

### Recommendation: Shields.io Static URLs

**DO NOT generate SVG locally.** Use shields.io URLs in README output. This is simpler, requires no dependencies, and produces industry-standard badges.

### URL Format

```
https://img.shields.io/badge/{label}-{message}-{color}?{params}
```

**Encoding rules:**
- Space: `_` or `%20`
- Underscore: `__`
- Dash: `--`

**Source:** [Shields.io Static Badge](https://shields.io/badges)

### Implementation

```go
// internal/output/badge.go
package output

import (
    "fmt"
    "net/url"
)

// BadgeFormat specifies the output format for badges
type BadgeFormat string

const (
    BadgeFormatURL      BadgeFormat = "url"
    BadgeFormatMarkdown BadgeFormat = "markdown"
    BadgeFormatHTML     BadgeFormat = "html"
)

// GenerateBadgeURL creates a shields.io badge URL for the ARS score
func GenerateBadgeURL(score float64, tier string, style string) string {
    color := tierToColor(tier)
    label := "ARS"
    // URL-encode the score with slash: "8.5/10" -> "8.5%2F10"
    message := url.PathEscape(fmt.Sprintf("%.1f/10", score))

    baseURL := fmt.Sprintf("https://img.shields.io/badge/%s-%s-%s", label, message, color)
    if style != "" && style != "flat" {
        baseURL += "?style=" + style
    }
    return baseURL
}

// GenerateTierBadgeURL creates a badge showing the tier name
func GenerateTierBadgeURL(tier string, style string) string {
    color := tierToColor(tier)
    label := "ARS"
    // Escape dashes in tier name: "Agent-Ready" -> "Agent--Ready"
    escapedTier := strings.ReplaceAll(tier, "-", "--")

    baseURL := fmt.Sprintf("https://img.shields.io/badge/%s-%s-%s", label, escapedTier, color)
    if style != "" && style != "flat" {
        baseURL += "?style=" + style
    }
    return baseURL
}

func tierToColor(tier string) string {
    switch tier {
    case "Agent-Ready":
        return "brightgreen"
    case "Agent-Assisted":
        return "green"
    case "Agent-Limited":
        return "yellow"
    case "Agent-Hostile":
        return "red"
    default:
        return "lightgrey"
    }
}

// FormatBadge returns the badge in the specified format
func FormatBadge(url string, format BadgeFormat, altText string) string {
    switch format {
    case BadgeFormatMarkdown:
        return fmt.Sprintf("![%s](%s)", altText, url)
    case BadgeFormatHTML:
        return fmt.Sprintf(`<img src="%s" alt="%s">`, url, altText)
    default:
        return url
    }
}
```

### Badge Examples

| Score | Tier | URL |
|-------|------|-----|
| 8.5 | Agent-Ready | `https://img.shields.io/badge/ARS-8.5%2F10-brightgreen` |
| 6.2 | Agent-Assisted | `https://img.shields.io/badge/ARS-6.2%2F10-green` |
| 4.8 | Agent-Limited | `https://img.shields.io/badge/ARS-4.8%2F10-yellow` |
| 3.1 | Agent-Hostile | `https://img.shields.io/badge/ARS-3.1%2F10-red` |

### Style Options

| Style | Best For | Example |
|-------|----------|---------|
| `flat` (default) | General use | ![flat](https://img.shields.io/badge/ARS-8.5-brightgreen) |
| `flat-square` | Modern repos | Add `?style=flat-square` |
| `for-the-badge` | Feature highlight | Add `?style=for-the-badge` |

### CLI Flag

Add `--badge` flag to `scan` command:

```go
// cmd/scan.go
var badgeFormat string
scanCmd.Flags().StringVar(&badgeFormat, "badge", "", "Output badge: url, markdown, or html")
```

**No new dependencies required.**

---

## 3. HTML Collapsible Sections

### Recommendation: Native `<details>/<summary>` Elements

**NO JavaScript required.** Use HTML5 semantic elements with CSS styling.

### Browser Support (Verified)

Supported in all modern browsers since January 2020:
- Chrome 12+
- Firefox 49+
- Safari 6.1+
- Edge 79+

**Source:** [Can I Use: details](https://caniuse.com/details)

### Template Changes

Modify `internal/output/templates/report.html`:

```html
<!-- Before: non-collapsible -->
<div class="category">
    <h2>{{.DisplayName}} <span class="cat-score">{{printf "%.1f" .Score}}/10</span></h2>
    <table class="metric-table">...</table>
</div>

<!-- After: collapsible -->
<details class="category-details" {{if lt .Score 6.0}}open{{end}}>
    <summary>
        <span class="category-header">
            {{.DisplayName}}
            <span class="cat-score score-{{.ScoreClass}}">{{printf "%.1f" .Score}}/10</span>
        </span>
    </summary>
    <div class="category-content">
        <table class="metric-table">
            <thead>
                <tr><th>Metric</th><th>Value</th><th>Score</th><th>Weight</th></tr>
            </thead>
            <tbody>
                {{range .SubScores}}
                <tr>
                    <td>{{.DisplayName}}</td>
                    <td>{{.FormattedValue}}</td>
                    <td class="score-cell score-{{.ScoreClass}}">{{printf "%.1f" .Score}}</td>
                    <td>{{printf "%.0f" .WeightPct}}%</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{if .ImpactDescription}}<p class="impact">{{.ImpactDescription}}</p>{{end}}
    </div>
</details>
```

### CSS Additions

Add to `internal/output/templates/styles.css`:

```css
/* Collapsible category sections */
details.category-details {
    border: 1px solid #ddd;
    border-radius: 4px;
    margin-bottom: 1rem;
    background: #fff;
}

details.category-details summary {
    padding: 1rem;
    cursor: pointer;
    list-style: none;
    background: #f8f9fa;
    border-radius: 4px 4px 0 0;
}

details.category-details summary::-webkit-details-marker {
    display: none;
}

details.category-details summary::before {
    content: '\25B6';  /* Right triangle */
    display: inline-block;
    margin-right: 0.5rem;
    transition: transform 0.2s ease;
    font-size: 0.75rem;
}

details[open].category-details summary::before {
    transform: rotate(90deg);  /* Point down when open */
}

details[open].category-details summary {
    border-bottom: 1px solid #ddd;
    border-radius: 4px 4px 0 0;
}

details.category-details .category-content {
    padding: 1rem;
}

details.category-details .category-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
}

/* Focus styles for keyboard accessibility */
details.category-details summary:focus {
    outline: 2px solid #007bff;
    outline-offset: 2px;
}
```

### Smart Defaults

Categories with scores below 6.0 start **expanded** (need attention). High-scoring categories start **collapsed** (working well).

```html
{{if lt .Score 6.0}}open{{end}}
```

### Accessibility Benefits

- Keyboard accessible (Enter/Space to toggle)
- Screen reader announces expanded/collapsed state
- Works without CSS (graceful degradation)
- Content indexed by search engines (not hidden from DOM)

**No new dependencies required.**

---

## What NOT to Add

### DO NOT Add: SVG Generation Libraries

| Library | Why Avoid |
|---------|-----------|
| `github.com/ajstarks/svgo` | Overkill for badges |
| `github.com/fogleman/gg` | Image generation, wrong tool |
| Any SVG library | Shields.io URLs are simpler, reliable, standard |

**Shields.io advantages:**
- Zero code maintenance
- Always up-to-date styling
- Industry recognition
- No binary size increase

### DO NOT Add: JavaScript for Collapsibles

| Approach | Why Avoid |
|----------|-----------|
| Alpine.js | Unnecessary dependency |
| Inline `<script>` | Adds complexity, CSP concerns |
| jQuery | Massive overkill |
| Any JS framework | `<details>` does the same thing natively |

**Native `<details>` advantages:**
- Zero JavaScript
- Built-in accessibility
- Works offline
- No load-time overhead

### DO NOT Keep: Anthropic SDK

**Remove:** `github.com/anthropics/anthropic-sdk-go`

| Reason | Details |
|--------|---------|
| Auth duplication | Claude Code handles auth already |
| Cost tracking | Claude Code has built-in cost tracking |
| Retry logic | Claude Code handles retries |
| Dependency bloat | ~1 MB binary size freed |
| Maintenance | One less thing to update |

---

## Integration Points

| Existing | v0.0.3 Change | Integration |
|----------|---------------|-------------|
| `os/exec` (stdlib) | Extended for C4 | Same pattern as C7 |
| `html/template` | Add `<details>` | Template syntax only |
| `embed` | CSS additions | Update `styles.css` |
| `encoding/json` | Parse CLI output | Already in use |
| `fmt.Sprintf` | Badge URLs | Simple formatting |
| `net/url` | Badge encoding | stdlib, already available |

**Net dependency change: -1 (remove Anthropic SDK)**

---

## Migration Checklist

```bash
# 1. Remove Anthropic SDK
go mod edit -droprequire github.com/anthropics/anthropic-sdk-go
go mod tidy

# 2. Delete old LLM client
rm internal/llm/client.go

# 3. Create new Claude Code wrapper
# (implement internal/llm/claude_code.go)

# 4. Add badge generation
# (implement internal/output/badge.go)

# 5. Update HTML template
# (modify internal/output/templates/report.html)

# 6. Add CSS for collapsibles
# (update internal/output/templates/styles.css)

# 7. Verify no build errors
go build ./...

# 8. Run tests
go test ./...
```

---

## Sources

### Claude Code Headless
- [Claude Code Headless Documentation](https://code.claude.com/docs/en/headless) -- Official headless mode docs (HIGH confidence)
- [Claude Code CLI Reference](https://code.claude.com/docs/en/cli-reference) -- All CLI flags (HIGH confidence)

### Shields.io Badges
- [Shields.io Static Badge](https://shields.io/badges) -- Badge URL format (HIGH confidence)
- [Shields.io Endpoint Badge](https://shields.io/badges/endpoint-badge) -- JSON endpoint schema (HIGH confidence)
- [Shields.io GitHub](https://github.com/badges/shields) -- Source and documentation (HIGH confidence)

### HTML Collapsible Sections
- [Can I Use: details](https://caniuse.com/details) -- Browser support (HIGH confidence)
- [MDN: details element](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/details) -- Semantic reference (HIGH confidence)

---
*Stack research for: ARS v0.0.3 -- Claude Code unification, badges, HTML collapsibles*
*Researched: 2026-02-03*

---

# Stack Research: Academic Citations (v0.0.4 Milestone)

**Domain:** Citation systems for software engineering documentation
**Researched:** 2026-02-04
**Confidence:** HIGH

## Executive Summary

This research investigates tools, formats, and approaches for adding academic citations to ARS's existing metric descriptions. The goal is credibility through research backing for engineering leaders, not academic paper density. Given the existing infrastructure (Go codebase, HTML reports with inline CSS, 33 metrics needing ~3-5 citations each), the recommendation is a **minimal, manual approach** that leverages existing patterns.

## Recommended Stack

### Core Approach: Manual (Author, Year) Format

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Manual inline citations | N/A | Inline `(Author, Year)` references | Already implemented in descriptions.go; no new dependencies needed |
| Go struct `Citation` | N/A | Reference data storage | Already exists in citations.go; extend for metric-level granularity |
| CSS `.citation` class | N/A | Visual styling of inline citations | Already exists in styles.css; ready for use |

**Rationale:** The project already has all infrastructure needed. The `Citation` struct, CSS styling, and HTML template patterns are in place. Adding citations is a content task, not a technology task.

### Citation Format: APA-Style (Author, Year)

| Aspect | Recommendation | Why |
|--------|----------------|-----|
| Inline format | `(Author, Year)` or `(Author et al., Year)` | Industry standard for technical documentation; already used in descriptions.go |
| Reference format | Author, Year. *Title*. URL | Matches existing citations.go pattern |
| Multiple authors | `et al.` for 3+ authors | Standard practice, reduces clutter |

**Example (already in codebase):**
```html
<span class="citation">(Borg et al., 2026)</span>
```

### URL Verification Tools

| Tool | Purpose | When to Use |
|------|---------|-------------|
| Manual HTTP check | Verify URLs resolve | During citation addition (one-time) |
| `curl -I [URL]` | Quick status code check | Command-line verification |
| Browser verification | Confirm content matches citation | Final validation |

**Rationale:** With ~100-150 total citations (33 metrics x 3-5 each), manual verification is faster than setting up automated tooling. Lychee or similar tools are overkill for a one-time batch verification.

### URL Permanence Strategy

| Source Type | URL Strategy | Why |
|-------------|--------------|-----|
| Academic papers | Prefer DOI links (`doi.org/...`) | DOIs are permanent; URLs change |
| Books | Publisher page or ISBN lookup | More stable than retailer links |
| Blog posts/docs | Archive.org backup + original | Protection against link rot |
| Official docs | Use versioned URLs when available | `/docs/v1.0/` over `/docs/latest/` |

**Example DOI format:**
```
https://doi.org/10.1145/361598.361623
```

## What NOT to Add

| Avoid | Why | Alternative |
|-------|-----|-------------|
| BibTeX/CSL tooling | Over-engineering for 150 citations | Manual Go structs |
| Zotero/Mendeley integration | Adds external dependency for one-time task | Manual entry |
| Automated link checkers in CI | Overkill; links checked once during addition | Manual verification |
| Footnote-based citations | Engineering audience expects inline | (Author, Year) format |
| Hover tooltips for citations | Adds JavaScript complexity; minimal UX benefit | Static inline text |
| Per-metric citation structs | Over-complicates data model | Inline in HTML content |

**Key Principle:** The existing `descriptions.go` pattern embeds citation markup directly in the `Detailed` HTML field. This is appropriate for ~150 citations that change rarely.

## Existing Infrastructure (No Changes Needed)

### Citation Data Structure (citations.go)
```go
type Citation struct {
    Category    string
    Title       string
    Authors     string
    Year        int
    URL         string
    Description string
}
```

### CSS Styling (styles.css)
```css
.citation {
  color: var(--color-muted);
  font-style: normal;
}
```

### HTML Template Pattern (report.html)
```html
{{range .Citations}}
<li><a href="{{.URL}}" target="_blank" rel="noopener">{{.Title}}</a>
    ({{.Authors}}, {{.Year}}) - {{.Description}}</li>
{{end}}
```

### Inline Citation Pattern (descriptions.go)
```html
<span class="citation">(McCabe, 1976)</span>
```

## Implementation Approach

### Phase 1: Foundational Sources (Pre-2021)

Sources like McCabe (1976), Fowler (1999), Parnas (1972) are well-established. Most URLs are stable institutional or publisher links.

**Verification approach:**
1. Check URL returns 200 status
2. Confirm page content matches citation
3. For academic papers, prefer DOI when available

### Phase 2: AI/Agent Era Sources (2021+)

Sources like Borg et al. (2026) are newer and may have less stable URLs.

**Verification approach:**
1. Check arXiv/DOI links (highly stable)
2. For conference papers, use ACM/IEEE digital library DOIs
3. For blog posts, consider archive.org snapshot

### Reference Section Strategy

Current pattern: Per-category references at category level.

**Recommendation:** Keep this pattern. Each category section already has:
```html
{{if .Citations}}
<div class="category-citations">
    <h4>References</h4>
    <ul>{{range .Citations}}...{{end}}</ul>
</div>
{{end}}
```

**Enhancement:** Inline citations in metric descriptions link conceptually to the category reference section. No hyperlink anchors needed for ~3-5 citations per metric.

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Manual (Author, Year) | IEEE numeric [1] | Never for this project; numeric requires back-reference which is poor UX |
| Inline in HTML | Separate citations per metric | If citations.go grows beyond 200 entries |
| Manual URL check | Lychee CI integration | If project adds many links frequently |
| DOI links | Direct publisher URLs | Only when DOI unavailable |

## Version Compatibility

| Component | Version | Notes |
|-----------|---------|-------|
| Go html/template | Go 1.21+ | Standard library, no compatibility concerns |
| CSS `.citation` class | CSS3 | Universal browser support |
| DOI resolver (doi.org) | N/A | International DOI Foundation, stable since 2000 |

## Citation Count Estimate

| Category | Metrics | Est. Citations/Metric | Total |
|----------|---------|----------------------|-------|
| C1: Code Health | 6 | 4 | 24 |
| C2: Semantics | 5 | 3 | 15 |
| C3: Architecture | 5 | 3 | 15 |
| C4: Documentation | 7 | 3 | 21 |
| C5: Temporal | 5 | 3 | 15 |
| C6: Testing | 5 | 3 | 15 |
| **Total** | **33** | **~3.2 avg** | **~105** |

This is a manageable size for manual curation. No tooling investment justified.

## Verification Protocol

For each citation added:

1. **URL Check:** `curl -I [URL]` returns 200 or 301/302 to valid destination
2. **Content Check:** Page title/abstract matches citation
3. **Author Check:** Listed authors match citation
4. **Year Check:** Publication year correct
5. **DOI Preference:** If academic paper, use `doi.org` URL when available

## Citation Format Reference

### Standard Inline Citation
```html
<span class="citation">(Author, Year)</span>
```

### Multiple Authors (3+)
```html
<span class="citation">(Author et al., Year)</span>
```

### Two Authors
```html
<span class="citation">(Author & Author, Year)</span>
```

### Multiple Citations
```html
<span class="citation">(Author, Year; Other, Year)</span>
```

## Sources

- [Purdue OWL - DOIs vs URLs](https://owl.purdue.edu/owl/research_and_citation/conducting_research/internet_references/urls_vs_dois.html) - DOI permanence guidance (HIGH confidence)
- [QuillBot - APA vs Chicago Author-Date](https://quillbot.com/blog/frequently-asked-questions/whats-the-difference-between-apa-and-chicago-author-date-citations/) - Citation format comparison (MEDIUM confidence)
- [BibGuru - Citation Style for Computer Science](https://www.bibguru.com/blog/citation-style-for-computer-science/) - Engineering citation standards (MEDIUM confidence)
- [Markdown Citations Guide](https://blog.markdowntools.com/posts/markdown-citations-and-references-guide) - Markdown citation patterns (MEDIUM confidence)
- [Lychee Link Checker](https://github.com/lycheeverse/lychee) - Link verification tool (evaluated, not recommended for this scope)
- [W3Schools CSS Tooltip](https://www.w3schools.com/css/css_tooltip.asp) - CSS tooltip patterns (evaluated, not recommended)

---
*Stack research for: Academic citation implementation in ARS HTML reports*
*Researched: 2026-02-04*

---

# Stack Research: C7 Debug Tooling and Heuristic Testing (v0.0.5 Milestone)

**Domain:** Debug infrastructure for heuristic-based agent evaluation scoring
**Researched:** 2026-02-06
**Confidence:** HIGH (all recommendations use Go stdlib; verified against codebase)

## Executive Summary

v0.0.5 requires **zero new Go dependencies**. The debug tooling for C7 heuristic scoring uses exclusively Go standard library packages that are already imported or trivially available. The four needs -- flag handling, response logging, heuristic testing, and response capture/replay -- each map cleanly to existing stdlib primitives and established Go testing patterns.

The key architectural insight: the `metrics.Executor` interface (already defined in `internal/agent/metrics/metric.go`) is the single seam through which all debug and test infrastructure plugs in. No new abstractions needed.

---

## 1. Debug Mode Flag: `--debug-c7`

### Recommended: Cobra Bool Flag + `io.Writer` Injection

**Use `spf13/cobra` (already in project) for flag registration. Use `io.Writer` (stdlib) for debug output routing.**

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `spf13/cobra` | v1.10.2 (existing) | `--debug-c7` flag registration | Already used for all CLI flags in `cmd/scan.go` |
| `io.Writer` | stdlib | Debug output destination | Already the pattern for `Pipeline.writer`; consistent |
| `io.Discard` | stdlib | No-op writer when debug disabled | Zero-cost disable; no conditional checks needed |
| `os.Stderr` | stdlib | Debug output target | Separates debug output from scan results on stdout |

### Why NOT `log/slog`

Go 1.21+ provides `log/slog` for structured logging. It is the correct choice for production services. It is the **wrong** choice here because:

1. **C7 debug output is investigative, not operational.** The developer wants to see "what did the agent say?" and "how did the heuristic score it?" -- not structured key-value log lines.
2. **Output format is human-readable prose + JSON blobs.** `slog` would add `level=DEBUG msg=...` framing that obscures the actual response content.
3. **The project has zero logging infrastructure today.** Adding `slog` for one flag is scope creep. The `io.Writer` pattern is already the project's convention (see `Pipeline.writer`, `C7Progress`).
4. **Debug mode is off by default.** When disabled, `io.Discard` is cheaper than a disabled `slog.Logger`.

### Why NOT a Custom `DebugLogger` Type

A type like `type DebugLogger struct { enabled bool; writer io.Writer }` with methods like `.Logf()` is over-engineering. The raw `io.Writer` is sufficient because:

- `fmt.Fprintf(w, ...)` does everything needed
- No log levels required (it is all debug-level)
- No timestamps needed (responses already have duration)
- No structured fields needed (responses are already in `MetricResult` structs)

### Integration Pattern

```go
// cmd/scan.go -- add flag
var debugC7 bool
scanCmd.Flags().BoolVar(&debugC7, "debug-c7", false, "dump C7 agent responses and heuristic scores to stderr")

// Pipeline threading -- pass debug writer through
var debugWriter io.Writer = io.Discard
if debugC7 {
    debugWriter = os.Stderr
}
// Pass debugWriter to C7 analyzer or parallel runner
```

The flag follows the existing pattern in `cmd/scan.go` (lines 16-24) where `enableC7`, `noLLM`, `jsonOutput`, etc. are declared as package-level `var` and registered in `init()`.

### What Gets Written to Debug Output

For each metric sample:
1. **Prompt sent** -- what the agent was asked
2. **Raw response** -- exactly what the agent returned
3. **Heuristic breakdown** -- which indicators matched, score before/after clamping
4. **Final score** -- the 1-10 score assigned

Format: plain text with clear delimiters (not JSON, not structured logs). Example:

```
=== M2: Code Behavior Comprehension ===
--- Sample 1: internal/scoring/config.go ---
PROMPT: Read the file at internal/scoring/config.go and explain...
RESPONSE (247 words):
  The file defines a ScoringConfig struct that holds...
HEURISTIC BREAKDOWN:
  Base score: 5
  +1 "returns" found
  +1 "error" found
  +1 "handling" found
  +1 "validates" found
  +1 word count > 100 (247 words)
  +1 word count > 200 (247 words)
  Final: 10 (clamped from 11)
---
```

---

## 2. Response Inspection/Logging

### Recommended: `fmt.Fprintf` to Injected `io.Writer`

**No logging library. No `slog`. Just `fmt.Fprintf`.**

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `fmt.Fprintf` | stdlib | Write debug info to writer | Already used throughout codebase; zero learning curve |
| `io.Writer` | stdlib | Abstraction for output destination | Already the project convention |
| `encoding/json.MarshalIndent` | stdlib | Pretty-print JSON responses | For when raw CLI JSON responses need inspection |
| `strings.Builder` | stdlib | Efficient string building for debug blocks | Avoids repeated small writes |

### Implementation Location

The debug output belongs **inside the metric Execute methods**, not in the parallel runner. Each metric's `Execute()` method has access to:
- The prompt it constructed
- The raw response from the executor
- The heuristic scoring result

The `io.Writer` for debug output should be threaded through the `Executor` interface or passed as a field on each metric. The cleaner approach: add an optional `DebugWriter io.Writer` field to the concrete metric types (`M2Comprehension`, `M3Navigation`, `M4Identifiers`).

```go
type M2Comprehension struct {
    sampleCount int
    timeout     time.Duration
    DebugWriter io.Writer // nil = no debug output
}
```

This is cleaner than modifying the `Executor` interface because:
- Debug output is about the **scoring heuristics**, not the executor
- The `Executor` interface should stay minimal (it has one method)
- Each metric knows its own scoring internals

### What NOT to Log

- Do NOT log the full file content (it is already on disk)
- Do NOT log timestamps (the `MetricResult.Duration` already tracks timing)
- Do NOT log system info (irrelevant to heuristic debugging)
- Do NOT log to files (stderr is sufficient; redirect with `2>debug.log`)

---

## 3. Testing Heuristic Scoring Functions

### Recommended: Table-Driven Tests with `testing` Package

**The project already has the right pattern.** See `internal/agent/metrics/metric_test.go` lines 329-487 which test `scoreComprehensionResponse`, `scoreNavigationResponse`, `scoreIdentifierResponse`, and `scoreDocumentationResponse` with table-driven tests using min/max score ranges.

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `testing` | stdlib | Test framework | Already used everywhere; no alternative needed |
| `testing.TB` | stdlib | Test/benchmark interface | Enables both `*testing.T` and `*testing.B` |

### Current Test Pattern (Already Exists)

```go
func TestM2_ScoreComprehensionResponse(t *testing.T) {
    m := NewM2Comprehension().(*M2Comprehension)
    tests := []struct {
        name     string
        response string
        minScore int
        maxScore int
    }{...}
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            score := m.scoreComprehensionResponse(tc.response)
            if score < tc.minScore || score > tc.maxScore {
                t.Errorf(...)
            }
        })
    }
}
```

### What to ADD: Real Response Corpus Tests

The existing tests use fabricated strings. The milestone needs tests against **real Claude responses** captured from actual runs. This requires:

1. **Golden files** in `testdata/` containing real agent responses
2. **Exact score assertions** (not ranges) once heuristics are fixed
3. **Regression tests** to prevent score drift

### Golden File Pattern for Heuristic Tests

```go
// internal/agent/metrics/testdata/m2_responses/
//   good_comprehension.txt     -- real response, expected score 8
//   shallow_comprehension.txt  -- real response, expected score 4
//   failed_comprehension.txt   -- real response, expected score 1

func TestM2_ScoreComprehensionResponse_RealResponses(t *testing.T) {
    m := NewM2ComprehensionMetric()

    tests := []struct {
        file     string
        expected int
    }{
        {"testdata/m2_responses/good_comprehension.txt", 8},
        {"testdata/m2_responses/shallow_comprehension.txt", 4},
        {"testdata/m2_responses/failed_comprehension.txt", 1},
    }

    for _, tc := range tests {
        t.Run(tc.file, func(t *testing.T) {
            response, err := os.ReadFile(tc.file)
            if err != nil {
                t.Fatalf("read golden file: %v", err)
            }
            score := m.scoreComprehensionResponse(string(response))
            if score != tc.expected {
                t.Errorf("score = %d, want %d", score, tc.expected)
            }
        })
    }
}
```

This uses Go's built-in `testdata/` convention (the `go` toolchain automatically ignores `testdata/` directories in builds). No golden file library needed -- `os.ReadFile` and direct comparison is sufficient.

### Why NOT `goldie` or Other Golden File Libraries

Libraries like `github.com/sebdah/goldie` add:
- Auto-update with `-update` flag
- Diff output on mismatch

For this use case, these are unnecessary because:
- Response files are **manually curated**, not auto-generated
- Expected scores are **explicitly declared**, not file-content-matches
- The "golden" file is the **input** (the response), not the **output**

Standard `os.ReadFile` + table-driven tests is the right fit.

### What NOT to Test

- Do NOT test `SelectSamples()` with mocked file systems -- it is already tested
- Do NOT test the `Executor` interface -- it is for integration tests
- Do NOT write tests that require Claude CLI -- those are integration tests gated by `t.Skip`

---

## 4. Capturing and Replaying Agent Responses

### Recommended: Mock Executor + `testdata/` Files

**The `metrics.Executor` interface is already the replay seam.** Implement a `MockExecutor` that returns canned responses from files.

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `metrics.Executor` interface | existing | Seam for mock injection | Already defined; no changes needed |
| `os.ReadFile` | stdlib | Load response fixtures | Reads `testdata/` golden files |
| `testdata/` convention | Go standard | Test fixture storage | Go toolchain ignores in builds automatically |
| `encoding/json` | stdlib | Serialize/deserialize response fixtures | Already in use for CLI response parsing |

### MockExecutor Implementation

```go
// internal/agent/metrics/mock_executor_test.go
package metrics

import (
    "context"
    "fmt"
    "os"
    "time"
)

// MockExecutor replays canned responses for deterministic testing.
type MockExecutor struct {
    // Map from prompt substring to response file path
    Responses map[string]string
    // Fallback response if no match found
    FallbackResponse string
    // Optional: simulate errors
    ErrorOnPrompt map[string]error
    // Track calls for assertions
    Calls []MockCall
}

type MockCall struct {
    WorkDir string
    Prompt  string
    Tools   string
    Timeout time.Duration
}

func (m *MockExecutor) ExecutePrompt(ctx context.Context, workDir, prompt, tools string, timeout time.Duration) (string, error) {
    m.Calls = append(m.Calls, MockCall{
        WorkDir: workDir,
        Prompt:  prompt,
        Tools:   tools,
        Timeout: timeout,
    })

    // Check for error simulation
    for substr, err := range m.ErrorOnPrompt {
        if strings.Contains(prompt, substr) {
            return "", err
        }
    }

    // Find matching response
    for substr, filePath := range m.Responses {
        if strings.Contains(prompt, substr) {
            data, err := os.ReadFile(filePath)
            if err != nil {
                return "", fmt.Errorf("mock: read response file: %w", err)
            }
            return string(data), nil
        }
    }

    return m.FallbackResponse, nil
}
```

### Capture Workflow

The `--debug-c7` flag (from section 1) serves double duty as the capture mechanism:

```bash
# Run with debug to capture responses
go run . scan --enable-c7 --debug-c7 ./path/to/project 2>c7_debug.log

# Extract individual responses from debug log
# (manual: copy paste the RESPONSE sections into testdata/ files)
```

No automated capture tool needed. The debug log contains the full responses in a readable format. The developer manually copies interesting responses into `testdata/` golden files with appropriate names.

### Why NOT Automated Capture/Replay (httptest, VCR, etc.)

| Tool | Why Not |
|------|---------|
| `net/http/httptest` | Claude CLI is subprocess-based, not HTTP |
| `go-vcr` / `httpreplay` | Same reason -- no HTTP to record |
| Custom subprocess recording | Over-engineering; responses are just text strings |
| Serialized `TaskResult` files | Too coupled to internal types; plain text is simpler |

The agent responses are **plain text strings**. There is no protocol, no headers, no request/response pairing to record. A text file in `testdata/` is the simplest and most maintainable format.

### Response File Organization

```
internal/agent/metrics/testdata/
    m2_responses/
        comprehensive_explanation.txt      # Good response: covers all paths
        shallow_explanation.txt            # Weak response: surface-level only
        uncertain_explanation.txt          # Hedging response: "might", "probably"
        empty_response.txt                # Edge case: empty string
    m3_responses/
        full_trace.txt                    # Good: complete cross-file trace
        partial_trace.txt                 # Partial: only direct deps
        failed_trace.txt                  # Bad: "cannot find file"
    m4_responses/
        accurate_interpretation.txt       # Good: correct inference
        vague_interpretation.txt          # Weak: generic guess
        wrong_interpretation.txt          # Bad: misinterprets identifier
```

Each file contains the **exact text** that the Claude CLI returned in its `result` field. No JSON wrapping, no metadata -- just the response string.

---

## Recommended Stack Summary

### New Technologies: ZERO

| Component | Technology | Status |
|-----------|-----------|--------|
| Flag handling | `spf13/cobra` BoolVar | Already in project |
| Debug output | `fmt.Fprintf` to `io.Writer` | stdlib, already used |
| Output routing | `io.Discard` / `os.Stderr` | stdlib |
| Test framework | `testing` package | stdlib, already used |
| Golden files | `os.ReadFile` + `testdata/` | stdlib + Go convention |
| Mock executor | Implements `metrics.Executor` | Interface already defined |
| JSON formatting | `encoding/json.MarshalIndent` | stdlib, already imported |
| String building | `strings.Builder` | stdlib |

**Net dependency change: 0. No new imports. No new go.mod entries.**

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `log/slog` | Over-engineering for debug dump; wrong output format | `fmt.Fprintf` to `io.Writer` |
| `logrus` / `zap` / `zerolog` | External dependency; structured logging is wrong paradigm | `fmt.Fprintf` to `io.Writer` |
| `github.com/sebdah/goldie` | Golden file lib adds auto-update feature we do not need | `os.ReadFile` + explicit assertions |
| `github.com/stretchr/testify` | Assertion library adds dependency for convenience | stdlib `testing` with `t.Errorf` |
| `github.com/golang/mock` / `gomock` | Mock generation tool; interface has 1 method | Hand-written `MockExecutor` |
| Delve debugger integration | Runtime debugger is for stepping, not output capture | `--debug-c7` flag |
| Custom replay framework | Responses are strings, not protocol messages | `testdata/` text files |
| `httptest.Server` | No HTTP involved; CLI is subprocess-based | `MockExecutor` |
| Per-test temporary directories | `testdata/` is persistent and version-controlled | `testdata/` convention |

---

## Integration with Existing Codebase

### Files to Modify

| File | Change | Why |
|------|--------|-----|
| `cmd/scan.go` | Add `--debug-c7` bool flag | Same pattern as `--enable-c7` on line 21 |
| `internal/pipeline/pipeline.go` | Thread `debugWriter` to C7 analyzer | Pipeline already threads `verbose`, `writer`, `evaluator` |
| `internal/analyzer/c7_agent/agent.go` | Accept `debugWriter`, pass to metrics | Currently creates metrics without debug context |
| `internal/agent/parallel.go` | Pass `debugWriter` to metric execution | Currently no debug output path |
| `internal/agent/metrics/m2_comprehension.go` | Add `DebugWriter` field, write during `Execute` | Scoring logic is in `scoreComprehensionResponse` |
| `internal/agent/metrics/m3_navigation.go` | Add `DebugWriter` field, write during `Execute` | Same pattern as M2 |
| `internal/agent/metrics/m4_identifiers.go` | Add `DebugWriter` field, write during `Execute` | Same pattern as M2 |

### Files to Create

| File | Purpose |
|------|---------|
| `internal/agent/metrics/mock_executor_test.go` | MockExecutor for deterministic tests |
| `internal/agent/metrics/testdata/m2_responses/*.txt` | Real M2 response fixtures |
| `internal/agent/metrics/testdata/m3_responses/*.txt` | Real M3 response fixtures |
| `internal/agent/metrics/testdata/m4_responses/*.txt` | Real M4 response fixtures |

### Dependency Chain

```
cmd/scan.go (--debug-c7 flag)
  -> pipeline.Pipeline (debugWriter field)
    -> c7_agent.C7Analyzer (debugWriter field)
      -> agent.RunMetricsParallel (debugWriter param)
        -> metrics.M2Comprehension.DebugWriter
        -> metrics.M3Navigation.DebugWriter
        -> metrics.M4Identifiers.DebugWriter
```

This follows the existing pattern where `enableC7` flows:
```
cmd/scan.go (--enable-c7 flag)
  -> pipeline.Pipeline.SetC7Enabled()
    -> c7_agent.C7Analyzer.Enable()
      -> agent.RunMetricsParallel (via evaluator)
```

---

## Sources

### Go Standard Library
- [Go slog package](https://pkg.go.dev/log/slog) -- Evaluated and rejected for this use case (HIGH confidence)
- [Go Structured Logging with slog](https://go.dev/blog/slog) -- Official Go blog on slog design philosophy (HIGH confidence)
- [Go testing package](https://pkg.go.dev/testing) -- Standard test framework (HIGH confidence)
- [Go testdata convention](https://pkg.go.dev/cmd/go#hdr-Test_packages) -- `testdata/` directory semantics (HIGH confidence)

### Testing Patterns
- [File-driven testing in Go](https://eli.thegreenplace.net/2022/file-driven-testing-in-go/) -- Golden file patterns without libraries (MEDIUM confidence)
- [Testing with golden files in Go](https://medium.com/soon-london/testing-with-golden-files-in-go-7fccc71c43d3) -- Community golden file patterns (MEDIUM confidence)
- [Advanced unit testing patterns in Go](https://blog.logrocket.com/advanced-unit-testing-patterns-go/) -- Table-driven test patterns (MEDIUM confidence)

### Existing Codebase (Primary Source)
- `internal/agent/metrics/metric.go` -- `Executor` interface definition (authoritative)
- `internal/agent/metrics/metric_test.go` -- Existing heuristic scoring tests (authoritative)
- `cmd/scan.go` -- Cobra flag registration pattern (authoritative)
- `internal/pipeline/pipeline.go` -- Pipeline dependency threading pattern (authoritative)

---
*Stack research for: ARS v0.0.5 -- C7 debug tooling and heuristic testing*
*Researched: 2026-02-06*

---

# Stack Research: Interactive HTML Report Enhancements (v0.0.5 Milestone)

**Domain:** Modal UI, syntax highlighting, and copy-to-clipboard for self-contained HTML reports
**Researched:** 2026-02-06
**Confidence:** HIGH (verified against MDN, Can I Use, and codebase inspection)

## Executive Summary

Adding modal overlays, syntax highlighting, and copy-to-clipboard to the existing HTML report requires **zero new Go dependencies** and **zero external JavaScript libraries**. The three capabilities map to well-supported web platform features:

1. **Modal overlays:** Native HTML `<dialog>` element with `showModal()` -- 95.81% global browser support, built-in focus trap, Escape dismissal, and `::backdrop` styling.
2. **Syntax highlighting:** Custom ~30-line inline JavaScript function for JSON/command highlighting -- purpose-built for our narrow use case (JSON objects and shell commands only).
3. **Copy-to-clipboard:** `navigator.clipboard.writeText()` API -- 95.68% global support, already partially used in the badge section of the existing report.

The report template (`internal/output/templates/report.html`) already contains inline JavaScript for expand/collapse and a `navigator.clipboard.writeText()` call for badge copying. These additions extend the existing pattern with minimal incremental complexity.

**Net Go dependency change: 0. Net external library change: 0.**

---

## 1. Modal Overlay UI

### Recommendation: Native HTML `<dialog>` Element with `showModal()`

**Use `<dialog>` with JavaScript `.showModal()` / `.close()` calls.** Do NOT use CSS-only modal hacks or the Invoker Commands API.

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| HTML `<dialog>` element | HTML5.2+ | Native modal container | 95.81% global support; built-in accessibility, focus trap, backdrop, Escape key handling |
| `HTMLDialogElement.showModal()` | Web API | Open dialog as modal | Moves to top layer, makes background inert, traps focus |
| `HTMLDialogElement.close()` | Web API | Close dialog | Fires close event, restores focus to trigger element |
| CSS `::backdrop` | CSS Selectors L4 | Style the overlay behind modal | Native pseudo-element, no extra DOM nodes needed |

### Browser Support (Verified via Can I Use)

| Browser | Minimum Version | Release Date |
|---------|----------------|--------------|
| Chrome | 37+ | 2014 |
| Firefox | 98+ | March 2022 |
| Safari | 15.4+ | March 2022 |
| Edge | 79+ | 2020 |

**Global support: 95.81%** -- Baseline Widely Available since March 2022.

**Source:** [Can I Use: dialog](https://caniuse.com/dialog) (HIGH confidence)

### Why NOT CSS-Only Modals (:target or checkbox hack)

| Approach | Why Avoid |
|----------|-----------|
| `:target` pseudo-class | Modifies URL hash; breaks back button; no Escape key dismissal; no focus trap |
| Checkbox hack | Requires invisible `<input>` elements; no semantic meaning; no accessibility support |
| Custom `<div>` modal with JS | Reinvents focus trap, Escape handling, backdrop, inert background -- all built into `<dialog>` |

The `<dialog>` element provides everything for free: focus trapping, Escape key dismissal, `::backdrop`, `aria-modal`, and top-layer rendering. Building these from scratch would be hundreds of lines of fragile JavaScript.

### Why NOT the Invoker Commands API (commandfor/command)

The Invoker Commands API (`<button commandfor="dialog" command="show-modal">`) would allow zero-JS modal opening. However:

- **Global support: only 73.81%** as of February 2026
- **Baseline status: "low"** -- achieved across all major browsers only in December 2025 (Safari 26.2 was last)
- Users on Safari < 26.2, Firefox < 144, or Chrome < 135 would see non-functional buttons

**Verdict:** Too new. Use `.showModal()` JavaScript calls for now. Revisit when Invoker Commands reaches 90%+ global support (likely mid-2027).

**Sources:**
- [Can I Use: Invoker Commands](https://caniuse.com/wf-invoker-commands) -- 73.81% global support (HIGH confidence)
- [InfoQ: HTML Invoker Commands](https://www.infoq.com/news/2026/01/html-invoker-commands/) -- Timeline context (MEDIUM confidence)

### Implementation Pattern

#### HTML Structure

```html
<!-- Trigger button (in each recommendation card) -->
<button class="modal-trigger" data-modal="modal-rec-{{.Rank}}">
    View Improvement Prompt
</button>

<!-- Dialog element (rendered once per recommendation) -->
<dialog id="modal-rec-{{.Rank}}" class="ars-modal">
    <div class="modal-header">
        <h3>{{.Summary}}</h3>
        <button class="modal-close" aria-label="Close">&times;</button>
    </div>
    <div class="modal-body">
        <!-- Content here: trace output, improvement prompt, etc. -->
    </div>
</dialog>
```

#### JavaScript (inline in `<script>` block)

```javascript
// Modal open/close -- extend existing <script> block
document.querySelectorAll('.modal-trigger').forEach(btn => {
    btn.addEventListener('click', () => {
        const modal = document.getElementById(btn.dataset.modal);
        if (modal) modal.showModal();
    });
});

document.querySelectorAll('.modal-close').forEach(btn => {
    btn.addEventListener('click', () => {
        btn.closest('dialog').close();
    });
});

// Close on backdrop click (click outside modal content)
document.querySelectorAll('.ars-modal').forEach(dialog => {
    dialog.addEventListener('click', (e) => {
        if (e.target === dialog) dialog.close();
    });
});
```

#### CSS

```css
/* Modal dialog */
.ars-modal {
    border: none;
    border-radius: 0.5rem;
    padding: 0;
    max-width: 700px;
    width: 90vw;
    max-height: 80vh;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.15);
}

.ars-modal::backdrop {
    background: rgba(0, 0, 0, 0.5);
}

.modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.5rem;
    border-bottom: 1px solid var(--color-border);
}

.modal-header h3 {
    font-size: 1rem;
    font-weight: 600;
    margin: 0;
}

.modal-close {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    color: var(--color-muted);
    padding: 0 0.25rem;
    line-height: 1;
}

.modal-close:hover {
    color: var(--color-text);
}

.modal-body {
    padding: 1.5rem;
    overflow-y: auto;
    max-height: calc(80vh - 4rem);
}
```

### Progressive Enhancement

The `<dialog>` element is hidden by default (display: none until opened). If JavaScript is disabled:
- Modal content is invisible but present in DOM
- Core report content (scores, charts, recommendations) remains fully accessible
- Only the "View Improvement Prompt" / "View Trace" buttons become non-functional

This is acceptable progressive enhancement: the modal content is supplementary detail, not core report data.

### Go Template Integration

The `<dialog>` elements are rendered server-side by Go's `html/template`. Each modal's content is generated at report creation time -- no client-side data fetching needed.

Key consideration: Go's `html/template` auto-escapes content in JavaScript contexts. Since modal content is in HTML (not inside `<script>` tags), standard template actions work correctly:

```html
<dialog id="modal-trace-{{.Rank}}">
    <div class="modal-body">
        <pre><code>{{.TraceContent}}</code></pre>
    </div>
</dialog>
```

The `{{.TraceContent}}` will be HTML-escaped by `html/template`, which is correct for display inside a `<code>` block. For content that contains safe HTML (like pre-formatted trace output), use `template.HTML` type in Go, following the existing pattern for `DetailedDescription` and `RadarChartSVG`.

---

## 2. Syntax Highlighting for Code Blocks

### Recommendation: Custom Inline JSON/Command Highlighter (~30 lines)

**Write a purpose-built ~30-line JavaScript function.** Do NOT use highlight.js, Prism.js, or microlight.js.

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Custom inline JS | N/A | Highlight JSON keys/values and shell commands | Exactly fits the narrow scope; zero external dependencies; ~30 lines total |

### Why Custom, Not a Library

The ARS report needs syntax highlighting for exactly two content types:

1. **JSON objects** -- Agent response payloads, structured output
2. **Shell commands** -- `claude -p "..." --output-format json`

This is fundamentally different from a code editor or documentation site that needs to highlight dozens of languages.

| Factor | highlight.js (custom build) | Prism.js | microlight.js | Custom 30-line function |
|--------|---------------------------|----------|---------------|------------------------|
| Size (minified) | ~15-20 KB (core + JSON + bash) | ~4-5 KB (core + JSON + bash) | 2.2 KB | ~0.8 KB |
| Languages | 39+ in common build | Extensible | Language-agnostic | JSON + shell commands only |
| CSS theme needed | Yes (~1 KB) | Yes (~1 KB) | No | No (uses existing CSS vars) |
| Build step for custom bundle | Yes (download page or npm) | Yes (download page or npm) | No | No |
| Self-contained embedding | Possible but awkward | Possible but awkward | Easy | Trivial -- it IS inline |
| Color scheme matches report | Requires theme customization | Requires theme customization | Uses text-shadow (different aesthetic) | Uses existing CSS variables |

**The math is clear:** Adding even the smallest library (microlight.js at 2.2 KB) introduces an external artifact that must be embedded, versioned, and maintained. A custom function uses the CSS variables already defined in `styles.css` and highlights exactly what the ARS report contains.

### Implementation

#### JSON Highlighting Function

```javascript
// Syntax highlighting for JSON and commands
function highlightJSON(text) {
    return text
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/"([^"]*)"(\s*:)/g, '<span class="hl-key">"$1"</span>$2')
        .replace(/"([^"]*)"/g, '<span class="hl-string">"$1"</span>')
        .replace(/\b(true|false|null)\b/g, '<span class="hl-literal">$1</span>')
        .replace(/\b(-?\d+\.?\d*)\b/g, '<span class="hl-number">$1</span>');
}

// Apply to all code blocks with data-lang attribute
document.querySelectorAll('code[data-lang="json"]').forEach(el => {
    el.innerHTML = highlightJSON(el.textContent);
});
```

This is a well-established pattern. The regex approach for JSON highlighting appears in multiple community implementations (CodePen, GitHub Gists) and is reliable for well-formed JSON output, which is what the ARS report produces.

**Source:** [JSON Syntax Highlighting Gist](https://gist.github.com/faffyman/6183311) -- Community pattern (~15 lines for core logic) (MEDIUM confidence)

#### CSS Classes

```css
/* Syntax highlighting -- uses existing report color palette */
.hl-key { color: #2563eb; }          /* Blue -- matches link color */
.hl-string { color: #059669; }       /* Green */
.hl-number { color: #d97706; }       /* Amber */
.hl-literal { color: #7c3aed; }      /* Purple -- true/false/null */

/* Code blocks in modals */
.modal-body pre {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 0.375rem;
    padding: 1rem;
    overflow-x: auto;
    font-size: 0.8rem;
    line-height: 1.5;
}

.modal-body code {
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}
```

#### Shell Command Highlighting

For shell commands (simpler -- just highlight the command name and flags):

```javascript
function highlightCommand(text) {
    return text
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/^(\s*)([\w./]+)/gm, '$1<span class="hl-cmd">$2</span>')
        .replace(/(--?\w[\w-]*)/g, '<span class="hl-flag">$1</span>')
        .replace(/"([^"]*)"/g, '<span class="hl-string">"$1"</span>');
}

document.querySelectorAll('code[data-lang="shell"]').forEach(el => {
    el.innerHTML = highlightCommand(el.textContent);
});
```

```css
.hl-cmd { color: #2563eb; font-weight: 600; }
.hl-flag { color: #7c3aed; }
```

### Template Usage

In Go templates, mark code blocks with `data-lang` for the highlighting script to target:

```html
<pre><code data-lang="json">{{.JSONContent}}</code></pre>
<pre><code data-lang="shell">{{.CommandContent}}</code></pre>
```

The `{{.JSONContent}}` will be HTML-escaped by `html/template`. The highlighting function reads `textContent` (which gives unescaped text), processes it, then writes to `innerHTML`. This is safe because the function does its own HTML entity escaping (`&amp;`, `&lt;`, `&gt;`) before applying `<span>` tags.

### When to Reconsider

If ARS later needs to highlight Go, Python, or TypeScript source code (e.g., showing "worst offender" files), then Prism.js with a custom build becomes worth the cost. For JSON and shell commands, the custom approach is clearly better.

---

## 3. Copy-to-Clipboard

### Recommendation: `navigator.clipboard.writeText()` with Visual Feedback

**Extend the existing pattern.** The report already uses `navigator.clipboard.writeText()` for badge markdown copying (line 34 of `report.html`).

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `navigator.clipboard.writeText()` | Clipboard API | Copy text to system clipboard | 95.68% global support; already used in report; async, promise-based |

### Browser Support (Verified via Can I Use)

| Browser | Minimum Version |
|---------|----------------|
| Chrome | 66+ |
| Firefox | 63+ |
| Safari | 13.1+ |
| Edge | 79+ |

**Global support: 95.68%**

**Important constraints:**
- Requires secure context (HTTPS or `file://` protocol). HTML reports opened from local filesystem (`file://`) work correctly.
- Requires user gesture (click event). This is already the case since we use button click handlers.

**Source:** [Can I Use: Clipboard writeText](https://caniuse.com/mdn-api_clipboard_writetext) (HIGH confidence)

### Current Implementation (Already in Report)

The badge section already has a working copy button (line 34 of `report.html`):

```html
<button onclick="navigator.clipboard.writeText(document.getElementById('badge-markdown').textContent)">Copy</button>
```

This inline approach works but lacks feedback. The new implementation should add visual confirmation.

### Enhanced Implementation with Feedback

```javascript
// Copy-to-clipboard with visual feedback
function copyToClipboard(btn, targetId) {
    const text = document.getElementById(targetId).textContent;
    navigator.clipboard.writeText(text).then(() => {
        const original = btn.textContent;
        btn.textContent = 'Copied!';
        btn.classList.add('copied');
        setTimeout(() => {
            btn.textContent = original;
            btn.classList.remove('copied');
        }, 1500);
    }).catch(() => {
        // Fallback: select text for manual copy
        const range = document.createRange();
        range.selectNodeContents(document.getElementById(targetId));
        window.getSelection().removeAllRanges();
        window.getSelection().addRange(range);
    });
}
```

#### CSS for Feedback State

```css
/* Copy button states */
.copy-btn {
    padding: 0.25rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 0.25rem;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
}

.copy-btn:hover {
    background: var(--color-border);
}

.copy-btn.copied {
    color: var(--color-green);
    border-color: var(--color-green);
}
```

#### Template Usage

```html
<!-- In modal for improvement prompt -->
<div class="prompt-container">
    <div class="prompt-header">
        <span>Improvement Prompt</span>
        <button class="copy-btn" onclick="copyToClipboard(this, 'prompt-{{.Rank}}')">Copy</button>
    </div>
    <pre><code id="prompt-{{.Rank}}">{{.PromptText}}</code></pre>
</div>
```

### Fallback Strategy

The `.catch()` handler selects the text content, making it ready for manual Ctrl+C/Cmd+C. This covers:
- Browsers without Clipboard API (very rare at 95.68% support)
- Non-secure contexts where the API is blocked
- Permission denials

No `document.execCommand('copy')` fallback is needed. That API is deprecated and the text-selection fallback is more reliable than attempting to use a deprecated API that browsers may remove.

**Source:** [MDN: Clipboard.writeText()](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText) (HIGH confidence)

---

## Integration with Existing Report Infrastructure

### Files to Modify

| File | Change | Why |
|------|--------|-----|
| `internal/output/templates/report.html` | Add `<dialog>` elements, expand `<script>` block, add copy buttons | Core template changes |
| `internal/output/templates/styles.css` | Add modal, syntax highlighting, and copy button styles | All CSS inline via `{{.InlineCSS}}` |
| `internal/output/html.go` | Add new fields to `HTMLReportData` and `HTMLRecommendation` for modal content | New template data needed |

### Files NOT to Modify

| File | Why Leave Alone |
|------|-----------------|
| `go.mod` | No new Go dependencies |
| `internal/output/descriptions.go` | Metric descriptions do not need modals (already have expand/collapse) |
| `internal/output/charts.go` | SVG charts unrelated to interactive UI |

### Embedding Strategy

All JavaScript and CSS remain inline in the template, following the existing pattern:
- CSS: embedded via `{{.InlineCSS}}` in `<style>` tag (line 8 of `report.html`)
- JavaScript: inline `<script>` block at bottom of `<body>` (lines 125-159 of `report.html`)

The new JavaScript (~40 lines total for modals + highlighting + copy) extends the existing ~30 lines. Total inline JavaScript remains under 80 lines -- acceptable for a self-contained report.

### Template Data Additions

```go
// New fields needed in HTMLRecommendation (or new struct)
type HTMLRecommendation struct {
    // ... existing fields ...
    ImprovementPrompt string // Text for copy-to-clipboard
    HasTrace          bool   // Whether call trace is available
    TraceContent      string // Formatted trace output
}
```

The trace content and improvement prompts are generated server-side in Go. The template renders them into `<dialog>` elements. The JavaScript only handles open/close/copy interactions -- no client-side data processing.

### Go `html/template` Safety

Go's `html/template` package auto-escapes content based on context. Key behaviors for this milestone:

| Context | Template Action | Behavior |
|---------|----------------|----------|
| Inside `<code>` | `{{.JSONContent}}` | HTML-escaped (correct for display; JS reads via `.textContent`) |
| Inside `<script>` | `{{.Value}}` | JS-escaped (not needed; we use `data-` attributes, not inline values) |
| CSS attribute | `{{.InlineCSS}}` | Already uses `template.CSS` type (trusted) |
| Pre-formatted HTML | `{{.DetailedDescription}}` | Already uses `template.HTML` type (trusted) |

No changes to the template safety model are needed. New content (trace output, prompts) goes into HTML context as plain text, which `html/template` handles correctly by default.

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| highlight.js | 15-20 KB for two content types; requires embedding minified bundle and CSS theme | Custom 30-line highlighter |
| Prism.js | 4-5 KB for two content types; requires build step for custom bundle | Custom 30-line highlighter |
| microlight.js | 2.2 KB; language-agnostic (different aesthetic from report); known DOS vulnerability with large inputs | Custom 30-line highlighter |
| Alpine.js / Petite-Vue | Reactive framework for what is static content with open/close behavior | Vanilla JS event listeners |
| Web Components | Over-engineering for one-off modal/copy pattern | Standard HTML + JS |
| Invoker Commands API | Only 73.81% browser support; too new for production | `showModal()` / `.close()` JS calls |
| CSS-only modals (`:target`) | Breaks back button; no focus trap; no Escape key; no accessibility | `<dialog>` element |
| `document.execCommand('copy')` | Deprecated API; browsers actively removing it | `navigator.clipboard.writeText()` |
| clipboard.js library | 3 KB library for one function that is 5 lines of native code | `navigator.clipboard.writeText()` |
| Any external CDN | Breaks self-contained constraint; offline usage fails; additional HTTP request | Inline everything |

### Size Budget Reasoning

Current report inline CSS: ~560 lines (~14 KB uncompressed).
Current report inline JS: ~30 lines (~1 KB uncompressed).
New JS additions: ~70 lines (~2 KB uncompressed).
New CSS additions: ~60 lines (~1.5 KB uncompressed).

**Total inline code after changes: ~640 lines CSS + ~100 lines JS = ~18.5 KB.**

This is well within acceptable limits for a self-contained HTML report. For comparison, embedding just highlight.js core would add 15-20 KB of minified JavaScript alone.

---

## Version Compatibility

| Component | Minimum Browser | Global Support | Notes |
|-----------|----------------|----------------|-------|
| `<dialog>` element | Chrome 37, FF 98, Safari 15.4 | 95.81% | Baseline since March 2022 |
| `::backdrop` | Same as `<dialog>` | 95.81% | Part of `<dialog>` spec |
| `showModal()` / `close()` | Same as `<dialog>` | 95.81% | Part of `<dialog>` spec |
| `navigator.clipboard.writeText()` | Chrome 66, FF 63, Safari 13.1 | 95.68% | Requires secure context |
| `element.textContent` | All browsers | ~100% | Fundamental DOM API |
| CSS custom properties | Chrome 49, FF 31, Safari 9.1 | 97%+ | Already used in `styles.css` |
| Template literals | ES6+ | 97%+ | Not used; vanilla string concatenation sufficient |

All technologies have 95%+ global support. The lowest common denominator is `navigator.clipboard.writeText()` at 95.68%.

---

## Recommended Stack Summary

### New Go Dependencies: ZERO

| Component | Technology | Status |
|-----------|-----------|--------|
| Modal overlay | HTML `<dialog>` + `showModal()` | Web standard, no Go deps |
| Syntax highlighting | Custom inline JS (~30 lines) | Written as part of template |
| Copy-to-clipboard | `navigator.clipboard.writeText()` | Already used in report |
| Modal styling | CSS `::backdrop` + custom classes | Added to `styles.css` |
| Code block styling | CSS `.hl-key`, `.hl-string`, etc. | Added to `styles.css` |

### New External Libraries: ZERO

Everything is inline HTML, CSS, and JavaScript within the Go template. The report remains fully self-contained: a single HTML file with no external dependencies.

---

## Sources

### HTML Dialog Element
- [MDN: dialog element](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/dialog) -- Comprehensive reference including accessibility (HIGH confidence)
- [Can I Use: dialog](https://caniuse.com/dialog) -- 95.81% global support, Baseline since March 2022 (HIGH confidence)
- [MDN: showModal()](https://developer.mozilla.org/en-US/docs/Web/API/HTMLDialogElement/showModal) -- API reference for modal behavior (HIGH confidence)
- [MDN: ::backdrop](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Selectors/::backdrop) -- Backdrop pseudo-element styling (HIGH confidence)

### Invoker Commands API (Evaluated, NOT Recommended)
- [Can I Use: Invoker Commands](https://caniuse.com/wf-invoker-commands) -- 73.81% global support, "low" baseline (HIGH confidence)
- [InfoQ: HTML Invoker Commands](https://www.infoq.com/news/2026/01/html-invoker-commands/) -- Chrome 135, FF 144, Safari 26.2 timeline (MEDIUM confidence)
- [CSS-Tricks: Invoker Commands](https://css-tricks.com/invoker-commands-additional-ways-to-work-with-dialog-popover-and-more/) -- Detailed API overview (MEDIUM confidence)

### Clipboard API
- [Can I Use: Clipboard writeText](https://caniuse.com/mdn-api_clipboard_writetext) -- 95.68% global support (HIGH confidence)
- [MDN: Clipboard.writeText()](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText) -- API reference and security requirements (HIGH confidence)
- [web.dev: Copy text](https://web.dev/patterns/clipboard/copy-text) -- Best practices for clipboard access (MEDIUM confidence)

### Syntax Highlighting (Evaluated Alternatives)
- [highlight.js](https://highlightjs.org/) -- Zero-dependency highlighter, ~20 KB common build (HIGH confidence)
- [Prism.js](https://prismjs.com/) -- 2 KB core + language plugins (HIGH confidence)
- [microlight.js](https://asvd.github.io/microlight/) -- 2.2 KB language-agnostic highlighter (MEDIUM confidence)
- [JSON Syntax Highlighting Gist](https://gist.github.com/faffyman/6183311) -- ~15-line regex-based JSON highlighter (MEDIUM confidence)

### Existing Codebase (Primary Source)
- `internal/output/templates/report.html` -- Current template with inline JS (authoritative)
- `internal/output/templates/styles.css` -- Current CSS with CSS custom properties (authoritative)
- `internal/output/html.go` -- Template data structures and rendering (authoritative)

---
*Stack research for: ARS v0.0.5 -- Interactive HTML report enhancements (modals, syntax highlighting, clipboard)*
*Researched: 2026-02-06*
