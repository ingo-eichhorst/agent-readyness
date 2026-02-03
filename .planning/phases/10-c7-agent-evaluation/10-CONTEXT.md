# Phase 10: C7 Agent Evaluation - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Headless Claude Code integration that executes standardized tasks against the user's codebase to measure agent-in-the-loop readiness. Produces C7 scores measuring intent clarity, modification confidence, cross-file coherence, and semantic completeness through genuine agent behavior.

This phase implements the agent evaluation capability only. Task definitions, scoring rubrics, and baseline tasks are scoped within this phase. Future task expansion or alternative agent backends belong in later work.

</domain>

<decisions>
## Implementation Decisions

### Task Definitions
- Start with 4 standardized tasks covering core agent capabilities:
  1. **Intent Clarity**: "Find the function that handles X and explain what it does" (reading/navigation)
  2. **Modification Confidence**: "Add a simple validation check to function Y" (targeted modification)
  3. **Cross-File Coherence**: "Trace the data flow from endpoint Z to storage" (multi-file navigation)
  4. **Semantic Completeness**: "Add error handling to workflow W following existing patterns" (context-aware modification)
- Tasks are language-agnostic where possible (adapt prompts based on detected language)
- Each task has a defined success rubric (not binary pass/fail, but graded 0-100)
- Tasks execute against the actual user codebase (not synthetic fixtures)

### Execution Approach
- Headless Claude Code via `claude` CLI subprocess invocation
- Sequential task execution (not parallel) to avoid state conflicts
- 5-minute timeout per task (agent should complete or fail gracefully)
- Read-only mode with temp directory for agent workspace (no writes to actual codebase)
- Git worktree or shallow clone for isolation
- Agent output captured (tool calls, messages, final response) for scoring

### Success Measurement
- Each task scored 0-100 based on rubric criteria:
  - **Intent Clarity**: Correct identification (40%), accuracy of explanation (40%), use of codebase context (20%)
  - **Modification Confidence**: Correctness of change (50%), appropriate scope (30%), follows patterns (20%)
  - **Cross-File Coherence**: Completeness of trace (50%), accuracy (30%), efficiency (20%)
  - **Semantic Completeness**: Functional correctness (40%), pattern matching (40%), edge case handling (20%)
- Overall C7 score = average of 4 task scores
- Metadata captured: execution time, tool call count, error count, completion status
- Failures (timeout, crash, refusal) score 0 for that task

### User Experience
- Cost estimation shown before execution (based on expected token usage ~10k tokens/task)
- Explicit confirmation required (similar to --enable-c4-llm pattern)
- Progress reporting per task: "Running C7 Task 1/4: Intent Clarity..."
- Clear error message if `claude` CLI not found (not a crash)
- Output includes per-task breakdown in JSON, summary in terminal
- Failures reported gracefully: "Task 2 timed out (5m limit)" → continue to next task
- Optional `--c7-verbose` flag to show agent tool calls (debugging)

### Claude's Discretion
- Exact task prompt phrasing (as long as intent is clear)
- Rubric scoring implementation details
- Subprocess invocation mechanics (stdin/stdout vs temp files)
- How to sample/select target code locations for tasks
- Error recovery strategies for flaky subprocess communication

</decisions>

<specifics>
## Specific Ideas

- Tasks should feel like "real work an agent would be asked to do" (not toy examples)
- C7 is the most novel metric — it should differentiate ARS from static analysis tools
- Cost transparency is critical (users must know before ~40k tokens are consumed)
- Graceful degradation if `claude` CLI unavailable (C7 shows "unavailable" not crash)
- Consider sampling strategy for large repos (run tasks on a representative subset, not entire codebase)

</specifics>

<deferred>
## Deferred Ideas

- Support for alternative agent backends (Cursor, Copilot, etc.) — future work
- Expanding task library beyond initial 4 tasks — future iteration
- User-defined custom tasks — potential v3 feature
- Comparing multiple agent versions (A/B testing) — research prototype
- Recording agent sessions as GIFs/videos — nice-to-have, not MVP

</deferred>

---

*Phase: 10-c7-agent-evaluation*
*Context gathered: 2026-02-03*
