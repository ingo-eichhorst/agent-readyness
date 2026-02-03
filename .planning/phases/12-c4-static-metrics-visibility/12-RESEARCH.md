# Phase 12: C4 Static Metrics Visibility - Research

**Researched:** 2026-02-03
**Domain:** Go terminal rendering, analyzer availability patterns
**Confidence:** HIGH

## Summary

This phase focuses on making C4 documentation quality metrics visible in terminal output without requiring the `--enable-c4-llm` flag. The current codebase already has C4 analysis and terminal rendering implemented, but the terminal output is always displayed without distinguishing between static and LLM-based metrics.

The key changes needed are:
1. Add `Available` field to C4Metrics (matching C5/C7 pattern)
2. Update renderC4 to show LLM metrics as "N/A" when LLM is disabled
3. Add verbose mode with file-level documentation coverage breakdown

The implementation is straightforward because all patterns exist in the codebase - we are applying established patterns, not inventing new ones.

**Primary recommendation:** Follow the C7 terminal rendering pattern (recently implemented in 11-01-PLAN.md) for displaying opt-in metrics with "N/A" states.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| fatih/color | latest | Terminal color output | Already used throughout terminal.go |
| go standard lib | 1.21+ | io.Writer, fmt | Standard Go patterns |

### Supporting
No additional libraries needed - all required functionality exists in the codebase.

### Alternatives Considered
None - this is an extension of existing patterns, not new technology.

## Architecture Patterns

### Existing File Structure (No Changes Needed)
```
internal/
├── analyzer/
│   └── c4_documentation.go    # Add Available field logic
├── output/
│   ├── terminal.go            # Modify renderC4 function
│   └── terminal_test.go       # Add C4 test coverage
└── pkg/types/
    └── types.go               # Add Available field to C4Metrics
```

### Pattern 1: Availability Flag for Optional Analyzers

**What:** Categories with optional features use an `Available` bool to signal whether full analysis was performed.

**When to use:** When a category has features gated behind flags (like `--enable-c4-llm`, `--enable-c7`).

**Existing examples in codebase:**

```go
// Source: pkg/types/types.go (C5Metrics, line 196)
type C5Metrics struct {
    Available            bool  // false if no .git directory
    ChurnRate            float64
    // ... other fields
}

// Source: pkg/types/types.go (C7Metrics, line 254)
type C7Metrics struct {
    Available              bool  // false if claude CLI not found or user declined
    IntentClarity          int
    // ... other fields
}
```

**Pattern for C4:**
```go
// In pkg/types/types.go - C4Metrics struct
type C4Metrics struct {
    Available       bool    // true when static metrics computed (always true for valid repos)
    LLMEnabled      bool    // true if --enable-c4-llm was used (already exists!)
    // ... rest of fields
}
```

Note: C4Metrics already has `LLMEnabled` field (line 242 in types.go). We need to add `Available` for consistency.

### Pattern 2: Terminal Rendering with Optional Metrics

**What:** Categories show core metrics always, with optional metrics displayed as "N/A" when not enabled.

**When to use:** When category has both static and opt-in (LLM) metrics.

**Existing example (C7 pattern from 11-01-PLAN.md):**

```go
// Source: internal/output/terminal.go (renderC7, lines 495-547)
func renderC7(w io.Writer, ar *types.AnalysisResult, verbose bool) {
    // ... extract metrics

    if !m.Available {
        fmt.Fprintln(w, "  Not available (--enable-c7 not specified)")
        return
    }

    // Render metrics normally when available
    // ...
}
```

**Pattern for C4 (show static metrics, mark LLM as N/A):**
```go
func renderC4(w io.Writer, ar *types.AnalysisResult, verbose bool) {
    // ... existing static metric display

    // LLM metrics section
    if m.LLMEnabled {
        // Show actual LLM scores
    } else {
        // Show "N/A (requires --enable-c4-llm)"
    }
}
```

### Pattern 3: Verbose Mode with Per-File Details

**What:** Non-verbose shows summary metrics; verbose adds per-file breakdown.

**When to use:** When file-level detail is available and useful for debugging.

**Existing examples:**

```go
// Source: internal/output/terminal.go (renderC1, lines 171-201)
if verbose && len(m.Functions) > 0 {
    fmt.Fprintln(w)
    bold.Fprintln(w, "  Top complex functions:")
    // ... list functions
}

// Source: internal/output/terminal.go (renderC6, lines 471-481)
if verbose && len(m.TestFunctions) > 0 {
    fmt.Fprintln(w)
    bold.Fprintln(w, "  Test functions:")
    // ... list test functions
}
```

**Pattern for C4 verbose:**
- Show files missing documentation (undocumented public APIs)
- Similar to how C1 shows "top complex functions"

### Anti-Patterns to Avoid

- **Hiding the category when LLM disabled:** Don't return early or skip C4 entirely - always show static metrics.
- **Making Available=false when LLM is nil:** Static metrics are always computable, so Available should reflect static capability.
- **Inconsistent N/A formatting:** Use the same pattern as C7 for opt-in metrics.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Color thresholds | New helper functions | Existing colorForFloat, colorForFloatInverse | Consistent color coding across categories |
| Metric extraction | Custom extraction | Existing extractC4 in scorer.go | Already handles all C4 metrics |
| Terminal formatting | Custom spacing | Match existing renderC4 patterns | Visual consistency |

**Key insight:** All the infrastructure exists - this is purely wiring existing patterns together.

## Common Pitfalls

### Pitfall 1: Breaking Existing C4 Terminal Output

**What goes wrong:** Changes to renderC4 accidentally remove or alter existing static metric display.

**Why it happens:** renderC4 already exists and displays static metrics; modifications might overwrite working code.

**How to avoid:**
1. Read existing renderC4 carefully (lines 366-436 in terminal.go)
2. Preserve all existing metric display logic
3. ADD LLM metric section, don't restructure existing code

**Warning signs:** Tests for existing C4 output start failing.

### Pitfall 2: Available Field Confusion

**What goes wrong:** Setting Available=false when LLM is nil, causing C4 to be marked as unavailable.

**Why it happens:** Confusion between "LLM not enabled" and "category not available".

**How to avoid:**
- Available=true when static analysis succeeds (which is always, for valid repos)
- LLMEnabled=false when llmClient is nil
- These are separate concerns

**Warning signs:** C4 scores showing as 0 or category being skipped entirely.

### Pitfall 3: Inconsistent N/A Display

**What goes wrong:** LLM metrics show inconsistent formatting compared to how C7 handles unavailable features.

**Why it happens:** Not following established pattern from C7.

**How to avoid:**
- Copy the exact formatting pattern from renderC7
- Use same "Not available" wording style
- Match indentation and color choices

**Warning signs:** Visual inconsistency between C4 and C7 terminal output.

### Pitfall 4: Missing Verbose Mode Implementation

**What goes wrong:** Phase is marked complete but verbose mode doesn't show file-level details.

**Why it happens:** Focusing on basic display and forgetting the verbose requirement from CONTEXT.md.

**How to avoid:**
- Explicitly test verbose output
- Add per-file undocumented API list in verbose mode
- Follow C1/C6 verbose patterns

**Warning signs:** `ars scan . -v` shows same C4 output as non-verbose.

## Code Examples

Verified patterns from existing codebase:

### Existing renderC4 Function (Reference)
```go
// Source: internal/output/terminal.go (lines 366-436)
func renderC4(w io.Writer, ar *types.AnalysisResult, verbose bool) {
    bold := color.New(color.Bold)
    green := color.New(color.FgGreen)
    red := color.New(color.FgRed)

    raw, ok := ar.Metrics["c4"]
    if !ok {
        return
    }
    m, ok := raw.(*types.C4Metrics)
    if !ok {
        return
    }

    fmt.Fprintln(w)
    bold.Fprintln(w, "C4: Documentation Quality")
    fmt.Fprintln(w, "────────────────────────────────────────")

    // README
    if m.ReadmePresent {
        green.Fprintf(w, "  README:              present (%d words)\n", m.ReadmeWordCount)
    } else {
        red.Fprintln(w, "  README:              absent")
    }

    // ... more static metrics (comment density, API docs, CHANGELOG, etc.)

    // Verbose mode (currently just shows counts)
    if verbose {
        fmt.Fprintln(w)
        bold.Fprintln(w, "  Detailed metrics:")
        fmt.Fprintf(w, "    Total source lines:  %d\n", m.TotalSourceLines)
        fmt.Fprintf(w, "    Comment lines:       %d\n", m.CommentLines)
        fmt.Fprintf(w, "    Public APIs:         %d\n", m.PublicAPIs)
        fmt.Fprintf(w, "    Documented APIs:     %d\n", m.DocumentedAPIs)
    }
}
```

### C7 Pattern for Opt-in Metrics (Reference)
```go
// Source: internal/output/terminal.go (lines 495-514)
func renderC7(w io.Writer, ar *types.AnalysisResult, verbose bool) {
    // ... extract metrics

    fmt.Fprintln(w)
    bold.Fprintln(w, "C7: Agent Evaluation")
    fmt.Fprintln(w, "────────────────────────────────────────")

    if !m.Available {
        fmt.Fprintln(w, "  Not available (--enable-c7 not specified)")
        return
    }

    // Core metrics (when available)
    ic := c7ScoreColor(m.IntentClarity)
    ic.Fprintf(w, "  Intent clarity:       %d/100\n", m.IntentClarity)
    // ...
}
```

### C5Metrics Available Pattern (Reference)
```go
// Source: pkg/types/types.go (lines 194-206)
type C5Metrics struct {
    Available            bool
    ChurnRate            float64
    TemporalCouplingPct  float64
    AuthorFragmentation  float64
    CommitStability      float64
    HotspotConcentration float64
    // ...
}

// Source: internal/output/terminal.go (renderC5, lines 323-326)
if !m.Available {
    fmt.Fprintln(w, "  Not available (no .git directory)")
    return
}
```

### Color Helper Pattern for Percentage Metrics
```go
// Source: internal/output/terminal.go (lines 110-118)
// colorForFloatInverse returns a color where higher is better (e.g., coverage).
func colorForFloatInverse(val, redBelow, yellowBelow float64) *color.Color {
    if val < redBelow {
        return color.New(color.FgRed)
    }
    if val < yellowBelow {
        return color.New(color.FgYellow)
    }
    return color.New(color.FgGreen)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| C4 always displayed | C4 displays with LLM opt-in marked | This phase | Users see static metrics without LLM flag |
| No Available field in C4 | Available field for consistency | This phase | Matches C5/C7 pattern |

**Deprecated/outdated:**
- None - this is an enhancement, not a replacement

## Open Questions

No significant open questions - the implementation path is clear:

1. **File-level documentation tracking:** Do we need to track which specific files are missing API docs for verbose mode?
   - What we know: C4Metrics currently tracks aggregate counts (PublicAPIs, DocumentedAPIs) but not per-file breakdown
   - What's unclear: Whether adding per-file tracking is in scope or needs types.go changes
   - Recommendation: Keep verbose mode focused on aggregate counts (matching current verbose output), not per-file lists. This matches the existing verbose implementation and avoids scope creep.

## Sources

### Primary (HIGH confidence)
- `/Users/ingo/agent-readyness/internal/output/terminal.go` - Existing renderC4, renderC5, renderC6, renderC7 implementations
- `/Users/ingo/agent-readyness/pkg/types/types.go` - C4Metrics, C5Metrics, C7Metrics struct definitions
- `/Users/ingo/agent-readyness/internal/analyzer/c4_documentation.go` - C4Analyzer implementation
- `/Users/ingo/agent-readyness/internal/output/terminal_test.go` - Test patterns
- `.planning/phases/11-terminal-output-integration/11-01-PLAN.md` - Recent C7 terminal rendering implementation

### Secondary (MEDIUM confidence)
- `/Users/ingo/agent-readyness/internal/scoring/scorer.go` - extractC4 function (lines 241-278)
- `/Users/ingo/agent-readyness/cmd/scan.go` - CLI flag handling for --enable-c4-llm

### Tertiary (LOW confidence)
- None - all findings from codebase inspection

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Using existing codebase patterns only
- Architecture: HIGH - Patterns verified from existing code (renderC5, renderC7)
- Pitfalls: HIGH - Based on actual code structure and common refactoring errors

**Research date:** 2026-02-03
**Valid until:** Indefinite (internal codebase patterns)

---

## Implementation Checklist (For Planner)

Based on this research, the implementation should:

1. **types.go changes:**
   - Add `Available bool` field to C4Metrics struct (for consistency with C5/C7)

2. **c4_documentation.go changes:**
   - Set `metrics.Available = true` in Analyze() (static metrics always computable)
   - No other changes needed (LLMEnabled already set in runLLMAnalysis)

3. **terminal.go changes (renderC4):**
   - Keep all existing static metric display
   - Add LLM metrics section after static metrics:
     - If LLMEnabled: show actual scores (ReadmeClarity, ExampleQuality, Completeness, CrossRefCoherence)
     - If !LLMEnabled: show "N/A (requires --enable-c4-llm)" for each LLM metric
   - Optionally enhance verbose mode (but current implementation already shows detailed counts)

4. **terminal_test.go changes:**
   - Add C4 to newTestAnalysisResults() (it's currently missing!)
   - Add tests for C4 with LLMEnabled=true and LLMEnabled=false
   - Add tests for verbose mode

5. **Verification:**
   - `go build ./...` compiles
   - `go test ./internal/output/... -v` passes
   - `ars scan .` shows C4 static metrics (no LLM flag)
   - `ars scan . --enable-c4-llm` shows C4 with LLM metrics (requires API key)
