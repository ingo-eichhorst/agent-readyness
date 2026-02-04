# Phase 22: C4 Documentation - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Add research-backed citations to all 7 C4 Documentation metrics. Work involves content expansion within existing citation infrastructure (inline citations + References section) — no code changes required. Inherits quality protocols from Phase 18.

</domain>

<decisions>
## Implementation Decisions

### Citation approach (inherited from C1-C3)
- Follow established protocol from Phases 18-21
- 2-3 citations per metric (avoid academic over-citation)
- DOI preferred, ArXiv acceptable for foundational sources
- Inline format: (Author, Year)
- Full references section with verified URLs

### Source quality (inherited)
- Balance foundational documentation research with modern practices
- Duplicate entries per category allowed (self-contained references)
- URL verification via curl -I before adding
- Open-access versions preferred where available

### Research era coverage
- Documentation field has long history (Parnas 1970s, Knuth literate programming)
- Modern practices matter (Markdown, docs-as-code, README-driven development)
- Include both timeless principles and contemporary evidence
- AI-era sources (2021+) where relevant to agent consumption of docs

### Metric coverage consistency
- All 7 C4 metrics receive citations
- Depth follows style guide (2-6 citations per metric)
- README metrics may cite different sources than API doc metrics
- Empty state documentation can reference UX research if needed

### Claude's Discretion
- Exact mix of foundational vs modern sources per metric
- When to prioritize industry reports vs academic studies
- How to handle blog posts and practitioner guides (case-by-case)
- ArXiv vs published version selection

</decisions>

<specifics>
## Specific Ideas

- Documentation research spans multiple domains: technical writing, software engineering, UX (for examples/error messages)
- Classic works: Parnas on information hiding in documentation, Knuth on literate programming
- Modern practices: README-driven development, docs-as-code movement, Markdown standardization
- GitHub documentation guides and style guides may be acceptable as practitioner sources
- Consider citing documentation quality studies from industry (Google, Microsoft) if methodologically sound

</specifics>

<deferred>
## Deferred Ideas

None — discussion confirmed inheritance of established approach.

</deferred>

---

*Phase: 22-c4-documentation*
*Context gathered: 2026-02-04*
