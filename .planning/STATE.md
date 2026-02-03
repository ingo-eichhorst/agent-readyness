# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-01)

**Core value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.
**Current focus:** v2 Complete Analysis Framework - All phases complete

## Current Position

Phase: 10 of 10 (C7 Agent Evaluation)
Plan: 2 of 2 in current phase
Status: Phase complete
Last activity: 2026-02-03 -- Completed 10-02-PLAN.md (LLM Scoring and CLI Integration)

Progress: [####################] 100% (29/29 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 29
- Average duration: 6 min
- Total execution time: 176 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation | 3 | 9 min | 3 min |
| 02-core-analysis | 5 | 44 min | 9 min |
| 03-scoring-model | 3 | 10 min | 3 min |
| 04-recommendations-and-output | 3 | 14 min | 5 min |
| 05-hardening | 2 | 5 min | 3 min |
| 06-multi-language-foundation | 4 | 29 min | 7 min |
| 07-python-typescript-c1-c3-c6 | 2 | 19 min | 10 min |
| 08-c5-temporal-dynamics | 2 | 8 min | 4 min |
| 09-c4-documentation-quality | 3 | 28 min | 9 min |
| 10-c7-agent-evaluation | 2 | 10 min | 5 min |

**Recent Trend:**
- Last 5 plans: 09-02 (12 min), 09-03 (8 min), 10-01 (3 min), 10-02 (7 min)
- Trend: All phases complete; v2 roadmap delivered

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- v2 scope: Multi-language (Go/Python/TypeScript) + C2/C4/C5/C7 categories
- Tree-sitter for Python/TS parsing (not language runtimes), requires CGO_ENABLED=1
- Native git CLI for C5 (not go-git, 10-100x faster)
- Anthropic SDK for C4/C7 LLM features (single provider, Haiku for cost)
- Tiered execution: free/fast default (C1-C3/C5-C6), LLM features opt-in (C4-LLM/C7)
- Dual-parser: keep go/packages for Go, Tree-sitter for Python/TypeScript
- GoAwareAnalyzer bridge pattern: Go analyzers use SetGoPackages, new analyzers use AnalysisTarget directly
- Separate .ts and .tsx Tree-sitter parsers (different grammars for TypeScript vs TSX)
- Extension-based language routing in walker via langExtensions map
- Map-based ScoringConfig with Categories map[string]CategoryConfig (extensible for C4/C5/C7)
- Extractor pattern for scoring (metricExtractors map decouples scoring from extraction)
- C2 Python analyzer uses Tree-sitter node walking (not queries) for type annotation counting
- TypeScript any types penalized in coverage score; null safety = strictNullChecks + optional chaining
- .arsrc.yml project config with version 1, category weight overrides
- Pipeline auto-creates Tree-sitter parser; degrades gracefully if CGO unavailable
- Language dispatch via switch/case in each analyzer's Analyze method (C1/C3/C6 match C2 pattern)
- Thread-safe TreeSitterParser: sync.Mutex added to ParseFile for concurrent analyzer safety
- NewCxAnalyzer(tsParser) constructor pattern for all analyzers
- tsNormalizePath strips /index suffix for TypeScript module resolution matching
- Test detection via call_expression name matching (describe/it/test) for Jest/Vitest/Mocha
- Assertion counting uses expect() as anchor, not chain methods
- C5Analyzer is repo-level (uses RootDir, not per-file targets); no Tree-sitter dependency
- C5 uses 6-month git log window; 90-day sub-window for churn/author metrics
- Skip commits >50 files for coupling; min 5 commits per file for qualification
- C5 tests use real repo (not fixtures) for integration confidence
- C4Analyzer is repo-level like C5 (uses RootDir for file existence checks)
- C4 boolean metrics (changelog, examples, etc) converted to 0/1 for scoring
- TypeScript JSDoc detection uses simpler regex approach vs full Tree-sitter
- LLM client uses Anthropic SDK with claude-3-5-haiku for cost-effective evaluation
- Prompt caching with cache_control ephemeral for system prompts (rubrics)
- Max 100 file sampling for LLM cost control in large repos
- User confirmation required before LLM analysis (cost transparency)
- go-charts/v2 for radar chart SVG generation (no external JS dependencies)
- HTML templates embedded via embed.FS for self-contained binary
- Radar chart requires min 3 categories (go-charts library constraint)
- Baseline trend comparison via previous JSON output parsing
- C7 tasks use Read-only tools (Read,Glob,Grep) - no writes to codebase
- C7 executor uses cmd.Cancel (SIGINT) + cmd.WaitDelay for graceful subprocess timeout
- Git worktree for C7 workspace isolation; fallback to read-only mode for non-git repos
- C7Analyzer disabled by default, requires Enable(client) call via --enable-c7 flag
- LLM-as-a-judge pattern for C7 scoring with task-specific rubrics
- Score scaling: 1-10 LLM scores multiplied by 10 for 0-100 consistency

### Pending Todos

None.

### Blockers/Concerns

- CGO requirement: v2 needs CGO_ENABLED=1 for Tree-sitter (v1 was pure Go)

## Session Continuity

Last session: 2026-02-03T15:17:37Z
Stopped at: Completed 10-02-PLAN.md (LLM Scoring and CLI Integration)
Resume file: None
