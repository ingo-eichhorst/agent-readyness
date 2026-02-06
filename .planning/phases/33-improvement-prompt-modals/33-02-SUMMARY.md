---
phase: 33-improvement-prompt-modals
plan: 02
subsystem: ui
tags: [html, prompt-modals, clipboard, progressive-enhancement, template-wiring]

# Dependency graph
requires:
  - phase: 33-improvement-prompt-modals
    provides: renderImprovementPrompt(), nextTarget(), PromptParams struct (plan 01)
  - phase: 32-call-trace-modals
    provides: openModal() shared API, <template> pattern, TraceData struct
provides:
  - HTMLSubScore.PromptHTML/HasPrompt fields for prompt rendering
  - Prompt population in buildHTMLSubScores for metrics below 9.0
  - Languages threading from pipeline to HTML generator
  - Improve button next to View Trace in metric rows
  - copyPromptText() 3-tier clipboard fallback (Clipboard API, execCommand, select-all)
  - Progressive enhancement details fallback for no-JS
affects: [33-03 progressive enhancement]

# Tech tracking
tech-stack:
  added: []
  patterns: [3-tier clipboard fallback chain, prompt-btn indigo styling distinct from trace buttons]

key-files:
  created: []
  modified:
    - internal/output/html.go
    - internal/pipeline/pipeline.go
    - internal/output/templates/report.html
    - internal/output/templates/styles.css
    - internal/output/html_test.go

key-decisions:
  - "Breakpoints variable shared between trace and prompt blocks to avoid duplicate lookup"
  - "C7 breakpoints looked up separately in prompt block since C7 trace uses different path"
  - "Languages field on TraceData as []string (converted from []types.Language in pipeline)"
  - "Prompt styling uses indigo (#6366f1) to visually distinguish from trace buttons"

patterns-established:
  - "prompt-copy-container with copyPromptText(this) onclick for clipboard operations"
  - "3-tier copy fallback: navigator.clipboard -> execCommand -> select-all"

# Metrics
duration: 7min
completed: 2026-02-07
---

# Phase 33 Plan 02: HTML Template Wiring Summary

**Improve buttons with 3-tier clipboard copy wired into HTML report for all metrics below 9.0, with progressive enhancement fallback**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-06T23:29:16Z
- **Completed:** 2026-02-06T23:36:22Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- HTMLSubScore extended with PromptHTML/HasPrompt for prompt modal rendering
- buildHTMLSubScores populates prompts for all available metrics scoring below 9.0
- Pipeline threads detected languages to TraceData for build/test command generation
- Improve buttons appear next to View Trace buttons with distinct indigo styling
- 3-tier clipboard fallback works on both HTTPS and file:// protocols
- Progressive enhancement: prompt content visible in details fallback without JS
- Report size stays at 301 KB (under 500KB budget)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add prompt fields to HTMLSubScore and wire rendering** - `495696e` (feat)
2. **Task 2: Template/CSS/JS changes** - `e79fb47` (test, committed in prior session as blocker fix for 33-03)

## Files Created/Modified
- `internal/output/html.go` - PromptHTML/HasPrompt fields, Languages on TraceData, prompt population in buildHTMLSubScores
- `internal/pipeline/pipeline.go` - langs field on Pipeline, language threading to TraceData
- `internal/output/templates/report.html` - Improve button, prompt template storage, details fallback, copyPromptText JS
- `internal/output/templates/styles.css` - prompt-copy-container, prompt-btn indigo styling, select-fallback
- `internal/output/html_test.go` - Tests for prompt modals, high-score suppression, all-categories coverage

## Decisions Made
- Shared breakpoints variable between trace and prompt blocks to avoid duplicate scoring config lookup
- C7 metrics get separate breakpoint lookup since C7 trace uses different rendering path
- Languages stored as []string on TraceData (converted from []types.Language in pipeline)
- Prompt button uses indigo (#6366f1) to visually distinguish from default trace button color

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed test for high-score prompt suppression**
- **Found during:** Task 2 verification
- **Issue:** Test checked for "prompt-copy-container" string in full HTML, but CSS class definition always present in inline styles
- **Fix:** Test already corrected by linter to check for `<template id="prompt-` instead
- **Files modified:** internal/output/html_test.go
- **Verification:** All tests pass
- **Committed in:** e79fb47

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor test assertion fix. No scope creep.

## Issues Encountered
- Template/CSS/JS changes were already committed in a prior session (33-03 blocker fix). Task 2 work was already present, only Task 1 Go-side wiring was needed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All prompt modal functionality complete and wired
- 33-03 progressive enhancement already committed
- Report generates correctly with Improve buttons for all metric categories

---
*Phase: 33-improvement-prompt-modals*
*Completed: 2026-02-07*
