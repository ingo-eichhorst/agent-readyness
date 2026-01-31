---
phase: 04-recommendations-and-output
plan: 03
subsystem: pipeline
tags: [cli-flags, json-output, threshold, recommendations, pipeline-wiring]

# Dependency graph
requires:
  - phase: 04-01
    provides: "Recommendation type and Generate() function"
  - phase: 04-02
    provides: "RenderRecommendations, BuildJSONReport, RenderJSON functions"
  - phase: 03-scoring-model
    provides: "ScoredResult, Scorer, ScoringConfig"
provides:
  - "Full CLI pipeline with recommendations, JSON output, and threshold enforcement"
  - "ExitError type in pkg/types for custom exit codes"
  - "--threshold and --json flags on scan command"
affects: [05-hardening]

# Tech tracking
tech-stack:
  added: []
  patterns: ["ExitError in shared types to avoid import cycles", "threshold check after rendering for complete output"]

key-files:
  created: []
  modified:
    - "cmd/root.go"
    - "cmd/scan.go"
    - "internal/pipeline/pipeline.go"
    - "pkg/types/scoring.go"
    - "internal/recommend/recommend.go"
    - "internal/pipeline/pipeline_test.go"

key-decisions:
  - "ExitError placed in pkg/types (not cmd) to avoid cmd<->pipeline import cycle"
  - "Threshold check runs AFTER rendering so output is always displayed before exit"
  - "SilenceUsage on scan command prevents usage dump on ExitError"
  - "SilenceErrors on root command prevents Cobra double-printing errors"

patterns-established:
  - "Custom exit codes via types.ExitError with errors.As in Execute()"
  - "Dual rendering path: jsonOutput bool selects JSON vs terminal rendering in pipeline"

# Metrics
duration: 8min
completed: 2026-01-31
---

# Phase 4 Plan 3: CLI Pipeline Wiring Summary

**Wired recommendations, JSON output, --threshold exit-code-2, and --json flags into scan pipeline with dual rendering paths**

## Performance

- **Duration:** 8 min
- **Started:** 2026-01-31T22:36:16Z
- **Completed:** 2026-01-31T22:43:55Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- `ars scan .` now shows scores followed by top-5 recommendations in terminal
- `ars scan . --json` produces valid ANSI-free JSON with scores and recommendations
- `ars scan . --json --verbose` includes per-metric breakdowns in JSON categories
- `ars scan . --threshold 7` displays full output then exits with code 2 if score below threshold
- ExitError type in pkg/types avoids import cycle between cmd and pipeline packages

## Task Commits

Each task was committed atomically:

1. **Task 1: ExitError type and Execute() update** - `a3c2c30` (feat)
2. **Task 2: Pipeline integration with recommendations, JSON, and threshold** - `a13f614` (feat)

## Files Created/Modified
- `pkg/types/scoring.go` - Added ExitError type with Code and Message fields
- `cmd/root.go` - Updated Execute() with errors.As for ExitError, SilenceErrors
- `cmd/scan.go` - Added --threshold and --json flags, SilenceUsage, pass to pipeline
- `internal/pipeline/pipeline.go` - Recommendation generation, dual rendering, threshold check
- `internal/recommend/recommend.go` - Fixed action template formatting for 5 metrics
- `internal/pipeline/pipeline_test.go` - Updated New() calls with new signature

## Decisions Made
- ExitError in pkg/types rather than cmd to avoid import cycle (cmd imports pipeline, pipeline needs ExitError)
- Threshold check placed AFTER all rendering per research pitfall #5: users always see full output
- SilenceUsage set on scan command so threshold violations don't dump usage text
- JSON mode skips rendering when scored is nil (no data to report)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed action template formatting for 5 metrics**
- **Found during:** Task 2 (smoke testing terminal output)
- **Issue:** Action templates for afferent_coupling_avg, efferent_coupling_avg, module_fanout_avg, circular_deps, and import_complexity_avg had no format verbs but buildAction's default case passed two float args, causing `%!(EXTRA ...)` output
- **Fix:** Added format verbs to templates and explicit switch cases for these metrics
- **Files modified:** internal/recommend/recommend.go
- **Verification:** Smoke test output clean, no %!(EXTRA) artifacts
- **Committed in:** a13f614 (Task 2 commit)

**2. [Rule 3 - Blocking] Updated pipeline_test.go for new signature**
- **Found during:** Task 1 (build verification)
- **Issue:** pipeline_test.go called New() with 3 args, but signature now takes 5
- **Fix:** Added `0, false` args to all pipeline.New() calls in tests
- **Files modified:** internal/pipeline/pipeline_test.go
- **Verification:** go build ./... passes
- **Committed in:** a3c2c30 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both essential for correct operation. No scope creep.

## Issues Encountered
- `go run` wraps child exit codes (always returns 1 for non-zero), verified correct exit code 2 by building binary directly

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- CLI fully functional with all output modes and threshold enforcement
- Ready for Phase 5 hardening (edge cases, performance optimization)
- No blockers

---
*Phase: 04-recommendations-and-output*
*Completed: 2026-01-31*
