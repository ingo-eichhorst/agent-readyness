# Architecture Research: C7 Debug Infrastructure

**Domain:** Debug mode integration for C7 agent evaluation
**Researched:** 2026-02-06
**Confidence:** HIGH (based on direct codebase analysis, not external research)

---

## Existing Architecture Overview

The C7 evaluation pipeline follows this execution flow:

```
cmd/scan.go (--enable-c7 flag)
  -> pipeline.Pipeline.SetC7Enabled()
    -> c7_agent.C7Analyzer.Enable(evaluator)
      -> c7_agent.C7Analyzer.Analyze(targets)
        -> agent.RunMetricsParallel(ctx, workDir, targets, progress)
          -> For each M1-M5:
            -> metric.SelectSamples(targets)
            -> metric.Execute(ctx, workDir, samples, executor)
              -> executor.ExecutePrompt(ctx, workDir, prompt, tools, timeout)
                -> agent.CLIExecutorAdapter.ExecutePrompt(...)
                  -> agent.Executor.ExecuteTask(ctx, task)
                    -> `claude -p <prompt> --output-format json`
              -> metric.score*Response(response)  // heuristic scoring
            -> MetricResult{Score, Samples[]{Score, Response, Duration}}
        -> c7_agent.buildMetrics(result, startTime)
          -> types.C7Metrics (stored in AnalysisResult)
  -> pipeline stages: score -> recommend -> output
    -> output.renderC7(w, ar, verbose)     // terminal
    -> output.BuildJSONReport(scored, ...)  // JSON
    -> html.GenerateReport(...)             // HTML
```

### Key Data Types in the Pipeline

| Type | Package | Purpose | Debug Relevance |
|------|---------|---------|-----------------|
| `metrics.MetricResult` | `internal/agent/metrics` | Holds per-metric aggregate | Contains `Samples[]` with `Response` field |
| `metrics.SampleResult` | `internal/agent/metrics` | Holds per-sample outcome | **Already captures `Response string`** |
| `types.C7Metrics` | `pkg/types` | Final output for renderers | Only stores scores/metadata, **not responses** |
| `types.C7MetricResult` | `pkg/types` | Per-metric output for renderers | Stores `Samples []string` (descriptions only) |
| `agent.ParallelResult` | `internal/agent` | Holds all metric results | Full `MetricResult` with samples available |

---

## Critical Insight: Responses Are Already Captured, Then Discarded

The most important architectural finding is that **agent responses are already captured in `SampleResult.Response`** during metric execution, but they are **discarded during the `buildMetrics` transformation** in `internal/analyzer/c7_agent/agent.go`:

```go
// c7_agent/agent.go lines 122-124 -- current code
for _, s := range mr.Samples {
    metricResult.Samples = append(metricResult.Samples, s.Sample.Description)
    // s.Response is AVAILABLE HERE but NOT stored
}
```

The fix is surgical: extend `C7MetricResult` to carry response data and pass it through when debug mode is active.

---

## Integration Point 1: CLI Flag Parsing

### Where: `cmd/scan.go`

**Current pattern:** Flags are declared as package-level `var` in `cmd/scan.go` (lines 15-23), registered in `init()` (lines 124-133), and consumed in the `RunE` closure.

**Existing flags follow a consistent pattern:**
```go
// Declaration (package level)
var enableC7 bool

// Registration (init function)
scanCmd.Flags().BoolVar(&enableC7, "enable-c7", false, "description")

// Consumption (RunE closure)
if enableC7 {
    p.SetC7Enabled()
}
```

**New flag `--debug-c7`:**

Add a `debugC7` boolean flag following the exact same pattern. The flag should:
1. Be declared alongside `enableC7` (line ~21)
2. Be registered in `init()` (line ~133)
3. Imply `--enable-c7` (if `debugC7` is true, also set `enableC7 = true`)
4. Call `p.SetC7Debug(true)` on the pipeline

**Rationale for implying --enable-c7:** Debug mode is useless without C7 running. Rather than requiring `--enable-c7 --debug-c7`, make `--debug-c7` auto-enable C7. This follows the principle of least surprise.

### Files to Modify

| File | Change | Complexity |
|------|--------|------------|
| `cmd/scan.go` | Add `debugC7` var, flag registration, pipeline call | Low |
| `internal/pipeline/pipeline.go` | Add `SetC7Debug(bool)`, pass to C7Analyzer | Low |
| `internal/analyzer/c7_agent/agent.go` | Add `debug bool` field, pass to `buildMetrics` | Low |

### Component Boundary

The flag value flows: `cmd/scan.go` -> `Pipeline` -> `C7Analyzer` -> `buildMetrics`. This is a one-directional data flow with no cross-cutting concerns.

---

## Integration Point 2: Response Capture in Metric Execution

### Where: `internal/agent/metrics/` (M1-M5) and `internal/analyzer/c7_agent/agent.go`

**Current state:** Each metric's `Execute()` method already stores responses:

```go
// Every metric does this (M2 example, m2_comprehension.go lines 156-159):
sr := SampleResult{
    Sample:   sample,
    Response: response,    // <-- ALREADY CAPTURED
    Duration: time.Since(sampleStart),
}
```

The responses flow through `MetricResult.Samples[]` into `RunMetricsParallel` and arrive at `C7Analyzer.buildMetrics()`. The loss happens in `buildMetrics()` where only `s.Sample.Description` is extracted.

### Recommended Approach: Extend `types.C7MetricResult` to carry debug data conditionally

Add a `DebugSamples []C7DebugSample` field to `C7MetricResult`:

```go
// New type in pkg/types/types.go
type C7DebugSample struct {
    FilePath    string  `json:"file_path"`
    Description string  `json:"description"`
    Prompt      string  `json:"prompt"`
    Response    string  `json:"response"`
    Score       int     `json:"score"`
    Duration    float64 `json:"duration"`
}

// Extended in C7MetricResult
type C7MetricResult struct {
    // ... existing fields ...
    DebugSamples []C7DebugSample `json:"debug_samples,omitempty"`
}
```

**Why `omitempty` matters:** In normal mode, this field is nil/empty and omitted from JSON output. No breaking change to existing consumers.

**Alternative considered and rejected: Separate debug log file.** Writing a separate file would decouple debug from the main output pipeline, requiring new infrastructure (file path management, cleanup). The data is already flowing through the pipeline -- just extend the existing types.

### Prompt Capture

The prompts are currently constructed inline within each metric's `Execute()` method (e.g., `m2_comprehension.go` line 143). To capture prompts for debug output, the simplest approach is to store the prompt string alongside each `SampleResult`.

**Recommended: Add `Prompt` field to `SampleResult`.**

```go
type SampleResult struct {
    Sample   Sample
    Score    int
    Response string
    Prompt   string        // NEW: the prompt sent to the agent
    Duration time.Duration
    Error    string
}
```

Each metric already constructs the prompt before calling `executor.ExecutePrompt()`. Just assign `sr.Prompt = prompt` alongside `sr.Response = response`.

**Alternative considered and rejected: Wrapping the executor to intercept prompts.** This would require modifying the `Executor` interface or adding middleware. Overly complex for capturing a string that is already local to the call site.

### Files to Modify

| File | Change | Complexity |
|------|--------|------------|
| `pkg/types/types.go` | Add `C7DebugSample`, extend `C7MetricResult` | Low |
| `internal/agent/metrics/metric.go` | Add `Prompt` field to `SampleResult` | Low |
| `internal/agent/metrics/m1_consistency.go` | Set `sr.Prompt = prompt` | Trivial |
| `internal/agent/metrics/m2_comprehension.go` | Set `sr.Prompt = prompt` | Trivial |
| `internal/agent/metrics/m3_navigation.go` | Set `sr.Prompt = prompt` | Trivial |
| `internal/agent/metrics/m4_identifiers.go` | Set `sr.Prompt = prompt` | Trivial |
| `internal/agent/metrics/m5_documentation.go` | Set `sr.Prompt = prompt` | Trivial |
| `internal/analyzer/c7_agent/agent.go` | Populate `DebugSamples` when debug=true | Medium |

### Data Flow Change

```
BEFORE:
  metric.Execute() -> SampleResult{Response: "..."} -> buildMetrics() -> C7MetricResult{Samples: ["desc"]}

AFTER (debug mode):
  metric.Execute() -> SampleResult{Response: "...", Prompt: "..."} -> buildMetrics(debug=true)
    -> C7MetricResult{Samples: ["desc"], DebugSamples: [{Prompt, Response, Score, ...}]}
```

---

## Integration Point 3: Output Rendering

### Where: `internal/output/terminal.go`, `internal/output/json.go`, `internal/output/html.go`

**Terminal output (`renderC7` in terminal.go, line 531):**

The `verbose` flag already controls per-task breakdown display (lines 601-611). Debug information should be gated on a separate `debug` parameter (not reuse `verbose`), because `verbose` controls detail level for ALL categories and debug is C7-specific with dramatically more output.

**Recommended: Add `debugC7` flag to render functions.**

```go
func renderC7(w io.Writer, ar *types.AnalysisResult, verbose bool, debugC7 bool) {
    // ... existing rendering ...
    if debugC7 {
        renderC7Debug(w, m)
    }
}
```

This requires threading the `debugC7` flag through `RenderSummary` and into `renderC7`. The existing `RenderSummary` signature is:

```go
func RenderSummary(w io.Writer, result *types.ScanResult,
    analysisResults []*types.AnalysisResult, verbose bool)
```

Adding another bool parameter is consistent with the existing pattern. If boolean proliferation becomes a problem later, an options struct can be introduced, but for now one additional bool is acceptable.

### Debug Output Format

For terminal output, the debug section should appear per-metric after the score:

```
  M2 Comprehension:     7/10
    [DEBUG] Sample 1: internal/agent/executor.go
      Prompt (200 chars): Read the file at internal/agent/executor.go and explain...
      Response (500 chars): The file implements the Executor struct which manages...
      Heuristic score: 7 | Duration: 12.3s
    [DEBUG] Sample 2: internal/pipeline/pipeline.go
      ...
```

**For JSON output:** The `debug_samples` field is already structured via `omitempty`. No additional JSON rendering changes needed -- `BuildJSONReport` will include it automatically when present in the `C7MetricResult`, because the types flow through `AnalysisResult` -> scorer -> `ScoredResult`. However, the current JSON report (`JSONReport`) does not directly carry `C7MetricResult` data; it summarizes through `CategoryScore` and `SubScore`. Debug data in JSON mode would need a dedicated debug section or a separate `--debug-c7-json` output. The simpler approach is to add an optional `C7Debug` field to `JSONReport`.

**For HTML output:** Out of scope for this milestone. HTML reports are polished artifacts; debug output belongs in terminal and JSON modes.

### Files to Modify

| File | Change | Complexity |
|------|--------|------------|
| `internal/output/terminal.go` | Add `renderC7Debug()`, thread `debugC7` through `RenderSummary` -> `renderC7` | Medium |
| `internal/pipeline/pipeline.go` | Pass `debugC7` to output renderer calls | Low |
| `internal/output/json.go` | Add optional `C7Debug` field to `JSONReport` | Low |
| `internal/output/html.go` | None (out of scope) | None |

---

## Integration Point 4: Testing Infrastructure

### Where: `internal/agent/metrics/metric_test.go` and new test files

**Current test pattern:**

The existing tests in `metric_test.go` follow these patterns:

1. **Registry tests:** Verify metric count, unique IDs, unique names
2. **SelectSamples tests:** Verify sample selection with crafted `AnalysisTarget` data
3. **Heuristic scoring tests:** Test `score*Response()` functions with known inputs and expected score ranges
4. **Utility function tests:** Test helpers like `calculateVariance`, `countIdentifierWords`

**Existing heuristic tests use broad ranges** (e.g., `minScore: 7, maxScore: 10`). New tests should be more precise.

### Heuristic Scoring Issues Found During Research

These findings should inform both the test cases and eventual scoring fixes:

**M2 `scoreComprehensionResponse` (m2_comprehension.go:189-242):**
- Base score 5, with 13 positive indicators each adding +1, 7 negative indicators each subtracting -1, plus 2 length bonuses
- Maximum theoretical score: 5 + 13 + 2 = 20, clamped to 10
- Problem: Many indicators are common English words ("returns", "error", "loop", "checks"). A mediocre response using typical programming vocabulary easily hits the ceiling
- Problem: Positive/negative indicators are checked independently. A response containing "I'm not sure what this returns" matches both "not sure" (-1) and "returns" (+1), netting zero -- missing the semantics

**M3 `scoreNavigationResponse` (m3_navigation.go:181-254):**
- Base score 5, weighted positive indicators (most +1, "->" gets +2), path count bonuses, negative indicators
- Problem: The `"->"` indicator with weight 2 rewards arrow notation but does not verify it connects actual files
- Problem: Counting "/" as path references is noisy -- Markdown, URLs, and prose all contain "/"

**M4 `scoreIdentifierResponse` (m4_identifiers.go:267-313):**
- Self-reported accuracy adds +2 for "accurate"/"correct"
- Problem: Trusts the agent's self-assessment. An agent that always says "my interpretation was accurate" gets +2 regardless of truth
- The structure check (`verification:`, `accuracy:`) rewards format compliance, not content quality

**M5 `scoreDocumentationResponse` (m5_documentation.go:212-292):**
- Rewards structured responses matching the exact prompt format requested
- Problem: A response with all markdown headings but no actual analysis scores high
- Problem: Length bonus (+1 for >100 words) can be gamed by verbose but empty content

### Test Strategy for Heuristic Scoring

**New test file: `internal/agent/metrics/scoring_test.go`**

Focus areas:

1. **Monotonicity:** A clearly good response must score higher than a clearly bad one
2. **Edge cases:** Empty string, single word, very long response, all-indicators, no-indicators
3. **Boundary testing:** Responses near the clamp boundaries (1 and 10)
4. **Indicator isolation:** Test specific indicators to verify their contribution
5. **Adversarial inputs:** Responses that match indicators syntactically but are semantically poor

Test pattern follows existing codebase conventions:

```go
func TestM2_ScoreComprehensionResponse_Detailed(t *testing.T) {
    m := NewM2Comprehension().(*M2Comprehension)

    tests := []struct {
        name     string
        response string
        wantMin  int
        wantMax  int
    }{
        {
            name:     "all positive indicators saturated",
            response: "The function returns a value after handling errors. It validates input using if conditions in a loop. It iterates for each element, checking edge cases. The side effect modifies the state and updates the record, ensuring the boundary condition is met.",
            wantMin:  10,
            wantMax:  10,
        },
        {
            name:     "only negative indicators",
            response: "I don't know. Unclear. Cannot determine. Might be wrong. Probably. Seems to. Not sure. Unsure.",
            wantMin:  1,
            wantMax:  2,
        },
        {
            name:     "mixed indicators should score middle",
            response: "The function returns data but I'm not sure about edge cases.",
            wantMin:  4,
            wantMax:  7,
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            score := m.scoreComprehensionResponse(tc.response)
            if score < tc.wantMin || score > tc.wantMax {
                t.Errorf("score = %d, want [%d, %d]", score, tc.wantMin, tc.wantMax)
            }
        })
    }
}
```

### Files to Add/Modify

| File | Change | Complexity |
|------|--------|------------|
| `internal/agent/metrics/scoring_test.go` | NEW: Detailed heuristic scoring tests for M2-M5 | Medium |
| `internal/agent/metrics/metric_test.go` | Unchanged (existing tests remain valid) | None |

---

## Recommended Architecture

### Component Diagram

```
cmd/scan.go
  |-- debugC7 bool (CLI flag)
  |
  v
pipeline.Pipeline
  |-- debugC7 bool (stored field)
  |-- SetC7Debug(bool)
  |
  v
c7_agent.C7Analyzer
  |-- debug bool (stored field)
  |-- SetDebug(bool)
  |
  +-> agent.RunMetricsParallel()       // unchanged interface
  |     |
  |     +-> metrics.MetricResult       // unchanged type
  |           |-- Samples []SampleResult
  |                 |-- Prompt string   // NEW field
  |                 |-- Response string // existing field
  |                 |-- Score int       // existing field
  |
  +-> buildMetrics(result, startTime)  // debug-aware
        |
        v
      types.C7Metrics
        |-- MetricResults []C7MetricResult
              |-- DebugSamples []C7DebugSample  // NEW, omitempty
                    |-- FilePath string
                    |-- Description string
                    |-- Prompt string
                    |-- Response string
                    |-- Score int
                    |-- Duration float64
  |
  v
output rendering
  |-- Terminal: renderC7(w, ar, verbose, debugC7)
  |     |-- Normal output (always)
  |     |-- Debug section (when debugC7=true)
  |           Per metric, per sample:
  |             File path, Prompt (truncated), Response (truncated), Score, Duration
  |
  |-- JSON: JSONReport with optional C7Debug field
  |-- HTML: Unchanged
```

### Complete Data Flow

```
1. User runs: ars scan . --debug-c7
2. cmd/scan.go: sets enableC7=true (implied), calls p.SetC7Debug(true)
3. Pipeline: stores debugC7=true, passes to C7Analyzer.SetDebug(true)
4. C7Analyzer.Analyze():
   a. Creates workspace, initializes M1-M5 metrics
   b. RunMetricsParallel() executes M1-M5 (no changes to execution path)
   c. Each metric.Execute() stores prompt in SampleResult.Prompt (new)
   d. buildMetrics() checks debug flag:
      - If debug: populates C7MetricResult.DebugSamples from SampleResult data
      - If not debug: DebugSamples remains nil (omitempty hides it)
5. Pipeline.Run() continues to score -> recommend -> output
6. Output rendering:
   a. Terminal: renderC7() checks debugC7, calls renderC7Debug() for each metric
   b. JSON: C7Debug field included when present
   c. HTML: Unchanged (no debug in polished reports)
```

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Global Debug State

**What:** Using a global variable or environment variable for debug mode.
**Why bad:** Global state makes testing hard and introduces coupling between packages. The existing architecture passes state through explicit method calls (`SetC7Enabled()`) -- debug must follow the same pattern.
**Instead:** Thread the debug flag from CLI -> Pipeline -> C7Analyzer via explicit method calls.

### Anti-Pattern 2: Modifying the Executor Interface

**What:** Adding a "debug mode" to the `metrics.Executor` interface to capture prompts.
**Why bad:** The executor interface is an abstraction boundary for testability. Adding debug concerns violates single responsibility. The prompts are already available in the call site (each `Execute()` method constructs them locally).
**Instead:** Capture the prompt in `SampleResult.Prompt` at the call site where it is already in scope.

### Anti-Pattern 3: Separate Debug Output Stream

**What:** Writing debug output to a separate file, stderr, or logging framework.
**Why bad:** Adds infrastructure complexity (file path management, format choices, cleanup). The debug data is already flowing through the pipeline -- just carry it to the renderers.
**Instead:** Extend existing types to carry debug data conditionally, render inline with normal output.

### Anti-Pattern 4: Response Truncation at Capture Time

**What:** Truncating responses when storing them in `SampleResult` or `C7DebugSample`.
**Why bad:** Loses data that might be needed. Truncation is a presentation concern, not a data concern.
**Instead:** Store full responses in data structures. Truncate only during terminal rendering (e.g., first 500 chars). JSON output should include full responses for programmatic consumption.

### Anti-Pattern 5: Coupling Debug Rendering to Verbose Mode

**What:** Reusing the existing `verbose` flag to gate debug output.
**Why bad:** `verbose` controls detail level for ALL categories (C1-C7). Debug output is C7-specific and dramatically more voluminous (full prompts and responses). A user who wants verbose C1 complexity stats does not want to see C7 agent response dumps.
**Instead:** Separate `debugC7` flag that only affects C7 rendering.

---

## Suggested Build Order

### Phase 1: Flag and Plumbing (Foundation)

**Goal:** Wire the `--debug-c7` flag through the stack without changing behavior.

**Files:** `cmd/scan.go`, `internal/pipeline/pipeline.go`, `internal/analyzer/c7_agent/agent.go`

1. Add `debugC7 bool` var and `--debug-c7` flag to `cmd/scan.go`
2. Add `SetC7Debug(bool)` method to `Pipeline`, store as field
3. Add `SetDebug(bool)` method to `C7Analyzer`, store as field
4. Wire: `scan.go` -> `Pipeline.SetC7Debug()` -> `C7Analyzer.SetDebug()`
5. Auto-enable C7 when debug flag is set (`if debugC7 { enableC7 = true }`)

**Verification:** Run `ars scan . --debug-c7` -- should behave identically to `--enable-c7` (flag plumbed but not yet consumed).

**Depends on:** Nothing (first in chain).

### Phase 2: Data Capture (Prompt Storage + Debug Sample Population)

**Goal:** Capture prompts in SampleResult and populate DebugSamples when debug=true.

**Files:** `internal/agent/metrics/metric.go`, `internal/agent/metrics/m1_consistency.go` through `m5_documentation.go`, `pkg/types/types.go`, `internal/analyzer/c7_agent/agent.go`

1. Add `Prompt string` field to `metrics.SampleResult` in `metric.go`
2. In each M1-M5 `Execute()` method, set `sr.Prompt = prompt` before the executor call
3. Add `C7DebugSample` type to `pkg/types/types.go`
4. Add `DebugSamples []C7DebugSample` to `C7MetricResult` with `json:"debug_samples,omitempty"`
5. In `C7Analyzer.buildMetrics()`: when `debug=true`, populate `DebugSamples` from `SampleResult` data (Prompt, Response, Score, Duration, FilePath, Description)

**Verification:** Run `ars scan . --debug-c7 --json | jq '.categories'` -- JSON output should include `debug_samples` arrays within C7 metric results if the data flows through to JSON. Note: May need Phase 4 JSON changes to verify fully.

**Depends on:** Phase 1 (debug flag must be plumbed).

### Phase 3: Heuristic Scoring Tests

**Goal:** Write comprehensive tests for the scoring functions to document current behavior and expose issues.

**Files:** `internal/agent/metrics/scoring_test.go` (new)

1. Write test cases for `scoreComprehensionResponse` (M2): empty, good, bad, mixed, edge cases
2. Write test cases for `scoreNavigationResponse` (M3): same pattern
3. Write test cases for `scoreIdentifierResponse` (M4): same pattern
4. Write test cases for `scoreDocumentationResponse` (M5): same pattern
5. Include adversarial cases that expose the issues documented above
6. Verify monotonicity: good > mediocre > bad for each scoring function

**Verification:** `go test ./internal/agent/metrics/ -run TestM[2345]_Score -v`

**Depends on:** Nothing (independent of phases 1-2, can run in parallel).

### Phase 4: Debug Output Rendering

**Goal:** Display captured debug data in terminal and JSON output.

**Files:** `internal/output/terminal.go`, `internal/pipeline/pipeline.go`, optionally `internal/output/json.go`

1. Thread `debugC7` through `Pipeline.Run()` to output rendering calls
2. Update `RenderSummary` signature to accept `debugC7 bool`
3. Update `renderC7` to accept and check `debugC7` parameter
4. Add `renderC7Debug(w io.Writer, m *types.C7Metrics)` function
5. Format: per-metric, per-sample: file path, prompt (truncated 200 chars), response (truncated 500 chars), heuristic score, duration
6. For JSON: add optional `C7Debug` section to `JSONReport` (or rely on verbose mode carrying the data through `SubScores`)

**Verification:** Run `ars scan . --debug-c7` -- terminal output shows prompts and response previews after each metric score.

**Depends on:** Phase 1 (flag) and Phase 2 (data capture).

### Phase 5: Scoring Logic Fixes (Informed by Tests)

**Goal:** Fix scoring heuristic issues revealed by Phase 3 tests.

**Files:** `internal/agent/metrics/m2_comprehension.go`, `m3_navigation.go`, `m4_identifiers.go`, `m5_documentation.go`

This phase is informed by Phase 3 test results. Likely fixes:
1. Adjust indicator weights to prevent ceiling saturation
2. Add compound indicators (require multiple words in proximity, not just presence)
3. Reduce self-report trust in M4 (lower weight for "accurate"/"correct")
4. Add discriminating indicators that are less common in generic text
5. Re-run Phase 3 tests with tighter expected ranges to verify improvements

**Verification:** All Phase 3 tests pass with narrower score ranges.

**Depends on:** Phase 3 (tests must exist to validate fixes).

---

## Scalability Considerations

| Concern | Current (5 metrics) | At 10 metrics | At 20 metrics |
|---------|---------------------|---------------|---------------|
| Debug output length | ~50 terminal lines | ~100 lines | Consider `--debug-c7-output FILE` |
| JSON debug payload size | ~10KB | ~20KB | Consider separate debug file |
| Memory for stored responses | Negligible | Negligible | Negligible (strings) |
| Terminal rendering time | Instant | Instant | Instant |

For the current 5-metric system, the inline approach is appropriate. No premature optimization needed.

---

## Sources

All findings are based on direct codebase analysis of the following files:

- `cmd/scan.go` -- CLI flag patterns (lines 15-23 declarations, 124-133 registration, 89-96 C7 handling)
- `internal/pipeline/pipeline.go` -- Pipeline orchestration (lines 26-43 struct, 49-100 New(), 119-124 SetC7Enabled(), 140-288 Run())
- `internal/analyzer/c7_agent/agent.go` -- C7 analyzer (lines 36-87 Analyze(), 98-153 buildMetrics(), 122-124 sample extraction gap)
- `internal/agent/executor.go` -- CLI subprocess execution (lines 42-118 ExecuteTask())
- `internal/agent/executor_adapter.go` -- Executor interface adapter (lines 22-50 ExecutePrompt())
- `internal/agent/parallel.go` -- Parallel metric execution (lines 22-87 RunMetricsParallel())
- `internal/agent/metrics/metric.go` -- Metric interface and SampleResult type (lines 39-57)
- `internal/agent/metrics/m1_consistency.go` -- M1 Execute() and scoring (lines 109-190)
- `internal/agent/metrics/m2_comprehension.go` -- M2 Execute() and scoreComprehensionResponse (lines 121-242)
- `internal/agent/metrics/m3_navigation.go` -- M3 Execute() and scoreNavigationResponse (lines 110-254)
- `internal/agent/metrics/m4_identifiers.go` -- M4 Execute() and scoreIdentifierResponse (lines 196-313)
- `internal/agent/metrics/m5_documentation.go` -- M5 Execute() and scoreDocumentationResponse (lines 131-292)
- `internal/agent/metrics/metric_test.go` -- Existing test patterns (lines 1-488)
- `internal/output/terminal.go` -- Terminal rendering, renderC7() (lines 531-611)
- `internal/output/json.go` -- JSON output structure (lines 12-115)
- `internal/output/html.go` -- HTML report generation (lines 94-138)
- `pkg/types/types.go` -- C7Metrics and C7MetricResult type definitions (lines 253-303)

No external sources were consulted. Confidence is HIGH based on direct code reading with specific line references.
