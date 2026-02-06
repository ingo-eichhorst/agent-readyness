---
phase: 30-evidence-data-flow
plan: 02
subsystem: scoring
tags: [evidence, scoring, worst-offenders, sort, top-5]

# Dependency graph
requires:
  - phase: 30-01
    provides: EvidenceItem type, MetricExtractor 3-return signature, SubScore.Evidence field
provides:
  - All 7 extractCx functions return populated evidence maps
  - Top-5 worst-offender extraction pattern for per-item metrics
  - Non-nil evidence arrays guaranteed for every metric key
affects: [30-03-json-output, 31-html-modals]

# Tech tracking
tech-stack:
  added: []
  patterns: ["sort-copy-limit-5 pattern for worst-offender extraction"]

key-files:
  created: []
  modified: ["internal/scoring/scorer.go"]

key-decisions:
  - "Combined Task 1+2 into single commit since all changes in one file"
  - "C5 unavailable and C7 unavailable paths also return non-nil evidence maps"

patterns-established:
  - "Top-5 extraction: copy slice, sort descending by metric, limit to 5, build EvidenceItem array"
  - "Coupling evidence uses local pkgCount struct for map-to-sorted-slice conversion"
  - "All evidence maps end with non-nil guarantee loop over metric key names"

# Metrics
duration: 6min
completed: 2026-02-06
---

# Phase 30 Plan 02: Evidence Population Summary

**Top-5 worst-offender evidence extraction for C1/C3/C5/C6 metrics with explicit empty arrays for C2/C4/C7 aggregate-only metrics**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-06T20:07:33Z
- **Completed:** 2026-02-06T20:13:33Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- extractC1 returns evidence for all 6 metrics: top-5 functions by complexity and line count, single largest file, top-5 coupling packages, top-5 duplicate blocks
- extractC3 returns evidence for circular_deps (cycle descriptions with arrow notation) and dead_exports (unused symbol details)
- extractC5 returns evidence from TopHotspots (commit count, author count, total changes) and CoupledPairs (coupling percentage)
- extractC6 returns evidence for test_isolation (external dependency flag) and assertion_density_avg (lowest assertion count)
- extractC2, extractC4, extractC7 return explicit empty evidence arrays for all metric keys
- Every evidence map value guaranteed non-nil, including unavailable-path returns for C5 and C7

## Task Commits

Each task was committed atomically:

1. **Task 1+2: Populate evidence in all 7 extractCx functions** - `90192c7` (feat)

## Files Created/Modified
- `internal/scoring/scorer.go` - Added evidence extraction logic to all 7 extractCx functions (+411 lines)

## Decisions Made
- Combined Task 1 and Task 2 into a single commit since both modify only scorer.go and the C2/C4/C7 empty-array changes were trivially interleaved with the C1/C3/C5/C6 work
- Used local `pkgCount` struct inside coupling evidence extraction rather than a package-level type (keeps scope minimal)
- C5 and C7 unavailable paths now also return non-nil evidence maps (was returning nil before)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] C5/C7 unavailable paths returned nil evidence**
- **Found during:** Task 1
- **Issue:** When C5 or C7 metrics are unavailable, the early return path returned `nil` for the evidence map, which violates the non-nil guarantee
- **Fix:** Added evidence map construction with empty arrays for all metric keys in both unavailable paths
- **Files modified:** internal/scoring/scorer.go
- **Verification:** go test ./internal/scoring/... passes
- **Committed in:** 90192c7

---

**Total deviations:** 1 auto-fixed (1 bug fix)
**Impact on plan:** Essential for non-nil guarantee. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All evidence maps are populated and ready for JSON serialization in Plan 30-03
- The sort-copy-limit-5 pattern is established and consistent across all extractors
- No blockers for downstream HTML modal rendering

---
*Phase: 30-evidence-data-flow*
*Completed: 2026-02-06*
