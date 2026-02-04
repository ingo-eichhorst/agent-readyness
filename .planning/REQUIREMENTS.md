# Requirements: Agent Readiness Score (ARS)

**Defined:** 2026-02-04
**Core Value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## v0.0.4 Requirements

Requirements for milestone v0.0.4 Metric Research. Each maps to roadmap phases.

### C1: Code Health Metrics (Issue #47)

- [ ] **C1-01**: Research foundational sources (pre-2021) for all 6 C1 metrics
- [ ] **C1-02**: Research AI/agent era sources (2021+) for all 6 C1 metrics
- [ ] **C1-03**: Add inline citations (Author, Year) to complexity_avg description
- [ ] **C1-04**: Add inline citations (Author, Year) to func_length_avg description
- [ ] **C1-05**: Add inline citations (Author, Year) to file_size_avg description
- [ ] **C1-06**: Add inline citations (Author, Year) to afferent_coupling_avg description
- [ ] **C1-07**: Add inline citations (Author, Year) to efferent_coupling_avg description
- [ ] **C1-08**: Add inline citations (Author, Year) to duplication_rate description
- [ ] **C1-09**: Add References section for all C1 metrics with verified URLs
- [ ] **C1-10**: Verify all C1 citation URLs accessible

### C2: Semantic Explicitness Metrics (Issue #48)

- [ ] **C2-01**: Research foundational sources (pre-2021) for all 5 C2 metrics
- [ ] **C2-02**: Research AI/agent era sources (2021+) for all 5 C2 metrics
- [ ] **C2-03**: Add inline citations to type_annotation_coverage description
- [ ] **C2-04**: Add inline citations to naming_consistency description
- [ ] **C2-05**: Add inline citations to magic_number_ratio description
- [ ] **C2-06**: Add inline citations to type_strictness description
- [ ] **C2-07**: Add inline citations to null_safety description
- [ ] **C2-08**: Add References section for all C2 metrics with verified URLs
- [ ] **C2-09**: Verify all C2 citation URLs accessible

### C3: Architecture Metrics (Issue #49)

- [ ] **C3-01**: Research foundational sources (pre-2021) for all 5 C3 metrics
- [ ] **C3-02**: Research AI/agent era sources (2021+) for all 5 C3 metrics
- [ ] **C3-03**: Add inline citations to max_dir_depth description
- [ ] **C3-04**: Add inline citations to module_fanout_avg description
- [ ] **C3-05**: Add inline citations to circular_deps description
- [ ] **C3-06**: Add inline citations to import_complexity_avg description
- [ ] **C3-07**: Add inline citations to dead_exports description
- [ ] **C3-08**: Add References section for all C3 metrics with verified URLs
- [ ] **C3-09**: Verify all C3 citation URLs accessible

### C4: Documentation Metrics (Issue #50)

- [ ] **C4-01**: Research foundational sources (pre-2021) for all 7 C4 metrics
- [ ] **C4-02**: Research AI/agent era sources (2021+) for all 7 C4 metrics
- [ ] **C4-03**: Add inline citations to readme_word_count description
- [ ] **C4-04**: Add inline citations to comment_density description
- [ ] **C4-05**: Add inline citations to api_doc_coverage description
- [ ] **C4-06**: Add inline citations to changelog_present description
- [ ] **C4-07**: Add inline citations to examples_present description
- [ ] **C4-08**: Add inline citations to contributing_present description
- [ ] **C4-09**: Add inline citations to diagrams_present description
- [ ] **C4-10**: Add References section for all C4 metrics with verified URLs
- [ ] **C4-11**: Verify all C4 citation URLs accessible

### C5: Temporal Dynamics Metrics (Issue #51)

- [ ] **C5-01**: Research foundational sources (pre-2021) for all 5 C5 metrics
- [ ] **C5-02**: Research AI/agent era sources (2021+) for all 5 C5 metrics
- [ ] **C5-03**: Add inline citations to churn_rate description
- [ ] **C5-04**: Add inline citations to temporal_coupling_pct description
- [ ] **C5-05**: Add inline citations to author_fragmentation description
- [ ] **C5-06**: Add inline citations to commit_stability description
- [ ] **C5-07**: Add inline citations to hotspot_concentration description
- [ ] **C5-08**: Add References section for all C5 metrics with verified URLs
- [ ] **C5-09**: Verify all C5 citation URLs accessible

### C6: Testing Metrics (Issue #52)

- [ ] **C6-01**: Research foundational sources (pre-2021) for all 5 C6 metrics
- [ ] **C6-02**: Research AI/agent era sources (2021+) for all 5 C6 metrics
- [ ] **C6-03**: Add inline citations to test_to_code_ratio description
- [ ] **C6-04**: Add inline citations to coverage_percent description
- [ ] **C6-05**: Add inline citations to test_isolation description
- [ ] **C6-06**: Add inline citations to assertion_density_avg description
- [ ] **C6-07**: Add inline citations to test_file_ratio description
- [ ] **C6-08**: Add References section for all C6 metrics with verified URLs
- [ ] **C6-09**: Verify all C6 citation URLs accessible

### C7: Agent Evaluation Metrics (Issue #53)

- [ ] **C7-01**: Research foundational sources (pre-2021) for C7 overall_score metric
- [ ] **C7-02**: Research AI/agent era sources (2021+) for C7 overall_score metric
- [ ] **C7-03**: Add inline citations to overall_score description
- [ ] **C7-04**: Add References section for C7 metric with verified URLs
- [ ] **C7-05**: Verify all C7 citation URLs accessible

### Quality Standards (Cross-Category)

- [ ] **QA-01**: Establish citation style guide (format, density, source quality)
- [ ] **QA-02**: Create URL verification protocol (curl -I + manual checks)
- [ ] **QA-03**: Define Retraction Watch check process
- [ ] **QA-04**: Document source quality checklist

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
| (To be filled by roadmapper) | | |

**Coverage:**
- v0.0.4 requirements: 70 total
- Mapped to phases: (pending)
- Unmapped: (pending)

---
*Requirements defined: 2026-02-04*
*Last updated: 2026-02-04 after initial definition*
