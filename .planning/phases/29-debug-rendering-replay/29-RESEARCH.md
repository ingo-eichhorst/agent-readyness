# Phase 29: Debug Rendering & Replay - Research

**Researched:** 2026-02-06
**Domain:** Terminal debug output rendering, JSON file persistence, response replay via Executor interface
**Confidence:** HIGH

## Summary

Phase 29 completes the C7 debug infrastructure by adding three capabilities: (1) rendering debug data to stderr during terminal output, (2) persisting captured responses to a `--debug-dir` directory as JSON files, and (3) replaying those files on subsequent runs to skip Claude CLI execution. The phase also includes CLI help documentation and updating GitHub issue #55.

The codebase is fully prepared for this work. Phases 26-28 built the complete foundation: the `--debug-c7` flag exists and threads through Pipeline to C7Analyzer, the `debugWriter io.Writer` channel routes to `os.Stderr` or `io.Discard`, `C7DebugSample` structs capture prompt/response/score/trace data, and heuristic scoring works correctly with grouped indicators. The `metrics.Executor` interface (`ExecutePrompt`) is the clean abstraction boundary for replay -- implementing a `ReplayExecutor` that reads from files requires zero changes to metric logic.

The three sub-plans map directly to the codebase: (29-01) add a `renderC7Debug` function in `internal/output/terminal.go` called from `pipeline.go` when `debugC7` is true; (29-02) add `--debug-dir` flag to `cmd/scan.go`, implement save/load in a new `internal/agent/replay.go`, and wire a `ReplayExecutor` into the parallel execution flow; (29-03) update flag descriptions, README usage section, and GitHub issue #55.

**Primary recommendation:** Implement replay at the `metrics.Executor` interface level by creating a `ReplayExecutor` that loads JSON files keyed by `{metric_id}_{sample_index}`. When `--debug-dir` points to a directory with existing files, swap the `CLIExecutorAdapter` for `ReplayExecutor`. When files are absent, use `CLIExecutorAdapter` and save responses after execution. This approach requires no changes to any metric implementation.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/json` | stdlib | Serialize/deserialize debug samples to JSON files | Already used throughout the codebase for JSON output |
| `os` | stdlib | Directory creation (`os.MkdirAll`), file I/O | Standard Go file operations |
| `path/filepath` | stdlib | Construct file paths for debug directory | Already used in pipeline.go, cmd/scan.go |
| `fmt.Fprintf` | stdlib | Write debug rendering to `io.Writer` (stderr) | Consistent with all existing terminal output |
| `spf13/cobra` | v1.10.2 | `--debug-dir` flag registration (`StringVar`) | Already used for all existing scan flags |
| `fatih/color` | existing | Colored debug output on stderr | Already used by all terminal rendering functions |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `strings.Builder` | stdlib | Efficient string construction for debug output | When formatting multi-line debug blocks |
| `io` | stdlib | `io.Writer` interface for debug channel | Already used throughout for output abstraction |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Raw JSON files per sample | Single SQLite database | Overkill -- JSON files are human-readable, greppable, and sufficient for 5 metrics x 1-5 samples |
| Custom binary format | Protobuf or msgpack | No benefit -- responses are text, JSON is native to Go, and human readability matters for debugging |
| Separate `--replay-from` flag | Automatic replay from `--debug-dir` | Extra flag adds UX complexity; auto-detecting existing files in `--debug-dir` is simpler and matches the success criteria |

**Installation:** No new dependencies required. All functionality uses stdlib and existing dependencies.

## Architecture Patterns

### Recommended File Layout for New Code
```
cmd/scan.go                              # Add --debug-dir flag
internal/output/terminal.go              # Add renderC7Debug function
internal/agent/replay.go                 # NEW: ReplayExecutor + save/load functions
internal/agent/replay_test.go            # NEW: Tests for replay
internal/pipeline/pipeline.go            # Wire debug rendering + replay executor
```

### Pattern 1: Debug Rendering via Separate Function in terminal.go
**What:** Add a `RenderC7Debug(w io.Writer, analysisResults []*types.AnalysisResult)` function in `internal/output/terminal.go` that writes per-metric, per-sample debug blocks to the provided writer (stderr).
**When to use:** Called from `pipeline.go` after analysis completes, gated on `p.debugC7`.
**Why this pattern:** Follows the exact same pattern as `RenderSummary`, `RenderScores`, `RenderRecommendations` -- each is a standalone function called from the pipeline. Debug rendering is no different.

```go
// internal/output/terminal.go

// RenderC7Debug writes detailed C7 debug data to w (typically os.Stderr).
// Shows per-metric, per-sample: prompt (truncated), response (truncated),
// score, duration, and indicator trace.
func RenderC7Debug(w io.Writer, analysisResults []*types.AnalysisResult) {
    // Find C7 result
    for _, ar := range analysisResults {
        if ar.Category != "C7" {
            continue
        }
        raw, ok := ar.Metrics["c7"]
        if !ok {
            return
        }
        m, ok := raw.(*types.C7Metrics)
        if !ok || !m.Available {
            return
        }

        bold := color.New(color.Bold)
        dim := color.New(color.FgHiBlack)

        fmt.Fprintln(w)
        bold.Fprintln(w, "C7 Debug: Agent Evaluation Details")
        fmt.Fprintln(w, strings.Repeat("=", 60))

        for _, mr := range m.MetricResults {
            fmt.Fprintln(w)
            bold.Fprintf(w, "[%s] %s  score=%d/10  (%.1fs)\n",
                mr.MetricID, mr.MetricName, mr.Score, mr.Duration)
            fmt.Fprintln(w, strings.Repeat("-", 50))

            for i, ds := range mr.DebugSamples {
                fmt.Fprintf(w, "  Sample %d: %s\n", i+1, ds.Description)
                fmt.Fprintf(w, "  File:     %s\n", ds.FilePath)
                fmt.Fprintf(w, "  Score:    %d/10  Duration: %.1fs\n", ds.Score, ds.Duration)

                // Prompt (truncated to 200 chars)
                prompt := ds.Prompt
                if len(prompt) > 200 {
                    prompt = prompt[:200] + "..."
                }
                dim.Fprintf(w, "  Prompt:   %s\n", prompt)

                // Response (truncated to 500 chars)
                response := ds.Response
                if len(response) > 500 {
                    response = response[:500] + "..."
                }
                fmt.Fprintf(w, "  Response: %s\n", response)

                // Score trace
                fmt.Fprintf(w, "  Trace:    base=%d", ds.ScoreTrace.BaseScore)
                for _, ind := range ds.ScoreTrace.Indicators {
                    if ind.Matched {
                        fmt.Fprintf(w, " %s(%+d)", ind.Name, ind.Delta)
                    }
                }
                fmt.Fprintf(w, " -> final=%d\n", ds.ScoreTrace.FinalScore)

                if ds.Error != "" {
                    color.New(color.FgRed).Fprintf(w, "  Error:    %s\n", ds.Error)
                }
                fmt.Fprintln(w)
            }
        }
    }
}
```

### Pattern 2: ReplayExecutor Implementing metrics.Executor Interface
**What:** A `ReplayExecutor` struct that implements `metrics.Executor.ExecutePrompt()` by loading responses from JSON files in a debug directory, keyed by prompt hash or `{metric_id}_{sample_index}`.
**When to use:** When `--debug-dir` is specified and the directory contains previously saved response files.
**Why this pattern:** The `metrics.Executor` interface is the clean abstraction boundary established in Phase 24. All 5 metrics call `executor.ExecutePrompt()` without knowing the implementation. Swapping `CLIExecutorAdapter` for `ReplayExecutor` requires zero changes to metric logic.

```go
// internal/agent/replay.go

// DebugResponse represents a single captured response for replay.
type DebugResponse struct {
    MetricID    string  `json:"metric_id"`
    SampleIndex int     `json:"sample_index"`
    FilePath    string  `json:"file_path"`
    Prompt      string  `json:"prompt"`
    Response    string  `json:"response"`
    Duration    float64 `json:"duration_seconds"`
    Error       string  `json:"error,omitempty"`
}

// SaveResponses writes captured responses to JSON files in debugDir.
func SaveResponses(debugDir string, results []metrics.MetricResult) error {
    if err := os.MkdirAll(debugDir, 0755); err != nil {
        return fmt.Errorf("create debug dir: %w", err)
    }
    for _, mr := range results {
        for i, sr := range mr.Samples {
            resp := DebugResponse{
                MetricID:    mr.MetricID,
                SampleIndex: i,
                FilePath:    sr.Sample.FilePath,
                Prompt:      sr.Prompt,
                Response:    sr.Response,
                Duration:    sr.Duration.Seconds(),
                Error:       sr.Error,
            }
            filename := fmt.Sprintf("%s_%d.json", mr.MetricID, i)
            path := filepath.Join(debugDir, filename)
            data, err := json.MarshalIndent(resp, "", "  ")
            if err != nil {
                return fmt.Errorf("marshal %s: %w", filename, err)
            }
            if err := os.WriteFile(path, data, 0644); err != nil {
                return fmt.Errorf("write %s: %w", filename, err)
            }
        }
    }
    return nil
}

// LoadResponses reads all captured responses from debugDir.
// Returns a map keyed by "{metric_id}_{sample_index}" for O(1) lookup.
func LoadResponses(debugDir string) (map[string]DebugResponse, error) {
    entries, err := os.ReadDir(debugDir)
    if err != nil {
        return nil, err
    }
    responses := make(map[string]DebugResponse)
    for _, entry := range entries {
        if filepath.Ext(entry.Name()) != ".json" {
            continue
        }
        data, err := os.ReadFile(filepath.Join(debugDir, entry.Name()))
        if err != nil {
            continue // Skip unreadable files
        }
        var resp DebugResponse
        if err := json.Unmarshal(data, &resp); err != nil {
            continue // Skip malformed files
        }
        key := fmt.Sprintf("%s_%d", resp.MetricID, resp.SampleIndex)
        responses[key] = resp
    }
    return responses, nil
}
```

### Pattern 3: Replay Detection via Directory Presence
**What:** When `--debug-dir` is specified, check if the directory already contains `.json` files. If yes, enter replay mode (load responses, skip CLI execution). If no, enter capture mode (execute CLI, save responses).
**When to use:** In the pipeline setup, before metric execution begins.
**Why this pattern:** Matches success criterion #3: "Running the same command a second time replays saved responses." No separate flag needed -- the presence of files is the signal.

```go
// In pipeline.go or c7_agent/agent.go

func (p *Pipeline) resolveC7Executor(workDir string) metrics.Executor {
    if p.debugDir != "" {
        // Check if debug dir has existing responses
        responses, err := agent.LoadResponses(p.debugDir)
        if err == nil && len(responses) > 0 {
            fmt.Fprintf(p.debugWriter, "[C7 DEBUG] Replay mode: loading %d responses from %s\n",
                len(responses), p.debugDir)
            return agent.NewReplayExecutor(responses)
        }
        // No existing responses -- use real CLI with capture
        fmt.Fprintf(p.debugWriter, "[C7 DEBUG] Capture mode: responses will be saved to %s\n",
            p.debugDir)
    }
    return agent.NewCLIExecutorAdapter(workDir)
}
```

### Anti-Patterns to Avoid
- **Saving entire C7Metrics struct as one file:** Each file should be one sample response. This enables per-sample inspection, selective deletion, and partial replay.
- **Using prompt text as file key:** Prompts change when metrics are updated. Use `{metric_id}_{sample_index}` which is stable.
- **Embedding replay logic inside metric Execute() functions:** Keep metric code unaware of replay. The Executor interface handles this transparently.
- **Creating debug-dir unconditionally:** Only create the directory when `--debug-dir` is explicitly provided. Never create `.ars-debug/` or similar implicitly.
- **Mixing debug output with stdout:** All debug rendering must go to stderr only. This is already enforced by the `debugWriter io.Writer` from Phase 26.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON serialization | Custom text format | `encoding/json` with `MarshalIndent` | JSON is self-documenting, parseable, and matches existing project patterns |
| File naming scheme | UUID-based names | `{metric_id}_{sample_index}.json` | Deterministic, human-readable, enables targeted replay |
| Directory check for replay mode | Custom filesystem walker | `os.ReadDir` + count `.json` files | Simple, idiomatic Go, sufficient for 5-25 files |
| Response truncation for display | Regex-based summarization | Simple `string[:N] + "..."` | Truncation is for display only; full response is in the JSON file |
| Colored terminal output | ANSI escape codes | `fatih/color` (already imported) | Already used by all other terminal rendering |

## Common Pitfalls

### Pitfall 1: Debug Output Corrupting JSON Mode
**What goes wrong:** Debug rendering writes to stdout, breaking `--json` piped output.
**Why it happens:** Easy to use `fmt.Println` instead of `fmt.Fprintf(debugWriter, ...)`.
**How to avoid:** All debug rendering functions take an `io.Writer` parameter. Pipeline passes `p.debugWriter` (which is `os.Stderr` when debug is on, `io.Discard` otherwise). Never use `p.writer` (stdout) for debug output.
**Warning signs:** `ars scan . --debug-c7 --json 2>/dev/null | jq` fails to parse.
**Verification test:** `go test` should include a test that captures stdout and verifies no debug content leaks.

### Pitfall 2: Replay File Key Mismatch
**What goes wrong:** Replay loads files but cannot match them to current metrics because the keying scheme changed.
**Why it happens:** If file naming uses prompt content or sample file paths, these change between runs when the codebase changes.
**How to avoid:** Use `{metric_id}_{sample_index}` as the stable key. Metric IDs are constants (`task_execution_consistency`, etc.). Sample indices are deterministic because `SelectSamples` sorts by `SelectionScore` descending.
**Warning signs:** Replay reports "0 of 5 responses loaded" despite files existing.
**Caveat:** If the scanned codebase changes significantly between capture and replay (files added/removed), sample selection may differ. This is acceptable -- replay is for heuristic iteration, not cross-project comparison.

### Pitfall 3: Race Condition in Parallel Save
**What goes wrong:** When 5 metrics complete in parallel and all try to save to the same directory, file writes can interleave.
**Why it happens:** `RunMetricsParallel` uses goroutines.
**How to avoid:** Save responses after all metrics complete (in `buildMetrics` or after `RunMetricsParallel` returns), not during parallel execution. The `ParallelResult.Results` slice contains all responses once `Wait()` returns.
**Warning signs:** Corrupt or missing JSON files in debug-dir.

### Pitfall 4: Large Response Files
**What goes wrong:** Some Claude CLI responses can be 1000+ words. Saving with full prompt + response + trace creates large files.
**Why it happens:** M2/M3 metrics ask for comprehensive explanations of complex code.
**How to avoid:** This is not actually a problem -- files will be 5-50KB each, totaling 50-250KB for a full run. Do not truncate saved responses; truncation is only for terminal display.
**Warning signs:** None expected. If files exceed 1MB, something is wrong with the CLI output parsing.

### Pitfall 5: Debug Samples Empty When Debug Not Enabled
**What goes wrong:** `renderC7Debug` is called but `DebugSamples` is nil because `SetDebug(true, ...)` was not called.
**Why it happens:** `--debug-c7` auto-enables C7 and sets debug, but if the code path is wrong, debug might not propagate to the analyzer.
**How to avoid:** The existing code in `cmd/scan.go` already handles this correctly: `debugC7 = true` triggers `enableC7 = true` and `p.SetC7Debug(true)`. Verify with a test that `DebugSamples` is populated when debug is on.
**Warning signs:** Debug output shows metric names and scores but no sample details.

### Pitfall 6: debug-dir Path Handling
**What goes wrong:** Relative paths like `./debug-out` resolve differently depending on working directory.
**Why it happens:** Go resolves relative paths from the process's working directory, not the scanned directory.
**How to avoid:** Convert `--debug-dir` to absolute path using `filepath.Abs()` at flag processing time (in `cmd/scan.go`), just like the scan directory is converted.
**Warning signs:** Files saved in unexpected locations.

## Code Examples

### Example 1: Wiring Debug Rendering in Pipeline
```go
// internal/pipeline/pipeline.go - in Run() method, after analysis and scoring

// Stage 3.7: Debug rendering (after analysis, before normal output)
if p.debugC7 && p.results != nil {
    output.RenderC7Debug(p.debugWriter, p.results)
}
```

### Example 2: Wiring --debug-dir Flag
```go
// cmd/scan.go - in var block
var debugDir string // Directory for response persistence/replay

// cmd/scan.go - in init()
scanCmd.Flags().StringVar(&debugDir, "debug-dir", "",
    "directory for C7 response persistence and replay (implies --debug-c7)")

// cmd/scan.go - in RunE (before pipeline setup)
if debugDir != "" {
    debugC7 = true  // --debug-dir implies --debug-c7
    var err error
    debugDir, err = filepath.Abs(debugDir)
    if err != nil {
        return fmt.Errorf("invalid debug-dir path: %w", err)
    }
}

// After pipeline creation
if debugDir != "" {
    p.SetDebugDir(debugDir)
}
```

### Example 3: ReplayExecutor
```go
// internal/agent/replay.go

type ReplayExecutor struct {
    responses map[string]DebugResponse
    callIndex map[string]int // Tracks call count per metric for sample indexing
    mu        sync.Mutex
}

func NewReplayExecutor(responses map[string]DebugResponse) *ReplayExecutor {
    return &ReplayExecutor{
        responses: responses,
        callIndex: make(map[string]int),
    }
}

// ExecutePrompt returns a pre-recorded response instead of calling Claude CLI.
// Implements metrics.Executor interface.
func (r *ReplayExecutor) ExecutePrompt(ctx context.Context, workDir, prompt, tools string, timeout time.Duration) (string, error) {
    r.mu.Lock()
    // Determine metric ID from prompt content (heuristic) or use call ordering
    // Since metrics call ExecutePrompt sequentially within each metric's Execute(),
    // we track calls per-goroutine using the prompt to identify the metric.
    metricID := identifyMetricFromPrompt(prompt)
    idx := r.callIndex[metricID]
    r.callIndex[metricID] = idx + 1
    r.mu.Unlock()

    key := fmt.Sprintf("%s_%d", metricID, idx)
    resp, ok := r.responses[key]
    if !ok {
        return "", fmt.Errorf("no replay data for %s", key)
    }
    if resp.Error != "" {
        return "", fmt.Errorf("replayed error: %s", resp.Error)
    }
    return resp.Response, nil
}

var _ metrics.Executor = (*ReplayExecutor)(nil)
```

### Example 4: Saving Responses After Parallel Execution
```go
// internal/agent/parallel.go or internal/pipeline/pipeline.go

// After RunMetricsParallel returns and before buildMetrics:
if p.debugDir != "" {
    if err := agent.SaveResponses(p.debugDir, result.Results); err != nil {
        fmt.Fprintf(p.debugWriter, "[C7 DEBUG] Warning: failed to save responses: %v\n", err)
    }
}
```

### Example 5: Debug JSON File Format
```json
{
  "metric_id": "code_behavior_comprehension",
  "sample_index": 0,
  "file_path": "internal/scoring/scorer.go",
  "prompt": "Read the file at internal/scoring/scorer.go and explain what the code does.\n\nFocus on:\n1. The main purpose/behavior of the code\n2. Important control flow paths (branches, loops)\n3. Error handling and edge cases\n4. Return values and side effects\n\nBe specific and reference actual code elements.",
  "response": "The file `internal/scoring/scorer.go` implements the scoring engine...",
  "duration_seconds": 12.5,
  "error": ""
}
```

## Design Decisions

### Decision 1: Replay Detection via File Presence (Not Separate Flag)
**Decided:** Auto-detect replay when `--debug-dir` contains `.json` files. No separate `--replay-from` flag.
**Rationale:** The success criteria specify "Running the same command a second time replays." This means the same `--debug-dir` flag triggers both save and replay. Auto-detection is simpler UX and matches the spec exactly.
**Alternative rejected:** Separate `--replay-from` flag adds cognitive load and does not match the success criteria.

### Decision 2: One JSON File Per Sample (Not Per Metric or Per Run)
**Decided:** Save as `{metric_id}_{sample_index}.json`, one file per sample evaluation.
**Rationale:** Granular files enable: (a) inspecting one sample without parsing large files, (b) deleting specific responses to force re-evaluation, (c) human-readable filenames like `code_behavior_comprehension_0.json`.
**Alternative rejected:** Single `responses.json` file would be harder to inspect and prevent partial replay.

### Decision 3: Metric Identification in Replay via Prompt Content Matching
**Decided:** The `ReplayExecutor` identifies which metric is calling `ExecutePrompt` by matching prompt patterns.
**Rationale:** The `Executor` interface only receives `prompt` and `tools` -- there is no `metricID` parameter. Rather than changing the interface (which would require updating all 5 metrics), match on known prompt patterns. Each metric has distinctive prompt text:
- M1: "list all function names"
- M2: "explain what the code does"
- M3: "Trace the dependencies" / "trace the complete dependency chain"
- M4: "interpret what the identifier"
- M5: "review the documentation" / "identify any inaccuracies"

**Alternative considered:** Add `metricID` to `Executor` interface. This would be cleaner but requires changing the interface signature and updating all 5 metrics and the CLIExecutorAdapter. The prompt-matching approach avoids this churn.

**Better alternative:** Store the metric_id in the saved file and use a `CaptureExecutor` wrapper that records prompt-to-metricID mappings during the first run. On replay, use the saved prompt text as the lookup key (exact string match on the prompt). This is more robust than pattern matching.

### Decision 4: Debug Rendering Shows Truncated Output, Files Store Full Data
**Decided:** Terminal debug output truncates prompts (200 chars) and responses (500 chars). JSON files store everything.
**Rationale:** Terminal is for quick inspection; files are for deep analysis. This matches the success criterion: "displays per-metric per-sample prompts, responses (truncated), scores, and durations on stderr."

## Integration Points

### Where New Code Touches Existing Code

| New | Existing | Change Type |
|-----|----------|-------------|
| `--debug-dir` flag | `cmd/scan.go` | Add `StringVar` in `init()`, handle in `RunE` |
| `Pipeline.SetDebugDir()` | `internal/pipeline/pipeline.go` | New method + `debugDir string` field |
| `RenderC7Debug()` | `internal/output/terminal.go` | New function, called from pipeline |
| `ReplayExecutor` | `internal/agent/` | New file, implements existing `metrics.Executor` interface |
| `SaveResponses()` | `internal/agent/` | New file, called from pipeline after analysis |
| Executor swap | `internal/analyzer/c7_agent/agent.go` OR `internal/pipeline/pipeline.go` | Thread executor choice to `RunMetricsParallel` |
| CLI help text | `cmd/scan.go` | Update flag descriptions |
| README section | `README.md` | Add debug/replay usage examples |

### Critical Integration: Executor Swap in C7Analyzer
The most architecturally significant change is intercepting the executor creation. Currently:
- `agent.go:90` calls `agent.RunMetricsParallel(ctx, workDir, targets, progress)`
- `parallel.go:36` creates `executor := NewCLIExecutorAdapter(workDir)`

The replay executor needs to replace this. Two approaches:

**Option A (Simpler):** Add `executor metrics.Executor` field to `C7Analyzer`, default to nil. If non-nil, pass to `RunMetricsParallel` instead of creating a new `CLIExecutorAdapter`.

**Option B (Cleaner):** Modify `RunMetricsParallel` to accept an `executor metrics.Executor` parameter. The pipeline decides which executor to use.

Recommend **Option B** because it keeps decision logic in the pipeline where `--debug-dir` state lives, rather than pushing file system concerns into the analyzer.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No debug output | `--debug-c7` flag with io.Writer channel | Phase 26 (2026-02-06) | Debug flag exists, routes to stderr |
| No data capture | `C7DebugSample` structs with ScoreTrace | Phase 27 (2026-02-06) | Full prompt/response/trace captured when debug on |
| 0/10 scores | Grouped indicators with variable BaseScore | Phase 28 (2026-02-06) | Scoring produces realistic 1-10 values |
| No persistence | **This phase: JSON files in --debug-dir** | Phase 29 (planned) | Offline analysis + replay |

## Open Questions

1. **Executor identity for replay lookup**
   - What we know: The `Executor` interface only passes `prompt` and `tools`, not `metricID`.
   - What's unclear: Whether prompt-based matching is robust enough, or if we need to change the interface.
   - Recommendation: Use prompt text as an exact-match key (hash the prompt string). This avoids both pattern matching fragility and interface changes. Store `prompt_hash` in the JSON file for lookup.

2. **M1 Consistency runs the same prompt 3 times**
   - What we know: M1 calls `ExecutePrompt` 3 times with the exact same prompt for consistency measurement.
   - What's unclear: Should replay return the same response 3 times (defeating the variance measurement) or store 3 separate responses?
   - Recommendation: Store all 3 responses as separate files (`task_execution_consistency_0.json`, `_1.json`, `_2.json`). Replay returns them in order. This preserves the original variance data. Note: this means replay M1 scores will always show zero variance (identical scores each run), which is expected behavior for replay.

3. **GitHub issue #55 update scope**
   - What we know: Issue #55 documents the 0/10 scoring problem for M2/M3/M4.
   - What's unclear: How much detail to include in the update (just "fixed" vs full root cause analysis).
   - Recommendation: Update with: (a) root cause (heuristic saturation from individual indicators), (b) fix (grouped indicators + variable BaseScore from Phase 28), (c) current score ranges from fixture tests, (d) debug mode usage instructions. This is DOC-01/DOC-02.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `internal/agent/metrics/metric.go` -- `Executor` interface definition (line 76-78) [Direct code reading]
- Codebase analysis: `internal/agent/executor_adapter.go` -- `CLIExecutorAdapter` implementation [Direct code reading]
- Codebase analysis: `internal/agent/parallel.go` -- `RunMetricsParallel` executor creation (line 36) [Direct code reading]
- Codebase analysis: `internal/analyzer/c7_agent/agent.go` -- `C7Analyzer` debug fields and `buildMetrics` [Direct code reading]
- Codebase analysis: `internal/output/terminal.go` -- `renderC7` existing output pattern [Direct code reading]
- Codebase analysis: `cmd/scan.go` -- Flag registration and pipeline wiring [Direct code reading]
- Codebase analysis: `pkg/types/types.go` -- `C7DebugSample`, `C7ScoreTrace` types [Direct code reading]
- Planning document: `.planning/research/FEATURES-c7-debug.md` -- Feature research including replay architecture [Project planning]
- Planning document: `.planning/ROADMAP.md` -- Phase 29 success criteria and plan outlines [Project planning]
- Prior research: `.planning/phases/26-debug-foundation/26-RESEARCH.md` -- Debug channel architecture [Project planning]

### Secondary (MEDIUM confidence)
- Go stdlib `encoding/json`: `MarshalIndent` for pretty-printed JSON files [Training data, verified by stdlib docs]
- Go stdlib `os.MkdirAll`: Recursive directory creation [Training data, verified by stdlib docs]
- `spf13/cobra`: `StringVar` for string flag registration [Verified by existing codebase usage]

### Tertiary (LOW confidence)
- None. All findings are based on direct codebase analysis.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- All stdlib, no new dependencies
- Architecture: HIGH -- Direct codebase analysis shows exact integration points
- Pitfalls: HIGH -- Based on actual code paths and existing patterns
- Replay design: MEDIUM -- The prompt-based lookup key needs validation during implementation

**Research date:** 2026-02-06
**Valid until:** 2026-03-06 (stable -- internal tool, no external dependencies changing)
