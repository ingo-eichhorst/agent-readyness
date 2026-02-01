---
phase: 05-hardening
plan: 02
subsystem: pipeline
tags: [spinner, progress, parallel, errgroup, isatty, performance]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "Pipeline orchestration and CLI scan command"
  - phase: 02-core-analysis
    provides: "C1, C3, C6 analyzers that are parallelized"
provides:
  - "Stderr spinner with TTY detection for user feedback"
  - "Parallel analyzer execution via errgroup"
  - "Deterministic result ordering regardless of completion order"
  - "ProgressFunc callback pattern for pipeline stages"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "errgroup for parallel goroutine execution with error collection"
    - "ProgressFunc callback for decoupled progress reporting"
    - "TTY detection to suppress interactive output in CI/piped environments"

key-files:
  created:
    - "internal/pipeline/progress.go"
  modified:
    - "internal/pipeline/pipeline.go"
    - "internal/pipeline/pipeline_test.go"
    - "cmd/scan.go"

key-decisions:
  - "Spinner writes to os.Stderr only, never stdout, preventing --json corruption"
  - "TTY detection via go-isatty (already indirect dep) gates all spinner output"
  - "Analyzer errors in parallel mode return nil to avoid aborting sibling analyzers"
  - "Results sorted by Category string for deterministic C1/C3/C6 ordering"
  - "Parallel test uses baseline measurement to isolate analyzer timing from pipeline overhead"

patterns-established:
  - "ProgressFunc callback: decouples progress UI from pipeline logic"
  - "Spinner lifecycle: Start -> Update (per stage) -> Stop pattern"

# Metrics
duration: 3min
completed: 2026-02-01
---

# Phase 5 Plan 2: Progress Indicators and Parallel Analyzers Summary

**Stderr spinner with TTY detection and parallel analyzer execution via errgroup for reduced scan wall-clock time**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-01T10:46:18Z
- **Completed:** 2026-02-01T10:49:31Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Spinner displays animated progress on stderr during scans, auto-suppressed in non-TTY environments
- Three analyzers (C1, C3, C6) now execute in parallel via errgroup, reducing analysis wall-clock time
- Results are deterministically sorted by category regardless of goroutine completion order
- ProgressFunc callback pattern decouples progress reporting from pipeline internals

## Task Commits

Each task was committed atomically:

1. **Task 1: Create stderr spinner with TTY detection** - `c40d2e7` (feat)
2. **Task 2: Parallelize analyzers, add progress callbacks, wire spinner** - `eca30c7` (feat)

## Files Created/Modified
- `internal/pipeline/progress.go` - Spinner struct with TTY detection, ProgressFunc type
- `internal/pipeline/pipeline.go` - Parallel errgroup execution, progress callbacks, deterministic sorting
- `internal/pipeline/pipeline_test.go` - TestParallelAnalyzers (timing + ordering), TestProgressCallbackInvoked
- `cmd/scan.go` - Spinner wired into scan command lifecycle

## Decisions Made
- Spinner writes exclusively to os.Stderr, preventing corruption of --json stdout output
- TTY detection uses go-isatty (already an indirect dependency via fatih/color) -- no new deps
- Analyzer errors in parallel mode return nil rather than error, so sibling analyzers continue
- Results sorted by Category string (C1 < C3 < C6) for deterministic output
- Parallel timing test uses baseline measurement approach to isolate analyzer time from pipeline overhead

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Adjusted parallel test timing approach**
- **Found during:** Task 2 (TestParallelAnalyzers)
- **Issue:** Original test with 50ms sleep and 200ms threshold failed because pipeline discovery/parsing overhead (~700ms) was included in the measurement
- **Fix:** Added baseline measurement (run pipeline without analyzers) and compare only the analyzer portion of execution time
- **Files modified:** internal/pipeline/pipeline_test.go
- **Verification:** Test passes reliably, correctly proves parallel execution
- **Committed in:** eca30c7 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Test timing approach needed adjustment for reliability. No scope creep.

## Issues Encountered
None beyond the test timing fix documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Spinner and parallel execution are ready for production use
- No blockers for remaining Phase 5 plans

---
*Phase: 05-hardening*
*Completed: 2026-02-01*
