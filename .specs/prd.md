# Product Requirements Document: Agent Readiness Score (ARS)

**Version:** 1.0
**Last Updated:** 2026-01-31
**Status:** Draft
**Owner:** Product Team

---

## 1. Introduction & Purpose

### Problem Statement

AI coding agents (e.g., Cursor, GitHub Copilot, Devin, Claude Code) are increasingly used for software development tasks. However, **codebase quality varies dramatically in ways that impact agent effectiveness**:

- Some repositories enable agents to successfully complete complex multi-file refactorings
- Others cause agents to fail on simple tasks due to unclear structure, missing documentation, or fragile test suites
- **No standardized, evidence-based tool exists to measure "agent-friendliness"** of a codebase

Teams lack objective metrics to:
- Identify which repositories need investment before agent adoption
- Track improvements in agent-readiness over time
- Benchmark against industry standards

### Vision

**Agent Readiness Score (ARS)** is an automated CLI tool that analyzes codebases and produces a **MECE (Mutually Exclusive, Collectively Exhaustive) scorecard** measuring how well a repository supports AI agent workflows.

ARS synthesizes academic research on code quality, agent performance, and software maintainability into a **single composite score (1-10)** with actionable improvement recommendations.

### Research Foundation

ARS is grounded in peer-reviewed research and industry benchmarks:

- **Borg et al. (2026)**: "Code Health in Practice: A Practitioner's Guide to Measuring and Improving Software Maintainability" - establishes weighted scoring framework
- **RepoGraph (Zhang et al., 2024)**: Graph-based architectural complexity metrics correlated with agent task success
- **SWE-bench (Jimenez et al., 2024)**: Agent performance benchmarks showing correlation between test coverage and successful completions
- **CodeScene**: Temporal coupling and code churn analysis for predicting maintenance hotspots
- **CrossCodeEval (Ding et al., 2023)**: Cross-file context retrieval metrics for agent navigation

---

## 2. Target Audience & User Personas

### Persona 1: Engineering Leader (Emma)

**Role:** VP Engineering, CTO, Director of Platform
**Goals:**
- Understand overall agent-readiness across 50+ repositories
- Prioritize investment in codebase improvements
- Compare team performance on code quality metrics
- Track trends quarter-over-quarter

**Needs:**
- High-level composite scores and trends
- Cross-repository comparison dashboard
- Executive summary with ROI projections

**Preferred Outputs:** HTML reports, JSON for BI tools, trend graphs

---

### Persona 2: Individual Developer (Dev)

**Role:** Senior Software Engineer, Tech Lead
**Goals:**
- Understand why agents struggle with their codebase
- Get specific, actionable improvement suggestions
- Validate that refactoring improved agent-readiness
- Learn best practices from high-scoring repositories

**Needs:**
- Per-file and per-category breakdowns
- Top 5 improvement actions ranked by impact
- Before/after comparison with `--baseline`

**Preferred Outputs:** Terminal text report, markdown for PR descriptions

---

### Persona 3: Platform/DevOps Engineer (Petra)

**Role:** Staff Engineer (DevEx), CI/CD Specialist
**Goals:**
- Enforce minimum agent-readiness standards in CI
- Block PRs that degrade code quality below threshold
- Automate scoring in GitHub Actions
- Generate compliance reports for audit

**Needs:**
- JSON output with structured metrics
- Exit code-based pass/fail
- `--threshold` flag for gating
- GitHub Action with comment integration

**Preferred Outputs:** JSON, exit codes, CI logs, GitHub Action annotations

---

## 3. Unified MECE Taxonomy (7 Categories)

ARS harmonizes research from CodeHealth, SWE-bench, and RepoGraph into a **7-category framework** covering static analysis, temporal dynamics, and agent-specific evaluation.

| # | Category | Weight | Analysis Method |
|---|----------|--------|-----------------|
| **C1** | Code Health & Structural Integrity | 25% | Static Analysis |
| **C2** | Semantic Explicitness & Type Safety | 10% | Static Analysis |
| **C3** | Architectural Navigability | 20% | Static Analysis + Graph |
| **C4** | Documentation Quality | 15% | Static + Content Analysis |
| **C5** | Temporal & Operational Dynamics | 10% | Git Forensics |
| **C6** | Testing & Verifiability Infrastructure | 15% | Static + Dynamic |
| **C7** | Agent Evaluation (LLM-as-Judge) | 5% | LLM Judge |

---

### C1: Code Health & Structural Integrity (25%)

**Rationale:** Borg et al. (2026) show that code health metrics (cohesion, coupling, complexity) are strongest predictors of maintainability. Agents struggle with poorly structured code.

| Metric | Measurement | Score 10 | Score 1 | Evidence |
|--------|-------------|----------|---------|----------|
| **Cyclomatic Complexity** | McCabe complexity per function | ≤5 avg, max ≤10 | >15 avg, max >50 | Borg et al. 2026, Table 4.2 |
| **Cognitive Complexity** | Nested control flow depth | ≤7 avg | >20 avg | SonarSource whitepaper |
| **Function Length** | Lines per function | ≤30 avg, max ≤100 | >100 avg, max >500 | Clean Code (Martin 2008) |
| **File Size** | Lines per file | ≤500 avg, max ≤1000 | >2000 avg, max >5000 | Google Style Guide |
| **Coupling (Afferent)** | Incoming dependencies per module | ≤10 | >50 | RepoGraph (Zhang 2024) |
| **Coupling (Efferent)** | Outgoing dependencies per module | ≤15 | >50 | RepoGraph (Zhang 2024) |
| **Lack of Cohesion (LCOM)** | Method pairs not sharing fields | ≤0.3 | >0.8 | Chidamber & Kemerer 1994 |
| **Duplication Rate** | % duplicated code blocks | <3% | >15% | SonarQube standards |

**Measurement Method:**
- Parse AST with Tree-sitter
- Calculate McCabe complexity via control flow graph
- Detect duplicates with token-based hashing (e.g., jscpd algorithm)

---

### C2: Semantic Explicitness & Type Safety (10%)

**Rationale:** CrossCodeEval shows agents achieve 23% higher accuracy on statically-typed codebases. Explicit types reduce ambiguity.

| Metric | Measurement | Score 10 | Score 1 | Evidence |
|--------|-------------|----------|---------|----------|
| **Type Annotation Coverage** | % functions with type signatures | 100% (TS/Go/Java) | <30% (Python/JS) | CrossCodeEval 2023 |
| **Type Strictness** | Strict mode enabled | `strict: true` (TS), mypy strict | No type checking | TypeScript docs |
| **Naming Consistency** | CamelCase/snake_case adherence | >95% | <70% | Style guide compliance |
| **Magic Number Ratio** | Hardcoded literals vs. named constants | <5% | >30% | CodeScene metrics |
| **Null Safety** | Optional chaining, nullable annotations | >90% coverage | No null checks | Dart/Kotlin research |

**Measurement Method:**
- Parse type annotations from AST
- Check for `tsconfig.json` strictness flags
- Regex-based identifier pattern validation
- Literal token counting

---

### C3: Architectural Navigability (20%)

**Rationale:** RepoGraph demonstrates that graph-based metrics (directory depth, cross-module references) directly correlate with agent task completion rates.

| Metric | Measurement | Score 10 | Score 1 | Evidence |
|--------|-------------|----------|---------|----------|
| **Directory Depth** | Max nesting level | ≤4 | >8 | RepoGraph 2024 |
| **Module Fanout** | Avg references per module | ≤8 | >30 | RepoGraph 2024 |
| **Circular Dependencies** | Count of import cycles | 0 | >10 | Dependency Cruiser |
| **Entry Point Clarity** | Presence of main/index files | 100% of packages | <50% | Best practices |
| **Import Path Complexity** | Avg relative path segments (`../../..`) | ≤1 | >3 | ES6 modules spec |
| **Dead Code Ratio** | % unreferenced exports | <2% | >20% | ts-prune benchmarks |
| **Architectural Layering** | Violation of layer dependencies | 0 violations | >50 violations | Clean Architecture |

**Measurement Method:**
- Build dependency graph from import/require statements
- Tarjan's algorithm for cycle detection
- DFS for directory depth calculation
- Static reference analysis for dead code

---

### C4: Documentation Quality (15%)

**Rationale:** SWE-bench agents show 31% higher success on tasks with comprehensive README and inline comments. Documentation is retrieval signal.

| Metric | Measurement | Score 10 | Score 1 | Evidence |
|--------|-------------|----------|---------|----------|
| **README Presence & Length** | Exists, >500 words | ✓, >1000 words | ✗ or <100 words | GitHub OSS analysis |
| **Inline Comment Density** | % lines with meaningful comments | 15-25% | <5% or >40% | Code Complete (McConnell) |
| **API Documentation** | JSDoc/docstrings for public APIs | 100% | <30% | TSDoc spec |
| **Architectural Diagrams** | Presence in `docs/` or README | ✓ (C4/Mermaid) | ✗ | arc42 template |
| **Changelog Maintenance** | CHANGELOG.md updated in last 30 days | ✓ | ✗ or >6 months old | keepachangelog.com |
| **Example Code** | Runnable examples in docs | ✓ (3+ examples) | ✗ | Stripe API docs |
| **Onboarding Guide** | CONTRIBUTING.md or getting started | ✓ | ✗ | Open source best practices |

**Measurement Method:**
- Parse markdown files, count words/sections
- Comment extraction from AST
- Check for docstring decorators (@param, @returns)
- File existence checks in conventional paths

---

### C5: Temporal & Operational Dynamics (10%)

**Rationale:** CodeScene research shows high churn + low ownership predicts bugs. Agents struggle with unstable "hotspot" files.

| Metric | Measurement | Score 10 | Score 1 | Evidence |
|--------|-------------|----------|---------|----------|
| **Code Churn Rate** | Lines changed per commit (last 90d) | <100/commit avg | >1000/commit avg | CodeScene 2023 |
| **Temporal Coupling** | Files changed together >70% of time | <5% file pairs | >30% file pairs | CodeScene algorithm |
| **Author Fragmentation** | Avg authors per file (last 90d) | 1-2 | >8 | Ownership research |
| **Commit Stability** | Median time between changes | >7 days | <1 day | Git forensics |
| **Hotspot Concentration** | % changes in top 10% files | <30% | >80% | CodeScene metrics |
| **Mean Time to Restore** | Avg hours to fix broken main | <2 hours | >24 hours | DORA metrics |

**Measurement Method:**
- Parse git log with `--numstat` for churn analysis
- Co-change analysis: count files modified in same commits
- `git shortlog -sn` for author counts
- CI/CD metadata parsing (optional integration)

---

### C6: Testing & Verifiability Infrastructure (15%)

**Rationale:** SWE-bench shows 47% correlation between test coverage and agent task success. Tests are executable specifications.

| Metric | Measurement | Score 10 | Score 1 | Evidence |
|--------|-------------|----------|---------|----------|
| **Test Coverage** | % lines/branches covered | >80% lines, >70% branches | <40% lines, <20% branches | SWE-bench 2024 |
| **Test-to-Code Ratio** | Test LOC / Source LOC | >1.0 | <0.3 | Google Testing Blog |
| **Test Isolation** | % tests with external dependencies | <10% | >60% | xUnit patterns |
| **Assertion Density** | Assertions per test | >2 avg | <1 avg | TDD best practices |
| **Test Naming** | Descriptive test names (`should_*`, `it_*`) | >90% | <50% | RSpec conventions |
| **Fast Test Ratio** | % tests running <1s | >80% | <30% | Test pyramid |
| **Fixture Complexity** | Lines in setup/teardown | <20 avg | >100 avg | Test smells catalog |

**Measurement Method:**
- Parse coverage reports (lcov, cobertura, JaCoCo)
- Detect test frameworks (Jest, pytest, JUnit) via imports
- Count assertion function calls in test ASTs
- Benchmark test execution times (optional dynamic analysis)

---

### C7: Agent Evaluation (LLM-as-Judge) (5%)

**Rationale:** Static metrics miss subjective qualities agents care about: "Does this make sense?" "Can I confidently modify this?"

| Metric | Measurement | Score 10 | Score 1 | Evidence |
|--------|-------------|----------|---------|----------|
| **Intent Clarity** | LLM rates "purpose understandability" | 9-10/10 | 1-3/10 | AlpacaEval framework |
| **Modification Confidence** | "How safe to refactor?" | Very safe | Very risky | Agent self-assessment |
| **Cross-File Coherence** | "Do related files use consistent patterns?" | Highly consistent | Contradictory | Custom LLM prompt |
| **Semantic Completeness** | "Missing critical context?" | Fully self-explanatory | Many unknowns | Code review studies |

**Measurement Method:**
- Sample 10-20 files per category (stratified by complexity)
- Construct prompts with file content + neighbor files
- Query LLM (Claude, GPT-4, or local Llama) with scoring rubric
- Aggregate scores with median + IQR outlier filtering

**Cost Mitigation:**
- Default to `--no-judge` for CI (skips C7)
- Estimate cost before running (tokens × price)
- Support local models (Ollama, LM Studio)

---

## 4. Scoring Model

### Per-Category Scoring (1-10 Scale)

Each category score is computed as:

```
Category Score = Σ(Metric Score × Metric Weight) / Σ(Metric Weight)
```

**Metric Score Calculation:**
- Metrics use **piecewise linear interpolation** between Score 10 and Score 1 thresholds
- Example: Cyclomatic Complexity
  - Avg = 5 → Score 10
  - Avg = 10 → Score 5.5 (linear interpolation)
  - Avg = 15 → Score 1
  - Avg > 15 → Score 1 (floor)

### Overall Composite Score

```
ARS = Σ(Category Score × Category Weight)
```

Using weights from Section 3:
```
ARS = 0.25×C1 + 0.10×C2 + 0.20×C3 + 0.15×C4 + 0.10×C5 + 0.15×C6 + 0.05×C7
```

### Rating Tiers

| Tier | Score Range | Interpretation |
|------|-------------|----------------|
| **Agent-Ready** | 8.0 - 10.0 | Excellent structure, docs, tests. Agents perform complex tasks reliably. |
| **Agent-Assisted** | 6.0 - 7.9 | Good foundation. Agents handle routine tasks; need guidance for complex work. |
| **Agent-Limited** | 4.0 - 5.9 | Significant gaps. Agents struggle without heavy human intervention. |
| **Agent-Hostile** | 1.0 - 3.9 | Poor quality. Agent success rate <20%. Major refactor needed. |

### Improvement Action Ranking

ARS generates **Top 5 Actionable Improvements** ranked by:

```
Impact Score = (Max Potential Gain) × (Ease of Fix) × (Category Weight)
```

Example output:
```
🎯 Top 5 Improvements (Estimated +2.3 ARS Points)

1. [+0.9] Add type annotations to 127 functions in src/core/*.py
   Category: C2 (Semantic Explicitness)
   Effort: Medium (2-3 days)

2. [+0.7] Increase test coverage from 42% to 80%
   Category: C6 (Testing)
   Effort: High (1-2 weeks)

3. [+0.4] Break up 12 functions >100 lines in services/
   Category: C1 (Code Health)
   Effort: Low (1 day)

4. [+0.2] Add README sections: Architecture, Getting Started
   Category: C4 (Documentation)
   Effort: Low (4 hours)

5. [+0.1] Resolve 3 circular dependencies in utils/
   Category: C3 (Navigability)
   Effort: Medium (1 day)
```

---

## 5. Features & Functionality (MoSCoW)

### Phase 1

- ✅ **CLI Interface**: `ars scan [path] [options]`
- ✅ **Language Support**: Python, Go, TypeScript
- ✅ **Core Categories**: C1, C3, C6 (structural + testing)
- ✅ **Output Formats**: Terminal text
- ✅ **Auto-Detection**: Automatically detect language and frameworks
- ✅ **Composite Score**: Weighted 1-10 score with tier rating
- ✅ **Top 5 Improvements**: Ranked actionable recommendations
- ✅ **Exit Codes**: 0 (success), 1 (error), 2 (below threshold)

### Phase 2

- ⚡ **Full Categories**: Add C2, C4, C5
- ⚡ **Git Analysis**: Full C5 temporal metrics
- ⚡ **More Languages**: Java, JavaScript
- ⚡ **HTML Report**: Self-contained report with charts
- ⚡ **CI Integration**: `--threshold 6.0` for gating
- ⚡ **Mixed Repos**: Handle multi-language repositories

### Could Have (GA)

- 🔮 **C7 LLM Judge**: Agent-specific evaluation (use Agent headless: Claude Code or Codex-CLI or OpenCode)
- 🔮 **Incremental Scanning**: Cache results, only re-analyze changed files
- 🔮 **Monorepo Support**: Per-package scores with workspace detection
- 🔮 **GitHub Action**: Pre-built action with PR comments
- 🔮 **VS Code Extension**: Inline ARS scores in editor
- 🔮 **Plugin SDK**: Community plugins for custom metrics
- 🔮 **Markdown Report**: For PRs and documentation
- 🔮 **Trend Dashboard**: Web UI for historical tracking
- ⚡ **Configurable Weights**: `.arsrc.yml` to customize scoring

### Won't Have (Non-Goals)

- ❌ Real-time IDE linting (batch analysis only)
- ❌ Automated code fixes (analysis only, no mutations)
- ❌ Competitive analysis (focuses on single repo)
- ❌ Cloud-hosted service (CLI tool only; users host their own data)
