---
phase: 27-data-capture
plan: 02
subsystem: agent-evaluation
tags: [c7, debug, types, omitempty, score-trace]

# Dependency graph
requires:
  - phase: 27-data-capture-01
    provides: "ScoreTrace, IndicatorMatch, Prompt/Response fields on SampleResult"
  - phase: 26-debug-foundation
    provides: "debug flag threading (SetDebug), io.Writer pattern"
provides:
  - "C7DebugSample, C7ScoreTrace, C7IndicatorMatch output types in pkg/types"
  - "Conditional debug sample population in buildMetrics()"
  - "JSON omitempty behavior for zero-cost non-debug output"
affects: [29-debug-output, 28-scoring-fix]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Type bridging: internal metrics types -> output types via convertScoreTrace helper"
    - "Conditional population: debug data only allocated when a.debug is true"

key-files:
  modified:
    - "pkg/types/types.go"
    - "internal/analyzer/c7_agent/agent.go"
    - "internal/analyzer/c7_agent/agent_test.go"

key-decisions:
  - "Separate output types from internal types: C7ScoreTrace mirrors metrics.ScoreTrace but lives in pkg/types for output boundary"
  - "omitempty only on DebugSamples field: existing C7MetricResult fields lack json tags, maintained consistency"

patterns-established:
  - "convertScoreTrace pattern: explicit type mapping between internal and output packages"
  - "Conditional debug population: zero allocations when debug inactive"

# Metrics
duration: 3min
completed: 2026-02-06
---

# Phase 27 Plan 02: C7 Debug Sample Types and Population Summary

**C7DebugSample/C7ScoreTrace/C7IndicatorMatch output types with conditional population in buildMetrics() gated by a.debug flag**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-06T13:11:35Z
- **Completed:** 2026-02-06T13:14:47Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added C7DebugSample, C7ScoreTrace, C7IndicatorMatch types to pkg/types/types.go
- Extended C7MetricResult with DebugSamples field using json omitempty for zero-impact non-debug JSON
- Implemented convertScoreTrace helper for clean internal-to-output type mapping
- Conditional population in buildMetrics() -- nil DebugSamples when debug off, full data when debug on
- Three new tests: debug-off nil check, debug-on full data verification, JSON omitempty behavior

## Task Commits

Each task was committed atomically:

1. **Task 1: Add C7 debug types to pkg/types and extend C7MetricResult** - `3b4cc5f` (feat)
2. **Task 2: Populate DebugSamples in buildMetrics() and add tests** - `c095c74` (feat)

## Files Created/Modified
- `pkg/types/types.go` - Added C7DebugSample, C7ScoreTrace, C7IndicatorMatch types; added DebugSamples field to C7MetricResult
- `internal/analyzer/c7_agent/agent.go` - Added convertScoreTrace helper; conditional debug sample population in sample loop
- `internal/analyzer/c7_agent/agent_test.go` - Three new tests for debug sample population and JSON omitempty

## Decisions Made
- Separate output types from internal types: C7ScoreTrace in pkg/types mirrors metrics.ScoreTrace but maintains package boundary separation. The convertScoreTrace helper performs explicit field mapping.
- Only added json tag to new DebugSamples field: existing C7MetricResult fields have no json tags (they are serialized through a separate output structure), so adding tags only to the new field maintains consistency.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- C7 debug data pipeline complete: internal metric data (Plan 01) now bridges to output types (Plan 02)
- Phase 29 can render debug information by reading C7MetricResult.DebugSamples
- Phase 28 scoring fix can leverage ScoreTrace data for diagnosing M2/M3/M4 scoring issues

---
*Phase: 27-data-capture*
*Completed: 2026-02-06*
