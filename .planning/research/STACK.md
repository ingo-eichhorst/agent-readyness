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
