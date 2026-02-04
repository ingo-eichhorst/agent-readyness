# Roadmap: ARS v0.0.3

## Overview

ARS v0.0.3 simplifies LLM integration by unifying on Claude Code CLI (removing Anthropic SDK), adds badge generation for visibility, enhances HTML reports with research-backed expandable descriptions, reorganizes the analyzer codebase into category subdirectories, and adds polish items (README badges, testing improvements). The milestone follows a risk-ordered approach: low-risk additive features first, then the highest-risk behavior change (Claude Code migration), followed by structural reorganization after features stabilize.

## Milestones

- v1.0 MVP - Phases 1-5 (shipped 2026-02-01)
- v0.0.2 Complete Analysis Framework - Phases 6-12 (shipped 2026-02-03)
- v0.0.3 Simplification & Polish - Phases 13-17 (in progress)

## Phases

- [x] **Phase 13: Badge Generation** - shields.io badge URL generation with CLI flag
- [x] **Phase 14: HTML Enhancements** - Expandable metric descriptions with research citations
- [x] **Phase 15: Claude Code Integration** - Unified CLI for C4/C7, remove Anthropic SDK
- [x] **Phase 16: Analyzer Reorganization** - Category subdirectories with re-exports
- [ ] **Phase 17: README & Testing** - Status badges and coverage improvements

## Phase Details

### Phase 13: Badge Generation
**Goal**: Users can generate shields.io badge URLs to display ARS scores in READMEs
**Depends on**: Nothing (first phase of milestone)
**Requirements**: BADGE-01, BADGE-02, BADGE-03
**Success Criteria** (what must be TRUE):
  1. Running `ars scan --badge` outputs a shields.io markdown URL to stdout
  2. Badge color reflects the tier (red for Agent-Hostile, orange for Agent-Limited, yellow for Agent-Assisted, green for Agent-Ready)
  3. Badge displays both tier name and numeric score (e.g., "Agent-Ready 8.2/10")
**Plans**: 1 plan

Plans:
- [x] 13-01-PLAN.md — Badge URL generation, CLI flag, and output integration

### Phase 14: HTML Enhancements
**Goal**: HTML reports provide educational context with expandable research-backed metric descriptions
**Depends on**: Nothing (independent of Phase 13)
**Requirements**: HTML-01, HTML-02, HTML-03, HTML-04
**Success Criteria** (what must be TRUE):
  1. Each metric in HTML report shows a brief 1-2 sentence description
  2. Each metric has an expandable section with detailed explanation and research citations
  3. Expandable sections work without JavaScript (CSS-only details/summary)
  4. Categories scoring below 6.0 have their detail sections expanded by default
**Plans**: 1 plan

Plans:
- [x] 14-01-PLAN.md — Metric descriptions data, HTML struct updates, template with details/summary

### Phase 15: Claude Code Integration
**Goal**: All LLM features use Claude Code CLI, eliminating Anthropic SDK dependency
**Depends on**: Nothing (independent feature work)
**Requirements**: LLM-01, LLM-02, LLM-03, LLM-04, LLM-05
**Success Criteria** (what must be TRUE):
  1. C4 documentation quality analysis uses Claude Code CLI (`claude -p`) instead of Anthropic SDK
  2. C7 agent evaluation continues working with Claude Code CLI (regression check)
  3. LLM analysis runs automatically when Claude CLI is available (no `--enable-c4-llm` flag)
  4. No `ANTHROPIC_API_KEY` environment variable required
  5. Anthropic SDK removed from go.mod
**Plans**: 2 plans

Plans:
- [x] 15-01-PLAN.md — CLI detection, evaluator infrastructure, C4 refactor to use CLI
- [x] 15-02-PLAN.md — Pipeline auto-detection, flag cleanup, SDK removal

### Phase 16: Analyzer Reorganization
**Goal**: Analyzer code organized into category subdirectories for improved navigability
**Depends on**: Phase 15 (do structural changes after features stabilize)
**Requirements**: REORG-01, REORG-02, REORG-03, REORG-04
**Success Criteria** (what must be TRUE):
  1. Each category has its own subdirectory (internal/analyzer/c1/, c2/, ..., c7/)
  2. All analyzer files moved to appropriate subdirectories
  3. All import paths work correctly (no broken imports)
  4. Root-level analyzer.go provides re-exports for backward compatibility
**Plans**: 2 plans

Plans:
- [x] 16-01-PLAN.md — Foundation: create directories, shared.go, root re-exports
- [x] 16-02-PLAN.md — Move all category files, update imports, verify tests

### Phase 17: README & Testing
**Goal**: Project has standard status badges and test commands include coverage
**Depends on**: Nothing (independent polish work)
**Requirements**: README-01, README-02, README-03, README-04, TEST-01, TEST-02
**Success Criteria** (what must be TRUE):
  1. README displays Go Reference badge
  2. README displays Go Report Card badge
  3. README displays License badge
  4. README displays Release badge
  5. Test commands in documentation include `-coverprofile` flag
  6. Coverage data is available for C6 self-analysis
**Plans**: 1 plan

Plans:
- [ ] 17-01-PLAN.md — Add LICENSE, README badges, fix coverage filename

## Progress

**Execution Order:**
Phases 13-17 execute in order. Phases 13, 14, 15, 17 are independent and could parallelize, but Phase 16 depends on Phase 15.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 13. Badge Generation | 1/1 | Complete | 2026-02-03 |
| 14. HTML Enhancements | 1/1 | Complete | 2026-02-03 |
| 15. Claude Code Integration | 2/2 | Complete | 2026-02-04 |
| 16. Analyzer Reorganization | 2/2 | Complete | 2026-02-04 |
| 17. README & Testing | 0/1 | Not started | - |

---
*Roadmap created: 2026-02-03*
*Milestone: v0.0.3 Simplification & Polish*
