# Roadmap: Agent Readiness Score (ARS)

## Milestones

- v1 Initial Release (shipped 2026-02-01) - Phases 1-5
- v2 Complete Analysis Framework - Phases 6-10 (in progress)

## Phases

<details>
<summary>v1 Initial Release (Phases 1-5) - SHIPPED 2026-02-01</summary>

Phases 1-5 delivered Go-only CLI with C1 (Code Health), C3 (Architecture), C6 (Testing) analysis.
5 phases, 16 plans completed. See MILESTONES.md for details.

</details>

### v2 Complete Analysis Framework (In Progress)

**Milestone Goal:** Transform ARS from a Go-specific structural analyzer into a comprehensive, multi-language agent-readiness assessment tool with all seven research-backed analysis categories (C1-C7), HTML reports, and configurable scoring.

**Phase Numbering:**
- Integer phases (6, 7, 8, 9, 10): Planned milestone work
- Decimal phases (e.g., 7.1): Urgent insertions if needed (marked with INSERTED)

- [x] **Phase 6: Multi-Language Foundation + C2 Semantic Explicitness** - Generalize abstractions for Go/Python/TypeScript, add Tree-sitter parsing, implement C2 across all languages, build config system
- [x] **Phase 7: Python + TypeScript Analysis (C1/C3/C6)** - Port existing code health, architecture, and testing analyzers to Python and TypeScript via Tree-sitter
- [x] **Phase 8: C5 Temporal Dynamics** - Git-based temporal analysis with native git CLI for churn, hotspots, author fragmentation, and temporal coupling
- [ ] **Phase 9: C4 Documentation Quality + HTML Reports** - Documentation analysis with LLM content evaluation, plus polished HTML report generation
- [ ] **Phase 10: C7 Agent Evaluation** - Headless Claude Code integration for genuine agent-in-the-loop readiness assessment

## Phase Details

### Phase 6: Multi-Language Foundation + C2 Semantic Explicitness
**Goal**: Users can analyze Go, Python, and TypeScript codebases for semantic explicitness and type safety, with configurable scoring weights and thresholds
**Depends on**: v1 (Phase 5)
**Requirements**: LANG-01, LANG-02, LANG-03, LANG-04, LANG-05, LANG-06, LANG-07, C2-GO-01, C2-GO-02, C2-GO-03, C2-GO-04, C2-PY-01, C2-PY-02, C2-PY-03, C2-PY-04, C2-TS-01, C2-TS-02, C2-TS-03, C2-TS-04, SCORE-01, SCORE-02, SCORE-03, SCORE-04, SCORE-05, SCORE-06, SCORE-07, SCORE-08, CLI-04, CLI-07, CLI-08
**Success Criteria** (what must be TRUE):
  1. User can run `ars scan` on a Python project and see C2 semantic explicitness scores (type annotation coverage, naming consistency, magic numbers)
  2. User can run `ars scan` on a TypeScript project and see C2 scores (type coverage, strict mode detection, null safety)
  3. User can run `ars scan` on a mixed-language repo and see per-language C2 analysis in the output
  4. User can provide a `.arsrc.yml` config file to customize category weights, metric thresholds, and per-language overrides
  5. Non-LLM analysis completes in under 30 seconds for a 50k LOC repository
**Plans**: 4 plans

Plans:
- [x] 06-01-PLAN.md -- AnalysisTarget abstraction + pipeline interface refactoring
- [x] 06-02-PLAN.md -- Multi-language discovery + Tree-sitter parser integration
- [x] 06-03-PLAN.md -- C2 Go analyzer + map-based scoring expansion
- [x] 06-04-PLAN.md -- C2 Python/TypeScript + config system + CLI wiring

### Phase 7: Python + TypeScript Analysis (C1/C3/C6)
**Goal**: Users get full code health (C1), architecture (C3), and testing (C6) analysis for Python and TypeScript projects, matching the depth of Go analysis
**Depends on**: Phase 6
**Requirements**: PY-01, PY-02, PY-03, PY-04, PY-05, PY-06, PY-07, PY-08, PY-09, TS-01, TS-02, TS-03, TS-04, TS-05, TS-06, TS-07, TS-08, TS-09
**Success Criteria** (what must be TRUE):
  1. User can run `ars scan` on a Python project and see C1 scores (cyclomatic complexity, function length, file size, duplication) comparable to Go analysis
  2. User can run `ars scan` on a TypeScript project and see C3 scores (import graph, dead code, directory depth) comparable to Go analysis
  3. User can run `ars scan` on a Python project and see C6 scores with pytest/unittest detection and coverage.py parsing
  4. User can run `ars scan` on a TypeScript project and see C6 scores with Jest/Mocha/Vitest detection and Istanbul/lcov parsing
**Plans**: 2 plans

Plans:
- [x] 07-01-PLAN.md -- Python C1/C3/C6 analyzers + dispatcher refactoring + pipeline wiring
- [x] 07-02-PLAN.md -- TypeScript C1/C3/C6 analyzers + end-to-end verification

### Phase 8: C5 Temporal Dynamics
**Goal**: Users can see git-based temporal analysis revealing code churn hotspots, ownership patterns, and change coupling that affect agent effectiveness
**Depends on**: Phase 6 (uses AnalysisTarget + scoring infrastructure)
**Requirements**: C5-01, C5-02, C5-03, C5-04, C5-05, C5-06, C5-07, C5-08
**Success Criteria** (what must be TRUE):
  1. User can run `ars scan` on a git repository and see C5 temporal dynamics scores (churn rate, hotspot concentration, author fragmentation)
  2. User sees temporal coupling detection identifying files that change together more than 70% of the time
  3. User gets a clear error message when scanning a directory without a .git directory (C5 unavailable, not a crash)
  4. C5 analysis completes within the 30-second performance budget even on repos with 12+ months of history
**Plans**: 2 plans

Plans:
- [x] 08-01-PLAN.md -- C5 analyzer + git log parsing + metrics + scoring + pipeline wiring
- [x] 08-02-PLAN.md -- Unit tests + end-to-end verification

### Phase 9: C4 Documentation Quality + HTML Reports
**Goal**: Users get documentation quality analysis with optional LLM-based content evaluation, and can generate polished, self-contained HTML reports with visual score presentation and research citations
**Depends on**: Phase 6 (scoring infrastructure), Phase 7 and 8 (categories to render)
**Requirements**: C4-01, C4-02, C4-03, C4-04, C4-05, C4-06, C4-07, C4-08, C4-09, C4-10, C4-11, C4-12, C4-13, C4-14, HTML-01, HTML-02, HTML-03, HTML-04, HTML-05, HTML-06, HTML-07, HTML-08, HTML-09, HTML-10, CLI-01, CLI-03, CLI-05, CLI-06
**Success Criteria** (what must be TRUE):
  1. User can run `ars scan` and see C4 documentation quality scores (README presence, comment density, API doc coverage) without any LLM dependency
  2. User can run `ars scan --enable-c4-llm` and see LLM-evaluated content quality ratings (README clarity, example quality, completeness) with cost shown before execution
  3. User can run `ars scan --output-html` and get a self-contained HTML file with radar chart, metric breakdowns, research citations, and recommendations
  4. HTML report renders correctly offline with no external CSS/JS dependencies and is protected against XSS from code content
  5. User can run `ars scan --baseline previous.json --output-html` and see a trend comparison chart showing score changes over time
**Plans**: 3 plans

Plans:
- [ ] 09-01-PLAN.md -- C4 static documentation metrics (README, comments, API docs, CHANGELOG, examples)
- [ ] 09-02-PLAN.md -- LLM client abstraction + C4 content quality evaluation (--enable-c4-llm)
- [ ] 09-03-PLAN.md -- HTML report generation (templates, charts, research citations, --output-html)

### Phase 10: C7 Agent Evaluation
**Goal**: Users can opt in to a genuine agent-in-the-loop assessment where headless Claude Code attempts standardized tasks against their codebase, producing the most novel and differentiated ARS metric
**Depends on**: Phase 9 (shares LLM infrastructure, cost estimation patterns)
**Requirements**: C7-01, C7-02, C7-03, C7-04, C7-05, C7-06, C7-07, C7-08, C7-09, C7-10, CLI-02
**Success Criteria** (what must be TRUE):
  1. User can run `ars scan --enable-c7` and see C7 agent evaluation scores measuring intent clarity, modification confidence, cross-file coherence, and semantic completeness
  2. User sees cost estimation and must confirm before C7 evaluation runs
  3. C7 handles agent errors, timeouts, and failures gracefully without crashing the overall scan
  4. User without `claude` CLI installed gets a clear error when requesting C7, not a crash
**Plans**: TBD

Plans:
- [ ] 10-01: Headless Claude Code integration + task definitions
- [ ] 10-02: C7 scoring, sampling strategy, and error handling

## Progress

**Execution Order:**
Phases execute in numeric order: 6 -> 7 -> 8 -> 9 -> 10
(Phase 8 could potentially overlap with Phase 7 since C5 is git-based, not AST-based)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 6. Multi-Language + C2 | v2 | 4/4 | Complete | 2026-02-01 |
| 7. Python + TS (C1/C3/C6) | v2 | 2/2 | Complete | 2026-02-01 |
| 8. C5 Temporal | v2 | 2/2 | Complete | 2026-02-02 |
| 9. C4 Docs + HTML | v2 | 3/3 | Complete | 2026-02-03 |
| 10. C7 Agent Eval | v2 | 0/2 | Not started | - |
