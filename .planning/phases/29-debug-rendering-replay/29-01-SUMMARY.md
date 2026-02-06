---
phase: 29-debug-rendering-replay
plan: 01
subsystem: output
tags: [debug, c7, terminal, rendering, stderr]

requires:
  - phase: 27-c7-debug-capture
    provides: "C7DebugSample and C7ScoreTrace types, debug data capture in analyzer"
  - phase: 26-c7-debug-infra
    provides: "io.Writer debug pattern, SetC7Debug pipeline method"
provides:
  - "RenderC7Debug function for terminal debug output"
  - "Pipeline Stage 3.7 wiring for debug rendering"
  - "Score trace rendering with indicator breakdown"
affects: [29-02, 29-03]

tech-stack:
  added: []
  patterns:
    - "Separate debug rendering function (RenderC7Debug) vs normal rendering (renderC7)"
    - "Score trace formatting: base={N} indicator(+delta) -> final={N}"

key-files:
  created: []
  modified:
    - internal/output/terminal.go
    - internal/output/terminal_test.go
    - internal/pipeline/pipeline.go

key-decisions:
  - "Prompt truncated to 200 chars, response to 500 chars for readable debug output"
  - "Dim color (FgHiBlack) for prompts, red for errors, bold for headers"
  - "Only matched indicators shown in trace line (unmatched omitted for brevity)"
  - "Stage 3.7 placed after scoring (3.6) and before render output (4)"

patterns-established:
  - "renderScoreTrace helper for formatting indicator breakdown"
  - "truncateString utility for safe string truncation with ellipsis"

duration: 5min
completed: 2026-02-06
---

# Phase 29 Plan 01: Debug Rendering Summary

**RenderC7Debug function renders per-metric, per-sample debug data with truncated prompts, score traces, and indicator breakdowns to stderr**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-06T15:34:51Z
- **Completed:** 2026-02-06T15:40:21Z
- **Tasks:** 2/2
- **Files modified:** 3

## Accomplishments

- Implemented `RenderC7Debug` in `internal/output/terminal.go` with full per-metric, per-sample rendering
- Prompts truncated to 200 chars, responses to 500 chars, with dim/red/bold color coding
- Score trace shows `base={N} indicator(+delta) -> final={N}` format with only matched indicators
- Wired into pipeline as Stage 3.7 (after scoring, before render output) via `p.debugWriter`
- Added 3 tests: normal rendering, empty debug samples, and missing C7 result

## Task Details

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Implement RenderC7Debug in terminal.go | 8384f7c | internal/output/terminal.go |
| 2 | Wire into pipeline + add tests | e4d884d | internal/pipeline/pipeline.go, internal/output/terminal_test.go |

## Deviations from Plan

None -- plan executed exactly as written.

## Verification

- `go build ./...` -- compiles without errors
- `go test ./internal/output/ -run TestRenderC7Debug -v` -- all 3 tests pass
- `go test ./internal/pipeline/ -v` -- all pipeline tests pass (no regression)
- `go test ./... -count=1` -- full test suite passes (all packages green)
