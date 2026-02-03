# Phase 13: Badge Generation - Research

**Researched:** 2026-02-03
**Domain:** shields.io static badge URL generation in Go
**Confidence:** HIGH

## Summary

This phase adds `--badge` flag support to generate shields.io badge markdown URLs that display ARS tier and score. The implementation is straightforward: construct a shields.io static badge URL with properly URL-encoded parameters, then output it in markdown format alongside existing output modes (terminal, JSON, HTML).

The shields.io static badge API uses a simple URL pattern: `https://img.shields.io/badge/LABEL-MESSAGE-COLOR`. Special characters must be percent-encoded (spaces as `%20`, dashes as `--`). Go's standard library `net/url.PathEscape()` handles encoding correctly for URL path segments.

The codebase already has all necessary infrastructure: `ScoredResult` contains `Composite` (float64) and `Tier` (string), output rendering is centralized in `internal/output/`, and the CLI uses cobra flags. The implementation requires minimal changes: add flag to `cmd/scan.go`, create badge URL builder in `internal/output/badge.go`, integrate with existing output flow.

**Primary recommendation:** Use Go's `net/url.PathEscape()` for encoding and shields.io's named colors (`green`, `yellow`, `orange`, `red`) for tier-based coloring.

## Standard Stack

This phase requires no new dependencies. All functionality is provided by Go's standard library.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/url` | stdlib | URL path encoding | Go standard library, `PathEscape()` correctly encodes spaces as `%20` for URL paths |
| `fmt` | stdlib | String formatting | Score formatting with one decimal precision |

### Supporting
None required.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `net/url.PathEscape()` | `net/url.QueryEscape()` | QueryEscape encodes spaces as `+` which is incorrect for URL paths |
| Named colors | Hex codes | Named colors (`green`, `orange`, etc.) are more readable and widely supported |

**Installation:**
```bash
# No new dependencies needed
```

## Architecture Patterns

### Recommended Project Structure
```
internal/output/
├── badge.go        # NEW: Badge URL generation
├── badge_test.go   # NEW: Badge generation tests
├── json.go         # Modify: Add badge_url and badge_markdown fields
├── terminal.go     # Modify: Append badge markdown when --badge flag
└── html.go         # Modify: Add "Copy badge" section
```

### Pattern 1: Badge URL Builder
**What:** Pure function that takes score and tier, returns badge URL and markdown
**When to use:** Called by output renderers when `--badge` flag is set
**Example:**
```go
// Source: shields.io documentation + Go net/url package
package output

import (
    "fmt"
    "net/url"
)

const (
    arsRepoURL = "https://github.com/ingo-eichhorst/agent-readyness"
)

// BadgeInfo contains badge URL and markdown for output.
type BadgeInfo struct {
    URL      string // Raw shields.io URL
    Markdown string // Complete markdown with link
}

// GenerateBadge creates a shields.io badge URL and markdown for the given score and tier.
func GenerateBadge(composite float64, tier string) BadgeInfo {
    // Format: "Agent-Ready 8.2/10"
    message := fmt.Sprintf("%s %.1f/10", tier, composite)

    // Encode for URL path: spaces -> %20, dashes -> --
    encodedMessage := encodeBadgeText(message)

    // Map tier to color
    color := tierToColor(tier)

    // Build URL: https://img.shields.io/badge/ARS-MESSAGE-COLOR
    badgeURL := fmt.Sprintf("https://img.shields.io/badge/ARS-%s-%s", encodedMessage, color)

    // Build markdown with link to repo
    markdown := fmt.Sprintf("[![ARS](%s)](%s)", badgeURL, arsRepoURL)

    return BadgeInfo{
        URL:      badgeURL,
        Markdown: markdown,
    }
}

// encodeBadgeText encodes text for shields.io badge URL path.
// Shields.io requires: spaces as %20 (or _), dashes as --
func encodeBadgeText(s string) string {
    // First escape dashes (before general encoding)
    result := strings.ReplaceAll(s, "-", "--")
    // Then URL-encode (spaces become %20)
    return url.PathEscape(result)
}

// tierToColor maps tier classification to shields.io color.
func tierToColor(tier string) string {
    switch tier {
    case "Agent-Ready":
        return "green"
    case "Agent-Assisted":
        return "yellow"
    case "Agent-Limited":
        return "orange"
    default: // Agent-Hostile
        return "red"
    }
}
```

### Pattern 2: Flag Integration
**What:** Add `--badge` flag that controls badge output across all modes
**When to use:** CLI flag parsed in `cmd/scan.go`, passed to pipeline
**Example:**
```go
// Source: Existing cmd/scan.go pattern
var badgeOutput bool

func init() {
    scanCmd.Flags().BoolVar(&badgeOutput, "badge", false, "generate shields.io badge markdown URL")
}
```

### Pattern 3: Output Mode Integration
**What:** Badge output appended/included based on output mode
**When to use:** Terminal appends markdown, JSON includes fields, HTML adds copy section
**Example:**
```go
// Terminal mode: append after recommendations
if badgeOutput && scored != nil {
    badge := output.GenerateBadge(scored.Composite, scored.Tier)
    fmt.Fprintln(w)
    fmt.Fprintln(w, "Badge:")
    fmt.Fprintln(w, badge.Markdown)
}

// JSON mode: add to report
type JSONReport struct {
    // ... existing fields ...
    BadgeURL      string `json:"badge_url,omitempty"`
    BadgeMarkdown string `json:"badge_markdown,omitempty"`
}
```

### Anti-Patterns to Avoid
- **Using QueryEscape for paths:** QueryEscape encodes spaces as `+`, which is incorrect for URL path segments. Use `PathEscape` instead.
- **Hardcoding URL without encoding:** Special characters in tier names and scores must be encoded.
- **Forgetting double-dash for literal dashes:** Shields.io interprets single dash as separator; use `--` for literal dash.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| URL encoding | Custom string replacement | `net/url.PathEscape()` | Handles all edge cases correctly, standard library |
| Float formatting | Manual rounding/string ops | `fmt.Sprintf("%.1f", score)` | Consistent precision, locale-independent |

**Key insight:** The shields.io URL format is simple enough that no external library is needed. Go's standard library handles all encoding requirements.

## Common Pitfalls

### Pitfall 1: Wrong URL Encoding Function
**What goes wrong:** Using `url.QueryEscape()` encodes spaces as `+`, resulting in `Agent+Ready` instead of `Agent%20Ready` in the badge
**Why it happens:** QueryEscape is designed for query string parameters where `+` means space
**How to avoid:** Always use `url.PathEscape()` for URL path segments
**Warning signs:** Badge displays `+` characters instead of spaces

### Pitfall 2: Single Dash Interpreted as Separator
**What goes wrong:** Tier name "Agent-Ready" gets split incorrectly because `-` is the separator
**Why it happens:** Shields.io uses single dash to separate label-message-color
**How to avoid:** Replace `-` with `--` before other encoding
**Warning signs:** Badge shows truncated or malformed text

### Pitfall 3: Color Name Mismatch
**What goes wrong:** Using color names not recognized by shields.io (e.g., "brightred" doesn't exist)
**Why it happens:** Assuming all CSS color names work
**How to avoid:** Stick to shields.io's named color palette: `green`, `yellow`, `orange`, `red`
**Warning signs:** Badge displays default gray color instead of tier color

### Pitfall 4: Forgetting Empty/Zero Score Edge Cases
**What goes wrong:** Badge generated with score 0.0 or empty tier
**Why it happens:** Not checking if scoring succeeded before generating badge
**How to avoid:** Only generate badge when `scored != nil && scored.Composite > 0`
**Warning signs:** Badge shows "0.0/10" or empty tier name

## Code Examples

### Complete Badge Generation
```go
// Source: shields.io static badge docs + Go net/url package
package output

import (
    "fmt"
    "net/url"
    "strings"

    "github.com/ingo/agent-readyness/pkg/types"
)

const arsRepoURL = "https://github.com/ingo-eichhorst/agent-readyness"

// BadgeInfo holds generated badge URL and markdown.
type BadgeInfo struct {
    URL      string
    Markdown string
}

// GenerateBadge creates shields.io badge from scored result.
func GenerateBadge(scored *types.ScoredResult) BadgeInfo {
    if scored == nil {
        return BadgeInfo{}
    }

    // Format message: "Agent-Ready 8.2/10"
    message := fmt.Sprintf("%s %.1f/10", scored.Tier, scored.Composite)

    // Encode for URL: dashes -> --, then PathEscape
    encoded := encodeBadgeText(message)

    // Get tier color
    color := tierToColor(scored.Tier)

    // Build URL
    badgeURL := fmt.Sprintf("https://img.shields.io/badge/ARS-%s-%s", encoded, color)

    // Build markdown with clickable link
    markdown := fmt.Sprintf("[![ARS](%s)](%s)", badgeURL, arsRepoURL)

    return BadgeInfo{URL: badgeURL, Markdown: markdown}
}

func encodeBadgeText(s string) string {
    // 1. Escape dashes first (shields.io separator)
    s = strings.ReplaceAll(s, "-", "--")
    // 2. URL path encoding (spaces -> %20)
    return url.PathEscape(s)
}

func tierToColor(tier string) string {
    colors := map[string]string{
        "Agent-Ready":    "green",
        "Agent-Assisted": "yellow",
        "Agent-Limited":  "orange",
        "Agent-Hostile":  "red",
    }
    if c, ok := colors[tier]; ok {
        return c
    }
    return "red" // default for unknown tiers
}
```

### JSON Report Integration
```go
// Source: Existing internal/output/json.go pattern
type JSONReport struct {
    Version         string               `json:"version"`
    CompositeScore  float64              `json:"composite_score"`
    Tier            string               `json:"tier"`
    Categories      []JSONCategory       `json:"categories"`
    Recommendations []JSONRecommendation `json:"recommendations"`
    BadgeURL        string               `json:"badge_url,omitempty"`
    BadgeMarkdown   string               `json:"badge_markdown,omitempty"`
}

// In BuildJSONReport:
func BuildJSONReport(scored *types.ScoredResult, recs []recommend.Recommendation, verbose bool, includeBadge bool) *JSONReport {
    report := &JSONReport{
        // ... existing fields ...
    }

    if includeBadge {
        badge := GenerateBadge(scored)
        report.BadgeURL = badge.URL
        report.BadgeMarkdown = badge.Markdown
    }

    return report
}
```

### Terminal Output Integration
```go
// Source: Existing internal/output/terminal.go pattern

// RenderBadge prints badge markdown to terminal.
func RenderBadge(w io.Writer, scored *types.ScoredResult) {
    if scored == nil {
        return
    }

    badge := GenerateBadge(scored)
    bold := color.New(color.Bold)

    fmt.Fprintln(w)
    bold.Fprintln(w, "Badge")
    fmt.Fprintln(w, strings.Repeat("-", 40))
    fmt.Fprintln(w, badge.Markdown)
}
```

### HTML Copy Section
```go
// Source: Existing internal/output/html.go pattern

// In HTMLReportData struct:
type HTMLReportData struct {
    // ... existing fields ...
    BadgeMarkdown string // For copy-to-clipboard section
}

// Template snippet:
// <div class="badge-section">
//   <h3>Badge</h3>
//   <code id="badge-code">{{.BadgeMarkdown}}</code>
//   <button onclick="copyBadge()">Copy</button>
// </div>
```

### Unit Test Example
```go
// Source: Go testing patterns
func TestGenerateBadge(t *testing.T) {
    tests := []struct {
        name      string
        composite float64
        tier      string
        wantURL   string
    }{
        {
            name:      "Agent-Ready high score",
            composite: 8.2,
            tier:      "Agent-Ready",
            wantURL:   "https://img.shields.io/badge/ARS-Agent--Ready%208.2%2F10-green",
        },
        {
            name:      "Agent-Hostile low score",
            composite: 3.5,
            tier:      "Agent-Hostile",
            wantURL:   "https://img.shields.io/badge/ARS-Agent--Hostile%203.5%2F10-red",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            scored := &types.ScoredResult{
                Composite: tt.composite,
                Tier:      tt.tier,
            }
            badge := GenerateBadge(scored)
            if badge.URL != tt.wantURL {
                t.Errorf("URL = %q, want %q", badge.URL, tt.wantURL)
            }
        })
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A | shields.io static badge API | Stable since 2015+ | Standard for README badges |

**Deprecated/outdated:**
- None. The shields.io static badge format has been stable for years.

## Open Questions

None. The implementation approach is straightforward and all technical details are verified.

## Sources

### Primary (HIGH confidence)
- [shields.io Static Badge Documentation](https://shields.io/badges/static-badge) - URL format, encoding rules
- [Go net/url Package](https://pkg.go.dev/net/url) - PathEscape function behavior
- [shields.io badge-maker README](https://github.com/badges/shields/tree/master/badge-maker) - Named colors list

### Secondary (MEDIUM confidence)
- Existing codebase analysis - `internal/output/*.go`, `internal/scoring/config.go` for tier thresholds

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Go stdlib, no external dependencies
- Architecture: HIGH - Simple integration with existing output patterns
- Pitfalls: HIGH - Verified encoding behavior with official documentation

**Research date:** 2026-02-03
**Valid until:** 2026-05-03 (shields.io API is stable)
