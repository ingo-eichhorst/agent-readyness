# Requirements Archive: v1 Initial Release

**Archived:** 2026-02-01
**Status:** ✅ SHIPPED

This is the archived requirements specification for v1.
For current requirements, see `.planning/REQUIREMENTS.md` (will be created for next milestone).

---

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
- [x] **FOUND-07**: Handles edge cases (symlinks, syntax errors, Unicode paths)
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

- [x] **SCORE-01**: Generates per-category score (1-10) for C1, C3, C6
- [x] **SCORE-02**: Calculates composite score using weighted average (C1: 25%, C3: 20%, C6: 15%)
- [x] **SCORE-03**: Assigns tier rating (Agent-Ready 8-10, Agent-Assisted 6-8, Agent-Limited 4-6, Agent-Hostile 1-4)
- [x] **SCORE-04**: Uses piecewise linear interpolation between threshold values
- [x] **SCORE-05**: Provides verbose mode showing per-metric breakdown
- [x] **SCORE-06**: Scoring thresholds are configurable (foundation for tuning)

### Recommendations

- [x] **REC-01**: Generates Top 5 improvement recommendations
- [x] **REC-02**: Ranks recommendations by impact (max potential gain x ease x category weight)
- [x] **REC-03**: Includes estimated score improvement for each recommendation
- [x] **REC-04**: Provides effort estimate (Low/Medium/High) for each improvement
- [x] **REC-05**: Frames recommendations in agent-readiness terms

### Output & CLI

- [x] **OUT-01**: Terminal text output with ANSI colors for readability
- [x] **OUT-02**: Summary section showing composite score and tier
- [x] **OUT-03**: Category breakdown section with individual scores
- [x] **OUT-04**: Recommendations section with Top 5 improvements
- [x] **OUT-05**: Optional `--threshold X` flag for CI gating (exit 2 if score < X)
- [x] **OUT-06**: Optional `--verbose` flag for detailed metric breakdown
- [x] **OUT-07**: Performance completes in <30s for 50k LOC repos
- [x] **OUT-08**: Progress indicators for long-running scans

## Traceability

| Requirement | Phase | Status | Outcome |
|-------------|-------|--------|---------|
| FOUND-01 | Phase 1 | ✅ Complete | scan command with directory arg |
| FOUND-02 | Phase 1 | ✅ Complete | Auto-detection works |
| FOUND-03 | Phase 1 | ✅ Complete | Help flag functional |
| FOUND-04 | Phase 1 | ✅ Complete | Version flag functional |
| FOUND-05 | Phase 1 | ✅ Complete | Clear error messages |
| FOUND-06 | Phase 1,4 | ✅ Complete | Exit codes 0/1/2 working |
| FOUND-07 | Phase 5 | ✅ Complete | Edge cases handled gracefully |
| FOUND-08 | Phase 1 | ✅ Complete | Vendor/generated exclusion |
| FOUND-09 | Phase 1 | ✅ Complete | Go file classification |
| C1-01 | Phase 2 | ✅ Complete | Cyclomatic complexity via gocyclo |
| C1-02 | Phase 2 | ✅ Complete | Function length measurement |
| C1-03 | Phase 2 | ✅ Complete | File size measurement |
| C1-04 | Phase 2 | ✅ Complete | Afferent coupling |
| C1-05 | Phase 2 | ✅ Complete | Efferent coupling |
| C1-06 | Phase 2 | ✅ Complete | Duplication detection |
| C3-01 | Phase 2 | ✅ Complete | Directory depth |
| C3-02 | Phase 2 | ✅ Complete | Module fanout |
| C3-03 | Phase 2 | ✅ Complete | Circular dependency detection |
| C3-04 | Phase 2 | ✅ Complete | Import complexity |
| C3-05 | Phase 2 | ✅ Complete | Dead code detection |
| C6-01 | Phase 2 | ✅ Complete | Test file detection |
| C6-02 | Phase 2 | ✅ Complete | Test-to-code ratio |
| C6-03 | Phase 2 | ✅ Complete | Coverage parsing (3 formats) |
| C6-04 | Phase 2 | ✅ Complete | Test isolation analysis |
| C6-05 | Phase 2 | ✅ Complete | Assertion density |
| SCORE-01 | Phase 3 | ✅ Complete | Per-category scores |
| SCORE-02 | Phase 3 | ✅ Complete | Composite score with weights |
| SCORE-03 | Phase 3 | ✅ Complete | Tier classification |
| SCORE-04 | Phase 3 | ✅ Complete | Piecewise interpolation |
| SCORE-05 | Phase 3 | ✅ Complete | Verbose breakdown |
| SCORE-06 | Phase 3 | ✅ Complete | Configurable thresholds |
| REC-01 | Phase 4 | ✅ Complete | Top 5 recommendations |
| REC-02 | Phase 4 | ✅ Complete | Impact ranking |
| REC-03 | Phase 4 | ✅ Complete | Score improvement estimates |
| REC-04 | Phase 4 | ✅ Complete | Effort estimates |
| REC-05 | Phase 4 | ✅ Complete | Agent-readiness framing |
| OUT-01 | Phase 4 | ✅ Complete | ANSI colors with TTY detection |
| OUT-02 | Phase 4 | ✅ Complete | Summary section |
| OUT-03 | Phase 4 | ✅ Complete | Category breakdown |
| OUT-04 | Phase 4 | ✅ Complete | Recommendations section |
| OUT-05 | Phase 4 | ✅ Complete | --threshold flag with exit 2 |
| OUT-06 | Phase 4 | ✅ Complete | --verbose flag |
| OUT-07 | Phase 5 | ✅ Complete | Performance <30s (14s for 50k LOC) |
| OUT-08 | Phase 5 | ✅ Complete | Progress spinner |

**Coverage:**
- v1 requirements: 44 total
- Shipped: 44 (100%)
- Adjusted: 0
- Dropped: 0

---

## Milestone Summary

**Shipped:** 44 of 44 v1 requirements (100%)

**Adjusted during implementation:**
- None - All requirements delivered as specified

**Dropped:**
- None - All planned v1 requirements completed

**Validation:**
- All requirements verified via automated tests (81 test functions)
- Integration verified via milestone audit (v1-MILESTONE-AUDIT.md)
- End-to-end flows tested (6 complete user workflows)
- Cross-phase wiring verified (14/14 key exports connected)

---

*Archived: 2026-02-01 as part of v1 milestone completion*
