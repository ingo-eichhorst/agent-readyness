---
phase: 29-debug-rendering-replay
plan: 02
subsystem: agent
tags: [debug, c7, replay, persistence, cli-flag]

requires:
  - phase: 29-01
    provides: "RenderC7Debug function, Pipeline Stage 3.7 wiring"
  - phase: 24-03
    provides: "RunMetricsParallel, ParallelResult, metrics.Executor interface"
provides:
  - "ReplayExecutor implementing metrics.Executor for response replay"
  - "SaveResponses/LoadResponses for JSON persistence of C7 metric responses"
  - "--debug-dir CLI flag for response capture and replay"
  - "Pipeline.SetDebugDir for threading debug directory to C7Analyzer"
affects: [29-03]

tech-stack:
  added: []
  patterns:
    - "Executor injection: RunMetricsParallel accepts optional executor (nil = default CLI)"
    - "Capture/replay mode detection: LoadResponses success + non-empty = replay, otherwise capture"
    - "Prompt-based metric identification via substring matching for replay routing"

key-files:
  created:
    - internal/agent/replay.go
    - internal/agent/replay_test.go
  modified:
    - cmd/scan.go
    - internal/pipeline/pipeline.go
    - internal/agent/parallel.go
    - internal/agent/parallel_test.go
    - internal/analyzer/c7_agent/agent.go

key-decisions:
  - "Prompt-based metric identification over passing metricID through Executor interface -- preserves interface stability"
  - "Nil executor parameter over separate function -- simpler API, backward compatible with nil default"
  - "--debug-dir implies --debug-c7 -- single flag convenience for common workflow"
  - "Capture/replay mode auto-detected from directory contents -- no explicit mode flag needed"

patterns-established:
  - "identifyMetricFromPrompt helper for prompt-to-metric routing"
  - "DebugResponse struct as canonical JSON schema for C7 response persistence"
  - "Executor injection pattern for RunMetricsParallel/Sequential"

duration: 6min
completed: 2026-02-06
---

# Phase 29 Plan 02: Response Persistence & Replay Summary

**ReplayExecutor + SaveResponses/LoadResponses enable --debug-dir flag for C7 response JSON persistence and instant replay without Claude CLI**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-06T15:43:44Z
- **Completed:** 2026-02-06T15:50:06Z
- **Tasks:** 2/2
- **Files created:** 2
- **Files modified:** 5

## Accomplishments

- Created `DebugResponse` struct with metric_id, sample_index, file_path, prompt, response, duration_seconds, error fields
- Implemented `SaveResponses` to persist each sample as individual `{metric_id}_{sample_index}.json` files
- Implemented `LoadResponses` to read all JSON files into a keyed map for replay lookup
- Created `ReplayExecutor` implementing `metrics.Executor` with prompt-based metric identification
- `identifyMetricFromPrompt` detects M1-M5 via case-insensitive substring matching on distinctive prompt patterns
- Added `--debug-dir` CLI flag that implies `--debug-c7` and resolves to absolute path
- Threaded debugDir through `Pipeline.SetDebugDir` -> `C7Analyzer.SetDebugDir`
- Modified `RunMetricsParallel` and `RunMetricsSequential` to accept optional `executor metrics.Executor` parameter (nil = default CLIExecutorAdapter)
- Auto-detects replay mode when debugDir contains existing JSON files, capture mode otherwise
- Capture mode saves all responses after RunMetricsParallel completes
- Replay mode prints `[C7 DEBUG] Replay mode:` on stderr, capture prints `[C7 DEBUG] Capture mode:`
- 7 new tests covering save/load round-trip, replay behavior, error cases, and prompt identification

## Task Details

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Create replay.go with DebugResponse, SaveResponses, LoadResponses, ReplayExecutor | 940e5dc | internal/agent/replay.go, internal/agent/replay_test.go |
| 2 | Wire --debug-dir flag through CLI, pipeline, and C7 analyzer | 32d0507 | cmd/scan.go, internal/pipeline/pipeline.go, internal/agent/parallel.go, internal/analyzer/c7_agent/agent.go |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed `contains` function name collision in replay_test.go**

- **Found during:** Task 1
- **Issue:** `contains` helper in replay_test.go collided with same-named function in scorer_test.go (same package)
- **Fix:** Replaced local `contains` helper with `strings.Contains` from standard library
- **Files modified:** internal/agent/replay_test.go

## Verification

- `go build ./...` -- compiles without errors
- `go test ./... -count=1` -- full test suite passes (all 18 packages green, zero failures)
- `go test ./internal/agent/ -run TestSaveLoad -v` -- save/load round-trip passes
- `go test ./internal/agent/ -run TestReplayExecutor -v` -- replay executor passes (including error cases)
- `go test ./internal/agent/ -run TestIdentifyMetric -v` -- all 11 prompt identification cases pass
- No regressions from RunMetricsParallel signature change (all existing parallel_test.go tests updated and passing)
