# Pitfalls Research: Debug Infrastructure for C7 Agent Evaluation

**Domain:** Adding debug mode to an existing CLI tool (ARS C7 evaluation)
**Researched:** 2026-02-06
**Confidence:** HIGH (grounded in actual codebase analysis of 25+ source files)

This document catalogs pitfalls specific to adding debug/diagnostic infrastructure to the ARS CLI tool, with emphasis on the C7 agent evaluation subsystem (M2/M3/M4 scoring investigation).

> **Supersedes:** Previous v0.0.3 pitfalls document. That document covered CLI subprocess invocation, package reorganization, badges, and HTML reporting -- all now shipped in v0.0.4.

---

## Critical Pitfalls

These mistakes can cause broken output, CI failures, or require significant rework.

### 1. Debug Output Polluting Structured Output Channels

**What goes wrong:**
Debug information leaks into stdout, corrupting JSON output consumed by downstream tools. ARS has three output modes (terminal, JSON, HTML) and two separate consumers already write to stdout vs stderr:

- `pipeline.Pipeline.writer` (set to `cmd.OutOrStdout()`) carries all structured output: terminal rendering, JSON, scores
- `pipeline.Spinner` and `agent.C7Progress` write to `os.Stderr` for progress indicators
- `discovery.Walker` writes warnings directly to `os.Stderr`
- `workspace.go` uses `log.Printf` (which defaults to `os.Stderr`)

If debug output is routed through `p.writer` (stdout) or through `fmt.Fprintf(w, ...)` in any render function, it will:
1. Break `--json` output parsing (invalid JSON)
2. Corrupt piped output (`ars scan . | jq .composite_score`)
3. Pollute HTML report generation

**Why it happens:**
The codebase has no unified logging abstraction. Output goes through four different mechanisms: `io.Writer` parameter, `fmt.Fprintf(os.Stderr, ...)`, `log.Printf()`, and `color.Fprintf(w, ...)`. A developer adding debug statements naturally reaches for `fmt.Fprintf` to the nearest writer, which is often stdout.

**How to avoid:**
- Route ALL debug output exclusively through stderr. Never write debug output through the `io.Writer` parameter passed to render functions.
- Create a single `DebugWriter` that wraps `os.Stderr` and is gated by a flag. All debug output flows through this one channel.
- Verify the separation with a test: run `ars scan --json --debug <dir> 2>/dev/null` and confirm the output is valid JSON. Run `ars scan --json --debug <dir> 2>&1 1>/dev/null` and confirm only debug lines appear.

**Warning signs:**
- Any `fmt.Fprintf(w, ...)` call in new code where `w` is the pipeline writer
- Debug output appearing in `--json` mode output
- CI pipeline JSON parsing failures after enabling debug mode
- HTML reports containing `[DEBUG]` lines in rendered text

**Phase to address:** Phase 1 (foundation) -- establish the debug output channel before writing any debug content.

---

### 2. Debug Flags Causing Performance Regression in Normal Mode

**What goes wrong:**
Debug infrastructure adds overhead even when debug mode is disabled. In ARS, the C7 evaluation already takes 120+ seconds with 5 metrics running in parallel (`RunMetricsParallel` in `parallel.go`). Each metric makes 3-5 Claude CLI subprocess calls. Poorly implemented debug hooks add latency on every call:

- String formatting of debug messages that are never printed (Go still evaluates `fmt.Sprintf` arguments)
- Mutex contention from debug state tracking in the hot path of `C7Progress` (already has `sync.Mutex`)
- File I/O for debug log writing on every sample evaluation in `M2Comprehension.Execute()`
- Capturing and storing full CLI responses (which can be large) even when not debugging

**Why it happens:**
The natural approach is to add `if debug { log(...) }` everywhere. But the arguments to `log()` are evaluated before the `if` check in Go. And accumulating debug data (storing full responses, timing breakdowns) costs memory and CPU even when gated behind a flag, because the data structures exist and are allocated.

**How to avoid:**
- Use a `DebugLogger` interface with a no-op implementation for production. The no-op implementation should have zero allocations -- not even format string arguments should be evaluated.
- Avoid storing debug data in hot-path structs (`MetricResult`, `SampleResult`). Instead, emit debug events through a channel or callback that is nil-checked before invocation.
- Benchmark before and after: `go test -bench=. -benchmem ./internal/agent/...` with debug disabled must show zero regression.
- For the C7 parallel execution path specifically: debug logging must not introduce serialization points. The `errgroup.Group` in `RunMetricsParallel` runs 5 goroutines concurrently. Debug writes must be non-blocking (use buffered channel or separate goroutine for writes).

**Warning signs:**
- C7 evaluation time increases by more than 2% with debug mode OFF
- `go test -benchmem` shows new allocations in the metric execution hot path
- Lock contention visible in `go test -race` output (new mutex for debug state)
- CI timeout for C7 evaluation increases

**Phase to address:** Phase 1 (foundation) -- the debug logger interface must be zero-cost when disabled, established before any debug instrumentation is added.

---

### 3. Debug Mode Breaking in CI/Non-Interactive Environments

**What goes wrong:**
Debug features that rely on TTY capabilities fail or produce garbage in CI pipelines. The existing codebase already handles this carefully:

- `C7Progress` checks `isatty.IsTerminal(w.Fd())` and skips rendering in non-TTY mode
- `Spinner` does the same check and becomes a no-op
- ANSI color codes are automatically suppressed by `fatih/color` when stdout is not a TTY

But debug mode introduces new TTY-dependent behaviors:
1. ANSI escape codes in debug output (colors, cursor movement) produce raw escape sequences in CI logs
2. Carriage return (`\r`) for progress overwriting creates garbled multi-line output in non-TTY mode
3. Debug output width assumptions (the existing `%-130s` padding in `C7Progress.render()`) produce excessive whitespace in log files
4. Interactive prompts or confirmations in debug mode block forever in CI

**Why it happens:**
Developers test debug mode locally in terminals where ANSI codes render correctly. CI environments (GitHub Actions, Jenkins, etc.) capture output as plain text. The `isatty` check that protects `Spinner` and `C7Progress` is not automatically inherited by new debug code.

**How to avoid:**
- The debug writer must detect non-TTY and strip ANSI codes automatically. Use the same `isatty` pattern already established in `progress.go` line 67: `isatty.IsTerminal(w.Fd()) || isatty.IsCygwinTerminal(w.Fd())`.
- Debug output format must be plain text with line-based output (no `\r`, no cursor movement). Each debug line must be a complete, self-contained message ending with `\n`.
- Never use interactive features (prompts, pagers) in debug mode. Debug mode is observe-only.
- Add a CI test that runs `ars scan --enable-c7 --debug <dir>` in a non-TTY environment and verifies the output contains expected debug markers without ANSI artifacts.

**Warning signs:**
- Debug output contains `\033[` or `\r` escape sequences when piped to a file
- CI logs show garbled or overlapping lines when debug mode is enabled
- Debug output width exceeds 200 characters per line (CI log viewers truncate)
- Test `go test` output contains ANSI escape codes in debug assertions

**Phase to address:** Phase 1 (foundation) -- non-TTY behavior must be a design constraint for the debug writer, not an afterthought.

---

### 4. Test Fixtures Diverging from Real Claude CLI Responses

**What goes wrong:**
The C7 evaluation chain has three layers of response parsing, each with test fixtures:

1. `executor.go` `parseJSONOutput()` -- parses `{"type":"result","session_id":"...","result":"..."}`
2. `evaluator.go` `EvaluateContent()` -- parses `{"session_id":"...","result":"...","structured_output":{...}}`
3. `metrics/m2_comprehension.go` `scoreComprehensionResponse()` -- heuristic scoring of free-text responses

Test fixtures for these parsers (see `executor_test.go` lines 200-253) use hardcoded JSON strings. When the Claude CLI evolves (it has already changed `output_format` to `output_config.format` in late 2025), the test fixtures pass but production parsing breaks because:

- Fixtures are static snapshots of a previous CLI version
- No automated check compares fixture format against the current Claude CLI response format
- The `CombinedOutput()` call in `executor.go` line 80 captures both stdout and stderr from the CLI subprocess -- if the CLI starts writing warnings to stderr, they pollute the JSON response and fixtures do not reflect this

For the debug feature specifically: debug mode needs to capture and display these raw CLI responses. If the fixture responses in debug-mode tests differ from real CLI responses, the debug output will be misleading -- showing "expected" data that never actually occurs in production.

**Why it happens:**
Testing against a real Claude CLI is slow (120s+), expensive ($0.15+ per run), and non-deterministic. So developers write unit tests with static fixtures. Over time, the real response format drifts while fixtures remain frozen. The heuristic scorers in M2/M3/M4 (`scoreComprehensionResponse`, `scoreNavigationResponse`, `scoreIdentifierResponse`) are already tested against synthetic responses (see `metric_test.go` lines 329-487), not real agent output.

**How to avoid:**
- Record real CLI responses during manual testing and use them as "golden" fixtures. Tag each fixture with the Claude CLI version and date it was captured.
- Create a `testdata/cli-responses/` directory with versioned response files. Debug-mode tests should use the same fixture files as parsing tests.
- Add a CI job (weekly or on-demand) that runs one real CLI call and compares the response structure against the fixture schema. This is a contract test, not a functional test.
- In debug mode, when displaying raw responses, always show the actual response alongside the parsed result. This makes fixture drift visible immediately when a developer uses debug mode on a real project.

**Warning signs:**
- `parseJSONOutput` tests pass but `--enable-c7` fails on real projects with JSON parse errors
- Debug mode shows response structures that don't match what users see in raw CLI output
- Heuristic scorer tests pass with synthetic responses but produce unexpected scores on real code
- New fields appear in real CLI responses that are silently dropped by the parser

**Phase to address:** Phase 2 (instrumentation) -- when adding debug capture of raw responses, simultaneously update test fixtures from real captured data.

---

### 5. Over-Engineering Debug Infrastructure Beyond the Investigation Need

**What goes wrong:**
The debug mode is being added to investigate a specific bug: M2/M3/M4 scoring anomalies. But "debug infrastructure" naturally expands into:

- Structured logging framework with configurable levels (DEBUG, TRACE, INFO)
- Debug output formatting system with templates
- Debug data persistence (writing to files, databases)
- Debug UI (web dashboard for viewing debug data)
- Configuration file for debug settings
- Per-metric debug toggles
- Debug data aggregation and analytics

This scope expansion delays the actual investigation. The M2/M3/M4 bug may have a simple root cause (e.g., the heuristic scorers in `scoreComprehensionResponse` have threshold issues), but a 2-week debug infrastructure project obscures a 2-hour fix.

**Why it happens:**
Debug infrastructure feels like "investment" rather than "overhead." Developers think "while we're adding debug mode, we should also..." But ARS is a CLI tool with a specific evaluation pipeline, not a distributed system that needs production observability. The existing `--verbose` flag already provides expanded output for all categories.

**How to avoid:**
- Define the debug mode scope explicitly: "Show raw CLI prompts, raw CLI responses, sample selection details, and heuristic scoring breakdowns for C7 metrics." Nothing more.
- Set a time box: if the debug infrastructure takes more than 1 phase to implement, it is over-engineered.
- Ask "does this help investigate M2/M3/M4 scoring?" for every proposed debug feature. If the answer is "no, but it would be nice for future debugging," defer it.
- Do not introduce a logging framework. Use `fmt.Fprintf(debugWriter, "[DEBUG] ...")` directly. A logging framework is warranted only if debug mode is used by end users in production, which is not the case here.

**Warning signs:**
- Debug mode implementation requires changes to more than 5 files
- A configuration system is being built for debug settings
- Debug output has more than 3 verbosity levels
- Debug infrastructure takes longer to build than the investigation it enables
- Someone proposes storing debug data for later analysis (file persistence)

**Phase to address:** Phase 0 (requirements) -- scope must be locked before implementation begins. The requirements document should list exactly what debug mode shows and explicitly state what it does not show.

---

### 6. Debug State Leaking Between Concurrent Metric Executions

**What goes wrong:**
C7 runs 5 metrics in parallel via `errgroup.Group` in `RunMetricsParallel` (`parallel.go`). Each metric goroutine calls `SelectSamples()` then `Execute()`, which makes multiple Claude CLI calls. If debug state is stored in shared structures, concurrent goroutines corrupt each other's debug context:

- Metric M2's debug output shows M3's prompt because a shared debug buffer was overwritten
- Debug timestamps are interleaved, making it impossible to trace a single metric's execution flow
- Sample selection debug info for M4 appears under M1's heading because the debug context was not goroutine-local

The existing `C7Progress` struct handles this correctly with a mutex and per-metric status (`metrics map[string]*MetricProgress`). But debug output is more verbose and higher-frequency than progress updates. A mutex around every debug log line would serialize the parallel execution, defeating the purpose.

**Why it happens:**
Adding debug logging to the metric execution path seems simple: `debug("M2: sending prompt: %s", prompt)`. But `debug()` writes to a shared writer. Without per-goroutine buffering, the output interleaves. With per-goroutine buffering, the output loses real-time visibility. This is the classic concurrent logging problem.

**How to avoid:**
- Prefix every debug line with the metric ID: `[M2] [sample 2/3] Sending prompt...`. This makes interleaved output readable without requiring serialization.
- Use a `sync.Mutex`-protected writer for debug output (acceptable for debug mode since performance is secondary). The mutex is only held during the `Write()` call, not during format string evaluation.
- Do NOT store debug state on the `Metric` interface implementations (`M2Comprehension`, etc.). They are shared singletons created in `registry.go`. Pass debug context through `Execute()` parameters or use a per-invocation callback.
- Buffer each metric's debug output independently and flush in order after all metrics complete. This gives clean, per-metric output at the cost of delayed display. Offer both modes: `--debug` for buffered/clean output, `--debug-live` for real-time interleaved output (if needed at all).

**Warning signs:**
- Debug output for one metric contains prompts or responses from a different metric
- Debug output has inconsistent metric ID prefixes within a single line
- Race detector (`go test -race`) reports data races on debug state
- Debug output lines are truncated or garbled (concurrent writes without synchronization)

**Phase to address:** Phase 1 (foundation) -- the debug writer must handle concurrent writes correctly from day one, since C7 is inherently parallel.

---

### 7. Debug Flag Proliferation Cluttering the CLI Interface

**What goes wrong:**
ARS already has 8 flags on the `scan` command: `--config`, `--threshold`, `--json`, `--no-llm`, `--enable-c7`, `--output-html`, `--baseline`, `--badge`. Plus the global `--verbose` flag. Adding debug flags expands this:

- `--debug` (enable debug mode)
- `--debug-c7` (debug only C7)
- `--debug-metric M2` (debug specific metric)
- `--debug-output /path/to/file` (write debug to file)
- `--debug-level trace` (debug verbosity)

This clutters the CLI, confuses users, and creates a combinatorial testing matrix (`--debug --json`, `--debug --output-html`, `--debug --verbose`, etc.).

**Why it happens:**
Each debug capability feels like it needs its own flag. "What if the user only wants to debug M2?" "What if they want debug output in a file?" These are legitimate needs, but individual flags for each are the wrong abstraction.

**How to avoid:**
- Add exactly ONE flag: `--debug`. No sub-flags, no verbosity levels, no per-metric toggles.
- If per-metric filtering is needed, use an environment variable: `ARS_DEBUG_METRICS=M2,M3`. Environment variables are appropriate for developer-facing debug settings that should not be part of the public CLI contract.
- Debug output always goes to stderr. If the user wants it in a file: `ars scan --enable-c7 --debug . 2>debug.log`. Shell redirection is more flexible than a `--debug-output` flag.
- Test the combinatorial interactions: `--debug` must work correctly with every existing flag combination. At minimum: `--debug --json`, `--debug --output-html`, `--debug --verbose`, `--debug --no-llm`.

**Warning signs:**
- More than one new flag added for debug functionality
- Debug flag documentation is longer than the feature documentation
- Test matrix grows by more than 4 test cases for flag interactions
- Users ask "what's the difference between `--verbose` and `--debug`?"

**Phase to address:** Phase 0 (requirements) -- the flag design must be decided before implementation. Phase 2 (integration) -- test flag interactions.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Using `fmt.Println` for debug output instead of a debug writer | Fast to implement, no new abstractions | Cannot be disabled without removing code; pollutes stdout | Never -- even a one-line `debugf()` wrapper is worth it |
| Storing debug data in existing `MetricResult` struct | No new types needed | Bloats the struct for all users; breaks JSON serialization; shipped in non-debug builds | Never -- debug data should be separate from result data |
| Skipping non-TTY testing for debug output | Saves 30 minutes of test writing | Debug mode breaks every CI pipeline that enables it | Never -- CI is a primary use environment for ARS |
| Adding debug logging inside `scoreComprehensionResponse` heuristic directly | Directly instruments the suspected bug location | Tightly couples debug infrastructure to one scoring implementation; cannot reuse for M3/M4 | Acceptable for quick investigation only; must be replaced with callback pattern before merge |
| Using `log.Printf` for debug output (matching `workspace.go` pattern) | Consistent with existing code | `log.Printf` cannot be easily captured in tests, has no gating mechanism, always writes to stderr | Acceptable as interim step in Phase 1 only if wrapped |

## Integration Gotchas

Common mistakes when connecting debug mode to existing ARS subsystems.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| JSON output (`--json`) | Debug output interleaved with JSON, producing invalid output | Debug writes exclusively to stderr; JSON output is only on stdout; validate with `jq` in tests |
| HTML report (`--output-html`) | Debug lines captured in HTML report template data | Debug output never flows through `io.Writer` parameter; HTML generator receives only result data |
| Verbose mode (`--verbose`) | Confusing overlap: both `--verbose` and `--debug` show "more info" | Clear separation: `--verbose` shows expanded results (per-metric scores, per-file details); `--debug` shows execution internals (prompts, raw responses, timing) |
| C7 Progress display | Debug output and progress spinner fight for stderr cursor position | Debug mode disables the interactive progress spinner (`C7Progress`); shows line-by-line progress instead (one line per event, no `\r` overwriting) |
| Spinner (`pipeline.Spinner`) | Spinner animation interleaves with debug output | Stop spinner before debug output begins, or disable spinner entirely when `--debug` is active |
| Color output (`fatih/color`) | Debug output uses colors that are unreadable in some terminals or garbled in CI | Debug output is always plain text, no color. Prefix with `[DEBUG]` for grep-ability instead |

## Performance Traps

Patterns that work during testing but degrade production performance.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Formatting debug strings unconditionally | 5-10% CPU overhead from `fmt.Sprintf` on every metric sample call, even with debug off | Use closure-based debug: `debugf(func() string { return fmt.Sprintf(...) })` -- closure is only evaluated when debug is on | At 5 metrics x 3 samples x 2 CLI calls each = 30 format operations per scan |
| Capturing full CLI responses in memory for debug | Memory usage spikes by 50-100MB for large project scans | Only capture responses when debug mode is active; truncate to first 2000 chars for display | When scanning large monorepos with many files |
| Synchronizing debug writes with mutex in parallel metric execution | C7 evaluation time increases 10-30% due to lock contention | Use per-goroutine buffering with a final flush, or accept interleaved output with metric ID prefixes | When 5 metrics run concurrently (always, in production C7 mode) |
| Debug mode enabling additional CLI calls (e.g., re-running failed samples) | Doubles API cost and evaluation time | Debug mode must be observe-only: capture what happens, never add extra actions | Always -- debug mode must not change behavior |

## UX Pitfalls

Common user experience mistakes when adding debug features to CLI tools.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Debug output is an undifferentiated wall of text | Users cannot find the information they need; give up on debug mode | Structure debug output with clear sections: `=== M2: Code Behavior Comprehension ===`, then indented subsections for each sample |
| Debug output does not include enough context to reproduce the issue | User sees "score: 3" but does not know what prompt produced it or what response was scored | Every debug block must include: (1) the prompt sent, (2) the raw response received, (3) the scoring breakdown, (4) the final score |
| Debug mode changes scoring behavior | Users think debug mode reveals the bug, but actually debug mode causes different behavior (different timeouts, sequential vs parallel execution) | Debug mode must execute the exact same code path as normal mode. It only adds observation, never modifies execution |
| No clear relationship between `--verbose` and `--debug` | Users do not know which flag to use | Document in `--help`: "`--verbose` shows detailed results; `--debug` shows execution internals for troubleshooting" |
| Debug output has no machine-parseable structure | Users cannot pipe debug output to analysis tools | Use consistent prefix format: `[DEBUG] [M2] [sample:1/3] [phase:prompt] <message>` -- parseable with grep/awk |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Debug flag added:** Often missing interaction test with `--json` mode -- verify JSON output is still valid with `jq` when `--debug` is active
- [ ] **Debug output for prompts:** Often missing the system prompt -- only shows the user prompt, but the system prompt (rubric in `scorer.go`) is equally important for debugging scoring issues
- [ ] **Debug output for scoring:** Often missing the heuristic breakdown -- shows final score but not which positive/negative indicators fired in `scoreComprehensionResponse` (the 13 positive and 7 negative indicators)
- [ ] **Non-TTY testing:** Often missing -- verify debug output works when `os.Stderr` is not a terminal (pipe to file, run in Docker, run in GitHub Actions)
- [ ] **Debug mode with `--enable-c7` disabled:** Often missing edge case -- what happens when someone passes `--debug` without `--enable-c7`? Should show "C7 not enabled, debug mode has nothing to show" rather than silently doing nothing
- [ ] **Debug mode with `--no-llm`:** Often missing -- if LLM is disabled, C7 returns `Available: false`. Debug mode should explain WHY it is unavailable, not just silently skip
- [ ] **Sample selection debugging:** Often missing -- shows which samples were evaluated but not which files were considered and rejected, making it impossible to debug "wrong file was picked for M2"

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Debug output polluting JSON | LOW | Move all debug `fmt.Fprintf` calls from `w` to `os.Stderr`; add `--json` + `--debug` integration test |
| Performance regression with debug off | LOW | Profile with `go tool pprof`; replace `fmt.Sprintf` with closure-based debug; verify with benchmark |
| CI/non-TTY breakage | LOW | Add `isatty` check to debug writer; strip ANSI codes; add non-TTY test |
| Fixture drift from real responses | MEDIUM | Record new fixtures from real CLI run; update all tests; add weekly contract test CI job |
| Over-engineered debug system | MEDIUM | Delete the framework; replace with `fmt.Fprintf(debugW, "[DEBUG] [%s] %s\n", metricID, msg)` directly in the 5 metric files |
| Concurrent debug state corruption | LOW | Add metric ID prefix to all debug lines; wrap debug writer with `sync.Mutex`; run `go test -race` |
| Flag proliferation | LOW-MEDIUM | Remove extra flags; consolidate to single `--debug`; move developer settings to environment variables |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Debug output polluting stdout | Phase 1: Debug writer foundation | Test: `ars scan --json --debug . 2>/dev/null \| jq .` succeeds |
| Performance regression | Phase 1: Debug writer with zero-cost disabled path | Benchmark: no measurable regression with debug OFF |
| CI/non-TTY breakage | Phase 1: Debug writer TTY detection | Test: run in non-TTY env, output has no ANSI codes |
| Fixture divergence | Phase 2: Debug instrumentation with real response capture | Fixtures updated from real CLI responses; contract test exists |
| Over-engineering | Phase 0: Requirements scoping | Requirements doc lists exactly what debug shows and what it does not |
| Concurrent state corruption | Phase 1: Debug writer concurrency design | `go test -race` passes; debug output has correct metric prefixes |
| Flag proliferation | Phase 0: CLI design decision | Single `--debug` flag; env var for advanced settings |

## Sources

- Codebase analysis: `internal/agent/progress.go` (TTY detection pattern), `internal/pipeline/pipeline.go` (output routing), `internal/agent/parallel.go` (concurrent execution), `internal/agent/metrics/m2_comprehension.go` (heuristic scoring), `cmd/scan.go` (flag definitions)
- [Airflow PR: Print debug mode warning to stderr to avoid polluting stdout JSON output](http://www.mail-archive.com/commits@airflow.apache.org/msg486101.html) -- real-world example of this exact pitfall
- [AWS CLI Issue #5187: --debug output is written to stderr instead of stdout](https://github.com/aws/aws-cli/issues/5187) -- precedent for debug-to-stderr pattern
- [Orhun's Blog: Why stdout is faster than stderr?](https://blog.orhun.dev/stdout-vs-stderr/) -- performance implications of output channel choice
- [Observability Anti-Patterns](https://observability-antipatterns.github.io/) -- over-instrumentation and noise patterns
- [Honeybadger: Logging in Go](https://www.honeybadger.io/blog/golang-logging/) -- Go-specific logging best practices
- [Julien Harbulot: How and when to use stdout and stderr](https://julienharbulot.com/python-cli-streams.html) -- stream separation best practices

---
*Pitfalls research for: ARS C7 Debug Infrastructure*
*Researched: 2026-02-06*
