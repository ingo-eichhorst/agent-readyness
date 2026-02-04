# Phase 21 Plan 01: C3 Architecture Citations Summary

---
phase: 21-c3-architecture
plan: 01
subsystem: output/documentation
tags: [citations, c3, architecture, research]
depends_on:
  requires: [18-01, 18-02]
  provides: [c3-citations, architecture-documentation]
  affects: [html-reports, metric-descriptions]
tech-stack:
  added: []
  patterns: [inline-citations, research-evidence-sections]
key-files:
  created: []
  modified: [internal/output/descriptions.go, internal/output/citations.go]
decisions:
  - id: martin-practitioner
    choice: "Label Martin's principles as influential practitioner perspective"
    reason: "Not derived from peer-reviewed empirical research"
  - id: c3-foundational-sources
    choice: "Parnas (1972) and Stevens et al. (1974) as primary foundational sources"
    reason: "Seminal works establishing module decomposition and coupling/cohesion principles"
metrics:
  duration: "3m24s"
  completed: 2026-02-04
---

**One-liner:** C3 Architecture metrics now backed by foundational SE research (Parnas 1972, Stevens et al. 1974) plus empirical studies (Oyetoyan 2015, Romano 2018) and AI-era evidence (Borg 2026).

## What Was Built

Added research-backed citations to all 5 C3 Architecture metrics following the quality protocols established in Phase 18.

### C3 Metrics Documented

| Metric | Brief Citation | Key Sources |
|--------|---------------|-------------|
| max_dir_depth | Parnas, 1972 | Parnas (1972), MacCormack (2006), Borg (2026) |
| module_fanout_avg | Stevens et al., 1974 | Stevens (1974), Chidamber & Kemerer (1994), Martin (2003), Borg (2026) |
| circular_deps | Lakos, 1996 | Parnas (1972), Martin (2003), Lakos (1996), Oyetoyan (2015), Borg (2026) |
| import_complexity_avg | Sangal et al., 2005 | Parnas (1972), Sangal (2005), Pisch (2024), Borg (2026) |
| dead_exports | Fowler et al., 1999 | Fowler (1999), Romano (2018), Borg (2026) |

### Citation Inventory

**Total C3 citations added:** 13 (was 2)

| Source | Year | Type | Key Contribution |
|--------|------|------|------------------|
| Parnas | 1972 | Foundational | Module decomposition, information hiding |
| Stevens, Myers & Constantine | 1974 | Foundational | Coupling/cohesion definitions |
| Gamma et al. | 1994 | Foundational | Design patterns for dependencies |
| Chidamber & Kemerer | 1994 | Empirical | CBO metric validation |
| Lakos | 1996 | Practitioner | Acyclic design techniques |
| Fowler et al. | 1999 | Practitioner | Dead Code smell classification |
| Martin | 2003 | Practitioner | ADP/SDP principles |
| Sangal et al. | 2005 | Empirical | DSM for architecture management |
| MacCormack et al. | 2006 | Empirical | Modularity benefits validation |
| Oyetoyan et al. | 2015 | Empirical | Circular deps change-proneness |
| Romano et al. | 2018 | Empirical | Dead code comprehensibility study |
| Pisch et al. | 2024 | Empirical | M-score modularity metric |
| Borg et al. | 2026 | AI-Era | Code health predicts agent reliability |

## Decisions Made

### Martin's Principles Labeled as Practitioner Perspective

**Decision:** Clearly label Martin (2003) as "influential practitioner perspective widely adopted in industry, though not derived from empirical research."

**Rationale:** Martin's Acyclic Dependencies Principle and Stable Dependencies Principle are widely taught and practiced, but they represent architectural guidance rather than findings from controlled studies. Distinguishing practitioner wisdom from peer-reviewed research maintains intellectual honesty.

### Foundational Sources Prioritized

**Decision:** Use Parnas (1972) and Stevens et al. (1974) as primary foundational sources for all architecture concepts.

**Rationale:** These seminal works established the vocabulary and principles that later research built upon. Parnas's information hiding and Stevens's coupling/cohesion remain the conceptual foundation for all architecture metrics.

### Research Evidence Sections Added

**Decision:** Each C3 metric detailed description includes a dedicated "Research Evidence" subsection with 3-5 citations.

**Rationale:** Follows the pattern established in C1 (Phase 18) and C6 (Phase 19). Consolidates citations in one location for readability while keeping metrics evidence-based.

## Deviations from Plan

None - plan executed exactly as written.

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| internal/output/descriptions.go | Added inline citations to 5 C3 metrics | +24/-10 |
| internal/output/citations.go | Added 11 new C3 reference entries | +88 |

## Verification Results

- `go build ./...` passes
- `go test ./internal/output/...` passes
- All 5 C3 metrics have inline citations with Research Evidence sections
- 13 C3 entries in citations.go (existing 2 + 11 new)
- All 13 URLs verified accessible (302 redirects for DOI = success)

## Next Phase Readiness

**Phase 22:** C4 Documentation citations
- All infrastructure in place
- Same patterns apply (inline citations, Research Evidence sections)
- C4 may need different source types (documentation quality research vs architecture)

**No blockers identified.**
