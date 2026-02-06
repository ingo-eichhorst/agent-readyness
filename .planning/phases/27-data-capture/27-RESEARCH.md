# Phase 27: Data Capture - Research

**Researched:** 2026-02-06
**Domain:** Go struct extension, conditional data population, heuristic score tracing
**Confidence:** HIGH

## Summary

Phase 27 adds three data capture capabilities to the existing C7 debug infrastructure: (1) storing the full prompt sent to Claude CLI in each `SampleResult`, (2) ensuring the full response is preserved through the pipeline into `C7MetricResult`, and (3) generating score traces that show exactly which heuristic indicators matched and their individual point contributions.

The critical architectural finding is that **responses are already captured in `SampleResult.Response` during metric execution but discarded during the `buildMetrics` transformation** in `c7_agent/agent.go` (line 136: only `s.Sample.Description` is stored). The prompt is constructed locally in each metric's `Execute()` method but never stored. The fix is surgical: add a `Prompt` field to `SampleResult`, add a `ScoreTrace` type for heuristic indicator tracking, add a `C7DebugSample` type to `pkg/types`, extend `C7MetricResult` with a `DebugSamples` field (using `json:",omitempty"` for zero-cost when debug is off), and populate these fields conditionally in `buildMetrics()` when `a.debug` is true.

The zero-allocation requirement when debug is inactive (success criterion 4) is satisfied by design: `SampleResult.Prompt` is always populated (a single string assignment that occurs alongside the existing prompt construction -- negligible cost), but the expensive `C7DebugSample` slice is only allocated in `buildMetrics()` when `a.debug == true`. The score trace functions replace a simple `int` return with a `(int, ScoreTrace)` return, but `ScoreTrace` is a small struct allocated on the stack when debug is inactive and never escaped to heap.

**Primary recommendation:** Extend `SampleResult` with `Prompt` and `ScoreTrace` fields, add `C7DebugSample` to `pkg/types/types.go`, and conditionally populate `C7MetricResult.DebugSamples` in `buildMetrics()`. Each scoring function (M1-M5) returns a `ScoreTrace` alongside the score. All five metric Execute() methods store prompts. No interface changes. No new dependencies.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `io.Discard` | stdlib | Zero-cost debug writer when disabled | Already established in Phase 26 |
| `fmt.Fprintf` | stdlib | Writing debug content to `io.Writer` | Consistent with Phase 26 debugWriter pattern |
| `encoding/json` (omitempty) | stdlib | Conditional JSON serialization | `DebugSamples` only appears in JSON when populated |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `testing` | stdlib | Unit tests for score trace functions | Verify indicator contributions sum correctly |
| `strings` | stdlib | String matching in heuristic scorers | Already used in all M2-M5 scoring functions |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `Prompt` field on SampleResult | Executor wrapper intercepting prompts | Wrapper adds interface complexity; prompt is already local to call site |
| `ScoreTrace` struct return value | Side-effect writing trace to debugWriter inside scorer | Makes scoring functions impure; harder to test; trace data not available to `buildMetrics()` |
| `C7DebugSample` in `pkg/types` | Separate debug-specific package | Over-engineering; debug data is part of the C7 result, belongs in types |
| `omitempty` for conditional JSON | Separate debug JSON envelope | Adds rendering complexity; `omitempty` achieves same zero-cost goal |

**Installation:**
```bash
# No new dependencies. Zero go.mod changes.
```

## Architecture Patterns

### Recommended Change Topology

```
internal/agent/metrics/metric.go           (add Prompt + ScoreTrace to SampleResult)
internal/agent/metrics/m1_consistency.go   (set sr.Prompt, return ScoreTrace)
internal/agent/metrics/m2_comprehension.go (set sr.Prompt, return ScoreTrace)
internal/agent/metrics/m3_navigation.go    (set sr.Prompt, return ScoreTrace)
internal/agent/metrics/m4_identifiers.go   (set sr.Prompt, return ScoreTrace)
internal/agent/metrics/m5_documentation.go (set sr.Prompt, return ScoreTrace)
pkg/types/types.go                         (add C7DebugSample, extend C7MetricResult)
internal/analyzer/c7_agent/agent.go        (populate DebugSamples in buildMetrics when debug=true)
```

### Pattern 1: Adding Prompt to SampleResult (follow existing Response pattern)

**What:** Add a `Prompt string` field to `SampleResult` and populate it at the same point where `Response` is set.
**When to use:** When data constructed locally in Execute() needs to flow downstream.
**Source:** Direct codebase reading of `internal/agent/metrics/metric.go:39-46` and each metric's Execute().

**Example:**
```go
// internal/agent/metrics/metric.go -- extend SampleResult
type SampleResult struct {
    Sample     Sample
    Score      int
    Response   string        // Agent's response (EXISTING)
    Prompt     string        // NEW: the prompt sent to the agent
    ScoreTrace ScoreTrace    // NEW: heuristic scoring trace
    Duration   time.Duration
    Error      string
}
```

Each metric already constructs the prompt as a local variable before calling `executor.ExecutePrompt()`. The change is one line per metric:

```go
// Example from M2 (m2_comprehension.go Execute method)
prompt := fmt.Sprintf(`Read the file at %s and explain what the code does...`, sample.FilePath)
response, err := executor.ExecutePrompt(sampleCtx, workDir, prompt, "Read,Grep", timePerSample)

sr := SampleResult{
    Sample:   sample,
    Response: response,
    Prompt:   prompt,      // NEW: one line added
    Duration: time.Since(sampleStart),
}
```

### Pattern 2: ScoreTrace Return from Heuristic Scorers

**What:** Change scoring functions to return `(int, ScoreTrace)` instead of just `int`, where `ScoreTrace` documents exactly which indicators matched and their point contributions.
**When to use:** When scoring logic needs to be transparent for debugging.
**Source:** Analysis of all 5 scoring functions (M1 inline, M2-M5 via score*Response methods).

**Example:**
```go
// New type in internal/agent/metrics/metric.go
type ScoreTrace struct {
    BaseScore   int              // Starting score before adjustments
    Indicators  []IndicatorMatch // Each indicator checked and result
    FinalScore  int              // Score after clamping to 1-10
}

type IndicatorMatch struct {
    Name    string // e.g., "positive:returns", "negative:unclear", "length>100"
    Matched bool   // Whether the indicator was found
    Delta   int    // Point contribution: +1, -1, +2, etc.
}
```

The scoring function internals change minimally. Instead of:
```go
score := 5
if strings.Contains(response, "returns") { score++ }
```

It becomes:
```go
score := 5
trace := ScoreTrace{BaseScore: 5}
// ... for each indicator:
matched := strings.Contains(response, "returns")
if matched { score++ }
trace.Indicators = append(trace.Indicators, IndicatorMatch{
    Name: "positive:returns", Matched: matched, Delta: boolToInt(matched),
})
```

**Key insight:** The trace overhead is minimal -- a small slice of structs. When debug is off, the trace is constructed but never rendered or stored beyond the immediate function scope (the `ScoreTrace` in `SampleResult` is populated but `buildMetrics()` only reads it when `a.debug == true`).

**Performance note on success criterion 4:** The `SampleResult.ScoreTrace` is always populated (even when debug is off) because the scoring function runs regardless. However, `ScoreTrace` is a small struct (~200 bytes for 15 indicators) allocated per-sample. This is negligible compared to the Claude CLI response (typically 1-5KB) already stored in `SampleResult.Response`. The expensive allocation -- creating `C7DebugSample` slices and populating them in `buildMetrics()` -- is gated on `a.debug`. If truly zero additional allocations are required when debug is off, an alternative is to have scoring functions accept an optional `*ScoreTrace` pointer (nil when debug off), but this adds complexity to all 5 scoring functions for negligible gain. Recommendation: populate `ScoreTrace` always; gate `DebugSamples` on debug flag.

### Pattern 3: Conditional Debug Data Population in buildMetrics()

**What:** When `a.debug` is true, populate `C7MetricResult.DebugSamples` from the full `SampleResult` data including Prompt, Response, Score, Duration, and ScoreTrace.
**When to use:** When debug data needs to flow from internal types to output types.
**Source:** `c7_agent/agent.go` buildMetrics() lines 112-166.

**Example:**
```go
// In buildMetrics(), within the sample iteration loop
for _, s := range mr.Samples {
    metricResult.Samples = append(metricResult.Samples, s.Sample.Description)

    // NEW: populate debug data when debug mode is active
    if a.debug {
        metricResult.DebugSamples = append(metricResult.DebugSamples, types.C7DebugSample{
            FilePath:    s.Sample.FilePath,
            Description: s.Sample.Description,
            Prompt:      s.Prompt,
            Response:    s.Response,
            Score:       s.Score,
            Duration:    s.Duration.Seconds(),
            ScoreTrace:  convertScoreTrace(s.ScoreTrace),
        })
    }
}
```

### Pattern 4: C7DebugSample Type with omitempty

**What:** New type in `pkg/types/types.go` that carries all debug data for one sample evaluation, with `json:",omitempty"` on the `DebugSamples` slice in `C7MetricResult`.
**When to use:** When optional data needs to be present in JSON only when populated.
**Source:** Analysis of existing `C7MetricResult` in `pkg/types/types.go:294-303`.

**Example:**
```go
// pkg/types/types.go -- new type
type C7DebugSample struct {
    FilePath    string            `json:"file_path"`
    Description string            `json:"description"`
    Prompt      string            `json:"prompt"`
    Response    string            `json:"response"`
    Score       int               `json:"score"`
    Duration    float64           `json:"duration_seconds"`
    ScoreTrace  C7ScoreTrace      `json:"score_trace"`
}

type C7ScoreTrace struct {
    BaseScore  int                `json:"base_score"`
    Indicators []C7IndicatorMatch `json:"indicators"`
    FinalScore int                `json:"final_score"`
}

type C7IndicatorMatch struct {
    Name    string `json:"name"`
    Matched bool   `json:"matched"`
    Delta   int    `json:"delta"`
}

// Extend existing C7MetricResult
type C7MetricResult struct {
    MetricID     string          `json:"metric_id"`
    MetricName   string          `json:"metric_name"`
    Score        int             `json:"score"`
    Status       string          `json:"status"`
    Duration     float64         `json:"duration"`
    Reasoning    string          `json:"reasoning"`
    Samples      []string        `json:"samples"`
    DebugSamples []C7DebugSample `json:"debug_samples,omitempty"` // NEW
}
```

### Pattern 5: M1 Consistency Special Case (inline scoring, no score*Response method)

**What:** M1 does not have a separate scoring function -- it scores inline in Execute() with simple string prefix/suffix checks. The score trace for M1 tracks the JSON format checks and variance calculation.
**When to use:** When a metric has inline scoring instead of a dedicated scoring function.
**Source:** `m1_consistency.go` lines 148-158 (inline scoring logic).

**Example:**
```go
// M1 inline scoring with trace
trace := ScoreTrace{BaseScore: 0}

if strings.HasPrefix(response, "[") && strings.HasSuffix(response, "]") {
    sr.Score = 10
    trace.Indicators = append(trace.Indicators, IndicatorMatch{
        Name: "json_array_format", Matched: true, Delta: 10,
    })
} else if strings.Contains(response, "[") {
    sr.Score = 7
    trace.Indicators = append(trace.Indicators, IndicatorMatch{
        Name: "partial_json_array", Matched: true, Delta: 7,
    })
}
// ... etc

// After all runs: variance-based aggregate scoring also traced
switch {
case variancePct < 5:
    result.Score = 10
    // trace for aggregate: "variance<5%"
}
```

### Anti-Patterns to Avoid

- **Modifying the Executor interface:** Do NOT add debug parameters to `metrics.Executor.ExecutePrompt()`. The executor is an abstraction boundary for testability. Prompts are already in scope at the call site.
- **Truncating responses at capture time:** Store full responses in data structures. Truncation is a presentation concern for Phase 29 (rendering). Full data is needed for Phase 28 (testing with real fixtures) and Phase 29 (replay).
- **Wrapping the Executor to intercept prompts:** The prompt is already a local variable in each `Execute()` method. A wrapper adds unnecessary indirection.
- **Creating separate debug log files in this phase:** Phase 27 is about data capture into existing pipeline types, not file I/O. File persistence is Phase 29 (`--debug-dir`).
- **Using debug flag to conditionally populate Prompt field:** Always populate `SampleResult.Prompt`. It is a single string assignment (negligible cost). The debug flag gates only the expensive `C7DebugSample` construction in `buildMetrics()`.
- **Making ScoreTrace a pointer type:** Use value type. The struct is small (~200 bytes) and benefits from stack allocation. Pointer indirection adds GC pressure for no gain.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Conditional JSON fields | Custom JSON marshaler with debug check | `json:",omitempty"` tag | stdlib handles nil/empty slice omission automatically |
| Debug data gating | `if debug { ... }` checks scattered in metrics | Single gate in `buildMetrics()` | Centralizes debug logic; metrics stay clean |
| Prompt capture | Executor middleware/interceptor | Direct `sr.Prompt = prompt` assignment | Prompt is already a local variable; no indirection needed |
| Score trace formatting | Custom string formatting in each scorer | Shared `ScoreTrace` struct with standard JSON marshaling | Consistent structure across all 5 metrics |
| Score indicator tracking | Manual bookkeeping alongside scoring | `IndicatorMatch` slice appended during scoring loop | Follows natural structure of the scoring code |

**Key insight:** The data already exists at every point in the pipeline. Phase 27 is about carrying it through instead of discarding it. No new infrastructure is needed -- just extending existing types.

## Common Pitfalls

### Pitfall 1: Breaking JSON Serialization by Adding Non-omitempty Fields

**What goes wrong:** Adding `DebugSamples []C7DebugSample` without `omitempty` causes empty `[]` arrays in JSON output when debug is off. This changes the JSON schema for existing consumers.
**Why it happens:** Developer forgets `omitempty` or uses a pointer type that serializes as `null` instead of being omitted.
**How to avoid:** Use `json:"debug_samples,omitempty"` on the slice field. A nil slice serializes as absent (not `null` or `[]`). Test: `ars scan . --enable-c7 --json | jq '.categories'` should NOT contain `debug_samples` key when debug is off.
**Warning signs:** JSON output changes when debug is off.

### Pitfall 2: Forgetting to Populate Prompt in One of the 5 Metrics

**What goes wrong:** 4 out of 5 metrics set `sr.Prompt = prompt` but one is missed. Debug output shows empty prompts for that metric.
**Why it happens:** The change is mechanical across 5 files. Easy to miss one.
**How to avoid:** After modifying all 5 files, write a test that creates a mock executor, runs each metric, and asserts `SampleResult.Prompt != ""` for all successful samples.
**Warning signs:** One metric's debug samples have empty `prompt` fields.

### Pitfall 3: ScoreTrace Indicators Not Matching Actual Scoring Logic

**What goes wrong:** The score trace says indicator X contributed +1 but the actual score calculation has a different value, or an indicator is checked in scoring but not tracked in the trace.
**Why it happens:** The trace construction and scoring logic are in the same function but maintained separately. They can diverge.
**How to avoid:** Refactor scoring so the trace IS the scoring mechanism. Instead of parallel code paths, iterate over `trace.Indicators` to compute the final score from the same data structure. The score trace becomes the source of truth, not a parallel record.
**Warning signs:** Sum of `IndicatorMatch.Delta` values + `BaseScore` does not equal `FinalScore`.

### Pitfall 4: Performance Regression from ScoreTrace Allocation

**What goes wrong:** Creating `ScoreTrace` with 15+ `IndicatorMatch` entries per sample causes measurable allocation overhead even when debug is off.
**Why it happens:** The scoring function always returns `ScoreTrace` regardless of debug mode.
**How to avoid:** Keep `IndicatorMatch` small (3 fields, no heap-allocated slices). Pre-allocate the `Indicators` slice with `make([]IndicatorMatch, 0, 20)`. At ~15 indicators per scorer and 1-5 samples per metric, total allocation is < 5KB per metric execution -- negligible vs the Claude CLI call overhead (seconds). But if success criterion 4 is interpreted strictly ("no additional allocations"), pass a `*ScoreTrace` pointer that is nil when debug is off, and only populate when non-nil.
**How to verify:** `go test -bench=BenchmarkScoring -benchmem` before and after, confirm zero additional allocations when ScoreTrace pointer is nil.

### Pitfall 5: M1 Aggregate Score Trace Confusion

**What goes wrong:** M1 runs 3 samples of the same file and computes an aggregate variance-based score. The per-sample trace and the aggregate trace are conflated.
**Why it happens:** M1 has two levels of scoring: per-run (JSON format check) and aggregate (variance calculation). Other metrics only have per-sample scoring.
**How to avoid:** M1's `SampleResult.ScoreTrace` tracks the per-run JSON format check. The aggregate variance scoring goes into the `MetricResult`-level trace (or is computed from the per-sample traces in `buildMetrics()`). Document this distinction clearly.
**Warning signs:** M1 debug output shows only per-run traces but not the variance calculation that produces the final score.

### Pitfall 6: Concurrent Access to Debug Data

**What goes wrong:** `RunMetricsParallel()` runs 5 metrics concurrently. If debug data writing is not thread-safe, race conditions occur.
**Why it happens:** The parallel executor uses `sync.Mutex` to protect `result.Results[i]` but debug data flows through the same `MetricResult` type.
**How to avoid:** No additional synchronization needed. Each metric writes to its own `MetricResult` (indexed by `i`) which is already protected by the mutex in `parallel.go:57-71`. The `buildMetrics()` call happens AFTER `RunMetricsParallel()` returns, so all results are settled. Debug data population in `buildMetrics()` is single-threaded.
**Warning signs:** `-race` flag detects races during `go test -race ./...`.

## Code Examples

Verified patterns from the existing codebase:

### Current SampleResult (to be extended)
```go
// Source: internal/agent/metrics/metric.go lines 39-46
type SampleResult struct {
    Sample   Sample
    Score    int
    Response string
    Duration time.Duration
    Error    string
}
```

### Current buildMetrics sample extraction (the discard point)
```go
// Source: internal/analyzer/c7_agent/agent.go lines 134-137
for _, s := range mr.Samples {
    metricResult.Samples = append(metricResult.Samples, s.Sample.Description)
    // s.Response is AVAILABLE but NOT stored
    // s.Prompt does NOT exist yet
}
```

### Current M2 scoring function signature (to be extended)
```go
// Source: internal/agent/metrics/m2_comprehension.go lines 189
func (m *M2Comprehension) scoreComprehensionResponse(response string) int {
    // ... indicator checks returning just an int
}
```

### Current M2 Execute prompt construction and scoring
```go
// Source: internal/agent/metrics/m2_comprehension.go lines 139-172
for _, sample := range samples {
    prompt := fmt.Sprintf(`Read the file at %s and explain what the code does...`, sample.FilePath)
    response, err := executor.ExecutePrompt(sampleCtx, workDir, prompt, "Read,Grep", timePerSample)

    sr := SampleResult{
        Sample:   sample,
        Response: response,
        Duration: time.Since(sampleStart),
    }
    // prompt is local variable -- not stored
    // sr.Score set from m.scoreComprehensionResponse(response) -- trace not captured
}
```

### Existing C7MetricResult (to be extended)
```go
// Source: pkg/types/types.go lines 294-303
type C7MetricResult struct {
    MetricID   string   `json:"metric_id"`
    MetricName string   `json:"metric_name"`
    Score      int      `json:"score"`
    Status     string   `json:"status"`
    Duration   float64  `json:"duration"`
    Reasoning  string   `json:"reasoning"`
    Samples    []string `json:"samples"`
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Responses discarded after scoring | Responses preserved for debug rendering | Phase 27 (this phase) | Enables debug output (Phase 29) and fixture capture (Phase 28) |
| Scoring returns only int | Scoring returns (int, ScoreTrace) | Phase 27 (this phase) | Enables trace rendering showing indicator contributions |
| Debug data not in type system | `C7DebugSample` with `omitempty` | Phase 27 (this phase) | Zero-cost when disabled, structured when enabled |

**Deprecated/outdated:**
- Nothing deprecated. All changes are additive to existing types.

## Open Questions

1. **Strict interpretation of "no additional allocations" (success criterion 4)**
   - What we know: Adding `Prompt string` and `ScoreTrace` to `SampleResult` happens on every execution (debug or not). `Prompt` is a single string assignment (negligible). `ScoreTrace` allocates a small slice of `IndicatorMatch` entries.
   - What's unclear: Does "no additional allocations" mean literally zero, or negligible? The Claude CLI call itself takes seconds and allocates megabytes of process memory. A 200-byte `ScoreTrace` is noise by comparison.
   - Recommendation: Use pointer-based `*ScoreTrace` field on `SampleResult`. Pass `nil` when `debug == false` in the Execute() method. Scoring functions accept `*ScoreTrace` and skip trace building when nil. This achieves strict zero additional allocations but requires threading a debug flag to each metric's Execute() method or making ScoreTrace optional. The simpler approach (always populate) is recommended unless strict zero-alloc is enforced.

2. **Should the Metric interface change to accept debug flag?**
   - What we know: Currently `Execute(ctx, workDir, samples, executor)` has no debug parameter. Adding one changes the interface all metrics implement.
   - What's unclear: Whether interface stability is more important than clean debug threading.
   - Recommendation: Do NOT change the `Metric` interface. Instead, have the concrete metric structs (M1-M5) receive debug state via a field set during construction or via a `SetDebug()` method, similar to how `C7Analyzer` received its debug flag. The `AllMetrics()` registry would need to support debug propagation. Alternatively, since ScoreTrace is cheap, always populate it.

3. **M1's dual-level scoring (per-run vs aggregate)**
   - What we know: M1 runs the same task 3 times, scores each run (JSON format check), then computes a variance-based aggregate. The aggregate scoring logic is separate from the per-sample trace.
   - What's unclear: Should the aggregate variance calculation also produce a trace?
   - Recommendation: Yes. Add a `MetricLevelTrace` or use the existing `Reasoning` field in `C7MetricResult` to capture the aggregate logic (e.g., "variance=2.67, threshold=<5%, score=10"). Per-sample traces handle per-run scoring.

## Sources

### Primary (HIGH confidence)

All findings verified by direct codebase reading:

- `internal/agent/metrics/metric.go` -- SampleResult type (lines 39-46), Metric interface (lines 19-27), Executor interface (lines 60-62)
- `internal/agent/metrics/m1_consistency.go` -- Execute() with inline scoring (lines 109-190), no separate scoring function
- `internal/agent/metrics/m2_comprehension.go` -- Execute() (lines 121-186), scoreComprehensionResponse() (lines 189-242), 13 positive + 7 negative indicators
- `internal/agent/metrics/m3_navigation.go` -- Execute() (lines 110-178), scoreNavigationResponse() (lines 181-254), weighted indicators
- `internal/agent/metrics/m4_identifiers.go` -- Execute() (lines 196-264), scoreIdentifierResponse() (lines 267-313), self-report accuracy check
- `internal/agent/metrics/m5_documentation.go` -- Execute() (lines 131-209), scoreDocumentationResponse() (lines 212-292), structured response checks
- `internal/analyzer/c7_agent/agent.go` -- buildMetrics() (lines 112-166), sample data discard point (lines 134-137), debug/debugWriter fields (lines 19-20)
- `pkg/types/types.go` -- C7MetricResult type (lines 294-303), C7Metrics type (lines 253-282)
- `internal/agent/parallel.go` -- RunMetricsParallel() (lines 22-87), mutex-protected result storage
- `internal/agent/metrics/registry.go` -- AllMetrics() singleton pattern (lines 4-15)
- `.planning/research/ARCHITECTURE-debug.md` -- Prior architecture research with detailed data flow
- `.planning/phases/26-debug-foundation/26-01-SUMMARY.md` -- Phase 26 completion state

### Secondary (HIGH confidence)

- `.planning/research/FEATURES-c7-debug.md` -- Feature landscape, ScoreTrace design
- `.planning/ROADMAP.md` -- Phase 27 success criteria and requirements
- `.planning/REQUIREMENTS.md` -- DBG-04, DBG-05, DBG-06 requirement text

### Tertiary (N/A)

No external sources consulted. All research is codebase-internal. Confidence is HIGH.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Zero new dependencies. All patterns verified in existing codebase.
- Architecture: HIGH - Direct analysis of all 8 files that need modification, with line-level references.
- Pitfalls: HIGH - 6 specific pitfalls identified from codebase analysis and prior architecture research.

**Research date:** 2026-02-06
**Valid until:** 2026-03-06 (stable; no external dependencies, all internal patterns)
