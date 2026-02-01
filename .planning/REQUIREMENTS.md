# Requirements: Agent Readiness Score (ARS)

**Defined:** 2026-02-01
**Core Value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## v2 Requirements

Requirements for v2 milestone: Complete Analysis Framework (multi-language + all 7 categories).

### Multi-Language Foundation

- [ ] **LANG-01**: Detect Python projects (setup.py, pyproject.toml, requirements.txt, .py files)
- [ ] **LANG-02**: Detect TypeScript projects (tsconfig.json, package.json with typescript dep, .ts files)
- [ ] **LANG-03**: Parse Python files with Tree-sitter (tree-sitter-python)
- [ ] **LANG-04**: Parse TypeScript files with Tree-sitter (tree-sitter-typescript)
- [ ] **LANG-05**: Unified AnalysisTarget interface abstracts Go/Python/TypeScript differences
- [ ] **LANG-06**: Handle mixed-language repositories (analyze each language separately)
- [ ] **LANG-07**: Tree-sitter memory management (explicit Close() on parsers)

### C2: Semantic Explicitness & Type Safety

**Go:**
- [ ] **C2-GO-01**: Measure interface{}/any usage rate
- [ ] **C2-GO-02**: Check naming consistency (CamelCase for exports, camelCase for unexports)
- [ ] **C2-GO-03**: Detect magic numbers (numeric/string literals vs named constants)
- [ ] **C2-GO-04**: Analyze nil safety patterns (nil checks before dereference)

**Python:**
- [ ] **C2-PY-01**: Measure type annotation coverage (PEP 484/585 hints on functions/methods)
- [ ] **C2-PY-02**: Check naming consistency (snake_case adherence, PEP 8)
- [ ] **C2-PY-03**: Detect magic numbers
- [ ] **C2-PY-04**: Detect mypy/pyright configuration (strict mode)

**TypeScript:**
- [ ] **C2-TS-01**: Measure type annotation coverage (explicit types vs implicit/any)
- [ ] **C2-TS-02**: Check tsconfig.json strict mode flags
- [ ] **C2-TS-03**: Detect magic numbers
- [ ] **C2-TS-04**: Analyze null safety (optional chaining, nullable types)

### C4: Documentation Quality

**Static metrics:**
- [ ] **C4-01**: Check README.md presence and word count
- [ ] **C4-02**: Calculate inline comment density (% lines with comments)
- [ ] **C4-03**: Measure API documentation coverage (docstrings/JSDoc/godoc for public APIs)
- [ ] **C4-04**: Check CHANGELOG.md presence and last update date
- [ ] **C4-05**: Detect architectural diagrams (docs/ directory or embedded in README)
- [ ] **C4-06**: Detect example code presence (examples/ or README code blocks)
- [ ] **C4-07**: Check CONTRIBUTING.md or onboarding guide presence

**LLM-based content quality:**
- [ ] **C4-08**: Rate README clarity (1-10 scale via LLM judge)
- [ ] **C4-09**: Rate example code quality (runnable, clear, relevant)
- [ ] **C4-10**: Assess documentation completeness (missing critical sections)
- [ ] **C4-11**: Check cross-reference coherence (consistent terminology, valid links)
- [ ] **C4-12**: Implement sampling strategy (50-100 files max to control cost)
- [ ] **C4-13**: Implement prompt caching for repeated LLM calls
- [ ] **C4-14**: Show cost estimation before running LLM analysis

### C5: Temporal & Operational Dynamics

- [ ] **C5-01**: Parse git log with native git CLI (not go-git)
- [ ] **C5-02**: Calculate code churn rate (lines changed per commit, 90-day window)
- [ ] **C5-03**: Detect temporal coupling (files co-changed >70% of time)
- [ ] **C5-04**: Calculate author fragmentation (avg authors per file, 90-day window)
- [ ] **C5-05**: Calculate commit stability (median time between changes)
- [ ] **C5-06**: Calculate hotspot concentration (% changes in top 10% of files)
- [ ] **C5-07**: Fail gracefully if .git directory missing (clear error, not silent skip)
- [ ] **C5-08**: Performance optimization (default to 6-month history window)

### C7: Agent Evaluation (LLM-as-Judge)

- [ ] **C7-01**: Implement headless Claude Code integration (claude -p mode)
- [ ] **C7-02**: Define agent evaluation tasks (understand this code, propose refactor, add feature)
- [ ] **C7-03**: Measure intent clarity (agent's understanding of purpose)
- [ ] **C7-04**: Measure modification confidence (agent's safety assessment for refactoring)
- [ ] **C7-05**: Measure cross-file coherence (pattern consistency detection)
- [ ] **C7-06**: Measure semantic completeness (missing context detection)
- [ ] **C7-07**: Implement sampling strategy (20-50 functions stratified by complexity)
- [ ] **C7-08**: Show cost estimation and get confirmation before running
- [ ] **C7-09**: Handle agent errors and timeouts gracefully
- [ ] **C7-10**: Opt-in flag (--agent-eval or --enable-c7)

### Python-Specific Analysis

- [x] **PY-01**: C1 cyclomatic complexity for Python (via Tree-sitter)
- [x] **PY-02**: C1 function length for Python
- [x] **PY-03**: C1 file size for Python
- [x] **PY-04**: C1 duplication detection for Python (token-based hashing)
- [x] **PY-05**: C3 import graph analysis for Python
- [x] **PY-06**: C3 dead code detection for Python (unreferenced definitions)
- [x] **PY-07**: C3 directory depth for Python
- [x] **PY-08**: C6 test detection (pytest, unittest frameworks)
- [x] **PY-09**: C6 coverage parsing (coverage.py XML output)

### TypeScript-Specific Analysis

- [x] **TS-01**: C1 cyclomatic complexity for TypeScript
- [x] **TS-02**: C1 function length for TypeScript
- [x] **TS-03**: C1 file size for TypeScript
- [x] **TS-04**: C1 duplication detection for TypeScript
- [x] **TS-05**: C3 import graph analysis for TypeScript
- [x] **TS-06**: C3 dead code detection for TypeScript
- [x] **TS-07**: C3 directory depth for TypeScript
- [x] **TS-08**: C6 test detection (Jest, Mocha, Vitest frameworks)
- [x] **TS-09**: C6 coverage parsing (Istanbul/NYC lcov output)

### Scoring & Configuration

- [ ] **SCORE-01**: Update composite score to include all 7 categories with PRD weights
- [ ] **SCORE-02**: Rebalance weights when categories unavailable (C7 opt-out redistributes)
- [ ] **SCORE-03**: Parse .arsrc.yml configuration file
- [ ] **SCORE-04**: Support custom category weights in config
- [ ] **SCORE-05**: Support custom metric thresholds in config
- [ ] **SCORE-06**: Support per-language threshold overrides
- [ ] **SCORE-07**: Support metric enable/disable flags in config
- [ ] **SCORE-08**: Validate config schema with clear error messages

### HTML Reports

- [ ] **HTML-01**: Generate HTML report with html/template
- [ ] **HTML-02**: Radar chart for composite + per-category scores (inline SVG or ECharts)
- [ ] **HTML-03**: Metric breakdown tables with thresholds and actual values
- [ ] **HTML-04**: Research citations for each metric (linked to papers)
- [ ] **HTML-05**: Impact explanations for each metric (why it matters for agents)
- [ ] **HTML-06**: Top 5 recommendations embedded in report
- [ ] **HTML-07**: Trend comparison chart (if --baseline provided)
- [ ] **HTML-08**: Self-contained single file (no external CSS/JS dependencies)
- [ ] **HTML-09**: Polished, technical design (avoid generic AI aesthetic)
- [ ] **HTML-10**: XSS protection (use html/template escaping, never template.HTML with user data)

### CLI & Integration

- [ ] **CLI-01**: Add --enable-c4-llm flag for C4 LLM analysis (opt-in)
- [ ] **CLI-02**: Add --enable-c7 flag for C7 agent evaluation (opt-in)
- [ ] **CLI-03**: Add --output-html flag to generate HTML report
- [ ] **CLI-04**: Add --config flag to specify .arsrc.yml path
- [ ] **CLI-05**: Add --baseline flag for trend comparison
- [ ] **CLI-06**: Show cost estimation for LLM features before execution
- [ ] **CLI-07**: Update --verbose to show per-language analysis details
- [ ] **CLI-08**: Maintain <30s performance for non-LLM analysis (C1-C3, C5-C6)

## v3+ Requirements (Future)

Deferred to post-v2 milestones.

### Incremental Scanning
- **CACHE-01**: Cache analysis results by file content hash
- **CACHE-02**: Only re-analyze changed files
- **CACHE-03**: Persist cache to disk (.ars-cache/)

### Additional Output Formats
- **OUT-01**: Markdown report for PR comments
- **OUT-02**: SARIF format for IDE integration

### CI/CD Integration
- **CI-01**: GitHub Action with PR comment integration
- **CI-02**: GitLab CI template
- **CI-03**: Pre-commit hook support

### Advanced Features
- **MONO-01**: Monorepo per-package scoring with workspace detection
- **TREND-01**: Historical tracking database (SQLite)
- **TREND-02**: Web UI dashboard for trends
- **PLUGIN-01**: Plugin SDK for custom metrics

## Out of Scope

| Feature | Reason |
|---------|--------|
| Automated code fixes | Analysis only, never mutations (safety principle) |
| Real-time IDE linting | Batch analysis only (architectural constraint) |
| Competitive benchmarking | Single repo focus (avoid maintaining external database) |
| Language-specific linters integration | Focus on structural metrics, not language idioms |
| Code review automation | Separate concern from agent-readiness assessment |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| LANG-01 | Phase 6 | Pending |
| LANG-02 | Phase 6 | Pending |
| LANG-03 | Phase 6 | Pending |
| LANG-04 | Phase 6 | Pending |
| LANG-05 | Phase 6 | Pending |
| LANG-06 | Phase 6 | Pending |
| LANG-07 | Phase 6 | Pending |
| C2-GO-01 | Phase 6 | Pending |
| C2-GO-02 | Phase 6 | Pending |
| C2-GO-03 | Phase 6 | Pending |
| C2-GO-04 | Phase 6 | Pending |
| C2-PY-01 | Phase 6 | Pending |
| C2-PY-02 | Phase 6 | Pending |
| C2-PY-03 | Phase 6 | Pending |
| C2-PY-04 | Phase 6 | Pending |
| C2-TS-01 | Phase 6 | Pending |
| C2-TS-02 | Phase 6 | Pending |
| C2-TS-03 | Phase 6 | Pending |
| C2-TS-04 | Phase 6 | Pending |
| SCORE-01 | Phase 6 | Pending |
| SCORE-02 | Phase 6 | Pending |
| SCORE-03 | Phase 6 | Pending |
| SCORE-04 | Phase 6 | Pending |
| SCORE-05 | Phase 6 | Pending |
| SCORE-06 | Phase 6 | Pending |
| SCORE-07 | Phase 6 | Pending |
| SCORE-08 | Phase 6 | Pending |
| CLI-04 | Phase 6 | Pending |
| CLI-07 | Phase 6 | Pending |
| CLI-08 | Phase 6 | Pending |
| PY-01 | Phase 7 | Pending |
| PY-02 | Phase 7 | Pending |
| PY-03 | Phase 7 | Pending |
| PY-04 | Phase 7 | Pending |
| PY-05 | Phase 7 | Pending |
| PY-06 | Phase 7 | Pending |
| PY-07 | Phase 7 | Pending |
| PY-08 | Phase 7 | Pending |
| PY-09 | Phase 7 | Pending |
| TS-01 | Phase 7 | Pending |
| TS-02 | Phase 7 | Pending |
| TS-03 | Phase 7 | Pending |
| TS-04 | Phase 7 | Pending |
| TS-05 | Phase 7 | Pending |
| TS-06 | Phase 7 | Pending |
| TS-07 | Phase 7 | Pending |
| TS-08 | Phase 7 | Pending |
| TS-09 | Phase 7 | Pending |
| C5-01 | Phase 8 | Pending |
| C5-02 | Phase 8 | Pending |
| C5-03 | Phase 8 | Pending |
| C5-04 | Phase 8 | Pending |
| C5-05 | Phase 8 | Pending |
| C5-06 | Phase 8 | Pending |
| C5-07 | Phase 8 | Pending |
| C5-08 | Phase 8 | Pending |
| C4-01 | Phase 9 | Pending |
| C4-02 | Phase 9 | Pending |
| C4-03 | Phase 9 | Pending |
| C4-04 | Phase 9 | Pending |
| C4-05 | Phase 9 | Pending |
| C4-06 | Phase 9 | Pending |
| C4-07 | Phase 9 | Pending |
| C4-08 | Phase 9 | Pending |
| C4-09 | Phase 9 | Pending |
| C4-10 | Phase 9 | Pending |
| C4-11 | Phase 9 | Pending |
| C4-12 | Phase 9 | Pending |
| C4-13 | Phase 9 | Pending |
| C4-14 | Phase 9 | Pending |
| HTML-01 | Phase 9 | Pending |
| HTML-02 | Phase 9 | Pending |
| HTML-03 | Phase 9 | Pending |
| HTML-04 | Phase 9 | Pending |
| HTML-05 | Phase 9 | Pending |
| HTML-06 | Phase 9 | Pending |
| HTML-07 | Phase 9 | Pending |
| HTML-08 | Phase 9 | Pending |
| HTML-09 | Phase 9 | Pending |
| HTML-10 | Phase 9 | Pending |
| CLI-01 | Phase 9 | Pending |
| CLI-03 | Phase 9 | Pending |
| CLI-05 | Phase 9 | Pending |
| CLI-06 | Phase 9 | Pending |
| C7-01 | Phase 10 | Pending |
| C7-02 | Phase 10 | Pending |
| C7-03 | Phase 10 | Pending |
| C7-04 | Phase 10 | Pending |
| C7-05 | Phase 10 | Pending |
| C7-06 | Phase 10 | Pending |
| C7-07 | Phase 10 | Pending |
| C7-08 | Phase 10 | Pending |
| C7-09 | Phase 10 | Pending |
| C7-10 | Phase 10 | Pending |
| CLI-02 | Phase 10 | Pending |

**Coverage:**
- v2 requirements: 95 total
- Mapped to phases: 95
- Unmapped: 0

---
*Requirements defined: 2026-02-01*
*Last updated: 2026-02-01 after v2 roadmap creation*
