# Phase 18: C1 Code Health - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish citation quality protocols (style guide, URL verification, source quality checklist) and add research-backed citations to all 6 C1 Code Health metrics: complexity_avg, func_length_avg, file_size_avg, afferent_coupling_avg, efferent_coupling_avg, duplication_rate.

This phase creates the quality standards that all subsequent categories (C2-C7) will inherit.

</domain>

<decisions>
## Implementation Decisions

### Citation Style & Density
- **Target density:** 2-3 citations per metric (focused approach)
- **Format:** Use (Author et al., Year) for papers with 3+ authors
- **Balance:** Equal weight — 1 foundational source + 1-2 AI-era sources per metric
- **Redundancy handling:** Claude's discretion — cite seminal work only if validations are weak; cite seminal + strongest validation if evidence is strong

### Source Quality Criteria
- **Foundational sources (pre-2021):** Books + papers acceptable — seminal books (Fowler, Martin) are industry standards
- **AI-era sources (2021+):** ArXiv preprints acceptable — quality preprints are standard in AI/ML research
- **Retraction checks:** Check only if suspicious — trust reputable sources, verify if concerns arise
- **Claim attribution:** Every quantified claim gets explicit citation — maximum traceability for readers

### URL Verification Approach
- **DOI preference:** DOI preferred, URLs acceptable — use DOI if easy to find, don't spend excessive time converting
- **Verification documentation:** Document process only — high-level protocol, not specific curl commands
- **Paywall handling:** Claude's discretion per paper — balance effort vs accessibility
- **Verification timing:** During research — verify URLs as sources are found, catch issues early

### Metric Description Structure
- **Citation placement:** Both brief and detailed descriptions — key citation in brief, supporting evidence in detailed
- **Evidence organization:** Dedicated "Research Evidence" subsection in detailed description — clear separation between explanation and evidence
- **References scope:** All C1 sources at category level — one References section for all 6 C1 metrics combined
- **Threshold justifications:** Thresholds need citations — every threshold value should be research-backed

### Claude's Discretion
- When to cite seminal work only vs seminal + validation (judgment based on validation strength)
- Open-access alternatives for paywalled sources (balance effort vs reader accessibility)
- Exact wording and flow of Research Evidence subsections

</decisions>

<specifics>
## Specific Ideas

- This phase establishes the quality protocols that C2-C7 inherit, so thoroughness here pays dividends later
- Focus on the "why" of thresholds (research-backed reasoning), not just the "what"
- Research showed 50%+ URL rot without DOIs — prefer DOIs when available but don't make it a blocker
- Borg et al. (2026) is the key AI-era paper for C1 — already known from research phase

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 18-c1-code-health*
*Context gathered: 2026-02-04*
