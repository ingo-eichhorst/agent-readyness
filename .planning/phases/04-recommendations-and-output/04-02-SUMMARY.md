---
phase: 04-recommendations-and-output
plan: 02
subsystem: output
tags: [terminal-rendering, json-output, recommendations, verbose-mode]

# Dependency graph
requires:
  - phase: 04-01
    provides: "Recommendation type with ranked improvement suggestions"
  - phase: 03-scoring-model
    provides: "ScoredResult type with categories and sub-scores"
provides:
  - "RenderRecommendations function for terminal display of ranked recommendations"
  - "JSONReport types and BuildJSONReport/RenderJSON functions for machine-readable output"
affects: [04-03 pipeline integration]

# Tech tracking
tech-stack:
  added: []
  patterns: ["dual rendering paths (terminal + JSON) from same data", "omitempty for verbose-controlled JSON detail"]

key-files:
  created:
    - "internal/output/json.go"
    - "internal/output/json_test.go"
  modified:
    - "internal/output/terminal.go"
    - "internal/output/terminal_test.go"

key-decisions:
  - "Impact color thresholds: green >= 0.5, yellow >= 0.2, red < 0.2 composite points"
  - "JSON version field set to '1' for future schema evolution"
  - "JSONMetric uses omitempty so non-verbose output has no metrics key at all"
  - "RenderJSON uses json.NewEncoder (not json.Marshal) for streaming to io.Writer"

patterns-established:
  - "Dual rendering: terminal uses fatih/color, JSON uses encoding/json -- completely separate paths"
  - "BuildJSONReport as intermediate step between internal types and JSON serialization"

# Metrics
duration: 2min
completed: 2026-01-31
---

# Phase 4 Plan 2: Terminal and JSON Output Summary

**Terminal recommendation rendering with impact/effort/action display and ANSI-free JSON report output with verbose metric detail control**

## Performance

- **Duration:** 2 min
- **Started:** 2026-01-31T22:35:03Z
- **Completed:** 2026-01-31T22:37:05Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- RenderRecommendations displays numbered recommendations with bold summary, colored impact, effort level, and concrete action
- Empty recommendations render graceful "all metrics are excellent" message in green
- JSONReport types with proper struct tags for clean serialization
- BuildJSONReport converts ScoredResult + recommendations into JSON-ready structs
- Verbose mode populates per-metric breakdowns in JSON categories
- Non-verbose omits metrics entirely from JSON output via omitempty
- JSON output verified ANSI-free (no escape sequences)
- 14 total tests in output package (6 existing + 2 terminal recommendation + 8 JSON)

## Task Commits

Each task was committed atomically:

1. **Terminal recommendation rendering** - `39a2222` (feat)
2. **JSON output types and renderer** - `9886212` (feat)

## Files Created/Modified
- `internal/output/terminal.go` - Added RenderRecommendations function with impact coloring (modified)
- `internal/output/terminal_test.go` - Added 2 tests for recommendation rendering (modified)
- `internal/output/json.go` - JSONReport/JSONCategory/JSONMetric/JSONRecommendation types, BuildJSONReport, RenderJSON (created, 107 lines)
- `internal/output/json_test.go` - 8 tests covering valid JSON, no ANSI, version, verbose/non-verbose, recommendations (created, 210 lines)

## Decisions Made
- Impact coloring uses 0.5/0.2 point thresholds (green/yellow/red) matching score significance
- JSON version "1" enables future breaking changes with version detection
- omitempty on Metrics field means non-verbose JSON has no metrics key (cleaner output)
- json.NewEncoder chosen over json.Marshal for streaming to io.Writer

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Both terminal and JSON rendering ready for pipeline integration in 04-03
- RenderRecommendations takes []recommend.Recommendation
- BuildJSONReport takes *types.ScoredResult + []recommend.Recommendation + verbose bool
- RenderJSON takes *JSONReport and writes to io.Writer
- No blockers for 04-03

---
*Phase: 04-recommendations-and-output*
*Completed: 2026-01-31*
