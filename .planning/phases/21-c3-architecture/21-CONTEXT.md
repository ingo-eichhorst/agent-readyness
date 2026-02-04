# Phase 21: C3 Architecture - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Add research-backed citations to all 5 C3 Architecture metrics (directory depth, module fanout, circular dependencies, dead exports, and related measurements). Establish scientific foundations for architectural quality claims using established quality protocols from Phase 18.

</domain>

<decisions>
## Implementation Decisions

### Citation Approach
- Follow established patterns from C1 (18-02), C6 (19-01), and C2 (20-01) phases
- Use (Author, Year) inline format with 2-6 citations per metric
- Foundational sources: Parnas 1972 (modularity), Stevens & Myers 1974 (coupling/cohesion), Martin 2003 (Clean Architecture)
- Modern evidence: Recent replication studies and empirical architecture research
- Per-category grouping: C3 References section matches existing HTML report structure

### Source Quality
- Inherit quality protocols from Phase 18: style guide, URL verification, source quality checklist
- Balance classic SE principles with contemporary empirical evidence
- Label practitioner opinion vs research findings (e.g., Martin as "influential practitioner perspective")
- Acknowledge when modern studies contradict classic principles — cite both, note controversy

### Metric Coverage Priorities
- All 5 metrics receive citations following density guidelines (2-6 per metric)
- Directory depth: Parnas modularity principles + modern repo analysis
- Module fanout: Coupling research (Stevens & Myers, Chidamber & Kemerer)
- Circular dependencies: Architecture research + build system literature
- Dead exports: Code quality + maintainability research

### Claude's Discretion
- Exact citation distribution across metrics (within 2-6 range)
- Choice between multiple valid sources for same claim
- Handling of paywalled sources (find open-access alternatives)
- Specific wording of inline citation context

</decisions>

<specifics>
## Specific Ideas

- "Use the last 3 milestones categories c1 ... c3 as template" — follow established patterns from completed phases
- Parnas 1972 is foundational for modularity concepts
- Martin's work influential but should be labeled as practitioner perspective
- Balance between timeless principles and modern empirical evidence (similar to C2 approach)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 21-c3-architecture*
*Context gathered: 2026-02-04*
