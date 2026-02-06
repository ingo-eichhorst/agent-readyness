# Feature Research: C7 Debug Modes and Heuristic Testing

**Domain:** CLI debug infrastructure for agent evaluation metrics
**Researched:** 2026-02-06
**Confidence:** HIGH (based on codebase analysis + established CLI patterns)

## Context

The C7 agent evaluation runs 5 MECE metrics (M1-M5) via Claude CLI. Metrics M2, M3, M4, and M5 score responses using keyword-based heuristics (`scoreComprehensionResponse`, `scoreNavigationResponse`, `scoreIdentifierResponse`, `scoreDocumentationResponse`). These heuristics produce scores but offer no visibility into why a score was assigned. Currently, metric scores are appearing as 0/10 with no way to see the raw agent responses or trace through the scoring logic.

The existing infrastructure already captures `Response` strings in `SampleResult` (defined in `internal/agent/metrics/metric.go:40-46`) but discards them after scoring. The data is there -- it just needs a path to the user.

## Feature Landscape

### Table Stakes (Users Expect These)

Features developers expect from any debug-capable CLI tool. Missing these makes the debug workflow impossible.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| `--debug` flag on scan command | Standard CLI convention; users need a single switch to enable diagnostic output | LOW | Add `BoolVar` to `cmd/scan.go` alongside existing `--verbose`. Thread through `Pipeline` to `C7Analyzer`. Already have `--verbose` pattern to follow. |
| Raw response logging to stderr | When debugging scoring, seeing the actual agent response is essential; without it, heuristic tuning is blind | LOW | `SampleResult.Response` is already captured in `metric.go:43`. Just needs conditional `fmt.Fprintf(os.Stderr, ...)` gated on debug flag. Write to stderr so it does not corrupt stdout JSON output. |
| Per-metric score breakdown in debug output | Knowing the final score per metric is not enough; need to see which heuristic indicators matched and which missed | MEDIUM | Each `score*Response` function needs to return a trace of which indicators matched. Currently returns only `int`. Change to return `(int, []string)` or a `ScoreTrace` struct. |
| Debug output to stderr (not stdout) | Debug output must not break `--json` or piped output; Go CLI convention is diagnostic info on stderr, structured output on stdout | LOW | Already follows this pattern: spinner writes to `os.Stderr`, main output to `cmd.OutOrStdout()`. Debug output should use `os.Stderr` exclusively. |
| Response dump to file (`--debug-dir`) | For long agent responses (500+ words from M2/M3), terminal output is unreadable; need to dump to files for offline analysis | LOW | Write `{metric_id}_{sample_index}.txt` files to a specified directory. Include prompt, response, and score trace. |
| Heuristic scoring unit tests with realistic fixtures | Current tests use synthetic strings (e.g., `"The function returns the result after handling errors."`). Need tests with actual Claude CLI response patterns to validate heuristic logic. | MEDIUM | Create `testdata/c7_responses/` with golden response files per metric. Load in `metric_test.go` table-driven tests. Existing test pattern in `metric_test.go:329-487` provides the structure. |

### Differentiators (Competitive Advantage)

Features that make this debug infrastructure notably useful, beyond the minimum.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Response replay mode (`--replay-from`) | Re-score previously captured responses without running Claude CLI again. Eliminates $0.15+ cost per debug iteration. Enables rapid heuristic tuning cycle: capture once, iterate scoring offline. | MEDIUM | Load saved responses from `--debug-dir` files, feed into `score*Response` functions. Need a `MockExecutor` that reads from files. The `metrics.Executor` interface already abstracts CLI execution -- implement a `FileReplayExecutor`. |
| Score trace annotations in JSON output | When `--json --debug` are combined, embed the score trace directly into the JSON structure. Enables programmatic analysis of why scores are what they are. | MEDIUM | Extend `C7MetricResult` type (in `pkg/types/types.go`) with optional `DebugInfo` field. Include matched/unmatched indicators, base score, adjustments. Only populated when debug is active. |
| Side-by-side prompt + response display | Format debug output showing the prompt sent alongside the response received, making it easy to verify the agent followed instructions. | LOW | Template: `--- PROMPT [M2, sample 1] ---\n{prompt}\n--- RESPONSE [score: 7] ---\n{response}\n--- TRACE ---\n{indicators}`. Print to stderr. |
| Heuristic sensitivity report | Run all fixture responses through scoring, report which indicators are never triggered (dead code in heuristics) and which always trigger (noise). | LOW | Post-test analysis function that aggregates indicator match rates across all fixtures. Add as a test helper, not a CLI feature. |
| Golden file test infrastructure | Use `goldie`-style testing where expected scores are stored in golden files alongside responses. Running with `-update` regenerates goldens after heuristic changes. | MEDIUM | Leverage Go's `testdata/` convention. Store as `{metric_id}_{case_name}.response` + `{metric_id}_{case_name}.golden` pairs. Use `go test -update` flag pattern. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem helpful for debugging but create more problems than they solve.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Interactive step-through debugger for scoring | "I want to step through each keyword check" | Massively complex to build; Go has `delve` for this already. Adding custom step-through logic is a maintenance burden that duplicates existing tooling. | Use score trace output (showing which indicators matched) combined with Go's standard debugger when deep investigation is needed. The trace gives you 95% of what step-through would. |
| Real-time response streaming from Claude CLI | "Show me the response as it's generated" | Claude CLI in headless mode (`-p` flag) returns complete JSON after execution, not streaming tokens. Would require switching to SDK/API integration, fundamentally changing the executor architecture. Also pointless for debugging heuristics -- you need the complete response to score it. | Dump responses to file after completion. The response is typically available within 10-60 seconds. |
| Complex debug query language / filters | "Let me filter debug output like `--debug='M2.indicators.positive'`" | Over-engineering for a tool with 5 metrics and ~15 indicators each. The query language itself becomes a maintenance burden and documentation requirement that exceeds the value. | Use `--debug` for everything, pipe through `grep` for filtering. Unix philosophy: simple tools, compose with pipes. |
| Automatic heuristic tuning / ML-based scoring | "Use the captured responses to automatically adjust keyword weights" | Premature optimization. The heuristic functions have ~15 indicators each. Manual tuning with visibility into responses is the right approach until the heuristics are proven inadequate. ML scoring adds massive complexity for marginal gain at this scale. | Fix the current heuristics first by seeing actual responses. Upgrade to LLM-as-judge scoring (already designed with rubrics in `m2_comprehension.go:103-118`) once heuristics are proven insufficient. |
| Debug log levels (DEBUG/TRACE/INFO) | "Different verbosity levels for different detail" | The existing `--verbose` flag already controls one level. Adding multi-level debug creates cognitive overhead (which level shows what?) and code complexity (checking levels everywhere). With only 5 metrics, there is not enough content to warrant level granularity. | Two modes: normal (no debug output) and debug (all debug output to stderr). The `--verbose` flag already exists for the scan summary; `--debug` handles C7 internals. Clean separation. |
| Persistent debug configuration file | "Save my debug preferences in .arsrc.yml" | Debug is a transient developer activity, not a project configuration. Persisting debug state leads to accidentally shipping noisy output in CI, or forgetting it is enabled. | Flags only: `--debug` and `--debug-dir`. Transient by design. If someone wants it persistent, they can alias the command. |

## Feature Dependencies

```
[--debug flag]
    |
    |-- enables --> [Raw response logging to stderr]
    |                   |
    |                   +-- enhances --> [Side-by-side prompt + response display]
    |
    |-- enables --> [Per-metric score breakdown]
    |                   |
    |                   +-- feeds --> [Score trace in JSON output] (when --json also set)
    |
    +-- optionally uses --> [--debug-dir path]
                               |
                               +-- writes --> [Response dump files]
                               |
                               +-- reads from --> [Response replay mode] (--replay-from)

[Heuristic scoring unit tests]
    |
    +-- requires --> [Realistic response fixtures in testdata/]
    |
    +-- enhances --> [Golden file test infrastructure]
    |
    +-- informs --> [Heuristic sensitivity report]

[Response replay mode]
    +-- requires --> [Response dump files exist]
    +-- requires --> [FileReplayExecutor implementing metrics.Executor]
```

### Dependency Notes

- **`--debug` flag requires nothing new:** It is a simple `BoolVar` threaded through the pipeline. All other debug features are gated behind it.
- **Raw response logging requires `--debug`:** Without the flag gate, response logging would pollute normal output. The gate ensures opt-in behavior.
- **Response replay requires response dumps:** You cannot replay what was not captured. The `--debug-dir` capture must run first to produce the files that `--replay-from` consumes.
- **Score trace in JSON requires both `--debug` and `--json`:** Adding debug info to JSON is only meaningful when both flags are active. Without `--json`, the trace goes to stderr instead.
- **Heuristic tests are independent of the debug flag:** Tests run in `go test`, not via the CLI. They need fixtures but do not depend on runtime debug infrastructure.
- **Golden file infrastructure enhances heuristic tests:** Golden files are an optional improvement over inline expected values. Tests work without them (current pattern), but goldens make maintenance easier as heuristics evolve.

## MVP Definition

### Launch With (v1 -- This Milestone)

Minimum viable debug infrastructure to diagnose and fix the 0/10 scoring problem.

- [ ] `--debug` flag added to scan command -- single switch to enable all debug output
- [ ] Raw response logging to stderr -- see what the agent actually returned
- [ ] Per-metric score trace -- see which heuristic indicators matched/missed
- [ ] Realistic response fixtures in `testdata/c7_responses/` -- test heuristics against real patterns
- [ ] Unit tests for all 4 heuristic scoring functions (M2/M3/M4/M5) with fixture responses

### Add After Validation (v1.x)

Features to add once the core debug mode is working and heuristics are fixed.

- [ ] `--debug-dir` for response file dumps -- add when responses need offline analysis
- [ ] Response replay mode (`--replay-from`) -- add when heuristic iteration cycle is too slow/expensive
- [ ] Score trace annotations in JSON output -- add when JSON consumers need debug info
- [ ] Golden file test infrastructure -- add when fixture count exceeds ~10 per metric

### Future Consideration (v2+)

Features to defer until the debug workflow is proven.

- [ ] Side-by-side prompt + response formatting -- polish item, not blocking
- [ ] Heuristic sensitivity report -- useful for heuristic health monitoring long-term
- [ ] LLM-as-judge scoring integration (rubrics already exist in code) -- when heuristics are proven inadequate

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority | Depends On |
|---------|------------|---------------------|----------|------------|
| `--debug` flag | HIGH | LOW | P1 | Nothing |
| Raw response logging to stderr | HIGH | LOW | P1 | `--debug` flag |
| Per-metric score trace | HIGH | MEDIUM | P1 | `--debug` flag |
| Realistic test fixtures | HIGH | MEDIUM | P1 | Nothing (test-only) |
| Heuristic scoring unit tests | HIGH | MEDIUM | P1 | Test fixtures |
| `--debug-dir` file dumps | MEDIUM | LOW | P2 | `--debug` flag |
| Response replay mode | MEDIUM | MEDIUM | P2 | `--debug-dir` |
| Score trace in JSON | MEDIUM | MEDIUM | P2 | `--debug` flag, `--json` |
| Golden file testing | LOW | MEDIUM | P3 | Test fixtures |
| Side-by-side display | LOW | LOW | P3 | Response logging |
| Heuristic sensitivity report | LOW | LOW | P3 | Test fixtures |

**Priority key:**
- P1: Must have for this milestone -- required to diagnose and fix 0/10 scoring
- P2: Should have -- improves iteration speed for ongoing heuristic tuning
- P3: Nice to have -- quality of life improvements

## Implementation Mapping to Existing Code

### Where Each Feature Touches the Codebase

| Feature | Primary Files | Change Type |
|---------|---------------|-------------|
| `--debug` flag | `cmd/scan.go`, `internal/pipeline/pipeline.go` | Add flag, thread to analyzer |
| Raw response logging | `internal/agent/metrics/m{2,3,4,5}_*.go` | Add conditional stderr writes in `Execute()` |
| Score trace | `internal/agent/metrics/m{2,3,4,5}_*.go` | Change `score*Response(string) int` to return trace struct |
| Debug output separation | All debug writers | Use `os.Stderr` exclusively (matches existing spinner pattern) |
| Test fixtures | `internal/agent/metrics/testdata/c7_responses/` | New directory with `.txt` response files |
| Heuristic tests | `internal/agent/metrics/metric_test.go` | Extend existing test table pattern (lines 329-487) |
| `--debug-dir` | `cmd/scan.go`, `internal/analyzer/c7_agent/agent.go` | Flag + file write in metric execution |
| Response replay | `internal/agent/executor_adapter.go` | New `FileReplayExecutor` implementing `metrics.Executor` |
| JSON debug info | `pkg/types/types.go`, `internal/output/json.go` | Extend `C7MetricResult` with optional `DebugInfo` |

### Key Architectural Constraint

The `metrics.Executor` interface (`internal/agent/metrics/metric.go:60-62`) is the correct abstraction boundary for replay mode. Implementing `FileReplayExecutor` requires no changes to metric logic -- it just provides pre-recorded responses instead of calling Claude CLI. This is a clean inversion: the same interface that enables testability enables replay.

### Scoring Function Refactor Pattern

Current signature:
```go
func (m *M2Comprehension) scoreComprehensionResponse(response string) int
```

Proposed signature:
```go
func (m *M2Comprehension) scoreComprehensionResponse(response string) (int, *ScoreTrace)
```

Where `ScoreTrace` contains:
```go
type ScoreTrace struct {
    BaseScore          int
    MatchedIndicators  []string
    MissedIndicators   []string  // indicators checked but not found
    Adjustments        []string  // e.g., "+1 word count > 100"
    FinalScore         int
}
```

This change is backward-compatible: callers that only need the score can ignore the trace. Callers with debug enabled can log the trace to stderr.

## Existing Infrastructure to Leverage

The codebase already has several patterns that directly support debug mode implementation:

1. **`--verbose` flag pattern** (`cmd/root.go:13`): Already threads a boolean through the pipeline to all renderers. `--debug` follows the same pattern.

2. **stderr for diagnostic output** (`pipeline.go:65-68`): The spinner already writes to `os.Stderr`. Debug output follows the same convention.

3. **`SampleResult.Response` field** (`metric.go:43`): Agent responses are already captured and stored. They just are not displayed anywhere.

4. **`metrics.Executor` interface** (`metric.go:60-62`): Clean abstraction for CLI execution. Mock/replay executors plug in without metric changes.

5. **Existing heuristic tests** (`metric_test.go:329-487`): Table-driven test pattern for all 4 scoring functions. Extend with fixture-loaded responses.

6. **`C7MetricResult` type** (`pkg/types/types.go`): Already has `Samples []string` and `Reasoning string`. Can be extended with debug fields.

7. **`renderC7` verbose mode** (`terminal.go:601-610`): Already shows per-task breakdown when `--verbose` is set. Debug mode adds response + trace detail below this.

## Sources

- Codebase analysis: `internal/agent/metrics/m{1-5}_*.go` -- heuristic scoring implementations [HIGH confidence]
- Codebase analysis: `internal/agent/metrics/metric.go` -- interface definitions and types [HIGH confidence]
- Codebase analysis: `cmd/scan.go`, `cmd/root.go` -- existing flag patterns [HIGH confidence]
- Codebase analysis: `internal/pipeline/pipeline.go` -- pipeline threading pattern [HIGH confidence]
- Codebase analysis: `internal/output/terminal.go` -- output rendering patterns [HIGH confidence]
- [Go slog documentation](https://pkg.go.dev/log/slog) -- structured logging patterns for debug output [HIGH confidence]
- [Goldie: golden file testing for Go](https://github.com/sebdah/goldie) -- golden file testing pattern [HIGH confidence]
- [File-driven testing in Go](https://eli.thegreenplace.net/2022/file-driven-testing-in-go/) -- testdata directory conventions [HIGH confidence]
- [Go os/exec patterns](https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/) -- stderr/stdout separation [HIGH confidence]

---
*Feature research for: C7 debug modes and heuristic testing infrastructure*
*Researched: 2026-02-06*
