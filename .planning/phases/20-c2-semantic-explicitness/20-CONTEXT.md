# Phase 20: C2 Semantic Explicitness - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Add research-backed citations to all 5 C2 Semantic Explicitness metrics: type_annotation_coverage, naming_consistency, magic_number_ratio, type_strictness, null_safety.

This phase inherits quality protocols from Phase 18 (citation style guide, URL verification, source quality checklist).

</domain>

<decisions>
## Implementation Decisions

### Citation Style & Density
- **Target density:** 2-3 citations per metric (same as C1/C6)
- **Format:** (Author et al., Year) for papers with 3+ authors
- **Balance:** 1 foundational source + 1-2 AI-era sources per metric
- **Redundancy handling:** Cite seminal work only if validations are weak; cite seminal + strongest validation if evidence is strong

### Source Quality Criteria
- **Foundational sources (pre-2021):** Books + papers acceptable (type theory classics like Pierce, Cardelli are industry standards)
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
- **References scope:** All C2 sources at category level (one References section for all 5 metrics)
- **Threshold justifications:** Thresholds need citations

### Claude's Discretion
- When to cite seminal work only vs seminal + validation
- Open-access alternatives for paywalled sources
- Exact wording and flow of Research Evidence subsections
- Balance between language-agnostic type theory vs language-specific empirical findings (Go/Python/TypeScript)
- How to handle dated empirical findings — distinguish timeless type theory from context-dependent observations

</decisions>

<specifics>
## Specific Ideas

- Type safety research spans timeless type theory (Pierce's Types and Programming Languages) and modern empirical studies
- Focus on concepts over language-specific features where possible (null safety principles rather than Go's nil vs TypeScript's undefined)
- Distinguish foundational type theory (timeless) from dated empirical findings (pre-2010 studies may reflect older language ecosystems)
- AI-era type research may focus on type inference for LLMs, static analysis improvements, gradual typing adoption

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 20-c2-semantic-explicitness*
*Context gathered: 2026-02-04*
