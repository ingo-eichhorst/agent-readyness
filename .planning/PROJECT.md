# Agent Readiness Score (ARS)

## What This Is

A CLI tool that analyzes Go codebases and produces a composite score (1-10) measuring how well the repository supports AI agent workflows. ARS evaluates code health, architectural navigability, and testing infrastructure, then generates actionable improvement recommendations ranked by impact.

## Core Value

Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] CLI accepts directory path and scans Go codebase
- [ ] Auto-detects Go projects (go.mod, .go files)
- [ ] C1: Code Health analysis (cyclomatic complexity, function length, file size, coupling, duplication)
- [ ] C3: Architectural Navigability analysis (directory depth, module fanout, circular dependencies, import complexity, dead code)
- [ ] C6: Testing Infrastructure analysis (test coverage, test-to-code ratio, test isolation, assertion density)
- [ ] Composite score calculation using weighted average (C1: 25%, C3: 20%, C6: 15%)
- [ ] Per-category scores with metric breakdowns
- [ ] Top 5 improvement recommendations ranked by impact
- [ ] Terminal text output with tier rating (Agent-Ready, Agent-Assisted, Agent-Limited, Agent-Hostile)
- [ ] Exit codes: 0 (success), 1 (error), 2 (below threshold if --threshold specified)
- [ ] Usage: `ars scan <directory>`

### Out of Scope

- Python/TypeScript analyzers — Phase 2
- C2 (Semantic Explicitness), C4 (Documentation), C5 (Temporal Dynamics) — Phase 2
- C7 (LLM Judge) — Future, high cost
- HTML reports — Phase 2
- JSON output — Phase 2
- GitHub Action — Future
- VS Code extension — Future
- Multi-language repository support — Phase 2
- Incremental scanning / caching — Future
- Automated code fixes — analysis only, never mutations

## Context

**Research Foundation:**
- Borg et al. (2026): Code Health metrics predict maintainability
- RepoGraph (Zhang et al., 2024): Graph-based architectural metrics correlate with agent task success
- SWE-bench (Jimenez et al., 2024): Test coverage correlates with agent task completion (47%)
- CrossCodeEval: Agents perform better on well-structured codebases

**Use Case:**
Internal tooling to identify which repositories need investment before agent adoption. Teams lack objective metrics to prioritize codebase improvements or track agent-readiness over time.

**Test Strategy:**
- First test case: this repository (agent-readiness)
- Validation: open source Go libraries
- Real repos inform metric thresholds and scoring model

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
| Start with Go only | Get one language right, validate scoring model before expanding | — Pending |
| Use weighted composite score | Research shows different metrics have different predictive power | — Pending |
| Focus on C1, C3, C6 first | Structural quality and testing are highest-impact, measurable categories | — Pending |
| KISS over frameworks | Fast iteration, easier to maintain, lower barrier to contribution | — Pending |
| Test on real repos | Synthetic tests won't reveal threshold accuracy issues | — Pending |

---
*Last updated: 2026-01-31 after initialization*
