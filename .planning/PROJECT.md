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

### Active

**Current Milestone: v0.0.3 — Simplification & Polish**

**Goal:** Simplify LLM integration by unifying on Claude Code CLI, add badge generation for visibility, improve HTML report with research-backed expandable descriptions, and reorganize codebase structure.

**Target features (22 requirements across 6 GitHub issues):**
- LLM Integration (#6): Remove Anthropic SDK, use Claude Code CLI for all LLM analysis
- Badge Generation (#5): `--badge` flag generates shields.io markdown URL
- HTML Report (#7): Expandable metric descriptions with research citations
- README (#4): Add status badges (Go Reference, Report Card, License, Release)
- Codebase Organization (#3): Reorganize analyzer/ into category subdirectories
- Testing (#2): Always run tests with coverage flag

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

**Current State (v0.0.2 shipped 2026-02-03):**
- 21,122 LOC Go
- Tech stack: Go 1.24, cobra CLI, Tree-sitter (Python/TypeScript), Anthropic SDK (LLM features), go-charts (HTML reports)
- 100+ tests passing across 11 packages, 85%+ coverage
- Validated on multi-language codebases (Go, Python, TypeScript)
- 12 phases total (v1: 5 phases, v0.0.2: 7 phases), 31 plans completed
- All 7 analysis categories operational (C1-C7)

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
| Anthropic SDK for LLM features (v0.0.2) | Single provider, Haiku for cost efficiency | ✓ Good - Cost-effective C4/C7 analysis |
| Tiered execution model (v0.0.2) | Free/fast default, opt-in LLM features | ✓ Good - Zero cost for static analysis, user controls LLM spend |
| LLM-as-judge for C7 (v0.0.2) | Genuine agent evaluation vs synthetic metrics | ✓ Good - Most novel and differentiated metric in the space |
| Git worktree isolation (v0.0.2) | Safe agent execution without modifying user's working tree | ✓ Good - C7 runs in isolated workspace |

---
*Last updated: 2026-02-03 after v0.0.3 milestone initialization*
