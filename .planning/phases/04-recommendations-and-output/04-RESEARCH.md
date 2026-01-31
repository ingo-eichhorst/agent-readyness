# Phase 4: Recommendations and Output - Research

**Researched:** 2026-01-31
**Domain:** Go CLI terminal output, recommendation engine, JSON serialization, exit code handling
**Confidence:** HIGH

## Summary

This phase adds three capabilities to the existing ARS pipeline: (1) a recommendation engine that analyzes scored results to generate Top 5 improvement suggestions ranked by composite score impact, (2) enhanced terminal output with score/tier display plus recommendations using ANSI colors and symbols, and (3) CLI flags for `--threshold X` (CI gating with exit code 2) and `--json` (machine-readable output).

The codebase already has significant infrastructure in place. The `internal/output` package already renders file discovery summaries, per-category metric details, and scored results with color coding. The `internal/scoring` package has full breakpoint interpolation, category weighting, composite calculation, and tier classification. The `cmd` package uses Cobra with `--verbose` and `--config` flags already wired. This phase extends these existing systems rather than building from scratch.

The primary technical challenges are: (1) designing a recommendation engine that computes "what-if" score improvements by simulating metric changes, (2) structuring JSON output types that mirror but don't duplicate the terminal rendering, and (3) implementing custom exit codes in Cobra's `RunE` pattern.

**Primary recommendation:** Build a `internal/recommend` package that takes `*types.ScoredResult` plus `*scoring.ScoringConfig` and produces ranked recommendations by simulating metric improvements through the existing interpolation functions. Keep terminal rendering in `internal/output` and add `--threshold`/`--json` flags to `cmd/scan.go`.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/fatih/color` | v1.18.0 | ANSI color output | Already in use; auto-disables on non-TTY |
| `github.com/spf13/cobra` | v1.10.2 | CLI framework | Already in use; RunE pattern for exit codes |
| `encoding/json` | stdlib | JSON output | Standard library, no external dependency needed |
| `text/tabwriter` | stdlib | Aligned column output | Standard library, elastic tabstops for tables |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `fmt` | stdlib | Formatted printing | All terminal output via Fprintf/Fprintln |
| `io` | stdlib | Writer interface | Output abstraction (already used) |
| `sort` | stdlib | Sorting recommendations | Ranking by impact |
| `math` | stdlib | Score clamping/rounding | Display formatting |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `text/tabwriter` | `github.com/olekukonez/tablewriter` | More features but adds dependency; tabwriter sufficient for aligned scores |
| `encoding/json` | `encoding/json/v2` | Experimental, requires GOEXPERIMENT flag; not production ready |
| manual table formatting | `github.com/charmbracelet/lipgloss` | Beautiful output but massive dependency; overkill for score tables |

**Installation:**
No new dependencies needed. All required packages are either already in go.mod or in the Go standard library.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── recommend/           # NEW: recommendation engine
│   ├── recommend.go     # Core recommendation generation logic
│   └── recommend_test.go
├── output/
│   ├── terminal.go      # EXTEND: add RenderRecommendations, update RenderScores
│   ├── json.go          # NEW: JSON output renderer
│   ├── json_test.go
│   └── terminal_test.go # EXTEND: new test cases
├── scoring/             # EXISTING: no changes needed
│   ├── config.go
│   └── scorer.go
├── pipeline/
│   └── pipeline.go      # EXTEND: integrate recommendations, threshold check, JSON mode
cmd/
├── scan.go              # EXTEND: add --threshold, --json flags
└── root.go              # EXTEND: update Execute() for custom exit codes
```

### Pattern 1: Recommendation Engine via Score Simulation
**What:** For each metric with a sub-optimal score, simulate what the composite score would be if that metric improved to a target value (e.g., the next breakpoint tier). The difference is the estimated impact.
**When to use:** Generating ranked improvement recommendations.
**Example:**
```go
// Source: Project-specific pattern based on existing scoring infrastructure
type Recommendation struct {
    Rank            int
    Category        string  // e.g., "C1"
    MetricName      string  // e.g., "complexity_avg"
    CurrentValue    float64
    CurrentScore    float64
    TargetValue     float64 // next breakpoint value
    TargetScore     float64
    ScoreImprovement float64 // estimated composite improvement
    Effort          string  // "Low", "Medium", "High"
    Summary         string  // agent-readiness framed description
    Action          string  // concrete action to take
}

func Generate(scored *types.ScoredResult, cfg *scoring.ScoringConfig) []Recommendation {
    var candidates []Recommendation
    // For each category and sub-score, compute improvement potential
    for _, cat := range scored.Categories {
        for _, ss := range cat.SubScores {
            if !ss.Available || ss.Score >= 9.0 {
                continue // skip unavailable or already-excellent
            }
            // Find the next breakpoint that would improve the score
            // Simulate the composite with improved metric
            // Calculate delta = new_composite - current_composite
        }
    }
    // Sort by ScoreImprovement descending, take top 5
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].ScoreImprovement > candidates[j].ScoreImprovement
    })
    if len(candidates) > 5 {
        candidates = candidates[:5]
    }
    return candidates
}
```

### Pattern 2: Custom Exit Code via Error Type
**What:** Define an `ExitError` type that carries a specific exit code, then check for it in `Execute()`.
**When to use:** `--threshold X` flag that needs exit code 2 when score < X.
**Example:**
```go
// Source: Cobra community pattern (github.com/spf13/cobra/issues/2124)
// In cmd/errors.go or cmd/scan.go
type ExitError struct {
    Code    int
    Message string
}

func (e *ExitError) Error() string { return e.Message }

// In cmd/root.go Execute():
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        var exitErr *ExitError
        if errors.As(err, &exitErr) {
            // Message already printed by RunE
            os.Exit(exitErr.Code)
        }
        os.Exit(1)
    }
}

// In scan RunE:
if threshold > 0 && scored.Composite < threshold {
    // Still render full output first
    return &ExitError{Code: 2, Message: fmt.Sprintf(
        "Score %.1f below threshold %.1f", scored.Composite, threshold)}
}
```

### Pattern 3: JSON Output Struct with Tags
**What:** Define dedicated JSON output structs with `json` tags, separate from internal types, to control the output schema.
**When to use:** `--json` flag for machine-readable output.
**Example:**
```go
// Source: Go encoding/json standard patterns
type JSONReport struct {
    Composite       float64              `json:"composite_score"`
    Tier            string               `json:"tier"`
    Categories      []JSONCategory       `json:"categories"`
    Recommendations []JSONRecommendation `json:"recommendations"`
}

type JSONCategory struct {
    Name      string        `json:"name"`
    Score     float64       `json:"score"`
    Weight    float64       `json:"weight"`
    Metrics   []JSONMetric  `json:"metrics,omitempty"` // only with --verbose
}

// Use json.NewEncoder(w).Encode(report) for streaming output
// Or json.MarshalIndent for pretty-printed output to TTY
```

### Pattern 4: Symbols for Non-Color Accessibility
**What:** Pair colors with Unicode symbols so the output remains meaningful when colors are stripped.
**When to use:** All score displays.
**Example:**
```go
func scoreSymbol(score float64) string {
    if score >= 8.0 {
        return "\u2713" // checkmark
    }
    if score >= 6.0 {
        return "\u26A0" // warning
    }
    return "\u2717" // X mark
}
```

### Anti-Patterns to Avoid
- **Coupling recommendation logic to output rendering:** Keep recommendation generation in its own package; the output package just displays what it receives.
- **Modifying scored data in-place:** Recommendations should be computed from scored data without mutating it. The scored result is read-only input.
- **Using os.Exit() directly in RunE:** This prevents deferred cleanup and testing. Always return errors from RunE and handle exit codes in Execute().
- **Building JSON by string concatenation:** Always use `encoding/json` with proper struct tags. String building leads to escaping bugs.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Column alignment | Manual space padding with Sprintf | `text/tabwriter` | Handles variable-width content, Unicode, dynamic widths |
| TTY detection | Custom isatty checks | `fatih/color` auto-detection | Already handles NO_COLOR env, pipe detection, CI environments |
| JSON serialization | Manual string building | `encoding/json` with struct tags | Handles escaping, nested objects, null values correctly |
| Score interpolation | New interpolation code in recommend pkg | `scoring.Interpolate()` | Already exists, tested, handles edge cases (clamping, empty breakpoints) |
| CLI flag parsing | Manual os.Args parsing | Cobra flag definitions | Already using Cobra, consistent with existing flags |
| Exit code handling | os.Exit() in command body | Custom error type + Execute() handler | Testable, doesn't skip defers |

**Key insight:** The scoring package already has all the computation infrastructure (Interpolate, breakpoints, weights, composite calculation). The recommendation engine should reuse these functions to simulate "what-if" scenarios, not reimplement scoring logic.

## Common Pitfalls

### Pitfall 1: Recommendation Impact Calculation Ignoring Weight Normalization
**What goes wrong:** Calculating impact as just `(new_metric_score - old_metric_score)` ignores that the composite normalizes by sum of active weights (0.60 currently), and each metric has a weight within its category, and categories have weights within the composite.
**Why it happens:** The scoring has three levels of weighting: metric weight within category, category weight, and normalization by active weight sum.
**How to avoid:** Compute full composite both ways: current composite vs. simulated composite with improved metric. The delta IS the impact. Use the actual scoring functions.
**Warning signs:** Recommendations showing unrealistic score improvements (e.g., "+3.0 points" from a single metric).

### Pitfall 2: Color Objects Not Being Reused
**What goes wrong:** Creating `color.New(...)` inside loops for every line of output. Each call allocates. The existing code already shows this issue in some render functions.
**Why it happens:** Convenience of inline creation.
**How to avoid:** Create color objects once at the top of render functions (as done in `RenderSummary`) and reuse them. For score-dependent colors, the `scoreColor()` helper is the right pattern.
**Warning signs:** Many `color.New()` calls inside loops.

### Pitfall 3: tabwriter Not Flushed
**What goes wrong:** Output appears empty or truncated because `tabwriter.Writer` buffers until `Flush()` is called.
**Why it happens:** Forgetting that tabwriter buffers to compute column widths.
**How to avoid:** Always `defer w.Flush()` immediately after creating the writer.
**Warning signs:** Missing table output, or output appearing in wrong order.

### Pitfall 4: JSON Output Containing ANSI Escape Sequences
**What goes wrong:** If `--json` mode uses the same render path that includes `fatih/color` formatting, the JSON output contains ANSI escape codes.
**Why it happens:** JSON mode and terminal mode share code paths.
**How to avoid:** JSON rendering must be a completely separate code path. Never pass colored strings into JSON structs. Set `color.NoColor = true` explicitly when `--json` is active, or better yet, have JSON rendering never touch the color package at all.
**Warning signs:** JSON output containing `\x1b[` sequences.

### Pitfall 5: Threshold Check Before Output Rendering
**What goes wrong:** If `--threshold` check returns an error before rendering, the user sees no output, just an error message.
**Why it happens:** Returning early from RunE on threshold failure.
**How to avoid:** Always render the full output (terminal or JSON) first, THEN check the threshold and return the ExitError. The user needs to see their scores to understand why the threshold failed.
**Warning signs:** `--threshold` mode showing only "Score X.X below threshold Y.Y" with no report.

### Pitfall 6: Effort Estimation Hardcoded Per-Metric Without Context
**What goes wrong:** A metric like "coverage_percent" is marked "Low effort" but the project has 0% coverage and 50k LOC.
**Why it happens:** Effort depends on how far the metric needs to move, not just which metric it is.
**How to avoid:** Base effort on the gap between current and target values, not just the metric name. Large gaps = higher effort. Small gaps = lower effort. Combine with metric-specific baseline effort (e.g., reducing complexity is inherently harder than adding tests).
**Warning signs:** All recommendations showing "Low" effort regardless of gap size.

## Code Examples

### Rendering Recommendations to Terminal
```go
// Builds on existing output package patterns
func RenderRecommendations(w io.Writer, recs []recommend.Recommendation) {
    bold := color.New(color.Bold)

    fmt.Fprintln(w)
    bold.Fprintln(w, "Top Recommendations")
    fmt.Fprintln(w, "════════════════════════════════════════")

    for i, rec := range recs {
        sc := scoreColor(rec.ScoreImprovement) // reuse existing helper
        fmt.Fprintf(w, "\n  %d. %s\n", i+1, rec.Summary)
        sc.Fprintf(w, "     Impact: +%.1f points\n", rec.ScoreImprovement)
        fmt.Fprintf(w, "     Effort: %s\n", rec.Effort)
        fmt.Fprintf(w, "     Action: %s\n", rec.Action)
    }
}
```

### JSON Output with Encoder
```go
// In internal/output/json.go
func RenderJSON(w io.Writer, report *JSONReport) error {
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    return enc.Encode(report)
}
```

### Threshold Gating in Pipeline
```go
// In pipeline Run(), after rendering output
if p.threshold > 0 && p.scored.Composite < p.threshold {
    return &cmd.ExitError{
        Code:    2,
        Message: fmt.Sprintf("Score %.1f is below threshold %.1f", p.scored.Composite, p.threshold),
    }
}
return nil
```

### Agent-Readiness Framing Map
```go
// Maps metric names to agent-readiness-focused descriptions
var agentImpact = map[string]string{
    "complexity_avg":        "High complexity makes functions harder for agents to reason about and modify safely",
    "func_length_avg":       "Long functions exceed agent context windows, forcing partial understanding",
    "file_size_avg":         "Large files make it harder for agents to locate and navigate relevant code",
    "duplication_rate":       "Duplicated code means agents must find and update multiple locations",
    "max_dir_depth":          "Deep directory nesting makes project navigation harder for agents",
    "module_fanout_avg":      "High module coupling means agent changes ripple across many packages",
    "circular_deps":          "Circular dependencies confuse agent dependency analysis",
    "dead_exports":           "Dead exports clutter the API surface agents must understand",
    "test_to_code_ratio":     "Low test coverage means agents cannot verify their changes",
    "coverage_percent":       "Without test coverage data, agents cannot assess change safety",
    "test_isolation":         "Non-isolated tests create flaky failures that block agent workflows",
    "assertion_density_avg":  "Low assertion density means tests may pass despite broken behavior",
}
```

### Impact Estimation via Score Simulation
```go
// Simulate composite score if a single metric improved
func simulateImprovement(scored *types.ScoredResult, cfg *scoring.ScoringConfig,
    catIdx int, metricIdx int, newRawValue float64) float64 {

    // Deep-copy categories to avoid mutation
    cats := make([]types.CategoryScore, len(scored.Categories))
    for i, c := range scored.Categories {
        cats[i] = c
        cats[i].SubScores = make([]types.SubScore, len(c.SubScores))
        copy(cats[i].SubScores, c.SubScores)
    }

    // Find the metric config to get breakpoints
    // Interpolate new score from new raw value
    // Recompute category score from updated sub-scores
    // Recompute composite from updated categories
    // Return new composite
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `fmt.Sprintf` with manual padding | `text/tabwriter` for aligned columns | Always available in stdlib | Cleaner table output |
| `os.Exit()` in command handlers | Custom error types with exit codes | Cobra best practice since v1.x | Testable, defer-safe |
| `encoding/json` v1 | Still `encoding/json` v1 (v2 experimental) | v2 not production-ready | Stick with v1 |
| `NO_COLOR` manual check | `fatih/color` auto-respects NO_COLOR | fatih/color v1.13+ | No manual checks needed |

**Deprecated/outdated:**
- None relevant to this phase. All libraries in use are current versions.

## Open Questions

1. **Effort estimation calibration**
   - What we know: Effort should combine metric-specific difficulty with gap size between current and target values.
   - What's unclear: The exact thresholds for Low/Medium/High effort categorization. Should a 2-point score gap always be "Medium"?
   - Recommendation: Start with a simple model (gap < 1 point = Low, < 2.5 = Medium, else High) and adjust. Include metric-specific multipliers (e.g., reducing complexity is harder than improving test ratio).

2. **JSON output schema stability**
   - What we know: The JSON output will be used by CI tools and other integrations.
   - What's unclear: Whether the schema should be versioned or if it's too early for a stability guarantee.
   - Recommendation: Include a `"version": "1"` field in the JSON output to allow future schema evolution. Keep it simple for now.

3. **--verbose + --json interaction**
   - What we know: Context says Claude's discretion on this.
   - What's unclear: Whether verbose JSON should add per-metric breakdowns or keep the same schema.
   - Recommendation: `--verbose` adds `metrics` array inside each category in JSON output. Without `--verbose`, categories only have name/score/weight. This mirrors the terminal behavior.

## Sources

### Primary (HIGH confidence)
- Existing codebase: `internal/output/terminal.go`, `internal/scoring/scorer.go`, `internal/scoring/config.go`, `cmd/scan.go`, `cmd/root.go`, `internal/pipeline/pipeline.go`, `pkg/types/scoring.go`, `pkg/types/types.go` - Full implementation review
- `github.com/fatih/color` v1.18.0 README - TTY detection, NoColor global, color.New API
- Go stdlib `text/tabwriter` - Column alignment API, Flush requirement
- Go stdlib `encoding/json` - Marshal/MarshalIndent, struct tags, NewEncoder

### Secondary (MEDIUM confidence)
- Cobra exit code patterns - https://github.com/spf13/cobra/issues/2124 - Custom ExitError type pattern verified across multiple sources
- Cobra SilenceUsage/SilenceErrors - https://www.jetbrains.com/guide/go/tutorials/cli-apps-go-cobra/error_handling/

### Tertiary (LOW confidence)
- None. All findings verified against official sources or existing codebase.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in use in the codebase, no new dependencies needed
- Architecture: HIGH - Extends existing patterns visible in codebase; recommendation engine is new but straightforward
- Pitfalls: HIGH - Derived from actual codebase review and established Go patterns
- Recommendation engine: MEDIUM - Impact estimation via simulation is sound but effort calibration is unverified

**Research date:** 2026-01-31
**Valid until:** 2026-03-01 (stable domain, no fast-moving dependencies)
