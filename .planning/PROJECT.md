# Agent Readiness Score (ARS)

## What This Is

A Go CLI tool that analyzes codebases and produces a 1-10 composite score measuring how well a repository supports AI coding agent workflows. Team leads use ARS to assess whether their codebases are ready for safe agent adoption, prioritize refactoring work, track improvements over time, and justify investment in code quality.

## Core Value

Team leads get an objective, research-backed assessment of whether their codebases are safe for AI agent adoption, with specific actionable improvements ranked by impact.

## Requirements

### Validated

(None yet — ship to validate)

### Active

#### CLI & Core Infrastructure
- [ ] CLI accepts repository path and options via `ars scan [path]`
- [ ] Auto-detect language from file extensions and project structure
- [ ] Exit codes: 0 (success), 1 (error)
- [ ] Output formatted results to terminal (text)

#### Language Support
- [ ] Analyze Python codebases (C1, C3, C6 metrics)
- [ ] Analyze Go codebases (C1, C3, C6 metrics)
- [ ] Analyze TypeScript codebases (C1, C3, C6 metrics)

#### C1: Code Health & Structural Integrity (25% weight)
- [ ] Measure cyclomatic complexity (McCabe) per function
- [ ] Measure cognitive complexity (nested control flow depth)
- [ ] Measure function length (lines per function)
- [ ] Measure file size (lines per file)
- [ ] Measure coupling - afferent (incoming dependencies)
- [ ] Measure coupling - efferent (outgoing dependencies)
- [ ] Measure cohesion (LCOM - Lack of Cohesion of Methods)
- [ ] Detect code duplication (duplicated code blocks %)

#### C3: Architectural Navigability (20% weight)
- [ ] Measure directory depth (max nesting level)
- [ ] Measure module fanout (avg references per module)
- [ ] Detect circular dependencies (import cycles)
- [ ] Check entry point clarity (main/index files present)
- [ ] Measure import path complexity (relative path segments)
- [ ] Detect dead code ratio (unreferenced exports %)
- [ ] Check architectural layering violations

#### C6: Testing & Verifiability Infrastructure (15% weight)
- [ ] Estimate test coverage (% lines/branches - static approximation)
- [ ] Measure test-to-code ratio (test LOC / source LOC)
- [ ] Estimate test isolation (% tests with external dependencies)
- [ ] Measure assertion density (assertions per test)
- [ ] Check test naming conventions (`should_*`, `it_*`, `test_*`)
- [ ] Estimate fast test ratio (% tests likely <1s - heuristic)
- [ ] Measure fixture complexity (lines in setup/teardown)

#### Scoring & Recommendations
- [ ] Calculate weighted composite score: 0.25×C1 + 0.20×C3 + 0.15×C6
- [ ] Normalize to 1-10 scale with piecewise linear interpolation
- [ ] Assign tier rating: Agent-Ready (8-10), Agent-Assisted (6-7.9), Agent-Limited (4-5.9), Agent-Hostile (1-3.9)
- [ ] Generate Top 5 improvement recommendations ranked by (Max Gain × Ease × Category Weight)
- [ ] Display per-category scores and overall composite score

### Out of Scope

**Deferred to Phase 2:**
- C2 (Semantic Explicitness & Type Safety) category — full type annotation analysis
- C4 (Documentation Quality) category — README, comments, API docs analysis
- C5 (Temporal & Operational Dynamics) category — requires git log parsing, churn analysis
- C7 (Agent Evaluation - LLM Judge) category — subjective LLM-based scoring
- Java language support — needed but deferred
- JavaScript (non-TypeScript) support — defer until Java complete
- HTML/JSON/Markdown output formats — terminal text only in v1
- `--threshold` flag for CI gating — no CI integration yet
- `--baseline` comparison mode for tracking — no persistence in v1
- Coverage report parsing (lcov, cobertura, JaCoCo) — use static approximation instead
- Dynamic test execution for accurate timing — use static heuristics

**Explicitly Excluded:**
- Cloud-hosted service — CLI tool only, users run locally
- Automated code fixes — analysis only, no mutations
- Real-time IDE linting — batch analysis only
- Competitive benchmarking — single repo focus

## Context

**Research Foundation:**
- Detailed PRD exists in `.specs/prd.md` with academic backing
- Based on Borg et al. (2026) Code Health framework, RepoGraph (Zhang 2024), SWE-bench (Jimenez 2024), CrossCodeEval (Ding 2023)
- 7-category MECE taxonomy defined with specific metrics and thresholds
- v1 implements 3 of 7 categories (C1, C3, C6) to ship faster and validate approach

**Organizational Context:**
- Multiple teams with Python, Go, TypeScript, and Java codebases (Java deferred to Phase 2)
- Team leads are primary users
- Need to assess codebase readiness before agent adoption rollout
- Will use scores to: gate agent adoption, prioritize refactoring, track progress, justify investment

**Technical Context:**
- Static analysis only in v1 (no running tests, no git operations beyond reading file tree)
- Must handle typical repos (<100k LOC) efficiently
- Cross-platform distribution requirement (Linux, macOS, Windows)

## Constraints

- **Tech Stack**: Go — single binary distribution, cross-platform, strong CLI/parser ecosystem
- **Timeline**: Target 1-2 weeks for v1, but quality over speed (full Phase 1 scope may take 3-4 weeks)
- **Performance**: Should scan typical repo (<100k LOC) in under 30 seconds
- **Languages**: Must support Python, Go, TypeScript in v1 (cannot ship without all three)
- **Output**: Terminal text only (no file I/O, no HTML generation in v1)
- **Dependencies**: Minimize external dependencies; prefer stdlib where possible

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go for implementation | Single binary, cross-platform, strong stdlib, good for CLI tools | — Pending |
| Phase 1: C1+C3+C6 only (3 of 7 categories) | Ship faster, validate approach with highest-impact metrics, defer C2/C4/C5/C7 | — Pending |
| Static analysis only in v1 | Avoid complexity of running coverage tools, git forensics; use heuristics where needed | — Pending |
| Terminal text output only | Immediate value, simple implementation, defer HTML/JSON formatting | — Pending |
| Normalize partial category scores | C1+C3+C6 = 60% of full model; scale to 1-10 for v1, add categories in v2 | — Pending |

---
*Last updated: 2026-01-31 after initialization*
