# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-06)

**Core value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.
**Current focus:** v0.0.5 complete - Debug Rendering & Replay

## Current Position

Phase: 29 of 29 (Debug Rendering & Replay)
Plan: 3 of 3 in current phase
Status: Phase complete -- v0.0.5 milestone shipped
Last activity: 2026-02-06 — Completed 29-03-PLAN.md

Progress: [##########] 100% (v0.0.5)

## Performance Metrics

**Velocity (all milestones):**
- Total plans completed: 63
- Phases completed: 29
- Total milestones shipped: 5

**By Milestone:**

| Milestone | Phases | Plans | Days |
|-----------|--------|-------|------|
| v1 | 5 | 16 | 2 |
| v0.0.2 | 7 | 15 | 2 |
| v0.0.3 | 5 | 7 | 2 |
| v0.0.4 | 8 | 14 | 5 |
| v0.0.5 | 4 | 9 | 1 |

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [24-01]: Heuristic-based response scoring - keyword pattern matching over additional LLM calls
- [24-01]: Per-metric sample selection formulas - complexity/sqrt(LOC), import count, comment density
- [24-04]: 1-10 scale for MECE metrics - aligned with C1-C6, legacy 0-100 preserved
- [24-04]: Weight distribution M2+M3=25% each, M1=20%, M4+M5=15% each
- [25-01]: Heuristic disclaimer notes on all C7 thresholds - nascent field lacks validation
- [26-01]: io.Writer debug pattern (io.Discard/os.Stderr) over log.Logger for zero-cost debug
- [26-01]: Method-based debug threading (SetC7Debug -> SetDebug) over global state
- [27-01]: ScoreTrace as source of truth - score computed from trace indicators, not duplicated
- [27-01]: All indicators tracked (matched and unmatched) with Delta=0 for unmatched
- [27-02]: Separate output types from internal types - convertScoreTrace bridges metrics.ScoreTrace to types.C7ScoreTrace
- [27-02]: omitempty only on DebugSamples field - existing C7MetricResult fields lack json tags
- [28-02]: extractC7 returns all 6 C7 metrics (overall_score + 5 MECE) - root cause fix for C7 scoring 0/1
- [28-01]: Realistic fixtures over live CLI capture when concurrent instances rate-limited
- [28-03]: Grouped indicators over individual - each thematic group contributes +1 max regardless of members matched
- [28-03]: Variable BaseScore per metric (M2=2, M3=2, M4=1, M5=3) tuned to target score ranges
- [28-03]: M4 uses "accurate" not "correct" for self_report_positive to avoid false-positive on "partially correct"
- [29-01]: Prompt truncated to 200 chars, response to 500 chars in debug output for readability
- [29-01]: Only matched indicators shown in score trace line (unmatched omitted for brevity)
- [29-02]: Prompt-based metric identification over passing metricID through Executor interface
- [29-02]: Nil executor parameter over separate function for RunMetricsParallel
- [29-02]: --debug-dir implies --debug-c7 for single flag convenience
- [29-02]: Capture/replay mode auto-detected from directory contents

### Pending Todos

None.

### Blockers/Concerns

None - v0.0.5 milestone complete. All M2/M3/M4 scoring issues resolved, debug infrastructure shipped.

## Session Continuity

Last session: 2026-02-06
Stopped at: Completed 29-03-PLAN.md (v0.0.5 milestone complete)
Resume file: None
