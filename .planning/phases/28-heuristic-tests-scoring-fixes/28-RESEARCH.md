# Phase 28: Heuristic Tests & Scoring Fixes - Research

**Researched:** 2026-02-06
**Domain:** Go heuristic testing, Claude CLI response capture, scoring function debugging
**Confidence:** HIGH

## Summary

Research investigated the current M2/M3/M4 scoring functions, the full execution pipeline from CLI invocation to score display, Go testing patterns for fixture-based heuristic validation, and strategies for capturing real Claude CLI responses.

The investigation uncovered **two distinct bugs** causing the reported "0/10" scores, plus a **scoring saturation issue** that would cause all good responses to score 10/10 with no differentiation:

1. **Bug 1 (likely primary):** The `extractC7` function in `internal/scoring/scorer.go:346` only returns `"overall_score"` (the legacy metric). It does NOT return M1-M5 raw values. The scoring pipeline therefore interpolates 0 for all MECE metrics, producing 0 or 1 in the formal scoring output.

2. **Bug 2 (display path):** The terminal output reads directly from `C7Metrics.CodeBehaviorComprehension` etc., which are set by `buildMetrics` in `agent.go`. If the Execute() functions return `Score: 0` (e.g., because all samples had execution errors), these stay at zero.

3. **Scoring saturation:** All three scoring functions (M2/M3/M4) have so many positive indicators that a typical good response hits 10+ matches, pushing the score from BaseScore(5) to 15-25 before clamping to 10. There is zero discrimination between adequate and excellent responses.

**Primary recommendation:** Fix the root causes first (extractC7 + any execution pipeline issues), validate with real response fixtures, then tune indicator weights to create meaningful score differentiation in the 1-10 range.

## Standard Stack

No new libraries needed. All work uses the existing Go standard library and the project's existing patterns.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `testing` | stdlib | Go test framework | Built-in, table-driven test support |
| `os` | stdlib | File I/O for fixtures | Read fixture files from testdata/ |
| `path/filepath` | stdlib | Portable path handling | Cross-platform testdata paths |
| `strings` | stdlib | String operations | Used by scoring functions already |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` | stdlib | Parse Claude CLI JSON | Fixture files store raw JSON responses |
| `fmt` | stdlib | String formatting | Debug output in test helpers |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Raw fixture files | go:embed | Simpler but adds compile-time dependency; raw files match existing patterns |
| Manual captures | Mock responses | Mocks don't validate against real format; real captures find the actual bugs |

## Architecture Patterns

### Recommended Testdata Structure
```
testdata/
  c7_responses/
    m2_comprehension/
      good_go_explanation.txt      # Real Claude response, well-structured
      good_py_explanation.txt      # Real Claude response for Python file
      minimal_explanation.txt       # Short but correct response
      uncertain_response.txt        # Response with hedging language
    m3_navigation/
      good_dependency_trace.txt    # Real trace with file paths
      shallow_trace.txt             # Only direct imports
      failed_trace.txt              # "Cannot find" type response
    m4_identifiers/
      accurate_interpretation.txt   # Correct interpretation with verification
      partial_interpretation.txt    # Mostly correct, some gaps
      wrong_interpretation.txt      # Misinterpretation
    capture_metadata.json          # Records when/how responses were captured
```

### Pattern 1: Fixture-Based Scoring Tests (Table-Driven)
**What:** Load real responses from testdata files, run scoring functions, assert exact scores
**When to use:** For each M2/M3/M4 scoring function
**Example:**
```go
// Source: Verified against existing codebase patterns (internal/agent/metrics/metric_test.go)
func TestM2_Score_RealFixtures(t *testing.T) {
    fixtureDir := filepath.Join("testdata", "c7_responses", "m2_comprehension")

    tests := []struct {
        name          string
        fixtureFile   string
        expectedScore int  // Exact expected score
        minScore      int  // Acceptable range minimum
        maxScore      int  // Acceptable range maximum
    }{
        {
            name:          "good Go explanation",
            fixtureFile:   "good_go_explanation.txt",
            expectedScore: 8,
            minScore:      7,
            maxScore:      9,
        },
        // ... more test cases
    }

    m := NewM2ComprehensionMetric()

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            response, err := os.ReadFile(filepath.Join(fixtureDir, tc.fixtureFile))
            if err != nil {
                t.Fatalf("failed to read fixture: %v", err)
            }

            score, trace := m.scoreComprehensionResponse(string(response))

            if score < tc.minScore || score > tc.maxScore {
                t.Errorf("score = %d, want [%d, %d]", score, tc.minScore, tc.maxScore)
                // Log trace for debugging
                for _, ind := range trace.Indicators {
                    if ind.Matched {
                        t.Logf("  MATCHED: %s (delta=%d)", ind.Name, ind.Delta)
                    }
                }
            }
        })
    }
}
```

### Pattern 2: ScoreTrace Assertion Helper
**What:** Helper function to assert both score range AND trace invariants
**When to use:** Every scoring test should verify trace consistency
**Example:**
```go
// Source: Derived from existing TestScoreTrace_SumsCorrectly pattern
func assertScoreTrace(t *testing.T, score int, trace ScoreTrace, minScore, maxScore int) {
    t.Helper()

    // 1. Score is within expected range
    if score < minScore || score > maxScore {
        t.Errorf("score = %d, want [%d, %d]", score, minScore, maxScore)
    }

    // 2. Trace is source of truth
    expected := trace.BaseScore
    for _, ind := range trace.Indicators {
        expected += ind.Delta
    }
    if expected < 1 { expected = 1 }
    if expected > 10 { expected = 10 }

    if trace.FinalScore != expected {
        t.Errorf("FinalScore=%d but computed=%d", trace.FinalScore, expected)
    }

    // 3. Score matches trace
    if score != trace.FinalScore {
        t.Errorf("returned score %d != trace.FinalScore %d", score, trace.FinalScore)
    }

    // 4. Delta=0 when Matched=false
    for _, ind := range trace.Indicators {
        if !ind.Matched && ind.Delta != 0 {
            t.Errorf("indicator %q: Matched=false but Delta=%d", ind.Name, ind.Delta)
        }
    }
}
```

### Pattern 3: Real Response Capture Script
**What:** Shell script to capture real Claude CLI responses for fixtures
**When to use:** One-time capture, then check responses into testdata/
**Example:**
```bash
# Capture M2 response against a known file
claude -p "Read the file at internal/scoring/scorer.go and explain what the code does. Focus on: 1. The main purpose/behavior of the code 2. Important control flow paths 3. Error handling and edge cases 4. Return values and side effects. Be specific and reference actual code elements." \
  --allowedTools "Read,Grep" \
  --output-format json \
  | jq -r '.result' > testdata/c7_responses/m2_comprehension/good_go_explanation.txt
```

### Anti-Patterns to Avoid
- **Fabricating test responses:** Don't write fake Claude responses by hand. They won't match the real format and will mask the actual bug. Capture real responses.
- **Testing only happy path:** The 0/10 bug is a pipeline issue. Tests must cover the case where executor returns errors, samples are empty, etc.
- **Over-constraining scores:** Use ranges (minScore, maxScore) not exact scores. Scoring will change when indicators are tuned.
- **Testing indicators in isolation:** Test the full scoreXResponse function, not individual string.Contains checks.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Test fixture loading | Custom file loader | `os.ReadFile(filepath.Join("testdata", ...))` | Go test binary runs in package dir; path is reliable |
| Response format validation | Custom parser | Capture real responses, verify format once | Real responses are the source of truth |
| Score trace verification | Ad-hoc assertions | Reusable `assertScoreTrace` helper | DRY; same invariants apply to all metrics |
| Golden file comparison | Custom diff | `t.Logf` + trace inspection | Scores are numeric; golden files aren't useful here |

**Key insight:** The testdata directory is automatically ignored by `go build` but included by `go test`. Fixture files placed in `testdata/c7_responses/` will be available to tests without any special build configuration.

## Common Pitfalls

### Pitfall 1: Confusing Two Different "Score 0" Sources
**What goes wrong:** Assuming the scoring functions produce 0. They cannot -- clamping ensures [1,10]. A score of 0 means the Execute() function returned 0 because all samples failed or no samples were selected.
**Why it happens:** The scoring functions are the obvious place to look, but the bug is upstream.
**How to avoid:** Write tests that distinguish between "scoring function returned X" and "Execute() returned 0 due to error." Test the scoring functions in isolation with known inputs first.
**Warning signs:** If fixture-based scoring tests all pass (scores 7-10 for good responses) but `ars scan --enable-c7` still shows 0/10, the bug is NOT in the scoring functions.

### Pitfall 2: extractC7 Only Returns Legacy Metrics
**What goes wrong:** The `extractC7` function in `internal/scoring/scorer.go:346` returns ONLY `{"overall_score": m.OverallScore}`. It does NOT return M1-M5 individual scores. This means the formal scoring pipeline has zero-value raw data for all MECE metrics.
**Why it happens:** `extractC7` was written for the legacy 4-task system and was not updated when MECE metrics were added.
**How to avoid:** Update `extractC7` to return all M1-M5 scores as float64 values mapped to their config names.
**Warning signs:** C7 category score is always 0 or 1 in the formal `ScoredResult`, even when terminal output shows non-zero MECE scores.

### Pitfall 3: Scoring Saturation (All Good Responses Get 10/10)
**What goes wrong:** M2 has 20 positive indicators each worth +1. A typical Claude response matches 10-15 of them. Starting from BaseScore=5, this pushes to 15-20 before clamping to 10. There is no differentiation between adequate and excellent.
**Why it happens:** Indicator deltas are all +1 and there are too many of them relative to the [1,10] scoring range.
**How to avoid:** Two approaches:
  1. **Cap positive contribution:** Use fractional weights (e.g., +0.3 per indicator) so matching 10 indicators = +3 not +10
  2. **Reduce BaseScore:** Start from 3 instead of 5, giving more room for positive signals
  3. **Group indicators:** Instead of 20 individual +1 indicators, group into 4-5 categories each worth +1, requiring multiple matches within a category to score
**Warning signs:** All fixture tests produce score 10/10 regardless of response quality.

### Pitfall 4: Testing Against Fabricated Responses
**What goes wrong:** Hand-written test responses match the expected format perfectly, but real Claude CLI responses use different phrasing, structure, or markdown formatting. Tests pass, real usage fails.
**Why it happens:** Developers write responses that contain the exact indicator strings.
**How to avoid:** Capture real Claude CLI responses for test fixtures. Run `claude -p ... --output-format json | jq -r '.result'` to get the actual text.
**Warning signs:** Tests pass but `ars scan --enable-c7` still produces unexpected scores.

### Pitfall 5: Fixture Files In Wrong Directory
**What goes wrong:** Test files cannot find fixtures because the working directory is the package directory, not the project root.
**Why it happens:** `go test` sets CWD to the package containing the test file. `testdata/c7_responses/` at project root requires `../../../testdata/c7_responses/` from `internal/agent/metrics/`.
**How to avoid:** Either use `runtime.Caller(0)` to get file path (existing pattern in codebase), or place testdata under the package directory: `internal/agent/metrics/testdata/c7_responses/`.
**Warning signs:** `os.ReadFile` fails with "no such file or directory" in tests.

### Pitfall 6: Not Testing the Full Execute() Path With Mock Executor
**What goes wrong:** Scoring function tests pass, but Execute() has a different code path that produces 0.
**Why it happens:** Execute() wraps the scoring call with sample selection, timeout management, error aggregation, and averaging. Issues in any of these produce Score=0.
**How to avoid:** Write integration tests that call Execute() with a mockExecutor returning fixture content. Verify that MetricResult.Score is non-zero.
**Warning signs:** `scoreComprehensionResponse` tests pass but `Execute` tests show Score=0.

## Code Examples

### Verified: Current M2 Scoring Function Behavior
```go
// Source: Verified by running scoring functions against simulated realistic responses
// internal/agent/metrics/m2_comprehension.go:191-271

// A typical good Claude response about code hits ~12 of 20 positive indicators
// BaseScore(5) + 12 positive + 0 negative + 2 length = 19 -> clamped to 10
// An empty response: BaseScore(5) + 0 = 5
// The function NEVER returns 0 (min is 1 due to clamping)
```

### Verified: extractC7 Missing MECE Metrics
```go
// Source: internal/scoring/scorer.go:345-366 (read from codebase)

// CURRENT (broken): Only returns legacy overall_score
func extractC7(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
    // ... existing code ...
    return map[string]float64{
        "overall_score": m.OverallScore,
    }, nil
}

// FIXED: Should return all M1-M5 scores for the scoring pipeline
func extractC7(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
    // ... validation ...
    return map[string]float64{
        "overall_score":                    m.OverallScore,
        "task_execution_consistency":       float64(m.TaskExecutionConsistency),
        "code_behavior_comprehension":      float64(m.CodeBehaviorComprehension),
        "cross_file_navigation":            float64(m.CrossFileNavigation),
        "identifier_interpretability":      float64(m.IdentifierInterpretability),
        "documentation_accuracy_detection": float64(m.DocumentationAccuracyDetection),
    }, nil
}
```

### Verified: Response Capture Via Claude CLI
```bash
# Source: https://code.claude.com/docs/en/headless
# Claude CLI with -p and --output-format json returns:
# {"type":"result","session_id":"...","result":"<the agent text response>"}
# The "result" field is what gets passed to scoring functions.

# Capture M2 fixture:
claude -p "Read the file at internal/scoring/scorer.go and explain what the code does. Focus on: 1. The main purpose/behavior of the code 2. Important control flow paths 3. Error handling and edge cases 4. Return values and side effects." \
  --allowedTools "Read,Grep" --output-format json | jq -r '.result' \
  > testdata/c7_responses/m2_comprehension/good_go_explanation.txt

# Capture M3 fixture:
claude -p "Examine the file at internal/pipeline/pipeline.go and trace its dependencies. List all imports, identify what each provides, and trace data flow from one function through other files." \
  --allowedTools "Read,Glob,Grep" --output-format json | jq -r '.result' \
  > testdata/c7_responses/m3_navigation/good_dependency_trace.txt

# Capture M4 fixture:
claude -p "Without reading the file, interpret what the identifier \"NewM2ComprehensionMetric\" means based ONLY on its name. Then read internal/agent/metrics/m2_comprehension.go (line 26) to verify." \
  --allowedTools "Read" --output-format json | jq -r '.result' \
  > testdata/c7_responses/m4_identifiers/accurate_interpretation.txt
```

### Verified: Testdata Location Pattern
```go
// Source: Existing codebase pattern from internal/analyzer/c1_code_quality/codehealth_test.go
// The test working directory is the package directory, so testdata/ is relative to it.

// Option A: Package-local testdata (RECOMMENDED - simpler paths)
// Place at: internal/agent/metrics/testdata/c7_responses/
func loadFixture(t *testing.T, path string) string {
    t.Helper()
    data, err := os.ReadFile(filepath.Join("testdata", "c7_responses", path))
    if err != nil {
        t.Fatalf("failed to load fixture %s: %v", path, err)
    }
    return string(data)
}

// Option B: Project-root testdata (requires path resolution)
// Place at: testdata/c7_responses/
func testdataDir() string {
    _, file, _, _ := runtime.Caller(0)
    return filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata")
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Legacy 4-task C7 (0-100 scale) | 5 MECE metrics (1-10 scale) | Phase 24 | extractC7 not updated, scores broken |
| No scoring trace | ScoreTrace with indicators | Phase 27 | Enables debugging but doesn't fix root cause |
| Fabricated test strings | Real CLI response fixtures | Phase 28 (this phase) | Will validate against actual response format |

**Deprecated/outdated:**
- `OverallScore` (legacy float64 0-100): Still referenced by `extractC7` but should not be the only metric extracted
- `IntentClarity`, `ModificationConfidence`, `CrossFileCoherence`, `SemanticCompleteness`: Legacy task scores, preserved for backward compat but not used in MECE scoring

## Root Cause Analysis

### Why M2/M3/M4 Display "0/10"

Two parallel issues produce the "0/10" display:

**Issue A: Display Path (terminal output)**
The terminal reads `C7Metrics.CodeBehaviorComprehension` directly. This is set by `agent.go:buildMetrics()` from `MetricResult.Score`. If Execute() returns Score=0, it stays 0.

Execute() returns Score=0 only when:
1. `len(samples) == 0` -- "no samples available for evaluation"
2. `successCount == 0` -- "all samples failed" (all executor calls errored)

The scoring functions themselves (scoreComprehensionResponse, etc.) can NEVER return 0 due to clamping to [1,10]. So Score=0 means the execution pipeline failed, not the heuristics.

**Issue B: Formal Scoring Path (ScoredResult)**
`extractC7()` only returns `overall_score`. The M1-M5 breakpoints in config.go look up raw values by name. Since M1-M5 are not in the returned map, `rawValues["code_behavior_comprehension"]` returns 0.0 (Go zero value). Interpolate(breakpoints, 0.0) returns 1.0 (below first breakpoint of Value:1, Score:1).

**Both issues need fixing**, but they are independent. Issue A may resolve itself once the Claude CLI execution succeeds (it may already work). Issue B definitely needs a code fix in extractC7.

### Why Scoring Functions Over-Saturate

The scoring functions have too many positive indicators (M2: 20, M3: 15+, M4: 12+) each worth +1, starting from BaseScore=5. A typical good response matches 10-15 indicators, producing scores of 15-20 before clamping to 10.

The effective scoring range is:
- Terrible response (errors/hedging): ~3-4/10
- Mediocre response (some matches): ~7-8/10
- Good response: 10/10
- Excellent response: 10/10

This means the scoring provides minimal signal for actual quality differences.

## Open Questions

1. **Why does Execute() return Score=0 in production?**
   - What we know: The scoring functions work correctly with realistic text input
   - What's unclear: Whether the issue is empty sample selection, CLI execution errors, or response parsing
   - Recommendation: Run `ars scan . --enable-c7 --debug-c7` (Phase 26 infrastructure) to capture actual prompts, responses, and errors. The debug output will reveal whether it's sample selection, CLI failures, or response issues.

2. **What is the ideal scoring distribution?**
   - What we know: Current scoring over-saturates to 10 for any good response
   - What's unclear: What score differentiation users actually want (should a competent response be 6 or 8?)
   - Recommendation: Capture 5-10 real responses, manually assign expected scores, then tune indicators to match. Use the ScoreTrace to verify each indicator's contribution.

3. **Should testdata fixtures go in package-local or project-root testdata?**
   - What we know: Both patterns exist in the codebase. Project-root `testdata/` is used by discovery/analyzer tests. Package-local would be simpler for metrics tests.
   - Recommendation: Use package-local `internal/agent/metrics/testdata/c7_responses/` for simpler relative paths. Note: the discovery walker skips `testdata/` directories, so project-root would also work.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `internal/agent/metrics/m2_comprehension.go`, `m3_navigation.go`, `m4_identifiers.go` -- scoring function implementations
- Codebase analysis: `internal/scoring/scorer.go:345-366` -- extractC7 function showing only overall_score returned
- Codebase analysis: `internal/scoring/config.go:443-525` -- C7 scoring config with M1-M5 breakpoints
- Codebase analysis: `internal/agent/metrics/metric_test.go` -- existing test patterns
- Diagnostic execution: Scoring functions verified to produce 5/10 for empty input, 10/10 for realistic input (never 0)
- [Claude Code Headless Docs](https://code.claude.com/docs/en/headless) -- JSON output format verification

### Secondary (MEDIUM confidence)
- [Dave Cheney - Test fixtures in Go](https://dave.cheney.net/2016/05/10/test-fixtures-in-go) -- testdata directory conventions
- [Go Testing Patterns Wiki](https://github.com/gotestyourself/gotest.tools/wiki/Go-Testing-Patterns) -- table-driven test patterns

### Tertiary (LOW confidence)
- The actual "0/10 in production" root cause -- requires running with --debug-c7 to confirm whether it's sample selection or CLI execution failure

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- Go stdlib, no new dependencies
- Architecture (testdata structure): HIGH -- follows existing codebase patterns
- Root cause analysis: HIGH -- verified via code reading and diagnostic execution
- Scoring saturation fix approach: MEDIUM -- approach is sound but exact weights need tuning against real data
- Production bug root cause: MEDIUM -- two issues identified with high confidence, but which one causes the user-visible "0/10" needs --debug-c7 confirmation

**Research date:** 2026-02-06
**Valid until:** No expiration (codebase-specific findings, not library-dependent)
