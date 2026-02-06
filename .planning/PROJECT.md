# Agent Readiness Score (ARS)

## What This Is

A CLI tool that analyzes codebases (Go, Python, TypeScript) and produces a composite score (1-10) measuring how well the repository supports AI agent workflows. ARS evaluates seven dimensions of agent-readiness (code health, semantic explicitness, architecture, documentation, temporal dynamics, testing, and agent evaluation), then generates actionable improvement recommendations ranked by impact.

## Core Value

Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## Requirements

### Validated

**v1 requirements:**
- ✓ CLI accepts directory path and scans Go codebase — v1
- ✓ Auto-detects Go projects (go.mod, .go files) — v1
- ✓ C1: Code Health analysis (cyclomatic complexity, function length, file size, coupling, duplication) — v1
- ✓ C3: Architectural Navigability analysis (directory depth, module fanout, circular dependencies, import complexity, dead code) — v1
- ✓ C6: Testing Infrastructure analysis (test coverage, test-to-code ratio, test isolation, assertion density) — v1
- ✓ Composite score calculation using weighted average (C1: 25%, C3: 20%, C6: 15%) — v1
- ✓ Per-category scores with metric breakdowns — v1
- ✓ Top 5 improvement recommendations ranked by impact — v1
- ✓ Terminal text output with tier rating (Agent-Ready, Agent-Assisted, Agent-Limited, Agent-Hostile) — v1
- ✓ JSON output for machine consumption — v1
- ✓ Exit codes: 0 (success), 1 (error), 2 (below threshold if --threshold specified) — v1
- ✓ Usage: `ars scan <directory>` with --verbose, --json, --threshold, --config flags — v1
- ✓ Edge case handling (symlinks, permissions, Unicode paths) — v1
- ✓ Performance <30s for 50k LOC repos — v1
- ✓ Progress indicators for long-running scans — v1

**v0.0.2 requirements (95 total):**
- ✓ Multi-language support (Go, Python, TypeScript) with Tree-sitter parsing — v0.0.2
- ✓ C2: Semantic Explicitness analysis (type coverage, naming consistency, magic numbers, null safety) for all 3 languages — v0.0.2
- ✓ C4: Documentation Quality analysis (static metrics + optional LLM content evaluation) — v0.0.2
- ✓ C5: Temporal Dynamics analysis (git-based churn, hotspots, temporal coupling, author fragmentation) — v0.0.2
- ✓ C7: Agent Evaluation (headless Claude Code with LLM-as-judge scoring) — v0.0.2
- ✓ HTML report generation with radar charts, research citations, and baseline comparison — v0.0.2
- ✓ .arsrc.yml configuration system for custom weights and thresholds — v0.0.2
- ✓ Cost-transparent opt-in LLM features (--enable-c4-llm, --enable-c7 flags) — v0.0.2
- ✓ Complete 7-category framework (C1-C7) with updated composite scoring — v0.0.2

**v0.0.3 requirements (22 total):**
- ✓ Unified LLM integration on Claude Code CLI (removed Anthropic SDK, no API key required) — v0.0.3
- ✓ Auto-enable LLM features when Claude CLI available, --no-llm flag for opt-out — v0.0.3
- ✓ Badge generation with `--badge` flag producing shields.io markdown URL — v0.0.3
- ✓ Expandable metric descriptions in HTML reports with 33 research-backed explanations — v0.0.3
- ✓ Analyzer reorganization into 7 category subdirectories with shared utilities subpackage — v0.0.3
- ✓ MIT LICENSE and standard Go project badges (Go Reference, Report Card, License, Release) — v0.0.3
- ✓ Test coverage filename standardization (cover.out) for C6 self-analysis — v0.0.3

**v0.0.4 requirements (66 total):**
- ✓ Citation quality protocols (style guide, URL verification, source quality checklist) — v0.0.4
- ✓ C1 Code Quality metrics: 6 metrics with foundational + AI-era citations — v0.0.4
- ✓ C2 Semantic Explicitness metrics: 5 metrics with foundational + AI-era citations — v0.0.4
- ✓ C3 Architecture metrics: 5 metrics with foundational + AI-era citations — v0.0.4
- ✓ C4 Documentation metrics: 7 metrics with foundational + AI-era citations — v0.0.4
- ✓ C5 Temporal metrics: 5 metrics with foundational + AI-era citations — v0.0.4
- ✓ C6 Testing metrics: 5 metrics with foundational + AI-era citations — v0.0.4
- ✓ C7 MECE metrics: 5 agent-assessable metrics with parallel execution framework — v0.0.4
- ✓ C7 Agent Evaluation citations: 5 metrics with foundational + AI-era citations — v0.0.4

### Active

**v0.0.5 requirements (C7 M2/M3/M4 Bug Fix):**
- [ ] C7 M2 (Code Behavior Comprehension) calculates non-zero scores
- [ ] C7 M3 (Cross-File Navigation) calculates non-zero scores
- [ ] C7 M4 (Identifier Interpretability) calculates non-zero scores
- [ ] `--debug-c7` flag enables response inspection for all metrics
- [ ] Unit tests for M2/M3/M4 scoring heuristics with realistic agent responses
- [ ] GitHub #55 updated with root cause analysis and resolution

### Out of Scope

- GitHub Action — v3 (requires CI integration testing)
- VS Code extension — v3 (requires IDE integration)
- Incremental scanning / caching — v3 (optimize after full feature set exists)
- Monorepo per-package scoring — v3 (requires workspace detection)
- Trend dashboard / web UI — Future (requires persistence layer)
- Markdown report output — v3 (additional output format)
- Automated code fixes — Never (analysis only, no mutations)
- Real-time IDE linting — Never (batch analysis only)
- Competitive analysis — Never (single repo focus)

## Context

**Current State (v0.0.4 shipped 2026-02-05):**
- 26,372 LOC Go (+3,748 from v0.0.3)
- Tech stack: Go 1.24, cobra CLI, Tree-sitter (Python/TypeScript), Claude Code CLI (LLM features), go-charts (HTML reports)
- 100+ tests passing across 11 packages, 72%+ coverage
- Validated on multi-language codebases (Go, Python, TypeScript)
- 25 phases total (v1: 5, v0.0.2: 7, v0.0.3: 5, v0.0.4: 8), 54 plans completed
- All 7 analysis categories operational with 58 research citations
- C7 now uses 5 MECE metrics with parallel execution (M1-M5)
- Complete citation system: docs/CITATION-GUIDE.md + 58 citations across C1-C7

**Research Foundation:**
- Borg et al. (2026): Code Health metrics predict maintainability
- RepoGraph (Zhang et al., 2024): Graph-based architectural metrics correlate with agent task success
- SWE-bench (Jimenez et al., 2024): Test coverage correlates with agent task completion (47%)
- CrossCodeEval: Agents perform better on well-structured codebases

**Use Case:**
Internal tooling to identify which repositories need investment before agent adoption. Teams lack objective metrics to prioritize codebase improvements or track agent-readiness over time.

**Validation Strategy:**
- ✓ v1 validated on this repository (agent-readiness)
- Next: Test on open source Go libraries (kubernetes, prometheus, etc.)
- Tune scoring thresholds based on real-world results

**Target Users:**
- Engineering leaders prioritizing investment
- Developers wanting specific improvement guidance
- Platform engineers enforcing quality standards in CI

## Constraints

- **Philosophy**: KISS + TDD, simplicity is king — avoid over-engineering even with expanded scope
- **Tech stack**: Go for core, leverage Tree-sitter for multi-language parsing
- **Performance**: <30s for 50k LOC repos (maintained from v1), C7 excluded from timing (opt-in, high latency)
- **Testing**: TDD approach, validate on diverse real-world codebases (Go, Python, TypeScript)
- **Git requirement**: C5 requires .git directory — fail with clear error if missing (no fallbacks)
- **LLM costs**: C7 uses headless Claude Code — estimate and warn about costs before running
- **Report quality**: HTML reports must be polished, research-backed, technical (not generic AI output)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Start with Go only (v1) | Get one language right, validate scoring model before expanding | ✓ Good - Focused execution, clean architecture |
| Use weighted composite score (v1) | Research shows different metrics have different predictive power | ✓ Good - Meaningful scores that predict agent readiness |
| Focus on C1, C3, C6 first (v1) | Structural quality and testing are highest-impact, measurable categories | ✓ Good - Complete analysis foundation |
| KISS over frameworks (v1) | Fast iteration, easier to maintain, lower barrier to contribution | ✓ Good - Clean codebase even at 21k LOC |
| Test on real repos (v1) | Synthetic tests won't reveal threshold accuracy issues | ✓ Good - Tool validated on this codebase itself |
| Piecewise linear interpolation (v1) | Simple, predictable, configurable scoring | ✓ Good - Easy to tune and explain |
| Parallel analyzer execution (v1) | Reduce wall-clock time for large codebases | ✓ Good - Performance meets <30s requirement |
| Tree-sitter for Python/TypeScript (v0.0.2) | Language-agnostic parsing without runtime dependencies | ✓ Good - Multi-language without embedded interpreters (requires CGO) |
| Native git CLI for C5 (v0.0.2) | 10-100x faster than go-git for log parsing | ✓ Good - Temporal analysis completes in seconds |
| Anthropic SDK for LLM features (v0.0.2) | Single provider, Haiku for cost efficiency | ⚠️ Replaced - Migrated to Claude Code CLI in v0.0.3 |
| Tiered execution model (v0.0.2) | Free/fast default, opt-in LLM features | ✓ Good - Zero cost for static analysis, user controls LLM spend |
| LLM-as-judge for C7 (v0.0.2) | Genuine agent evaluation vs synthetic metrics | ✓ Good - Most novel and differentiated metric in the space |
| Git worktree isolation (v0.0.2) | Safe agent execution without modifying user's working tree | ✓ Good - C7 runs in isolated workspace |
| Claude Code CLI for LLM features (v0.0.3) | Unified auth via CLI, no API key management | ✓ Good - Zero-config LLM features for CLI users |
| Auto-enable LLM when CLI available (v0.0.3) | Reduce friction for users with Claude CLI installed | ✓ Good - LLM features work out of the box |
| shared/ subpackage for analyzer utilities (v0.0.3) | Resolve import cycles when reorganizing analyzers | ✓ Good - Clean architecture, no cycles |
| HTML5 details/summary for expandables (v0.0.3) | CSS-only expand/collapse, progressive enhancement | ✓ Good - Works without JavaScript |

## Current Milestone: v0.0.5 C7 Scoring Bug Fix

**Goal:** Fix M2, M3, M4 metrics returning 0/10 and establish debug infrastructure for C7 validation.

**Target deliverables:**
- Working M2/M3/M4 scoring (non-zero scores for valid responses)
- Debug mode for response inspection
- Test coverage for scoring heuristics
- Documented root cause and resolution

---
*Last updated: 2026-02-06 after v0.0.5 milestone started*
