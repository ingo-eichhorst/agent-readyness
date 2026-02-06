---
phase: 33-improvement-prompt-modals
plan: 01
subsystem: ui
tags: [html, prompt-generation, scoring, breakpoints, evidence]

# Dependency graph
requires:
  - phase: 32-call-trace-modals
    provides: renderBreakpointTrace pattern, findCurrentBand ascending detection, scoring.Breakpoint type
provides:
  - renderImprovementPrompt() for generating copyable 4-section improvement prompts
  - nextTarget() for computing next achievable breakpoint from current score
  - PromptParams struct for prompt data assembly
  - getMetricTaskGuidance() extracting How to Improve bullets from descriptions.go
affects: [33-02 HTML integration, 33-03 modal wiring]

# Tech tracking
tech-stack:
  added: []
  patterns: [4-section prompt structure, HTML-escaped plain-text in pre/code blocks, extractHowToImprove regex parsing]

key-files:
  created:
    - internal/output/prompt.go
    - internal/output/prompt_test.go
  modified: []

key-decisions:
  - "Plain text prompt inside <pre><code> with HTML escaping for safe embedding"
  - "nextTarget reuses findCurrentBand ascending detection pattern from trace.go"
  - "extractHowToImprove uses regex to parse <li> items from Detailed HTML descriptions"
  - "C7 metrics use score+2 target instead of breakpoint-based targets"

patterns-established:
  - "4-section prompt: Context / Build & Test / Task / Verification"
  - "prompt-copy-container with copyPromptText(this) onclick handler"

# Metrics
duration: 4min
completed: 2026-02-07
---

# Phase 33 Plan 01: Prompt Rendering Engine Summary

**renderImprovementPrompt() with 4-section copyable prompts, nextTarget() breakpoint calculator, and How to Improve extraction from metric descriptions**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-06T23:22:39Z
- **Completed:** 2026-02-06T23:26:10Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Prompt rendering engine producing Context/Build&Test/Task/Verification sections for all C1-C7 metrics
- nextTarget() correctly handles both ascending and descending breakpoint directions
- How to Improve guidance extracted from descriptions.go HTML into plain text bullets
- C7 metrics get description-based guidance without breakpoint targets

## Task Commits

Each task was committed atomically:

1. **Task 1: Create prompt rendering engine** - `a658a2d` (feat)
2. **Task 2: Unit tests for prompt rendering** - `04eab4e` (test)

## Files Created/Modified
- `internal/output/prompt.go` - PromptParams struct, renderImprovementPrompt(), nextTarget(), languageBuildCommands(), getMetricTaskGuidance(), extractHowToImprove()
- `internal/output/prompt_test.go` - 7 test functions covering C1/C7 prompts, empty evidence, ascending/descending/max breakpoints, task guidance

## Decisions Made
- Plain text prompt content inside HTML-escaped `<pre><code>` block for safe copy-paste
- Reused ascending detection pattern from findCurrentBand() in trace.go for nextTarget()
- Used regex to parse `<li>` items from Detailed HTML descriptions rather than a full HTML parser
- C7 metrics without breakpoints use score+2 (capped at 10) as improvement target

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- renderImprovementPrompt() ready for HTML template integration in plan 33-02
- copyPromptText() JavaScript function needs to be added in styles/template (plan 33-02/03)
- prompt-copy-container CSS class needs styling (plan 33-02/03)

---
*Phase: 33-improvement-prompt-modals*
*Completed: 2026-02-07*
