# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-06)

**Core value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.
**Current focus:** Phase 28 - Heuristic Tests & Scoring Fixes

## Current Position

Phase: 27 of 29 (Data Capture)
Plan: 2 of 2 in current phase
Status: Phase complete, verified ✓
Last activity: 2026-02-06 — Completed Phase 27

Progress: [###.......] 33% (v0.0.5)

## Performance Metrics

**Velocity (all milestones):**
- Total plans completed: 57
- Phases completed: 27
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

### Pending Todos

None.

### Blockers/Concerns

- **M2/M3/M4 scoring 0/10:** Root cause under investigation (GitHub #55). Likely heuristic indicator saturation or response format mismatch. Phase 28 will diagnose and fix.

## Session Continuity

Last session: 2026-02-06
Stopped at: Completed 27-02-PLAN.md (Phase 27 complete)
Resume file: None
