# Requirements: Agent Readiness Score (ARS)

**Defined:** 2026-02-04
**Core Value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## v0.0.4 Requirements

Requirements for milestone v0.0.4 Metric Research. Each maps to roadmap phases.

### C1: Code Health Metrics (Issue #47)

- [x] **C1-01**: Research foundational sources (pre-2021) for all 6 C1 metrics
- [x] **C1-02**: Research AI/agent era sources (2021+) for all 6 C1 metrics
- [x] **C1-03**: Add inline citations (Author, Year) to complexity_avg description
- [x] **C1-04**: Add inline citations (Author, Year) to func_length_avg description
- [x] **C1-05**: Add inline citations (Author, Year) to file_size_avg description
- [x] **C1-06**: Add inline citations (Author, Year) to afferent_coupling_avg description
- [x] **C1-07**: Add inline citations (Author, Year) to efferent_coupling_avg description
- [x] **C1-08**: Add inline citations (Author, Year) to duplication_rate description
- [x] **C1-09**: Add References section for all C1 metrics with verified URLs
- [x] **C1-10**: Verify all C1 citation URLs accessible

### C2: Semantic Explicitness Metrics (Issue #48)

- [x] **C2-01**: Research foundational sources (pre-2021) for all 5 C2 metrics
- [x] **C2-02**: Research AI/agent era sources (2021+) for all 5 C2 metrics
- [x] **C2-03**: Add inline citations to type_annotation_coverage description
- [x] **C2-04**: Add inline citations to naming_consistency description
- [x] **C2-05**: Add inline citations to magic_number_ratio description
- [x] **C2-06**: Add inline citations to type_strictness description
- [x] **C2-07**: Add inline citations to null_safety description
- [x] **C2-08**: Add References section for all C2 metrics with verified URLs
- [x] **C2-09**: Verify all C2 citation URLs accessible

### C3: Architecture Metrics (Issue #49)

- [x] **C3-01**: Research foundational sources (pre-2021) for all 5 C3 metrics
- [x] **C3-02**: Research AI/agent era sources (2021+) for all 5 C3 metrics
- [x] **C3-03**: Add inline citations to max_dir_depth description
- [x] **C3-04**: Add inline citations to module_fanout_avg description
- [x] **C3-05**: Add inline citations to circular_deps description
- [x] **C3-06**: Add inline citations to import_complexity_avg description
- [x] **C3-07**: Add inline citations to dead_exports description
- [x] **C3-08**: Add References section for all C3 metrics with verified URLs
- [x] **C3-09**: Verify all C3 citation URLs accessible

### C4: Documentation Metrics (Issue #50)

- [x] **C4-01**: Research foundational sources (pre-2021) for all 7 C4 metrics
- [x] **C4-02**: Research AI/agent era sources (2021+) for all 7 C4 metrics
- [x] **C4-03**: Add inline citations to readme_word_count description
- [x] **C4-04**: Add inline citations to comment_density description
- [x] **C4-05**: Add inline citations to api_doc_coverage description
- [x] **C4-06**: Add inline citations to changelog_present description
- [x] **C4-07**: Add inline citations to examples_present description
- [x] **C4-08**: Add inline citations to contributing_present description
- [x] **C4-09**: Add inline citations to diagrams_present description
- [x] **C4-10**: Add References section for all C4 metrics with verified URLs
- [x] **C4-11**: Verify all C4 citation URLs accessible

### C5: Temporal Dynamics Metrics (Issue #51)

- [x] **C5-01**: Research foundational sources (pre-2021) for all 5 C5 metrics
- [x] **C5-02**: Research AI/agent era sources (2021+) for all 5 C5 metrics
- [x] **C5-03**: Add inline citations to churn_rate description
- [x] **C5-04**: Add inline citations to temporal_coupling_pct description
- [x] **C5-05**: Add inline citations to author_fragmentation description
- [x] **C5-06**: Add inline citations to commit_stability description
- [x] **C5-07**: Add inline citations to hotspot_concentration description
- [x] **C5-08**: Add References section for all C5 metrics with verified URLs
- [x] **C5-09**: Verify all C5 citation URLs accessible

### C6: Testing Metrics (Issue #52)

- [x] **C6-01**: Research foundational sources (pre-2021) for all 5 C6 metrics
- [x] **C6-02**: Research AI/agent era sources (2021+) for all 5 C6 metrics
- [x] **C6-03**: Add inline citations to test_to_code_ratio description
- [x] **C6-04**: Add inline citations to coverage_percent description
- [x] **C6-05**: Add inline citations to test_isolation description
- [x] **C6-06**: Add inline citations to assertion_density_avg description
- [x] **C6-07**: Add inline citations to test_file_ratio description
- [x] **C6-08**: Add References section for all C6 metrics with verified URLs
- [x] **C6-09**: Verify all C6 citation URLs accessible

### C7: MECE Metrics Implementation (Issue #54)

- [ ] **C7-IMPL-01**: Implement M1 (Task Execution Consistency) with 3-run variance measurement
- [ ] **C7-IMPL-02**: Implement M2 (Code Behavior Comprehension) with complexity-based sampling
- [ ] **C7-IMPL-03**: Implement M3 (Cross-File Navigation) with dependency tracing
- [ ] **C7-IMPL-04**: Implement M4 (Identifier Interpretability) with semantic clarity rating
- [ ] **C7-IMPL-05**: Implement M5 (Documentation Accuracy Detection) with mismatch detection
- [ ] **C7-IMPL-06**: Create CLI progress display with token counter and cost estimation
- [ ] **C7-IMPL-07**: Implement parallel execution capability for all 5 metrics
- [ ] **C7-IMPL-08**: Define research-based scoring thresholds (1-10 scale) for each metric

### C7: Agent Evaluation Citations (Issue #53)

- [x] **C7-01**: Research foundational sources for all 5 C7 metrics (M1-M5)
- [x] **C7-02**: Research AI/agent era sources (2021-2025) for all 5 C7 metrics
- [x] **C7-03**: Add inline citations to M1 (Task Execution Consistency) description
- [x] **C7-04**: Add inline citations to M2 (Code Behavior Comprehension) description
- [x] **C7-05**: Add inline citations to M3 (Cross-File Navigation) description
- [x] **C7-06**: Add inline citations to M4 (Identifier Interpretability) description
- [x] **C7-07**: Add inline citations to M5 (Documentation Accuracy Detection) description
- [x] **C7-08**: Add References section for all C7 metrics with verified URLs

### Quality Standards (Cross-Category)

- [x] **QA-01**: Establish citation style guide (format, density, source quality)
- [x] **QA-02**: Create URL verification protocol (curl -I + manual checks)
- [x] **QA-03**: Define Retraction Watch check process
- [x] **QA-04**: Document source quality checklist

## Out of Scope

| Feature | Reason |
|---------|--------|
| BibTeX/Zotero integration | Manual curation appropriate for ~100-150 citations |
| Automated link checking CI | Overkill for this scale; manual verification sufficient |
| Numbered citations [1] | Parenthetical (Author, Year) is standard for technical docs |
| JavaScript citation tooltips | CSS-only approach is CSP-safe and sufficient |
| Global bibliography | Per-category references keep context relevant |
| Code changes to citation infrastructure | Existing citations.go + descriptions.go sufficient |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| QA-01 | Phase 18 | Complete |
| QA-02 | Phase 18 | Complete |
| QA-03 | Phase 18 | Complete |
| QA-04 | Phase 18 | Complete |
| C1-01 | Phase 18 | Complete |
| C1-02 | Phase 18 | Complete |
| C1-03 | Phase 18 | Complete |
| C1-04 | Phase 18 | Complete |
| C1-05 | Phase 18 | Complete |
| C1-06 | Phase 18 | Complete |
| C1-07 | Phase 18 | Complete |
| C1-08 | Phase 18 | Complete |
| C1-09 | Phase 18 | Complete |
| C1-10 | Phase 18 | Complete |
| C6-01 | Phase 19 | Complete |
| C6-02 | Phase 19 | Complete |
| C6-03 | Phase 19 | Complete |
| C6-04 | Phase 19 | Complete |
| C6-05 | Phase 19 | Complete |
| C6-06 | Phase 19 | Complete |
| C6-07 | Phase 19 | Complete |
| C6-08 | Phase 19 | Complete |
| C6-09 | Phase 19 | Complete |
| C2-01 | Phase 20 | Complete |
| C2-02 | Phase 20 | Complete |
| C2-03 | Phase 20 | Complete |
| C2-04 | Phase 20 | Complete |
| C2-05 | Phase 20 | Complete |
| C2-06 | Phase 20 | Complete |
| C2-07 | Phase 20 | Complete |
| C2-08 | Phase 20 | Complete |
| C2-09 | Phase 20 | Complete |
| C3-01 | Phase 21 | Complete |
| C3-02 | Phase 21 | Complete |
| C3-03 | Phase 21 | Complete |
| C3-04 | Phase 21 | Complete |
| C3-05 | Phase 21 | Complete |
| C3-06 | Phase 21 | Complete |
| C3-07 | Phase 21 | Complete |
| C3-08 | Phase 21 | Complete |
| C3-09 | Phase 21 | Complete |
| C4-01 | Phase 22 | Complete |
| C4-02 | Phase 22 | Complete |
| C4-03 | Phase 22 | Complete |
| C4-04 | Phase 22 | Complete |
| C4-05 | Phase 22 | Complete |
| C4-06 | Phase 22 | Complete |
| C4-07 | Phase 22 | Complete |
| C4-08 | Phase 22 | Complete |
| C4-09 | Phase 22 | Complete |
| C4-10 | Phase 22 | Complete |
| C4-11 | Phase 22 | Complete |
| C5-01 | Phase 23 | Complete |
| C5-02 | Phase 23 | Complete |
| C5-03 | Phase 23 | Complete |
| C5-04 | Phase 23 | Complete |
| C5-05 | Phase 23 | Complete |
| C5-06 | Phase 23 | Complete |
| C5-07 | Phase 23 | Complete |
| C5-08 | Phase 23 | Complete |
| C5-09 | Phase 23 | Complete |
| C7-IMPL-01 | Phase 24 | Complete |
| C7-IMPL-02 | Phase 24 | Complete |
| C7-IMPL-03 | Phase 24 | Complete |
| C7-IMPL-04 | Phase 24 | Complete |
| C7-IMPL-05 | Phase 24 | Complete |
| C7-IMPL-06 | Phase 24 | Complete |
| C7-IMPL-07 | Phase 24 | Complete |
| C7-IMPL-08 | Phase 24 | Complete |
| C7-01 | Phase 25 | Complete |
| C7-02 | Phase 25 | Complete |
| C7-03 | Phase 25 | Complete |
| C7-04 | Phase 25 | Complete |
| C7-05 | Phase 25 | Complete |
| C7-06 | Phase 25 | Complete |
| C7-07 | Phase 25 | Complete |
| C7-08 | Phase 25 | Complete |

**Coverage:**
- v0.0.4 requirements: 66 total
- Mapped to phases: 66/66
- Unmapped: 0

---
*Requirements defined: 2026-02-04*
*Last updated: 2026-02-04 after roadmap creation*
