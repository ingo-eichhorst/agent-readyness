# Project Research Summary

**Project:** ARS v0.0.5 - C7 Debug Infrastructure
**Domain:** CLI diagnostic tooling for heuristic-based agent evaluation
**Researched:** 2026-02-06
**Confidence:** HIGH

## Executive Summary

The v0.0.5 milestone adds debug infrastructure to investigate C7 agent evaluation scoring anomalies, specifically M2/M3/M4 metrics returning unexpected scores. The investigation requires visibility into raw agent responses, heuristic scoring logic, and sample selection decisions. Research reveals that **zero new dependencies are needed** - the entire debug infrastructure can be built using Go stdlib primitives (`io.Writer`, `fmt.Fprintf`, `os.Stderr`) and existing patterns already established in the codebase (`--verbose` flag threading, `C7Progress` stderr separation).

The recommended approach is surgical: add a single `--debug-c7` flag that routes diagnostic output exclusively to stderr, extend existing types (`SampleResult`, `C7MetricResult`) to carry prompt and response data when debug is active, and implement debug rendering in the existing output layer. The debug mode must be zero-cost when disabled (no allocations in hot path), safe for CI environments (no ANSI codes in non-TTY mode), and strictly observe-only (never modify execution behavior).

The critical architectural insight is that `metrics.Executor` interface is already the perfect seam for both mocking (for tests) and replay (for iterative heuristic tuning). The existing `SampleResult.Response` field already captures agent responses but they are discarded during the `buildMetrics` transformation. The fix is to conditionally preserve this data when debug mode is active, not to rebuild the capture mechanism.

## Key Findings

### Recommended Stack

**All stdlib, zero new dependencies.** The debug infrastructure leverages existing patterns:

**Core technologies:**
- `spf13/cobra` (existing) — `--debug-c7` flag registration, follows same pattern as `--enable-c7`
- `fmt.Fprintf` + `io.Writer` (stdlib) — debug output routing, consistent with `Pipeline.writer` pattern
- `io.Discard` / `os.Stderr` (stdlib) — zero-cost disable / debug output channel
- `testing` package (stdlib) — heuristic scoring tests, extend existing table-driven pattern
- `os.ReadFile` + `testdata/` (stdlib + Go convention) — golden file fixtures for real responses
- `metrics.Executor` interface (existing) — seam for mock injection, already defined for testability

**Rejected approaches:**
- `log/slog` — wrong paradigm; debug is investigative (view responses), not operational (structured logs)
- `github.com/sebdah/goldie` — golden file library unnecessary for ~15 response fixtures
- `github.com/stretchr/testify` — assertion helpers add dependency for marginal convenience
- Any logging framework — overkill for observe-only debug mode gated by single flag

**Net dependency change: 0. No new imports. No new go.mod entries.**

### Expected Features

Research identified table stakes (MVP for bug investigation) vs. differentiators (iteration speed improvements) vs. anti-features (common requests that create problems).

**Must have (MVP for v0.0.5):**
- `--debug-c7` flag — single switch to enable all debug output
- Raw response logging to stderr — see what the agent actually returned (responses already captured in `SampleResult.Response`, just need display path)
- Per-metric score trace — see which heuristic indicators matched/missed in `scoreComprehensionResponse` etc.
- Realistic response fixtures in `testdata/c7_responses/` — test heuristics against real Claude CLI output patterns
- Unit tests for all 4 heuristic scoring functions (M2/M3/M4/M5) — document current behavior, expose saturation issues

**Should have (iteration speed, defer to v1.x):**
- `--debug-dir` for response file dumps — offline analysis when responses exceed terminal readability
- Response replay mode (`--replay-from`) — re-score saved responses without $0.15+ API calls per iteration
- Score trace in JSON output (`--json --debug-c7`) — programmatic analysis of scoring logic
- Golden file test infrastructure — manage expected scores as fixtures once heuristics stabilize

**Defer (v2+, not blocking):**
- Side-by-side prompt + response formatting — polish item
- Heuristic sensitivity report — indicator health monitoring, useful long-term
- LLM-as-judge scoring — fallback when heuristics proven inadequate (rubrics already exist in code)

**Anti-features (reject):**
- Interactive step-through debugger — Go has `delve`; don't duplicate
- Real-time response streaming — Claude CLI headless mode (`-p` flag) returns complete JSON, not streaming
- Complex debug query language — overkill for 5 metrics; use `grep` on stderr output
- Automatic heuristic tuning / ML-based scoring — premature optimization, fix heuristics first
- Debug log levels (DEBUG/TRACE/INFO) — two modes sufficient: normal (silent) and debug (all output)
- Persistent debug config file — debug is transient activity, flags only

### Architecture Approach

**The data is already there, just needs a display path.** Research found that agent responses are captured in `SampleResult.Response` during metric execution but discarded in `C7Analyzer.buildMetrics()` when transforming to `C7MetricResult` for output rendering.

**Major components:**

1. **Flag plumbing (cmd → Pipeline → C7Analyzer)** — Thread `debugC7 bool` through existing method calls (`SetC7Debug`), following established `enableC7` pattern. Auto-enable C7 when debug flag is set (debug without C7 running is meaningless).

2. **Data capture (extend existing types)** — Add `Prompt string` field to `metrics.SampleResult`. Add `DebugSamples []C7DebugSample` field to `types.C7MetricResult` with `json:"omitempty"`. Populate in `buildMetrics()` when debug=true. No changes to metric execution path.

3. **Output rendering (terminal + JSON)** — Add `renderC7Debug(w io.Writer, m *types.C7Metrics)` in `terminal.go`, called when `debugC7=true`. Format: per-metric, per-sample: file path, prompt (truncated), response (truncated), score, duration. Write exclusively to stderr. JSON output includes `debug_samples` automatically via extended types.

4. **Testing (heuristic scoring validation)** — New file `internal/agent/metrics/scoring_test.go` with comprehensive tests for `score*Response()` functions. Use real response fixtures from `testdata/c7_responses/m{2,3,4}_responses/*.txt`. Test monotonicity (good > mediocre > bad), edge cases (empty, saturated indicators), adversarial inputs (syntactic match, semantic miss).

5. **Mock infrastructure (replay mode foundation)** — `MockExecutor` implements `metrics.Executor`, reads canned responses from `testdata/` files matched by prompt substring. Enables deterministic tests and future `--replay-from` feature.

**Critical boundaries:**
- Debug output: NEVER to `io.Writer` parameter (stdout), ALWAYS to `os.Stderr` directly
- Debug state: NEVER on shared `Metric` singletons, ALWAYS threaded through `Execute()` parameters or per-invocation fields
- Debug behavior: NEVER modifies execution (no extra CLI calls, no changed timeouts), ALWAYS observe-only

**Existing patterns to leverage:**
- `--verbose` flag threading through Pipeline → analyzers → renderers
- `C7Progress` stderr output + `isatty` check for non-TTY environments
- `SampleResult` data structure already captures `Response` field
- Table-driven tests in `metric_test.go` lines 329-487 for heuristic scoring
- `metrics.Executor` interface abstraction for testability

### Critical Pitfalls

1. **Debug output polluting structured output channels** — If debug writes to stdout (via `Pipeline.writer`), it corrupts `--json` output and breaks piped consumption (`| jq`). The codebase already separates: stdout for results, stderr for diagnostics (`Spinner`, `C7Progress`). Debug must follow stderr exclusively. **Verify:** `ars scan --json --debug-c7 . 2>/dev/null | jq` produces valid JSON.

2. **Debug flags causing performance regression in normal mode** — Go evaluates `fmt.Sprintf` arguments before the `if debug` check. Capturing responses and formatting debug strings on every sample evaluation (5 metrics × 3 samples × 2 CLI calls = 30 operations) adds 5-10% overhead even when debug is off. **Avoid:** Use `io.Discard` pattern (zero-cost when disabled) and never allocate debug data structures in hot path unless debug is active. **Verify:** Benchmark shows zero regression with debug disabled.

3. **Debug mode breaking in CI/non-TTY environments** — ANSI color codes, carriage returns (`\r`), and cursor movement produce garbage in CI logs. The existing `C7Progress` checks `isatty.IsTerminal(w.Fd())` and becomes line-based in non-TTY mode. Debug output must follow the same pattern: plain text, line-based (no `\r`), no ANSI codes. **Verify:** Run in Docker or with `2>&1 | tee debug.log` and check for escape sequences.

4. **Test fixtures diverging from real Claude CLI responses** — Existing tests use synthetic responses (`"The function returns data after handling errors."`). When heuristics are tuned against fabricated strings but real Claude responses have different patterns, the fixes don't transfer. The milestone requires capturing real responses during manual runs and using them as golden fixtures. **Avoid:** Update `testdata/c7_responses/` with actual CLI output; tag with version and date.

5. **Over-engineering debug infrastructure beyond investigation need** — The goal is diagnosing M2/M3/M4 scoring issues, not building a production observability platform. Scope expansion (structured logging framework, debug UI, multi-level verbosity, configuration files, per-metric toggles) delays the investigation. **Limit:** Single `--debug-c7` flag, stderr output only, zero logging libraries, one-phase implementation.

6. **Debug state leaking between concurrent metric executions** — C7 runs 5 metrics in parallel via `errgroup.Group`. If debug state is stored in shared structures (the `Metric` singletons in `registry.go`), concurrent goroutines corrupt each other's context. **Avoid:** Prefix all debug lines with `[M2]` metric ID. Use mutex-protected stderr writer (acceptable for debug mode). Never store debug state on singleton metrics.

7. **Debug flag proliferation cluttering CLI interface** — ARS already has 8 flags on `scan` command. Adding `--debug`, `--debug-c7`, `--debug-metric M2`, `--debug-output FILE`, `--debug-level trace` creates combinatorial testing matrix and user confusion. **Limit:** Exactly ONE flag: `--debug-c7`. Per-metric filtering via environment variable if needed (`ARS_DEBUG_METRICS=M2,M3`). File output via shell redirection (`2>debug.log`).

## Implications for Roadmap

Based on research, the v0.0.5 milestone should follow a 4-phase structure with minimal scope, fast iteration, and ruthless focus on the investigation goal.

### Phase 1: Foundation (Flag Plumbing + Debug Channel)
**Rationale:** Establish the debug output path and flag threading before any instrumentation. This prevents the #1 pitfall (output pollution) and #2 pitfall (performance regression) by design rather than retrofit.

**Delivers:**
- `--debug-c7` bool flag in `cmd/scan.go` (auto-enables C7)
- `SetC7Debug(bool)` method on `Pipeline`, threaded to `C7Analyzer`
- `DebugWriter io.Writer` field on metrics, initialized to `io.Discard` (normal) or `os.Stderr` (debug)
- Non-TTY detection (`isatty` check) for plain text output in CI

**Addresses features:** `--debug-c7` flag (table stakes)

**Avoids pitfalls:** #1 (output pollution — stderr only), #2 (performance — zero-cost when disabled), #3 (CI breakage — non-TTY from start), #7 (flag proliferation — single flag)

**Files:** `cmd/scan.go`, `internal/pipeline/pipeline.go`, `internal/analyzer/c7_agent/agent.go`

**Verification:** `ars scan . --debug-c7` behaves identically to `--enable-c7` (flag plumbed but not yet consumed)

**Complexity:** LOW (3 files, established patterns, no new behavior)

### Phase 2: Data Capture (Prompt + Response Storage)
**Rationale:** Extend existing types to carry debug data conditionally. The `SampleResult.Response` field already exists; just needs `Prompt` alongside it. The `buildMetrics()` transformation already loops over samples; just needs to populate `DebugSamples` when debug=true.

**Delivers:**
- `Prompt string` field added to `metrics.SampleResult`
- All M1-M5 `Execute()` methods set `sr.Prompt = prompt`
- `C7DebugSample` type in `pkg/types/types.go` (FilePath, Description, Prompt, Response, Score, Duration)
- `DebugSamples []C7DebugSample` field on `C7MetricResult` with `json:"omitempty"`
- `buildMetrics()` populates `DebugSamples` when `debug=true`

**Addresses features:** Raw response capture (table stakes), data structure for display

**Avoids pitfalls:** #5 (over-engineering — minimal type changes), #6 (concurrent state — no shared state, per-sample data)

**Files:** `internal/agent/metrics/metric.go`, `internal/agent/metrics/m{1,2,3,4,5}_*.go` (5 files), `pkg/types/types.go`, `internal/analyzer/c7_agent/agent.go`

**Verification:** Run `ars scan . --debug-c7 --json` and inspect JSON output for `debug_samples` arrays (may need Phase 4 to verify terminal rendering)

**Complexity:** LOW (type additions, field assignments in existing methods)

**Depends on:** Phase 1 (debug flag must exist to gate population)

### Phase 3: Heuristic Tests (Real Response Fixtures)
**Rationale:** Document current heuristic behavior and expose scoring issues before attempting fixes. Tests run independently of CLI (`go test`), can proceed in parallel with Phase 2. Real response fixtures ground the tests in actual Claude CLI output patterns rather than fabricated strings.

**Delivers:**
- `internal/agent/metrics/scoring_test.go` with comprehensive tests for `score*Response()` (M2/M3/M4/M5)
- `testdata/c7_responses/m2_responses/*.txt`, `m3_responses/*.txt`, `m4_responses/*.txt` — real responses captured from manual runs
- Test cases: empty, good, bad, mixed, edge cases, adversarial (syntactic match but semantic miss)
- Monotonicity verification: good score > mediocre score > bad score for each metric
- `MockExecutor` implementation for deterministic testing and future replay mode

**Addresses features:** Realistic test fixtures (table stakes), unit tests for heuristic functions (table stakes), foundation for replay mode (defer to v1.x)

**Avoids pitfalls:** #4 (fixture divergence — use real responses, tag with version/date)

**Files:** `internal/agent/metrics/scoring_test.go` (new), `internal/agent/metrics/mock_executor_test.go` (new), `testdata/c7_responses/` (new directory tree)

**Verification:** `go test ./internal/agent/metrics/ -run TestM[2345]_Score -v` — all pass, expose scoring saturation issues

**Complexity:** MEDIUM (need to capture real responses, ~15 fixture files, test design)

**Depends on:** Nothing (independent of Phases 1-2, can run in parallel)

### Phase 4: Debug Rendering (Terminal + JSON Output)
**Rationale:** Display the captured debug data. By this point, the data is flowing through the pipeline (Phase 2) and tests exist for validation (Phase 3). This phase is pure presentation — format and write to stderr.

**Delivers:**
- `renderC7Debug(w io.Writer, m *types.C7Metrics)` in `internal/output/terminal.go`
- Thread `debugC7 bool` through `Pipeline.Run()` → `RenderSummary` → `renderC7`
- Debug output format: `[DEBUG] [M2] [sample 1/3] File: internal/pipeline/pipeline.go | Prompt (200 chars): "Read the file..." | Response (500 chars): "The file implements..." | Score: 7 | Duration: 12.3s`
- JSON output includes `debug_samples` via extended `C7MetricResult` (automatic, no code change needed)

**Addresses features:** Debug output rendering (table stakes), score trace display (table stakes)

**Avoids pitfalls:** #1 (output pollution — stderr only, verified with `| jq`), #3 (CI breakage — plain text in non-TTY)

**Files:** `internal/output/terminal.go`, `internal/pipeline/pipeline.go`

**Verification:** Run `ars scan . --debug-c7` and see prompts, responses, scores on stderr; run `ars scan . --debug-c7 --json 2>/dev/null | jq` and confirm valid JSON on stdout

**Complexity:** MEDIUM (output formatting, threading debugC7 parameter, truncation logic)

**Depends on:** Phase 1 (debug channel exists), Phase 2 (debug data populated)

### Phase Ordering Rationale

- **Phase 1 before Phase 2:** The debug output channel must exist before any data is captured. Trying to capture first leads to ad-hoc output paths that violate the stdout/stderr separation.
- **Phase 3 in parallel:** Tests are independent of the CLI infrastructure. Can be developed while Phase 1-2 are being implemented. Tests inform Phase 5 (scoring fixes, not in this milestone).
- **Phase 2 before Phase 4:** Data must be captured before it can be rendered. The `DebugSamples` field must exist in `C7MetricResult` before `renderC7Debug` can access it.
- **Phase 4 is last:** Pure presentation layer. All dependencies (flag, data capture, tests) are in place. This phase just formats and writes what already exists.

**Total implementation time estimate:** 1-2 days (Foundation + Data Capture: 4-6 hours, Heuristic Tests: 6-8 hours including fixture capture, Debug Rendering: 4-6 hours). The research has already identified all file touch points and patterns to follow.

### Research Flags

**All phases use standard patterns — skip `/gsd:research-phase` for this milestone.**

- **Phase 1:** Established flag threading and `io.Writer` patterns already exist in codebase (`--verbose`, `Pipeline.writer`, `C7Progress`)
- **Phase 2:** Type extensions follow existing patterns (`C7MetricResult` already has `Samples []string`, just adding `DebugSamples`)
- **Phase 3:** Table-driven tests already exist for scoring functions (lines 329-487 in `metric_test.go`), just extending with real fixtures
- **Phase 4:** Rendering follows `renderC7` pattern (lines 531-611 in `terminal.go`), adding debug subsection

**When to use `/gsd:research-phase`:** If Phase 5 (scoring logic fixes) is added to this milestone, use research to investigate compound indicator patterns, semantic similarity detection, or LLM-as-judge rubrics. Phase 5 deferred to v0.0.6.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All stdlib, zero new dependencies verified. Patterns exist in codebase (25+ file references). |
| Features | HIGH | Table stakes (5 features) derived from bug investigation needs. Differentiators (4 features) from iteration workflow analysis. Anti-features (6 features) grounded in CLI tool patterns and ARS-specific constraints (parallel execution, JSON output mode). |
| Architecture | HIGH | Based on direct codebase reading with specific line references. `SampleResult.Response` field existence verified (metric.go:43). `buildMetrics` transformation gap verified (agent.go:122-124). All integration points mapped to existing files. |
| Pitfalls | HIGH | All 7 critical pitfalls grounded in existing ARS codebase analysis (stdout/stderr separation, `C7Progress` TTY detection, `errgroup` parallel execution, flag patterns, 3 output modes). External sources confirm patterns (Airflow debug-to-stderr precedent, AWS CLI issue #5187). |

**Overall confidence:** HIGH

Research is grounded in codebase analysis of 25+ source files with specific line references, not external documentation or inference. The recommended approach follows established patterns already present in the project (`--verbose` flag, `C7Progress` stderr output, table-driven tests). Zero new dependencies reduces unknowns.

### Gaps to Address

No significant gaps. The research was comprehensive (STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md all based on codebase analysis, not external sources). Minor validation items during implementation:

- **Capture real Claude CLI responses:** Requires one manual `ars scan --enable-c7` run on a sample project to populate `testdata/c7_responses/` fixtures. This is a content task (copy-paste responses from debug log), not a research gap.
- **Verify JSON output compatibility:** The extended `C7MetricResult` type with `json:"omitempty"` should not break existing JSON consumers. Validate with a test comparing `--json` output before/after changes (without debug flag active).
- **Confirm heuristic scoring issues:** The research identified potential saturation problems in `scoreComprehensionResponse` (13 positive indicators, common words like "returns"/"error") and `scoreNavigationResponse` (path counting noise). Phase 3 tests will confirm whether these are actual issues or red herrings.

## Sources

### Primary (HIGH confidence - codebase analysis)

All findings based on direct reading of ARS codebase:

- `cmd/scan.go` — CLI flag patterns (lines 15-23, 124-133)
- `internal/pipeline/pipeline.go` — Pipeline orchestration, flag threading (lines 26-43, 119-124, 140-288)
- `internal/analyzer/c7_agent/agent.go` — C7 analyzer, buildMetrics transformation (lines 36-87, 98-153, 122-124 response discard)
- `internal/agent/executor.go` — Claude CLI subprocess execution (lines 42-118)
- `internal/agent/executor_adapter.go` — Executor interface adapter (lines 22-50)
- `internal/agent/parallel.go` — Parallel metric execution with errgroup (lines 22-87)
- `internal/agent/metrics/metric.go` — Metric interface, SampleResult type (lines 39-57, Response field line 43)
- `internal/agent/metrics/m{1,2,3,4,5}_*.go` — All 5 metrics Execute() and scoring functions
- `internal/agent/metrics/metric_test.go` — Existing test patterns (lines 329-487)
- `internal/agent/progress.go` — TTY detection pattern (line 67)
- `internal/output/terminal.go` — Terminal rendering, renderC7() (lines 531-611)
- `internal/output/json.go` — JSON output structure
- `pkg/types/types.go` — C7Metrics and C7MetricResult definitions (lines 253-303)

### Secondary (HIGH confidence - official documentation)

- [Claude Code Headless Documentation](https://code.claude.com/docs/en/headless) — CLI JSON response format (`{"type":"result","session_id":"...","result":"..."}`)
- [Claude Code CLI Reference](https://code.claude.com/docs/en/cli-reference) — All CLI flags (`-p`, `--output-format`, `--json-schema`)
- [Go slog package](https://pkg.go.dev/log/slog) — Evaluated and rejected for debug use case
- [Go testing package](https://pkg.go.dev/testing) — Standard test framework
- [Go testdata convention](https://pkg.go.dev/cmd/go#hdr-Test_packages) — `testdata/` directory semantics

### Tertiary (MEDIUM confidence - patterns and precedent)

- [Airflow PR: Print debug mode warning to stderr](http://www.mail-archive.com/commits@airflow.apache.org/msg486101.html) — Real-world example of debug-to-stderr separation
- [AWS CLI Issue #5187: --debug output to stderr](https://github.com/aws/aws-cli/issues/5187) — Industry precedent for output channel choice
- [File-driven testing in Go](https://eli.thegreenplace.net/2022/file-driven-testing-in-go/) — Golden file patterns without libraries
- [Testing with golden files in Go](https://medium.com/soon-london/testing-with-golden-files-in-go-7fccc71c43d3) — Community patterns
- [Observability Anti-Patterns](https://observability-antipatterns.github.io/) — Over-instrumentation patterns to avoid

---
*Research completed: 2026-02-06*
*Ready for roadmap: yes*
