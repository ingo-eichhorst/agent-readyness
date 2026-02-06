# Roadmap: ARS v0.0.5 - C7 Debug Infrastructure

## Overview

This milestone fixes the M2/M3/M4 scoring bug (GitHub #55) and establishes debug infrastructure for ongoing C7 validation. The work flows from plumbing a debug flag through the pipeline, to capturing prompt/response data, to testing and fixing the heuristic scoring functions, to rendering debug output and enabling response replay for fast iteration. Four phases deliver the complete investigation-to-fix workflow.

## Milestones

- [Archived] **v1 MVP** - Phases 1-5 (shipped 2026-02-01)
- [Archived] **v0.0.2 Complete Analysis Framework** - Phases 6-12 (shipped 2026-02-03)
- [Archived] **v0.0.3 Simplification & Polish** - Phases 13-17 (shipped 2026-02-04)
- [Archived] **v0.0.4 Metric Research & C7 Implementation** - Phases 18-25 (shipped 2026-02-05)
- Active **v0.0.5 C7 Debug Infrastructure** - Phases 26-29 (in progress)

## Phases

- [x] **Phase 26: Debug Foundation** - Flag plumbing and debug output channel
- [x] **Phase 27: Data Capture** - Prompt/response storage and score trace infrastructure
- [ ] **Phase 28: Heuristic Tests & Scoring Fixes** - Real response fixtures, unit tests, and M2/M3/M4 bug fixes
- [ ] **Phase 29: Debug Rendering & Replay** - Terminal/JSON debug output, response persistence, replay mode, documentation

## Phase Details

### Phase 26: Debug Foundation
**Goal**: Users can activate C7 debug mode via a single CLI flag that routes diagnostic output to stderr without affecting normal operation
**Depends on**: Nothing (first phase of milestone)
**Requirements**: DBG-01, DBG-02, DBG-03
**Success Criteria** (what must be TRUE):
  1. Running `ars scan . --debug-c7` activates C7 evaluation automatically (no need to also pass `--enable-c7`)
  2. Debug output appears exclusively on stderr; `ars scan . --debug-c7 --json 2>/dev/null | jq` produces valid JSON on stdout
  3. Running `ars scan .` without `--debug-c7` produces identical output and performance to current behavior (zero-cost when disabled)
**Plans**: 1 plan

Plans:
- [x] 26-01-PLAN.md -- Wire --debug-c7 flag from CLI through Pipeline to C7Analyzer with debugWriter pattern and tests

### Phase 27: Data Capture
**Goal**: Debug mode preserves full prompts, responses, and score traces that flow through the pipeline for downstream rendering
**Depends on**: Phase 26 (debug flag and writer must exist)
**Requirements**: DBG-04, DBG-05, DBG-06
**Success Criteria** (what must be TRUE):
  1. When debug is active, each metric's SampleResult contains the full prompt that was sent to Claude CLI
  2. When debug is active, each metric's SampleResult contains the full response received from Claude CLI
  3. When debug is active, C7MetricResult contains per-sample score traces showing which heuristic indicators matched and their individual contributions
  4. When debug is inactive, no additional allocations occur in the metric execution path
**Plans**: 2 plans

Plans:
- [x] 27-01-PLAN.md -- Extend SampleResult with Prompt + ScoreTrace fields, update M1-M5 scoring to produce traces
- [x] 27-02-PLAN.md -- Add C7DebugSample type, extend C7MetricResult with DebugSamples, populate in buildMetrics()

### Phase 28: Heuristic Tests & Scoring Fixes
**Goal**: M2, M3, and M4 scoring functions produce accurate non-zero scores validated against real Claude CLI response fixtures
**Depends on**: Nothing (test development is independent of CLI debug infrastructure)
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05, FIX-01, FIX-02, FIX-03, FIX-04
**Success Criteria** (what must be TRUE):
  1. `testdata/c7_responses/` contains real captured Claude CLI responses for M2, M3, and M4 metrics (not fabricated strings)
  2. `go test ./internal/agent/metrics/ -run TestM2_Score -v` passes with documented expected scores for each fixture
  3. `go test ./internal/agent/metrics/ -run TestM3_Score -v` passes with documented expected scores for each fixture
  4. `go test ./internal/agent/metrics/ -run TestM4_Score -v` passes with documented expected scores for each fixture
  5. Running `ars scan . --enable-c7` produces non-zero scores for M2, M3, and M4 on a real codebase (the bug is fixed)
**Plans**: 3 plans

Plans:
- [ ] 28-01-PLAN.md -- Capture real Claude CLI response fixtures into testdata/c7_responses/ directory
- [ ] 28-02-PLAN.md -- Fix extractC7 to return M1-M5 metric scores in formal scoring pipeline
- [ ] 28-03-PLAN.md -- Fixture-based unit tests for M2/M3/M4 + fix scoring saturation with grouped indicators

### Phase 29: Debug Rendering & Replay
**Goal**: Users can inspect C7 debug data in terminal output, persist responses to disk for offline analysis, and replay saved responses without re-executing Claude CLI
**Depends on**: Phase 26 (debug channel), Phase 27 (debug data), Phase 28 (working scoring)
**Requirements**: RPL-01, RPL-02, RPL-03, RPL-04, DOC-01, DOC-02, DOC-03, DOC-04
**Success Criteria** (what must be TRUE):
  1. Running `ars scan . --debug-c7` displays per-metric per-sample prompts, responses (truncated), scores, and durations on stderr
  2. Running `ars scan . --debug-c7 --debug-dir ./debug-out` saves captured responses as JSON files in the specified directory
  3. Running `ars scan . --debug-c7 --debug-dir ./debug-out` a second time replays saved responses without executing Claude CLI (fast iteration)
  4. `ars scan --help` documents the `--debug-c7` and `--debug-dir` flags with clear usage descriptions
  5. GitHub issue #55 is updated with root cause analysis, fixes applied, and test results
**Plans**: TBD

Plans:
- [ ] 29-01: Implement renderC7Debug in terminal.go, thread debugC7 through output rendering
- [ ] 29-02: Implement --debug-dir flag, response persistence (save) and replay (load) modes
- [ ] 29-03: Update CLI help, README/docs, and GitHub issue #55

## Progress

**Execution Order:** 26 -> 27 -> 28 -> 29 (Phase 28 can start in parallel with 26-27)

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 26. Debug Foundation | 1/1 | ✓ Complete | 2026-02-06 |
| 27. Data Capture | 2/2 | ✓ Complete | 2026-02-06 |
| 28. Heuristic Tests & Scoring Fixes | 0/3 | Not started | - |
| 29. Debug Rendering & Replay | 0/3 | Not started | - |
