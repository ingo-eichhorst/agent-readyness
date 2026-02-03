# Phase 11: Terminal Output Integration - Research

**Researched:** 2026-02-03
**Domain:** Go terminal output rendering, CLI output formatting, existing codebase patterns
**Confidence:** HIGH

## Summary

Phase 11 closes a gap identified in the v2 milestone audit: C7 agent evaluation metrics are computed and available via JSON output but have no terminal rendering function. The research is straightforward because this phase follows an established pattern - the codebase already has 6 working terminal renderers (C1-C6) that serve as templates.

The implementation involves adding a `renderC7` function to `internal/output/terminal.go` that displays the 4 C7 metrics (intent clarity, modification confidence, cross-file coherence, semantic completeness) in a format consistent with other categories. The verbose mode should show per-task breakdown including score, status, duration, and reasoning.

**Primary recommendation:** Implement `renderC7` following the exact patterns in `renderC4`, `renderC5`, and `renderC6` - using the same header format, metric display conventions, and verbose detail expansion.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/fatih/color` | existing | Colored terminal output | Already used throughout codebase for consistent UX |
| `io.Writer` | Go stdlib | Output target abstraction | Testable, pipe-friendly output |
| `fmt` | Go stdlib | Formatted printing | Standard Go formatting |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `sort` | Go stdlib | Ordering verbose output | Display tasks in consistent order |
| `strings` | Go stdlib | String manipulation | Formatting duration, percentages |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| fatih/color | lipgloss/bubbletea | More features but adds dependency; color is already in use |
| Manual ANSI codes | color library | Color handles TTY detection automatically |

**Installation:**
```bash
# No new dependencies needed - all libraries already imported in terminal.go
```

## Architecture Patterns

### Recommended Project Structure
```
internal/output/
├── terminal.go      # Add renderC7 function + switch case
└── terminal_test.go # Add C7 rendering tests
```

### Pattern 1: Category Renderer Function
**What:** A `renderCX` function that extracts typed metrics and renders them with color-coded thresholds
**When to use:** Displaying any analysis category in terminal
**Example:**
```go
// Source: internal/output/terminal.go (existing pattern from renderC5)
func renderC7(w io.Writer, ar *types.AnalysisResult, verbose bool) {
    bold := color.New(color.Bold)

    raw, ok := ar.Metrics["c7"]
    if !ok {
        return
    }
    m, ok := raw.(*types.C7Metrics)
    if !ok {
        return
    }

    fmt.Fprintln(w)
    bold.Fprintln(w, "C7: Agent Evaluation")
    fmt.Fprintln(w, "----------------------------------------")

    if !m.Available {
        fmt.Fprintln(w, "  Not available (--enable-c7 not specified)")
        return
    }

    // Display 4 core metrics with color coding
    renderC7Metric(w, "Intent clarity:", m.IntentClarity)
    renderC7Metric(w, "Modification confidence:", m.ModificationConfidence)
    renderC7Metric(w, "Cross-file coherence:", m.CrossFileCoherence)
    renderC7Metric(w, "Semantic completeness:", m.SemanticCompleteness)

    // Summary metrics
    fmt.Fprintf(w, "  Overall score:       %.1f\n", m.OverallScore)
    fmt.Fprintf(w, "  Duration:            %.1fs\n", m.TotalDuration)
    fmt.Fprintf(w, "  Estimated cost:      $%.4f\n", m.CostUSD)

    // Verbose: per-task breakdown
    if verbose && len(m.TaskResults) > 0 {
        fmt.Fprintln(w)
        bold.Fprintln(w, "  Per-task results:")
        for _, tr := range m.TaskResults {
            fmt.Fprintf(w, "    %s: score=%d status=%s (%.1fs)\n",
                tr.TaskName, tr.Score, tr.Status, tr.Duration)
            if tr.Reasoning != "" {
                fmt.Fprintf(w, "      Reasoning: %s\n", tr.Reasoning)
            }
        }
    }
}
```

### Pattern 2: Metric Color Coding
**What:** Apply color based on score thresholds (green >= 70, yellow >= 40, red < 40)
**When to use:** Any 0-100 scale metric display
**Example:**
```go
// Source: Adapted from existing colorForFloatInverse pattern
func c7ScoreColor(score int) *color.Color {
    if score >= 70 {
        return color.New(color.FgGreen)
    }
    if score >= 40 {
        return color.New(color.FgYellow)
    }
    return color.New(color.FgRed)
}

func renderC7Metric(w io.Writer, label string, score int) {
    c := c7ScoreColor(score)
    c.Fprintf(w, "  %-24s %d/100\n", label, score)
}
```

### Pattern 3: Switch Case Integration
**What:** Add C7 case to the existing RenderSummary switch statement
**When to use:** Integrating new category into terminal output
**Example:**
```go
// Source: internal/output/terminal.go:66-81
// Add case "C7" to the switch statement
for _, ar := range analysisResults {
    switch ar.Category {
    case "C1":
        renderC1(w, ar, verbose)
    case "C2":
        renderC2(w, ar, verbose)
    case "C3":
        renderC3(w, ar, verbose)
    case "C4":
        renderC4(w, ar, verbose)
    case "C5":
        renderC5(w, ar, verbose)
    case "C6":
        renderC6(w, ar, verbose)
    case "C7":
        renderC7(w, ar, verbose) // ADD THIS LINE
    }
}
```

### Pattern 4: Category Display Name Mapping
**What:** Register C7 human-readable name in categoryDisplayNames map
**When to use:** Adding new category to scoring display
**Example:**
```go
// Source: internal/output/terminal.go:483-490
// Add C7 to the map
var categoryDisplayNames = map[string]string{
    "C1": "Code Health",
    "C2": "Semantic Explicitness",
    "C3": "Architecture",
    "C4": "Documentation Quality",
    "C5": "Temporal Dynamics",
    "C6": "Testing",
    "C7": "Agent Evaluation", // ADD THIS LINE
}
```

### Anti-Patterns to Avoid
- **Inconsistent header style:** Use same separator characters as other categories (40 dashes)
- **Missing availability check:** Always check `m.Available` before rendering detailed metrics
- **Color on unavailable:** Don't apply color to "n/a" or unavailable messages
- **Breaking pipe compatibility:** Use `io.Writer` interface, never hardcode os.Stdout

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Color coding | Manual ANSI escape codes | `fatih/color` + existing helpers | TTY detection handled automatically |
| Score thresholds | New threshold calculation | Existing `colorForFloatInverse` pattern | Consistent with other categories |
| Metric extraction | Direct map access | Type assertion pattern (`raw.(*types.C7Metrics)`) | Nil-safe, follows existing pattern |
| Test fixtures | New mock data structures | Existing `newTestAnalysisResults()` pattern | Consistent test structure |

**Key insight:** Every pattern needed for C7 rendering already exists in the codebase. The implementation is purely following established conventions.

## Common Pitfalls

### Pitfall 1: Forgetting the Availability Check
**What goes wrong:** Crash or confusing output when C7 was not enabled
**Why it happens:** Direct field access on unavailable metrics
**How to avoid:** Check `m.Available` first, render "Not available" message if false
**Warning signs:** Panic on nil pointer, zero values displayed
```go
if !m.Available {
    fmt.Fprintln(w, "  Not available (--enable-c7 not specified)")
    return
}
```

### Pitfall 2: Missing Switch Case
**What goes wrong:** C7 metrics computed but never displayed
**Why it happens:** Added renderC7 function but forgot switch case in RenderSummary
**How to avoid:** Add both: the function AND the switch case
**Warning signs:** C7 appears in JSON but not terminal output

### Pitfall 3: Inconsistent Metric Formatting
**What goes wrong:** C7 output looks different from other categories
**Why it happens:** Using different spacing, label format, or separator style
**How to avoid:** Copy exact format from renderC5 or renderC6
**Warning signs:** Visual inconsistency in terminal output

### Pitfall 4: Not Updating categoryDisplayNames
**What goes wrong:** RenderScores shows "C7" instead of "Agent Evaluation"
**Why it happens:** Map lookup returns empty string, fallback to category code
**How to avoid:** Add "C7": "Agent Evaluation" to the categoryDisplayNames map
**Warning signs:** Raw "C7" showing in scoring section

### Pitfall 5: Color on Piped Output
**What goes wrong:** ANSI codes appear in piped/redirected output
**Why it happens:** Color applied when stdout is not a TTY
**How to avoid:** `fatih/color` handles this automatically - don't bypass it
**Warning signs:** `^[[32m` garbage in log files

## Code Examples

Verified patterns from existing codebase:

### Header Format (from renderC5)
```go
// Source: internal/output/terminal.go:305-319
fmt.Fprintln(w)
bold.Fprintln(w, "C5: Temporal Dynamics")
fmt.Fprintln(w, "----------------------------------------")

if !m.Available {
    fmt.Fprintln(w, "  Not available (no .git directory)")
    return
}
```

### Metric Display with Color (from renderC4)
```go
// Source: internal/output/terminal.go:390-396
cd := colorForFloatInverse(m.CommentDensity, 5, 15)
cd.Fprintf(w, "  Comment density:     %.1f%%\n", m.CommentDensity)
```

### Verbose Per-Item Breakdown (from renderC6)
```go
// Source: internal/output/terminal.go:469-479
if verbose && len(m.TestFunctions) > 0 {
    fmt.Fprintln(w)
    bold.Fprintln(w, "  Test functions:")
    for _, tf := range m.TestFunctions {
        extDep := ""
        if tf.HasExternalDep {
            extDep = " [external-dep]"
        }
        fmt.Fprintf(w, "    %s.%s  assertions=%d%s  (%s:%d)\n",
            tf.Package, tf.Name, tf.AssertionCount, extDep, tf.File, tf.Line)
    }
}
```

### Test Pattern (from terminal_test.go)
```go
// Source: internal/output/terminal_test.go:145-206
func TestRenderSummaryWithMetrics(t *testing.T) {
    var buf bytes.Buffer
    result := newTestResult()
    analysisResults := newTestAnalysisResults()

    RenderSummary(&buf, result, analysisResults, false)
    out := buf.String()

    c7Checks := []string{
        "C7: Agent Evaluation",
        "Intent clarity:",
        "Modification confidence:",
        "Cross-file coherence:",
        "Semantic completeness:",
    }
    for _, check := range c7Checks {
        if !strings.Contains(out, check) {
            t.Errorf("output missing C7 metric %q\nGot:\n%s", check, out)
        }
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A | fatih/color for TTY-aware coloring | Already in use | No change needed |
| N/A | io.Writer abstraction | Already in use | Testable output |

**Deprecated/outdated:**
- None - the codebase uses current best practices for Go CLI output

## Open Questions

Things that couldn't be fully resolved:

1. **Reasoning truncation in verbose mode**
   - What we know: C7TaskResult.Reasoning can be long (LLM explanation)
   - What's unclear: Should it be truncated? How many characters?
   - Recommendation: Display full reasoning in verbose mode; users opt into verbosity

2. **Color thresholds for C7 metrics**
   - What we know: Scores are 0-100 scale (from LLM 1-10 scaled)
   - What's unclear: Exact threshold values for green/yellow/red
   - Recommendation: Use 70/40 thresholds (matching typical "good/fair/poor")

3. **Display when all tasks failed/timeout**
   - What we know: TaskResults can have status "timeout" or "error"
   - What's unclear: Best UX for partial failures
   - Recommendation: Show completed task scores, flag failed tasks with status

## Sources

### Primary (HIGH confidence)
- `/Users/ingo/agent-readyness/internal/output/terminal.go` - Existing C1-C6 render functions (lines 118-480)
- `/Users/ingo/agent-readyness/pkg/types/types.go` - C7Metrics struct definition (lines 252-274)
- `/Users/ingo/agent-readyness/internal/output/terminal_test.go` - Testing patterns (lines 145-231)
- [fatih/color documentation](https://pkg.go.dev/github.com/fatih/color) - Color library API

### Secondary (MEDIUM confidence)
- `/Users/ingo/agent-readyness/.planning/v2-MILESTONE-AUDIT.md` - Gap identification and requirements
- `/Users/ingo/agent-readyness/.planning/phases/10-c7-agent-evaluation/10-RESEARCH.md` - C7 metrics context

### Tertiary (LOW confidence)
- None - this is entirely an internal codebase pattern application

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Using existing libraries already in codebase
- Architecture: HIGH - Following established patterns from 6 existing renderers
- Pitfalls: HIGH - Based on examination of existing code patterns

**Research date:** 2026-02-03
**Valid until:** 2026-05-03 (90 days - stable internal patterns)
