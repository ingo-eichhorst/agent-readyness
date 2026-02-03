---
phase: 15-claude-code-integration
plan: 02
subsystem: cli
tags: [claude-cli, sdk-removal, auto-detection, llm-features]

# Dependency graph
requires:
  - phase: 15-01
    provides: CLI detection and evaluator infrastructure
provides:
  - Auto-detection of Claude CLI at pipeline startup
  - LLM features auto-enabled when CLI available
  - --no-llm flag for disabling LLM features
  - SDK-free C7 scoring via CLI evaluator
  - Anthropic SDK removed from dependencies
affects: [testing, documentation, release]

# Tech tracking
tech-stack:
  added: []
  removed: [anthropic-sdk-go, tidwall/gjson]
  patterns: [cli-based-llm-evaluation, auto-detection-pattern]

key-files:
  modified:
    - internal/pipeline/pipeline.go
    - cmd/scan.go
    - internal/agent/scorer.go
    - internal/analyzer/c7_agent.go
    - go.mod
  deleted:
    - internal/llm/client.go
    - internal/llm/client_test.go
    - internal/llm/cost.go
    - internal/llm/prompts.go

key-decisions:
  - "Auto-enable LLM when CLI detected (user can opt-out with --no-llm)"
  - "Remove cost estimates and confirmation prompts (CLI handles billing)"
  - "Single evaluator instance shared between C4 and C7 analyzers"

patterns-established:
  - "CLI auto-detection at pipeline init: agent.GetCLIStatus()"
  - "Evaluator field mapping: Reason not Reasoning"

# Metrics
duration: 9min
completed: 2026-02-03
---

# Phase 15 Plan 02: SDK Removal and CLI Migration Summary

**Complete Claude CLI migration: auto-detect CLI, remove SDK dependency, simplify user experience with no API key required**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-03T22:47:00Z
- **Completed:** 2026-02-03T22:56:13Z
- **Tasks:** 5 (combined into 4 commits)
- **Files modified:** 8 (4 modified, 4 deleted)

## Accomplishments

- Pipeline auto-detects CLI availability at startup via `agent.GetCLIStatus()`
- LLM features auto-enabled when CLI is available, no user action needed
- Removed `--enable-c4-llm` flag; added `--no-llm` flag for opt-out
- Eliminated ANTHROPIC_API_KEY requirement for all operations
- Deleted entire `internal/llm/` package (800+ lines)
- Removed Anthropic SDK and transitive dependencies from go.mod

## Task Commits

Each task was committed atomically:

1. **Task 1: Update pipeline for CLI auto-detection** - `8dc6d61` (refactor)
2. **Task 2: Update scan command flags** - `7c31f3d` (refactor)
3. **Task 3+4: Scan command C4/C7 cleanup** - `d838ce8` (refactor)
4. **Task 5: Migrate C7 scorer and remove SDK** - `3877aa8` (refactor)

## Files Created/Modified

**Modified:**
- `internal/pipeline/pipeline.go` - Auto-detection, evaluator/cliStatus fields, DisableLLM(), GetCLIStatus(), updated SetC7Enabled()
- `cmd/scan.go` - Removed LLM client handling, added CLI status display, simplified C7 enablement
- `internal/agent/scorer.go` - Uses Evaluator instead of llm.Client, maps to Reason field
- `internal/analyzer/c7_agent.go` - Enable() accepts *agent.Evaluator
- `go.mod` / `go.sum` - Removed anthropic-sdk-go and tidwall dependencies

**Deleted:**
- `internal/llm/client.go` - SDK wrapper
- `internal/llm/client_test.go` - SDK tests
- `internal/llm/cost.go` - Cost estimation
- `internal/llm/prompts.go` - SDK prompts

## Decisions Made

1. **Auto-enable LLM by default** - When CLI is available, LLM features are enabled without user confirmation. Users can opt-out with `--no-llm`. Rationale: CLI handles authentication and billing; no risk of unexpected charges.

2. **Remove cost estimates** - CLI-based evaluation doesn't expose token counts or costs the same way SDK does. Users rely on Claude CLI's own billing.

3. **Field name mapping** - Updated scorer to use `Reason` field from `EvaluationResult` instead of old `Reasoning` from `llm.Evaluation`.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all tasks completed smoothly. The field name difference between `EvaluationResult.Reason` and the old `llm.Evaluation.Reasoning` was documented in the plan.

## User Setup Required

None - no external service configuration required. Claude CLI installation is the only prerequisite.

## Next Phase Readiness

- CLI migration complete, ready for 15-03 testing and validation
- All tests pass with SDK removed
- Scan command works without ANTHROPIC_API_KEY

---
*Phase: 15-claude-code-integration*
*Completed: 2026-02-03*
