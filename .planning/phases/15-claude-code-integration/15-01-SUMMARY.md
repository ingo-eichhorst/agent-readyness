---
phase: 15-claude-code-integration
plan: 01
subsystem: agent
tags: [claude-cli, llm-evaluation, documentation-analysis, structured-output]

# Dependency graph
requires:
  - phase: 14-html-enhancements
    provides: HTML report generation foundation
provides:
  - CLI detection infrastructure (DetectCLI, CLIStatus)
  - CLI-based content evaluator with structured JSON output
  - C4 documentation prompts migrated to agent package
  - C4 analyzer using CLI evaluator instead of Anthropic SDK
affects: [15-02, 15-03, 16-code-organization]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - CLI subprocess management with graceful SIGINT shutdown
    - JSON schema for structured LLM output
    - sync.Once caching for CLI detection

key-files:
  created:
    - internal/agent/cli.go
    - internal/agent/cli_test.go
    - internal/agent/evaluator.go
    - internal/agent/evaluator_test.go
    - internal/agent/prompts.go
  modified:
    - internal/analyzer/c4_documentation.go
    - internal/pipeline/pipeline.go

key-decisions:
  - "Use 60-second timeout per evaluation (sufficient for CLI startup + response)"
  - "Single retry with 2-second backoff (matches existing SDK retry pattern)"
  - "Keep llm import in pipeline temporarily (C7 still uses it until plan 02)"

patterns-established:
  - "CLI evaluation pattern: exec.CommandContext with --output-format json and --json-schema"
  - "Graceful shutdown: cmd.Cancel with SIGINT, cmd.WaitDelay for grace period"

# Metrics
duration: 5min
completed: 2026-02-03
---

# Phase 15 Plan 01: CLI Evaluation Infrastructure Summary

**CLI-based content evaluation infrastructure replacing Anthropic SDK for C4 documentation analysis using structured JSON output**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-03T22:38:23Z
- **Completed:** 2026-02-03T22:43:28Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments
- Created CLI detection with caching and installation hints
- Built CLI-based evaluator with structured JSON schema output
- Migrated C4 evaluation prompts from llm package to agent package
- Refactored C4 analyzer to use CLI evaluator instead of SDK

## Task Commits

Each task was committed atomically:

1. **Task 1: Create CLI detection module** - `53f5ae8` (feat)
2. **Task 2: Create CLI-based evaluator and migrate prompts** - `bd654c7` (feat)
3. **Task 3: Refactor C4 analyzer to use CLI evaluator** - `355f4f9` (refactor)

## Files Created/Modified
- `internal/agent/cli.go` - CLI detection with CLIStatus struct and DetectCLI function
- `internal/agent/cli_test.go` - Tests for CLI detection and caching
- `internal/agent/evaluator.go` - Evaluator struct with EvaluateContent and EvaluateWithRetry
- `internal/agent/evaluator_test.go` - Integration tests for CLI evaluation
- `internal/agent/prompts.go` - C4 evaluation prompts (ReadmeClarity, ExampleQuality, etc.)
- `internal/analyzer/c4_documentation.go` - Updated to use agent.Evaluator
- `internal/pipeline/pipeline.go` - Updated SetLLMClient to create agent.Evaluator for C4

## Decisions Made
- **60-second timeout per evaluation:** Sufficient for CLI startup overhead plus LLM response generation
- **Single retry with 2-second backoff:** Matches existing SDK retry behavior, adequate for transient failures
- **Keep llm import temporarily:** Pipeline still needs llm.Client for C7 analyzer until plan 02 completes

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed function naming conflict in tests**
- **Found during:** Task 1 (CLI detection tests)
- **Issue:** `contains` function in cli_test.go conflicted with same name in scorer_test.go
- **Fix:** Renamed to `containsSubstr` in cli_test.go
- **Files modified:** internal/agent/cli_test.go
- **Verification:** Tests compile and pass
- **Committed in:** 53f5ae8 (Task 1 commit)

**2. [Rule 3 - Blocking] Updated pipeline.go to use new evaluator API**
- **Found during:** Task 3 (C4 analyzer refactoring)
- **Issue:** Build failed because SetLLMClient called c4.SetLLMClient which no longer exists
- **Fix:** Updated pipeline to create agent.Evaluator and call c4.SetEvaluator
- **Files modified:** internal/pipeline/pipeline.go
- **Verification:** Build succeeds, scan command works
- **Committed in:** 355f4f9 (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes necessary for compilation. No scope creep.

## Issues Encountered
None - plan executed smoothly after addressing blocking issues.

## User Setup Required
None - no external service configuration required. Claude CLI must be installed for LLM features to work.

## Next Phase Readiness
- CLI evaluation infrastructure ready for use
- Plan 02 will delete internal/llm package and migrate C7 to CLI
- Plan 03 will add C4 flag wiring in cmd/root.go

---
*Phase: 15-claude-code-integration*
*Completed: 2026-02-03*
