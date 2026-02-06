---
phase: 32-call-trace-modals
plan: 01
subsystem: ui
tags: [html, modal, c7, trace, debug]

requires:
  - phase: 31-modal-ui-infrastructure
    provides: Native dialog modal with openModal/closeModal API
provides:
  - C7 trace modal rendering with score checklist and prompt/response display
  - TraceData struct threading analysis results to HTML generator
  - Unconditional C7 DebugSample population for trace data availability
affects: [32-02, 32-03]

tech-stack:
  added: []
  patterns:
    - "renderC7Trace pattern for metric-specific trace HTML generation"
    - "TraceData struct bundles ScoringConfig + AnalysisResults for HTML rendering"
    - "template.HTML pre-rendering for modal content stored in <template> elements"

key-files:
  created:
    - internal/output/trace.go
  modified:
    - internal/analyzer/c7_agent/agent.go
    - internal/analyzer/c7_agent/agent_test.go
    - internal/output/html.go
    - internal/output/html_test.go
    - internal/output/templates/report.html
    - internal/output/templates/styles.css
    - internal/pipeline/pipeline.go

key-decisions:
  - "DebugSamples populated unconditionally (debug flag only controls terminal output)"
  - "TraceData passed as pointer (nil-safe for backward compat)"
  - "Trace content stored in <template> elements, injected into modal via innerHTML"

patterns-established:
  - "renderC7Trace: metric-specific trace renderer returns HTML string"
  - "TraceData threading: pipeline builds TraceData, passes through GenerateReport to buildHTMLSubScores"

duration: 9min
completed: 2026-02-06
---

# Phase 32 Plan 01: C7 Call Trace Modals Summary

**C7 trace modals with indicator checklist, collapsible prompt/response, and copy buttons via TraceData threading**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-06T22:22:38Z
- **Completed:** 2026-02-06T22:31:22Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- C7 DebugSamples now populated unconditionally when C7 runs (not gated on debug flag)
- GenerateReport receives TraceData with scoring config and analysis results
- C7 trace modals render score checklist with matched/unmatched indicators and delta values
- Collapsible prompt/response sections with inline copy-to-clipboard buttons
- View Trace buttons appear only for C7 metrics with trace data (graceful degradation)

## Task Commits

1. **Task 1: Populate C7 DebugSamples unconditionally and thread data to HTML generator** - `12420d7` (feat)
2. **Task 2: Render C7 trace modal content and wire View Trace buttons** - `0794d3f` (feat)

## Files Created/Modified
- `internal/output/trace.go` - C7 trace HTML rendering (checklist, code blocks, copy buttons)
- `internal/output/html.go` - TraceData struct, updated GenerateReport signature, trace field population
- `internal/output/html_test.go` - Updated GenerateReport calls with new signature
- `internal/output/templates/report.html` - View Trace button column, template stores for trace content
- `internal/output/templates/styles.css` - Trace checklist, code block, copy button styles
- `internal/analyzer/c7_agent/agent.go` - Removed debug guard on DebugSample population
- `internal/analyzer/c7_agent/agent_test.go` - Updated test expectations for unconditional DebugSamples
- `internal/pipeline/pipeline.go` - Constructs and passes TraceData to GenerateReport

## Decisions Made
- DebugSamples populated unconditionally -- debug flag only controls terminal fprintf output, not data capture. This ensures trace data is always available for HTML modals.
- TraceData passed as nil-safe pointer to maintain backward compatibility with existing callers.
- Trace content stored in hidden `<template>` elements and injected via innerHTML on modal open.
- Added C7 to categoryDisplayName mapping (was missing).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added C7 to categoryDisplayName mapping**
- **Found during:** Task 2
- **Issue:** categoryDisplayName map did not include C7, so C7 category would display as raw "C7" instead of "C7: Agent Evaluation"
- **Fix:** Added "C7": "C7: Agent Evaluation" to the map
- **Files modified:** internal/output/html.go
- **Verification:** Build passes, C7 displays correctly
- **Committed in:** 0794d3f (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Necessary for consistent category display. No scope creep.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- C7 trace modals complete and working end-to-end
- Ready for plan 02 (C1-C6 trace modals with breakpoint tables)
- TraceData struct already carries ScoringConfig for breakpoint rendering

---
*Phase: 32-call-trace-modals*
*Completed: 2026-02-06*
