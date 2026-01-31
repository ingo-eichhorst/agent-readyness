---
phase: 04-recommendations-and-output
plan: 01
subsystem: scoring
tags: [recommendation-engine, score-simulation, tdd, composite-scoring]

# Dependency graph
requires:
  - phase: 03-scoring-model
    provides: "ScoredResult type, Interpolate function, ScoringConfig with breakpoints, categoryScore/computeComposite patterns"
provides:
  - "Recommendation type with ranked improvement suggestions"
  - "Generate() function producing top-5 recommendations by composite impact"
  - "Agent-readiness framed summaries and concrete action templates"
affects: [04-02 terminal output rendering, 04-03 pipeline integration]

# Tech tracking
tech-stack:
  added: []
  patterns: ["score simulation via deep-copy-patch-recompute", "effort estimation with metric difficulty multiplier"]

key-files:
  created:
    - "internal/recommend/recommend.go"
    - "internal/recommend/recommend_test.go"
  modified: []

key-decisions:
  - "findTargetBreakpoint selects the minimal next-better breakpoint score (smallest improvement step, not max)"
  - "Hard metrics (complexity_avg, duplication_rate) get +1 effort level bump"
  - "Effort thresholds: gap < 1.0 = Low, < 2.5 = Medium, >= 2.5 = High"
  - "simulateComposite deep-copies categories to avoid mutation of input ScoredResult"

patterns-established:
  - "Score simulation: deep-copy categories, patch one sub-score, recompute category then composite"
  - "Agent-readiness framing: agentImpact map + displayNames map + actionTemplates map"

# Metrics
duration: 4min
completed: 2026-01-31
---

# Phase 4 Plan 1: Recommendation Engine Summary

**Top-5 recommendation engine using full composite score simulation with agent-readiness framing and effort estimation**

## Performance

- **Duration:** 4 min
- **Started:** 2026-01-31T22:28:24Z
- **Completed:** 2026-01-31T22:32:10Z
- **Tasks:** 2 (TDD: RED + GREEN)
- **Files modified:** 2

## Accomplishments
- Generate() produces ranked recommendations sorted by composite score impact
- Impact calculated via actual score simulation (deep-copy, patch metric, recompute category + composite)
- Effort estimation combines score gap size with metric-specific difficulty multiplier
- Agent-readiness framed summaries with concrete improvement actions for all 16 metrics
- Edge cases handled: empty input, all-excellent scores, unavailable metrics, fewer than 5 candidates, nil config

## Task Commits

Each task was committed atomically:

1. **RED: Failing tests for recommendation engine** - `4c343fd` (test)
2. **GREEN: Implement recommendation engine** - `40353bb` (feat)

_TDD plan: RED phase wrote 11 comprehensive tests, GREEN phase implemented to pass all._

## Files Created/Modified
- `internal/recommend/recommend.go` - Recommendation type, Generate function, simulateComposite, findTargetBreakpoint, effortLevel, agent-readiness maps (366 lines)
- `internal/recommend/recommend_test.go` - 11 tests covering ranking, impact accuracy, effort estimation, difficulty bumps, edge cases (429 lines)

## Decisions Made
- findTargetBreakpoint picks the smallest improvement step (next-better breakpoint), not the maximum possible improvement, to keep recommendations achievable
- Hard metric difficulty bump applies to complexity_avg and duplication_rate only (inherently harder to refactor)
- Effort thresholds at gap < 1.0 (Low), < 2.5 (Medium), >= 2.5 (High) based on research recommendation
- Deep-copy approach for simulation ensures input ScoredResult is never mutated

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Recommendation engine ready for terminal rendering in 04-02
- Generate() takes ScoredResult + ScoringConfig, returns []Recommendation for output package to display
- No blockers for 04-02 or 04-03

---
*Phase: 04-recommendations-and-output*
*Completed: 2026-01-31*
