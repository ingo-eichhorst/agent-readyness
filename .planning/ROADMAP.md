# Roadmap: Agent Readiness Score (ARS)

## Overview

ARS delivers a Go CLI that scans codebases and produces a composite agent-readiness score with actionable improvement recommendations. The roadmap follows the natural dependency chain: establish the parsing foundation, build all three analyzer categories (C1/C3/C6), layer scoring on top of collected metrics, add user-facing output and recommendations, then harden for real-world use with edge case handling and performance optimization.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - CLI skeleton, Go file discovery, and pipeline architecture
- [x] **Phase 2: Core Analysis** - C1 (Code Health), C3 (Architecture), C6 (Testing) metric analyzers
- [x] **Phase 3: Scoring Model** - Per-category and composite scoring with tier ratings
- [x] **Phase 4: Recommendations and Output** - Terminal output, improvement recommendations, CI gating
- [ ] **Phase 5: Hardening** - Edge cases, performance optimization, progress indicators

## Phase Details

### Phase 1: Foundation
**Goal**: Users can point the CLI at a Go repository and see it correctly discover and classify all Go source files
**Depends on**: Nothing (first phase)
**Requirements**: FOUND-01, FOUND-02, FOUND-03, FOUND-04, FOUND-05, FOUND-06, FOUND-08, FOUND-09
**Success Criteria** (what must be TRUE):
  1. Running `ars scan <directory>` on a Go project discovers all .go files and reports file counts (source vs test)
  2. Running `ars scan` on a non-Go directory produces a clear error message explaining why it failed
  3. Vendor directories and generated code are automatically excluded from discovered files
  4. `ars --help` prints usage documentation and `ars --version` prints the version string
  5. The pipeline architecture processes files through discovery, parsing, and a stub analyzer, producing structured output
**Plans**: 3 plans

Plans:
- [x] 01-01-PLAN.md -- Project init, shared types, CLI skeleton with cobra
- [x] 01-02-PLAN.md -- TDD file discovery engine and Go file classifier
- [x] 01-03-PLAN.md -- Pipeline architecture, terminal output, wire scan command

### Phase 2: Core Analysis
**Goal**: The tool measures all C1, C3, and C6 metrics accurately across real Go codebases
**Depends on**: Phase 1
**Requirements**: C1-01, C1-02, C1-03, C1-04, C1-05, C1-06, C3-01, C3-02, C3-03, C3-04, C3-05, C6-01, C6-02, C6-03, C6-04, C6-05
**Success Criteria** (what must be TRUE):
  1. Running `ars scan` on a Go project reports per-function cyclomatic complexity, function length, and file size metrics with avg and max values
  2. Running `ars scan` reports coupling metrics (afferent and efferent) per module and detects duplicated code blocks
  3. Running `ars scan` reports directory depth, module fanout, circular dependencies, import complexity, and dead code
  4. Running `ars scan` detects test files, calculates test-to-code ratio, parses coverage reports, identifies test isolation issues, and reports assertion density
  5. All metrics produce correct results when validated against known Go repositories (e.g., this repository, standard library packages)
**Plans**: 5 plans

Plans:
- [x] 02-01-PLAN.md -- Parser with go/packages, types evolution, pipeline interface update
- [x] 02-02-PLAN.md -- C1 Code Health analyzer (complexity, function length, file size, coupling, duplication)
- [x] 02-03-PLAN.md -- C3 Architecture analyzer (directory depth, fanout, circular deps, import complexity, dead code)
- [x] 02-04-PLAN.md -- C6 Testing analyzer (test detection, ratio, coverage, isolation, assertions)
- [x] 02-05-PLAN.md -- Pipeline wiring, terminal output for metrics, end-to-end integration

### Phase 3: Scoring Model
**Goal**: Raw metrics are converted into meaningful per-category and composite scores that predict agent readiness
**Depends on**: Phase 2
**Requirements**: SCORE-01, SCORE-02, SCORE-03, SCORE-04, SCORE-05, SCORE-06
**Success Criteria** (what must be TRUE):
  1. Each category (C1, C3, C6) produces a 1-10 score derived from its collected metrics via piecewise linear interpolation
  2. A composite score is calculated using weighted average (C1: 25%, C3: 20%, C6: 15%) and displayed alongside the tier rating (Agent-Ready/Assisted/Limited/Hostile)
  3. Running with `--verbose` shows the per-metric breakdown contributing to each category score
  4. Scoring thresholds are configurable (not hardcoded), enabling future tuning without code changes
**Plans**: 3 plans

Plans:
- [x] 03-01-PLAN.md -- Scoring foundation: types, config, interpolation, composite, tier classification (TDD)
- [x] 03-02-PLAN.md -- Category scorers: C1/C3/C6 metric extraction and per-category scoring (TDD)
- [x] 03-03-PLAN.md -- Pipeline integration, terminal score rendering, --config flag

### Phase 4: Recommendations and Output
**Goal**: Users see a polished terminal report with scores, tier rating, and actionable improvement recommendations
**Depends on**: Phase 3
**Requirements**: REC-01, REC-02, REC-03, REC-04, REC-05, OUT-01, OUT-02, OUT-03, OUT-04, OUT-05, OUT-06
**Success Criteria** (what must be TRUE):
  1. Terminal output displays composite score, tier rating, per-category scores, and metric breakdowns with ANSI color formatting
  2. Top 5 improvement recommendations appear ranked by impact, each with estimated score improvement and effort level (Low/Medium/High)
  3. Recommendations are framed in agent-readiness terms (not generic code quality language)
  4. Running with `--threshold X` exits with code 2 when the composite score falls below X
  5. Running with `--verbose` shows detailed per-metric breakdown alongside the standard output
**Plans**: 3 plans

Plans:
- [x] 04-01-PLAN.md -- Recommendation engine: impact simulation, ranking, effort estimation (TDD)
- [x] 04-02-PLAN.md -- Terminal recommendation rendering and JSON output
- [x] 04-03-PLAN.md -- Pipeline wiring, --threshold/--json flags, exit code handling

### Phase 5: Hardening
**Goal**: The tool handles real-world edge cases gracefully and performs well on large codebases
**Depends on**: Phase 4
**Requirements**: FOUND-07, OUT-07, OUT-08
**Success Criteria** (what must be TRUE):
  1. Symlinks, files with syntax errors, and Unicode paths are handled without crashes or misleading results
  2. Scanning a 50k LOC repository completes in under 30 seconds
  3. Long-running scans display progress indicators so the user knows work is happening
**Plans**: 2 plans

Plans:
- [ ] 05-01-PLAN.md -- Edge case resilience: symlinks, permission errors, Unicode paths in walker
- [ ] 05-02-PLAN.md -- Parallel analyzers, stderr progress spinner, TTY detection

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 3/3 | Complete | 2026-01-31 |
| 2. Core Analysis | 5/5 | Complete | 2026-01-31 |
| 3. Scoring Model | 3/3 | Complete | 2026-01-31 |
| 4. Recommendations and Output | 3/3 | Complete | 2026-01-31 |
| 5. Hardening | 0/2 | Not started | - |
