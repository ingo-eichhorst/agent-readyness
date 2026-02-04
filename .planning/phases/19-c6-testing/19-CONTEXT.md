# Phase 19: C6 Testing - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Add research-backed citations to all 5 C6 Testing metrics: test_ratio, coverage_percent, test_isolation_score, assertion_density, test_run_time_avg.

This phase inherits quality protocols from Phase 18 (citation style guide, URL verification, source quality checklist).

</domain>

<decisions>
## Implementation Decisions

### Citation Style & Density
- **Target density:** 2-3 citations per metric (same as C1)
- **Format:** (Author et al., Year) for papers with 3+ authors
- **Balance:** 1 foundational source + 1-2 AI-era sources per metric
- **Redundancy handling:** Cite seminal work only if validations are weak; cite seminal + strongest validation if evidence is strong

### Source Quality Criteria
- **Foundational sources (pre-2021):** Books + papers acceptable (TDD classics like Beck, Meszaros are industry standards)
- **AI-era sources (2021+):** ArXiv preprints acceptable (standard in AI/ML research)
- **Retraction checks:** Check only if suspicious
- **Claim attribution:** Every quantified claim gets explicit citation

### URL Verification Approach
- **DOI preference:** DOI preferred, URLs acceptable
- **Verification timing:** During research phase
- **Paywall handling:** Claude's discretion per paper
- **Verification documentation:** High-level protocol, not specific curl commands

### Metric Description Structure
- **Citation placement:** Both brief and detailed descriptions
- **Evidence organization:** Dedicated "Research Evidence" subsection in detailed description
- **References scope:** All C6 sources at category level (one References section for all 5 metrics)
- **Threshold justifications:** Thresholds need citations

### Claude's Discretion
- When to cite seminal work only vs seminal + validation
- Open-access alternatives for paywalled sources
- Exact wording and flow of Research Evidence subsections
- Balance between tool-specific research (JUnit, pytest) vs language-agnostic testing concepts

</decisions>

<specifics>
## Specific Ideas

- Testing is a well-researched domain — plenty of foundational sources available (Beck's TDD, Meszaros' xUnit patterns, coverage research)
- Focus on concepts over tools where possible (test isolation principles rather than specific framework features)
- Coverage controversy (100% coverage debate) should be acknowledged if relevant to metric descriptions
- AI-era testing research may focus on LLM-generated tests, test maintenance with AI assistants

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 19-c6-testing*
*Context gathered: 2026-02-04*
