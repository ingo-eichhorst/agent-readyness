# Roadmap: Agent Readiness Score (ARS) v0.0.4

## Milestones

- ✅ **v1 Initial Release** - Phases 1-5 (shipped 2026-02-01)
- ✅ **v0.0.2 Complete Analysis Framework** - Phases 6-12 (shipped 2026-02-03)
- ✅ **v0.0.3 Simplification & Polish** - Phases 13-17 (shipped 2026-02-04)
- 🚧 **v0.0.4 Metric Research** - Phases 18-24 (in progress)

## Overview

This milestone establishes scientific foundations for all 33 ARS metric descriptions. Each of the 7 analysis categories (C1-C7) receives research-backed citations with foundational sources (pre-2021) and AI-era evidence (2021+). Phase 18 establishes quality protocols (citation style guide, URL verification, source quality checklist) that all subsequent phases inherit. The work is content expansion within the existing citation infrastructure — no code changes required.

## Phases

**Phase Numbering:**
- Phases 18-24 continue from v0.0.3 (ended at Phase 17)
- Each phase corresponds to one analysis category

- [x] **Phase 18: C1 Code Health** - Establish quality protocols + 6 metric citations
- [x] **Phase 19: C6 Testing** - 5 metric citations (well-researched domain)
- [x] **Phase 20: C2 Semantic Explicitness** - 5 metric citations
- [ ] **Phase 21: C3 Architecture** - 5 metric citations
- [ ] **Phase 22: C4 Documentation** - 7 metric citations
- [ ] **Phase 23: C5 Temporal Dynamics** - 5 metric citations
- [ ] **Phase 24: C7 Agent Evaluation** - 1 metric citation (nascent field)

## Phase Details

### 🚧 v0.0.4 Metric Research (In Progress)

**Milestone Goal:** Scientific foundations for all metric descriptions with verifiable citations

### Phase 18: C1 Code Health
**Goal**: Establish citation quality protocols and add research-backed citations to all C1 Code Health metrics
**Depends on**: Nothing (first phase of milestone)
**Requirements**: QA-01, QA-02, QA-03, QA-04, C1-01, C1-02, C1-03, C1-04, C1-05, C1-06, C1-07, C1-08, C1-09, C1-10
**Success Criteria** (what must be TRUE):
  1. Citation style guide exists documenting (Author, Year) format, density targets (2-6 per metric), and source quality requirements
  2. URL verification protocol documented with curl -I checks and manual verification steps
  3. All 6 C1 metrics have inline citations (Author, Year) in their descriptions referencing foundational and AI-era sources
  4. C1 References section contains complete citations with verified, accessible URLs
  5. Every quantified claim in C1 metric descriptions has an explicit source attribution
**Plans**: 2 plans

Plans:
- [x] 18-01-PLAN.md — Establish citation quality protocols (style guide, URL verification, source quality checklist)
- [x] 18-02-PLAN.md — Add citations to all 6 C1 metrics and verify URLs

### Phase 19: C6 Testing
**Goal**: Add research-backed citations to all C6 Testing metrics using established quality protocols
**Depends on**: Phase 18 (inherits quality protocols)
**Requirements**: C6-01, C6-02, C6-03, C6-04, C6-05, C6-06, C6-07, C6-08, C6-09
**Success Criteria** (what must be TRUE):
  1. All 5 C6 metrics have inline citations referencing foundational TDD/coverage research and AI-era testing studies
  2. C6 References section contains complete citations with verified, accessible URLs
  3. Citation density follows style guide (2-6 per metric, prioritizing seminal works)
  4. Every quantified claim in C6 metric descriptions has an explicit source attribution
**Plans**: 1 plan

Plans:
- [x] 19-01-PLAN.md — Add citations to all 5 C6 metrics and verify URLs

### Phase 20: C2 Semantic Explicitness
**Goal**: Add research-backed citations to all C2 Semantic Explicitness metrics
**Depends on**: Phase 18 (inherits quality protocols)
**Requirements**: C2-01, C2-02, C2-03, C2-04, C2-05, C2-06, C2-07, C2-08, C2-09
**Success Criteria** (what must be TRUE):
  1. All 5 C2 metrics have inline citations referencing type theory foundations and modern type safety research
  2. C2 References section contains complete citations with verified, accessible URLs
  3. Citations distinguish timeless type theory from dated empirical findings
  4. Every quantified claim in C2 metric descriptions has an explicit source attribution
**Plans**: 1 plan

Plans:
- [x] 20-01-PLAN.md — Add citations to all 5 C2 metrics and verify URLs

### Phase 21: C3 Architecture
**Goal**: Add research-backed citations to all C3 Architecture metrics
**Depends on**: Phase 18 (inherits quality protocols)
**Requirements**: C3-01, C3-02, C3-03, C3-04, C3-05, C3-06, C3-07, C3-08, C3-09
**Success Criteria** (what must be TRUE):
  1. All 5 C3 metrics have inline citations referencing foundational SE sources (Parnas 1972, Martin 2003) and modern replication studies
  2. C3 References section contains complete citations with verified, accessible URLs
  3. Classic architecture principles are balanced with contemporary evidence
  4. Every quantified claim in C3 metric descriptions has an explicit source attribution
**Plans**: TBD

Plans:
- [ ] 21-01: [TBD during planning]

### Phase 22: C4 Documentation
**Goal**: Add research-backed citations to all C4 Documentation metrics
**Depends on**: Phase 18 (inherits quality protocols)
**Requirements**: C4-01, C4-02, C4-03, C4-04, C4-05, C4-06, C4-07, C4-08, C4-09, C4-10, C4-11
**Success Criteria** (what must be TRUE):
  1. All 7 C4 metrics have inline citations referencing documentation quality research
  2. C4 References section contains complete citations with verified, accessible URLs
  3. Open-access versions provided for paywalled sources where possible
  4. Every quantified claim in C4 metric descriptions has an explicit source attribution
**Plans**: TBD

Plans:
- [ ] 22-01: [TBD during planning]

### Phase 23: C5 Temporal Dynamics
**Goal**: Add research-backed citations to all C5 Temporal Dynamics metrics
**Depends on**: Phase 18 (inherits quality protocols)
**Requirements**: C5-01, C5-02, C5-03, C5-04, C5-05, C5-06, C5-07, C5-08, C5-09
**Success Criteria** (what must be TRUE):
  1. All 5 C5 metrics have inline citations referencing Tornhill's work and D'Ambros et al. research
  2. C5 References section contains complete citations with verified, accessible URLs
  3. Book references include ISBN as stable backup to URLs
  4. Every quantified claim in C5 metric descriptions has an explicit source attribution
**Plans**: TBD

Plans:
- [ ] 23-01: [TBD during planning]

### Phase 24: C7 Agent Evaluation
**Goal**: Add research-backed citations to C7 Agent Evaluation metrics, explicitly acknowledging the nascent state of AI agent code quality research
**Depends on**: Phase 18 (inherits quality protocols)
**Requirements**: C7-01, C7-02, C7-03, C7-04, C7-05
**Success Criteria** (what must be TRUE):
  1. C7 overall_score metric has inline citations referencing Borg et al. 2026, SWE-bench, and adjacent AI code generation research
  2. C7 References section contains complete citations with verified, accessible URLs
  3. Research novelty is explicitly acknowledged (most relevant work is in preprints)
  4. Adjacent research (LLM code generation, human factors in AI-assisted development) is cited where direct agent research is unavailable
**Plans**: TBD

Plans:
- [ ] 24-01: [TBD during planning]

<details>
<summary>Completed Milestones (v1 through v0.0.3)</summary>

### v1 Initial Release (Phases 1-5) - SHIPPED 2026-02-01

See .planning/MILESTONES.md for details.

### v0.0.2 Complete Analysis Framework (Phases 6-12) - SHIPPED 2026-02-03

See .planning/MILESTONES.md for details.

### v0.0.3 Simplification & Polish (Phases 13-17) - SHIPPED 2026-02-04

See .planning/MILESTONES.md for details.

</details>

## Progress

**Execution Order:**
Phases 18-24 execute sequentially. Phase 18 must complete first (establishes protocols).

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 18. C1 Code Health | v0.0.4 | 2/2 | Complete | 2026-02-04 |
| 19. C6 Testing | v0.0.4 | 1/1 | Complete | 2026-02-04 |
| 20. C2 Semantic Explicitness | v0.0.4 | 1/1 | Complete | 2026-02-04 |
| 21. C3 Architecture | v0.0.4 | 0/TBD | Not started | - |
| 22. C4 Documentation | v0.0.4 | 0/TBD | Not started | - |
| 23. C5 Temporal Dynamics | v0.0.4 | 0/TBD | Not started | - |
| 24. C7 Agent Evaluation | v0.0.4 | 0/TBD | Not started | - |

---
*Roadmap created: 2026-02-04*
*Last updated: 2026-02-04*
