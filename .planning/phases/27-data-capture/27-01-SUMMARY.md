---
phase: 27-data-capture
plan: 01
subsystem: agent-metrics
tags: [score-trace, prompt-capture, heuristic-scoring, c7-debug]

# Dependency graph
requires:
  - phase: 24-c7-mece-metrics-implementation
    provides: M1-M5 metric scoring functions and SampleResult type
  - phase: 26-debug-foundation
    provides: io.Writer debug pattern and SetDebug threading
provides:
  - ScoreTrace and IndicatorMatch types in metric.go
  - Prompt field on SampleResult for all 5 metrics
  - ScoreTrace as source of truth for M2-M5 scoring functions
  - Per-run ScoreTrace for M1 inline scoring
  - mockExecutor test helper for metric Execute() testing
affects: [28-scoring-fix, 29-rendering]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ScoreTrace source-of-truth pattern: score computed FROM trace indicators, not duplicated"
    - "IndicatorMatch records both matched and unmatched indicators with zero delta for unmatched"

key-files:
  created: []
  modified:
    - internal/agent/metrics/metric.go
    - internal/agent/metrics/m1_consistency.go
    - internal/agent/metrics/m2_comprehension.go
    - internal/agent/metrics/m3_navigation.go
    - internal/agent/metrics/m4_identifiers.go
    - internal/agent/metrics/m5_documentation.go
    - internal/agent/metrics/metric_test.go

key-decisions:
  - "ScoreTrace is source of truth (not parallel record) - score computed from BaseScore + sum(Deltas)"
  - "M1 uses BaseScore 0 (absolute scoring) while M2-M5 use BaseScore 5 (adjustment-based scoring)"
  - "All indicators tracked (matched and unmatched) with Delta=0 for unmatched - enables full trace visibility"

patterns-established:
  - "ScoreTrace source-of-truth: FinalScore = clamp(BaseScore + sum(ind.Delta), 1, 10)"
  - "Indicator naming convention: prefix:keyword (positive:returns, negative:unclear, self_report:accurate)"
  - "mockExecutor pattern for testing metric Execute() without real Claude CLI"

# Metrics
duration: 8min
completed: 2026-02-06
---

# Phase 27 Plan 01: Data Capture Summary

**ScoreTrace and Prompt capture for all 5 MECE metrics enabling full scoring transparency and debug inspection**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-06T12:58:03Z
- **Completed:** 2026-02-06T13:05:34Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Added IndicatorMatch and ScoreTrace types to metric.go providing structured scoring breakdown
- Extended SampleResult with Prompt and ScoreTrace fields for data capture
- Refactored all 5 metrics (M1-M5) to capture prompts and produce ScoreTraces
- ScoreTrace is source of truth: scores computed FROM indicator deltas, not duplicated
- Added 2 new test functions (TestScoreTrace_SumsCorrectly, TestAllMetrics_CapturePrompt) verifying trace arithmetic and prompt capture
- All 29 tests pass, including 27 original (unchanged behavior) + 2 new

## Task Commits

Each task was committed atomically:

1. **Task 1: Add ScoreTrace types and Prompt field to SampleResult** - `68a708e` (feat)
2. **Task 2: Update M1-M5 to capture prompts and return ScoreTraces** - `9ed7986` (feat)

## Files Created/Modified
- `internal/agent/metrics/metric.go` - Added IndicatorMatch, ScoreTrace types; extended SampleResult with Prompt and ScoreTrace fields
- `internal/agent/metrics/m1_consistency.go` - Prompt capture and inline ScoreTrace for per-run JSON format scoring
- `internal/agent/metrics/m2_comprehension.go` - Prompt capture; scoreComprehensionResponse returns (int, ScoreTrace)
- `internal/agent/metrics/m3_navigation.go` - Prompt capture; scoreNavigationResponse returns (int, ScoreTrace)
- `internal/agent/metrics/m4_identifiers.go` - Prompt capture; scoreIdentifierResponse returns (int, ScoreTrace)
- `internal/agent/metrics/m5_documentation.go` - Prompt capture; scoreDocumentationResponse returns (int, ScoreTrace)
- `internal/agent/metrics/metric_test.go` - Updated 4 existing test call sites; added mockExecutor, TestScoreTrace_SumsCorrectly, TestAllMetrics_CapturePrompt

## Decisions Made
- ScoreTrace as source of truth (not a parallel record): the final score is computed FROM the trace indicators, preventing trace/score divergence
- M1 uses BaseScore 0 (absolute scoring: the first matching indicator sets the score) while M2-M5 use BaseScore 5 (adjustment-based: indicators add/subtract from baseline)
- All indicators are tracked including unmatched ones (Delta=0) to enable full trace visibility for debugging
- Indicator naming follows prefix:keyword convention (positive:returns, negative:unclear, structure:verification, self_report:accurate)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- ScoreTrace types and Prompt capture ready for Phase 28 (scoring fix/testing) to inspect heuristic breakdowns
- Phase 29 (rendering) can display ScoreTrace indicators in debug output
- mockExecutor pattern available for future metric testing

---
*Phase: 27-data-capture*
*Completed: 2026-02-06*
