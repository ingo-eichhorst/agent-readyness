# Requirements: Agent Readiness Score (ARS)

**Defined:** 2026-01-31
**Core Value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## v1 Requirements

Requirements for initial release (Go language only, C1/C3/C6 categories).

### Foundation

- [x] **FOUND-01**: CLI accepts directory path as primary argument
- [x] **FOUND-02**: Auto-detects Go projects (go.mod presence, .go files)
- [x] **FOUND-03**: Provides `--help` flag with usage documentation
- [x] **FOUND-04**: Provides `--version` flag showing current version
- [x] **FOUND-05**: Clear error messages with actionable guidance when inputs invalid
- [x] **FOUND-06**: Exit codes: 0 (success), 1 (error), 2 (below threshold)
- [ ] **FOUND-07**: Handles edge cases (symlinks, syntax errors, Unicode paths)
- [x] **FOUND-08**: Excludes vendor directories and generated code automatically
- [x] **FOUND-09**: Properly classifies Go files (_test.go, build tags, platform-specific)

### C1: Code Health & Structural Integrity

- [x] **C1-01**: Calculates cyclomatic complexity per function (avg and max)
- [x] **C1-02**: Measures function length in lines (avg and max)
- [x] **C1-03**: Measures file size in lines (avg and max)
- [x] **C1-04**: Calculates afferent coupling (incoming dependencies per module)
- [x] **C1-05**: Calculates efferent coupling (outgoing dependencies per module)
- [x] **C1-06**: Detects duplicated code blocks and reports duplication rate (%)

### C3: Architectural Navigability

- [x] **C3-01**: Measures directory depth (max nesting level)
- [x] **C3-02**: Calculates module fanout (avg references per module)
- [x] **C3-03**: Detects circular dependencies (import cycle count)
- [x] **C3-04**: Measures import path complexity (avg relative path segments)
- [x] **C3-05**: Identifies dead code (unreferenced exported functions/types)

### C6: Testing & Verifiability Infrastructure

- [x] **C6-01**: Detects test files (*_test.go pattern)
- [x] **C6-02**: Calculates test-to-code ratio (test LOC / source LOC)
- [x] **C6-03**: Parses coverage reports if present (lcov, cobertura formats)
- [x] **C6-04**: Identifies test isolation (% tests with external dependencies)
- [x] **C6-05**: Calculates assertion density (assertions per test function)

### Scoring Model

- [ ] **SCORE-01**: Generates per-category score (1-10) for C1, C3, C6
- [ ] **SCORE-02**: Calculates composite score using weighted average (C1: 25%, C3: 20%, C6: 15%)
- [ ] **SCORE-03**: Assigns tier rating (Agent-Ready 8-10, Agent-Assisted 6-8, Agent-Limited 4-6, Agent-Hostile 1-4)
- [ ] **SCORE-04**: Uses piecewise linear interpolation between threshold values
- [ ] **SCORE-05**: Provides verbose mode showing per-metric breakdown
- [ ] **SCORE-06**: Scoring thresholds are configurable (foundation for tuning)

### Recommendations

- [ ] **REC-01**: Generates Top 5 improvement recommendations
- [ ] **REC-02**: Ranks recommendations by impact (max potential gain x ease x category weight)
- [ ] **REC-03**: Includes estimated score improvement for each recommendation
- [ ] **REC-04**: Provides effort estimate (Low/Medium/High) for each improvement
- [ ] **REC-05**: Frames recommendations in agent-readiness terms

### Output & CLI

- [ ] **OUT-01**: Terminal text output with ANSI colors for readability
- [ ] **OUT-02**: Summary section showing composite score and tier
- [ ] **OUT-03**: Category breakdown section with individual scores
- [ ] **OUT-04**: Recommendations section with Top 5 improvements
- [ ] **OUT-05**: Optional `--threshold X` flag for CI gating (exit 2 if score < X)
- [ ] **OUT-06**: Optional `--verbose` flag for detailed metric breakdown
- [ ] **OUT-07**: Performance completes in <30s for 50k LOC repos
- [ ] **OUT-08**: Progress indicators for long-running scans

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Multi-Language Support

- **LANG-01**: Python codebase analysis with similar metrics
- **LANG-02**: TypeScript codebase analysis with similar metrics
- **LANG-03**: Mixed-language repository support

### Additional Categories

- **C2-01**: Semantic Explicitness & Type Safety analysis
- **C4-01**: Documentation Quality analysis
- **C5-01**: Temporal & Operational Dynamics (git forensics)

### Output Formats

- **FMT-01**: JSON output for machine consumption
- **FMT-02**: HTML report with charts and graphs
- **FMT-03**: Markdown report for PR descriptions

### Advanced Features

- **ADV-01**: Baseline comparison (--baseline flag to compare against previous scan)
- **ADV-02**: Historical trend tracking
- **ADV-03**: Incremental scanning (only re-analyze changed files)
- **ADV-04**: Configurable weights via .arsrc.yml

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| C7 LLM Judge evaluation | High cost, adds latency/non-determinism, needs validation |
| Auto-fix / code mutations | Dangerous without review, violates diagnostic-only principle |
| GitHub Action integration | Delivery mechanism, defer until core is validated |
| VS Code extension | Different interface, validate CLI first |
| Plugin/extension SDK | Public API commitment, avoid in v1 |
| Real-time watch mode | Static analysis not sub-second, false expectations |
| Cross-repo benchmarking | Requires database, comparisons mislead without context |
| Cloud-hosted service | CLI-only tool, users host their data |
| Java/Rust/Swift analyzers | Beyond Phase 1-2 scope |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FOUND-01 | Phase 1 | Complete |
| FOUND-02 | Phase 1 | Complete |
| FOUND-03 | Phase 1 | Complete |
| FOUND-04 | Phase 1 | Complete |
| FOUND-05 | Phase 1 | Complete |
| FOUND-06 | Phase 1 | Complete |
| FOUND-07 | Phase 5 | Pending |
| FOUND-08 | Phase 1 | Complete |
| FOUND-09 | Phase 1 | Complete |
| C1-01 | Phase 2 | Pending |
| C1-02 | Phase 2 | Pending |
| C1-03 | Phase 2 | Pending |
| C1-04 | Phase 2 | Pending |
| C1-05 | Phase 2 | Pending |
| C1-06 | Phase 2 | Pending |
| C3-01 | Phase 2 | Pending |
| C3-02 | Phase 2 | Pending |
| C3-03 | Phase 2 | Pending |
| C3-04 | Phase 2 | Pending |
| C3-05 | Phase 2 | Pending |
| C6-01 | Phase 2 | Pending |
| C6-02 | Phase 2 | Pending |
| C6-03 | Phase 2 | Pending |
| C6-04 | Phase 2 | Pending |
| C6-05 | Phase 2 | Pending |
| SCORE-01 | Phase 3 | Pending |
| SCORE-02 | Phase 3 | Pending |
| SCORE-03 | Phase 3 | Pending |
| SCORE-04 | Phase 3 | Pending |
| SCORE-05 | Phase 3 | Pending |
| SCORE-06 | Phase 3 | Pending |
| REC-01 | Phase 4 | Pending |
| REC-02 | Phase 4 | Pending |
| REC-03 | Phase 4 | Pending |
| REC-04 | Phase 4 | Pending |
| REC-05 | Phase 4 | Pending |
| OUT-01 | Phase 4 | Pending |
| OUT-02 | Phase 4 | Pending |
| OUT-03 | Phase 4 | Pending |
| OUT-04 | Phase 4 | Pending |
| OUT-05 | Phase 4 | Pending |
| OUT-06 | Phase 4 | Pending |
| OUT-07 | Phase 5 | Pending |
| OUT-08 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 44 total
- Mapped to phases: 44
- Unmapped: 0

---
*Requirements defined: 2026-01-31*
*Last updated: 2026-01-31 after roadmap creation*
