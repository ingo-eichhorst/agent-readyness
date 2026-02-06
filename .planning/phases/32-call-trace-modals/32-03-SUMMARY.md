---
phase: 32-call-trace-modals
plan: 03
subsystem: ui
tags: [html, progressive-enhancement, noscript, details-fallback]

# Dependency graph
requires:
  - phase: 32-02
    provides: "Modal dialog infrastructure, trace rendering for C1-C7"
provides:
  - "<details> fallback for trace content without JavaScript"
  - "File size reporting for HTML report generation"
  - "Complete Phase 32 call trace modal system"
affects: [33-improvement-modals]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "js-enabled class toggling for progressive enhancement"
    - "Native <details> as no-JS fallback for modal content"

key-files:
  created: []
  modified:
    - "internal/output/templates/report.html"
    - "internal/output/templates/styles.css"
    - "internal/pipeline/pipeline.go"

key-decisions:
  - "TraceHTML reused in both <template> (modal) and <details> (fallback) -- single source of truth"
  - "Copy buttons left in fallback content -- non-functional without JS but cause no harm"
  - "File size reported as informational only, no warning threshold"

patterns-established:
  - "js-enabled class on body: added by inline script at top of body, used to hide no-JS fallback elements"

# Metrics
duration: 3min
completed: 2026-02-06
---

# Phase 32 Plan 03: Progressive Enhancement and File Size Reporting Summary

**Native `<details>` fallback for trace content without JavaScript, file size reporting on HTML generation**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-06T22:45:28Z
- **Completed:** 2026-02-06T22:48:50Z
- **Tasks:** 1
- **Files modified:** 3

## Accomplishments
- Added `<details class="trace-fallback">` elements inside each metric details row for no-JS access to trace data
- Added `js-enabled` class toggling via inline script at start of `<body>` to hide fallbacks when JS is available
- Added file size reporting (KB) to terminal output after HTML report generation
- Complete Phase 32 -- all trace modals work with and without JavaScript

## Task Commits

Each task was committed atomically:

1. **Task 1: Add progressive enhancement fallback and file size reporting** - `fdf72ba` (feat)

## Files Created/Modified
- `internal/output/templates/report.html` - Added js-enabled script, `<details>` fallback for trace content
- `internal/output/templates/styles.css` - CSS for trace-fallback visibility and js-enabled hiding
- `internal/pipeline/pipeline.go` - File size reporting after HTML generation via f.Sync()+f.Stat()

## Decisions Made
- Reused same TraceHTML content in both modal `<template>` and `<details>` fallback (single source of truth)
- Copy buttons remain in fallback (non-functional without JS but harmless)
- File size is informational only -- no warning threshold per CONTEXT.md decision

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 32 complete: all call trace modals functional with progressive enhancement
- Phase 33 (improvement modals) can build on the modal infrastructure from Phase 31-32
- Modal API (openModal/closeModal), trace rendering, and progressive enhancement patterns are established

---
*Phase: 32-call-trace-modals*
*Completed: 2026-02-06*
