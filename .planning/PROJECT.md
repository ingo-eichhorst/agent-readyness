# Agent Readiness Score (ARS)

## What This Is

A CLI tool that analyzes Go codebases and produces a composite score (1-10) measuring how well the repository supports AI agent workflows. ARS evaluates code health, architectural navigability, and testing infrastructure, then generates actionable improvement recommendations ranked by impact.

## Core Value

Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## Requirements

### Validated

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

### Active

(To be defined for next milestone)

### Out of Scope

- Python/TypeScript analyzers — v2
- C2 (Semantic Explicitness), C4 (Documentation), C5 (Temporal Dynamics) — v2
- C7 (LLM Judge) — Future, high cost
- HTML reports — v2
- GitHub Action — Future
- VS Code extension — Future
- Multi-language repository support — v2
- Incremental scanning / caching — Future
- Automated code fixes — analysis only, never mutations

## Context

**Current State (v1 shipped 2026-02-01):**
- 7,508 LOC Go
- Tech stack: Go 1.24, cobra CLI, go/packages parser, gocyclo
- 81 tests passing, 85%+ coverage
- Validated on this codebase (scores 8.1/10 Agent-Ready)
- 5 phases, 16 plans completed

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

- **Timeline**: ASAP — ship working v1 quickly
- **Philosophy**: KISS + TDD, simplicity is king
- **Tech stack**: Go, no heavy frameworks
- **Performance**: Should handle large repos (10k+ files) in reasonable time (<5 min)
- **Testing**: TDD approach, test on real codebases
- **Parsing**: Use simple, reliable parsers (avoid over-engineering)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Start with Go only | Get one language right, validate scoring model before expanding | ✓ Good - Focused execution, clean architecture |
| Use weighted composite score | Research shows different metrics have different predictive power | ✓ Good - Meaningful scores that predict agent readiness |
| Focus on C1, C3, C6 first | Structural quality and testing are highest-impact, measurable categories | ✓ Good - Complete analysis foundation |
| KISS over frameworks | Fast iteration, easier to maintain, lower barrier to contribution | ✓ Good - 7,508 LOC with full functionality |
| Test on real repos | Synthetic tests won't reveal threshold accuracy issues | ✓ Good - Tool validated on this codebase itself |
| Piecewise linear interpolation | Simple, predictable, configurable scoring | ✓ Good - Easy to tune and explain |
| Parallel analyzer execution | Reduce wall-clock time for large codebases | ✓ Good - Performance meets <30s requirement |

---
*Last updated: 2026-02-01 after v1 milestone completion*
