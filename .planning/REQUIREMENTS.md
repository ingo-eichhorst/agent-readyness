# Requirements: Agent Readiness Score (ARS)

**Defined:** 2026-02-06
**Core Value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## v0.0.5 Requirements (C7 Debug Infrastructure)

Requirements for investigating and fixing M2/M3/M4 scoring anomalies.

### Debug Foundation

- [ ] **DBG-01**: CLI accepts `--debug-c7` flag to enable C7 debug mode
- [ ] **DBG-02**: Debug mode auto-enables C7 evaluation if not already enabled
- [ ] **DBG-03**: Debug output writes exclusively to stderr (preserves stdout for JSON/HTML)
- [ ] **DBG-04**: Debug mode captures full prompts sent to each metric
- [ ] **DBG-05**: Debug mode captures full Claude CLI responses for each sample
- [ ] **DBG-06**: Debug mode displays score traces showing heuristic indicator contributions

### Heuristic Testing

- [ ] **TEST-01**: Test fixtures with real captured Claude CLI responses in `testdata/c7_responses/`
- [ ] **TEST-02**: Unit tests for M2 (Code Behavior Comprehension) scoring function
- [ ] **TEST-03**: Unit tests for M3 (Cross-File Navigation) scoring function
- [ ] **TEST-04**: Unit tests for M4 (Identifier Interpretability) scoring function
- [ ] **TEST-05**: Tests document expected scores for each fixture response

### Response Replay

- [ ] **RPL-01**: `--debug-dir` flag specifies directory for response persistence
- [ ] **RPL-02**: Debug mode saves captured responses to JSON files in debug-dir
- [ ] **RPL-03**: Replay mode loads responses from debug-dir instead of executing Claude CLI
- [ ] **RPL-04**: Replay mode enables fast heuristic iteration without agent execution

### Scoring Fixes

- [ ] **FIX-01**: M2 scoring function produces non-zero scores for valid comprehension responses
- [ ] **FIX-02**: M3 scoring function produces non-zero scores for valid navigation responses
- [ ] **FIX-03**: M4 scoring function produces non-zero scores for valid interpretation responses
- [ ] **FIX-04**: Heuristic adjustments documented with rationale in code comments

### Documentation

- [ ] **DOC-01**: GitHub issue #55 updated with root cause analysis
- [ ] **DOC-02**: GitHub issue #55 documents fixes applied and test results
- [ ] **DOC-03**: `--debug-c7` flag documented in CLI help output
- [ ] **DOC-04**: Debug mode usage documented in README or docs/

## Out of Scope

| Feature | Reason |
|---------|--------|
| Real-time response streaming | High complexity, debug mode is for post-execution analysis |
| Live debugger integration (Delve) | Over-engineering for heuristic investigation |
| Complex debug query language | Scope creep beyond investigation needs |
| Full logging framework (slog) | Adds new paradigm for single debug flag, inconsistent with existing `io.Writer` pattern |
| --verbose flag reuse | Affects all categories, C7 debug is category-specific and more voluminous |
| M1/M5 scoring changes | M1 and M5 already score correctly (10/10) |
| Automated response capture in CI | Debug mode is for manual investigation, not automated testing |
| JavaScript-based debug UI | CLI tool, no web interface |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DBG-01 | TBD | Pending |
| DBG-02 | TBD | Pending |
| DBG-03 | TBD | Pending |
| DBG-04 | TBD | Pending |
| DBG-05 | TBD | Pending |
| DBG-06 | TBD | Pending |
| TEST-01 | TBD | Pending |
| TEST-02 | TBD | Pending |
| TEST-03 | TBD | Pending |
| TEST-04 | TBD | Pending |
| TEST-05 | TBD | Pending |
| RPL-01 | TBD | Pending |
| RPL-02 | TBD | Pending |
| RPL-03 | TBD | Pending |
| RPL-04 | TBD | Pending |
| FIX-01 | TBD | Pending |
| FIX-02 | TBD | Pending |
| FIX-03 | TBD | Pending |
| FIX-04 | TBD | Pending |
| DOC-01 | TBD | Pending |
| DOC-02 | TBD | Pending |
| DOC-03 | TBD | Pending |
| DOC-04 | TBD | Pending |

**Coverage:**
- v0.0.5 requirements: 23 total
- Mapped to phases: 0 (roadmap not yet created)
- Unmapped: 23 ⚠️

---
*Requirements defined: 2026-02-06*
*Last updated: 2026-02-06 after initial definition*
