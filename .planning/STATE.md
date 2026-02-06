# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-06)

**Core value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.
**Current focus:** Phase 29 in progress - Debug Rendering & Replay

## Current Position

Phase: 29 of 29 (Debug Rendering & Replay)
Plan: 1 of 3 in current phase
Status: In progress
Last activity: 2026-02-06 — Completed 29-01-PLAN.md

Progress: [######....] 43% (v0.0.5)

## Performance Metrics

**Velocity (all milestones):**
- Total plans completed: 61
- Phases completed: 28
- Total milestones shipped: 4

**By Milestone:**

| Milestone | Phases | Plans | Days |
|-----------|--------|-------|------|
| v1 | 5 | 16 | 2 |
| v0.0.2 | 7 | 15 | 2 |
| v0.0.3 | 5 | 7 | 2 |
| v0.0.4 | 8 | 14 | 5 |

**v0.0.5 (In Progress):**
- Phases: 4 (26-29)
- Plans estimated: 9
- Focus: C7 debug infrastructure + M2/M3/M4 scoring fix

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

### Pending Todos

None.

### Blockers/Concerns

None - M2/M3/M4 scoring issues fully resolved (Bug 1: extractC7 in 28-02, Bug 2: scoring saturation in 28-03).

## Session Continuity

Last session: 2026-02-06
Stopped at: Completed 29-01-PLAN.md
Resume file: None
