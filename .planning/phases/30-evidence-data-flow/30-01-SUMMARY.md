---
phase: 30-evidence-data-flow
plan: 01
subsystem: scoring
tags: [evidence, types, metric-extractor, subscore, json]

# Dependency graph
requires:
  - phase: 28-heuristic-tests-scoring-fixes
    provides: C7 scoring pipeline with 5 MECE metrics + deprecated overall_score
provides:
  - EvidenceItem type with file_path, line, value, description fields
  - SubScore.Evidence field with json tags (no omitempty)
  - MetricExtractor 3-return signature (rawValues, unavailable, evidence)
  - C7 config with exactly 5 MECE metrics (overall_score removed)
affects: [30-02, 30-03, 31-json-output, 32-html-evidence]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Evidence pipeline: extractors return stub evidence maps, scoreMetrics wires into SubScore"
    - "Non-nil guarantee: evidence arrays always initialized (never null in JSON)"

key-files:
  modified:
    - pkg/types/scoring.go
    - internal/scoring/scorer.go
    - internal/scoring/config.go
    - internal/scoring/scorer_test.go

key-decisions:
  - "SubScore.Evidence uses json:evidence without omitempty -- ensures [] not null in JSON output"
  - "C7 overall_score fully removed (not just zero-weight) -- clean config for v0.0.6"
  - "Stub evidence maps in extractors -- Plan 02 will populate real evidence"

patterns-established:
  - "Evidence return pattern: extractors return make(map[string][]types.EvidenceItem) on success, nil on early-return"
  - "Non-nil evidence guarantee: scoreMetrics initializes empty slice when evidence map has no entry"

# Metrics
duration: 5min
completed: 2026-02-06
---

# Phase 30 Plan 01: Evidence Types and Extractor Signatures Summary

**EvidenceItem type, SubScore.Evidence field, 3-return MetricExtractor, and C7 overall_score removal**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-06T19:59:45Z
- **Completed:** 2026-02-06T20:05:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Defined EvidenceItem struct with JSON tags for worst-offender data flow
- Extended SubScore with Evidence field guaranteeing non-null JSON arrays
- Updated MetricExtractor to 3-return signature across all 7 category extractors
- Removed deprecated C7 overall_score from config, extractor, and tests

## Task Commits

Each task was committed atomically:

1. **Task 1: Add EvidenceItem type and update SubScore** - `929630f` (feat)
2. **Task 2: Update MetricExtractor signature, remove overall_score** - `478d446` (feat)

## Files Created/Modified
- `pkg/types/scoring.go` - Added EvidenceItem struct, Evidence field on SubScore, json tags on all SubScore fields
- `internal/scoring/scorer.go` - 3-return MetricExtractor, all 7 extractors updated, scoreMetrics wires evidence
- `internal/scoring/config.go` - Removed C7 overall_score metric entry (5 MECE metrics remain)
- `internal/scoring/scorer_test.go` - Updated C7 tests for 5 metrics and 3-return signatures

## Decisions Made
- SubScore.Evidence uses `json:"evidence"` without omitempty to guarantee `[]` (not `null`) in JSON
- C7 overall_score fully removed rather than kept at zero-weight -- cleaner config going forward
- Extractors return stub empty evidence maps now; Plan 02 will populate real evidence data

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Type foundation complete: EvidenceItem and SubScore.Evidence ready for all extractors
- Plan 02 can immediately start populating real evidence in C1-C6 extractors
- Plan 03 can wire evidence through JSON and HTML output

---
*Phase: 30-evidence-data-flow*
*Completed: 2026-02-06*
