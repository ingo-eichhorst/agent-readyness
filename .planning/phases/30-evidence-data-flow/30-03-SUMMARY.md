---
phase: 30-evidence-data-flow
plan: 03
subsystem: output
tags: [json, evidence, sub_scores, backward-compatibility]

# Dependency graph
requires:
  - phase: 30-02
    provides: "EvidenceItem type and evidence extraction in scoring pipeline"
provides:
  - "JSON output with sub_scores and evidence arrays"
  - "Schema version 2 with always-present sub_scores"
  - "Backward-compatible baseline loading for v0.0.5 JSON"
affects: [31-html-evidence, 32-trend-comparison]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "nil-to-empty-slice for guaranteed [] JSON serialization"
    - "no omitempty on sub_scores/evidence for always-present arrays"

key-files:
  created: []
  modified:
    - "internal/output/json.go"
    - "internal/output/json_test.go"

key-decisions:
  - "JSON version bumped from 1 to 2 to signal schema change"
  - "verbose parameter deprecated but kept in signature to avoid multi-file churn"
  - "sub_scores always present (no omitempty) -- evidence visible without verbose flag"

patterns-established:
  - "nil-to-empty evidence: ev := ss.Evidence; if ev == nil { ev = make([]types.EvidenceItem, 0) }"

# Metrics
duration: 4min
completed: 2026-02-06
---

# Phase 30 Plan 03: JSON Output Summary

**JSON output wired with sub_scores field containing evidence arrays, version bumped to 2, backward-compatible with v0.0.5 baselines**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-06T20:14:53Z
- **Completed:** 2026-02-06T20:18:23Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Renamed JSON field from "metrics" to "sub_scores" with no omitempty (always present)
- Added Evidence field to JSONMetric with nil-to-empty conversion (guarantees [] not null)
- Removed verbose gate: sub_scores always populated regardless of verbose flag
- Bumped JSON schema version from "1" to "2"
- Full test coverage: 13 JSON tests including evidence serialization and backward compatibility

## Task Commits

Each task was committed atomically:

1. **Task 1: Update JSON types and BuildJSONReport** - `9ec5663` (feat)
2. **Task 2: Update JSON tests** - `4f0b870` (test)

## Files Created/Modified
- `internal/output/json.go` - Updated JSONCategory/JSONMetric structs, removed verbose gate, added evidence wiring
- `internal/output/json_test.go` - Updated all tests for sub_scores, added evidence and backward-compat tests

## Decisions Made
- JSON version bumped from "1" to "2" to signal the schema change to consumers
- verbose parameter kept in BuildJSONReport signature (deprecated via comment) to avoid changing callers across multiple files
- sub_scores field has no omitempty -- always serialized even when empty

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 30 (Evidence Data Flow) is now complete: evidence flows from analyzers through scoring into JSON output
- Ready for Phase 31 (HTML evidence rendering) and Phase 32 (trend comparison with new schema)
- Old v0.0.5 baseline JSON files remain loadable for comparison

---
*Phase: 30-evidence-data-flow*
*Completed: 2026-02-06*
