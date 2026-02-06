---
phase: 32-call-trace-modals
plan: 02
subsystem: ui
tags: [html, modal, trace, breakpoints, evidence, syntax-highlighting]

requires:
  - phase: 32-call-trace-modals
    plan: 01
    provides: C7 trace modal rendering, TraceData threading, modal infrastructure
provides:
  - C1-C6 breakpoint scoring tables with current band highlighting in trace modals
  - Top-5 worst offender evidence tables in trace modals
  - JSON syntax highlighting in all trace modal code blocks
affects: [32-03]

tech-stack:
  added: []
  patterns:
    - "renderBreakpointTrace pattern for C1-C6 metric trace HTML generation"
    - "findCurrentBand helper handles both ascending and descending breakpoint directions"
    - "highlightTraceCode() regex-based JSON highlighting in modal code blocks"

key-files:
  created: []
  modified:
    - internal/output/trace.go
    - internal/output/html.go
    - internal/output/templates/styles.css
    - internal/output/templates/report.html

key-decisions:
  - "findCurrentBand auto-detects ascending vs descending breakpoint direction"
  - "Breakpoint range display uses <=, >=, and range notation for clarity"
  - "JSON syntax highlighting uses 3 colors (keys blue, strings dark blue, numbers orange)"
  - "Highlighting is intentionally simple regex -- works well for JSON, no-ops on plain text"

patterns-established:
  - "renderBreakpointTrace: breakpoint table + evidence table in single HTML string"
  - "highlightTraceCode: post-process code blocks after modal innerHTML injection"

duration: 5min
completed: 2026-02-06
---

# Phase 32 Plan 02: C1-C6 Trace Modals with Breakpoint Tables Summary

**C1-C6 breakpoint scoring tables with current band highlighting, evidence offender tables, and JSON syntax highlighting in trace modals**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-06T22:37:31Z
- **Completed:** 2026-02-06T22:42:58Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- All C1-C6 metrics now show breakpoint scoring tables with the current band highlighted in amber
- Metrics with evidence display top-5 worst offender tables (file, line, value, description)
- JSON syntax highlighting (3 subtle colors) applied to all code blocks in trace modals
- View Trace buttons appear for all metrics with breakpoints or evidence data
- Both ascending (coverage-style) and descending (complexity-style) breakpoint tables render correctly

## Task Commits

1. **Task 1: Render C1-C6 breakpoint tables and evidence in trace modals** - `8888c09` (feat)
2. **Task 2: Add syntax highlighting CSS and trace table styling** - `990c871` (feat)

## Files Created/Modified
- `internal/output/trace.go` - Added renderBreakpointTrace() and findCurrentBand() functions
- `internal/output/html.go` - Wired breakpoint lookup and evidence into buildHTMLSubScores for non-C7 categories
- `internal/output/templates/styles.css` - Breakpoint table, evidence table, and syntax highlighting CSS
- `internal/output/templates/report.html` - Added highlightTraceCode() JS and call from openModal()

## Decisions Made
- findCurrentBand auto-detects ascending vs descending breakpoint direction by comparing first and last scores
- Breakpoint ranges display as "<=X" for first, ">=X" for last, "X-Y" for middle rows
- JSON highlighting uses simple regexes: works well for JSON, harmlessly no-ops on plain text
- Evidence table file paths show full path with title attribute for hover, ellipsis for overflow

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None.

## Next Phase Readiness
- All C1-C7 metrics now have functional View Trace modals
- Ready for plan 03 (any remaining trace polish or verification)
- Syntax highlighting infrastructure in place for future modal content

---
*Phase: 32-call-trace-modals*
*Completed: 2026-02-06*
