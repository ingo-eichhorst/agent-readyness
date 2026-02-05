# Phase 25: C7 Agent Evaluation Citations - Context

**Gathered:** 2026-02-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Add research-backed citations to the 5 new C7 metrics (M1: Task Execution Consistency, M2: Code Behavior Comprehension, M3: Cross-File Navigation, M4: Identifier Interpretability, M5: Documentation Accuracy Detection). Document scientific foundations while explicitly acknowledging the nascent state of AI agent code quality research.

</domain>

<decisions>
## Implementation Decisions

### Citation Approach
- Follow the same patterns established in C1-C6 (Phases 18-23)
- Apply established quality protocols from Phase 18 citation style guide
- Maintain consistent citation density (2-6 per metric)
- Use (Author, Year) inline format with complete References section
- Verify all URLs using curl -I checks

### Research Scarcity Handling
- Use adjacent research when direct agent evaluation studies don't exist
  - SWE-bench for task execution
  - RepoGraph for cross-file navigation
  - LLM code generation studies for comprehension
- Include preprints and 2024-2025 papers (field is emerging)
- Explicitly acknowledge research gaps where they exist

### Threshold Documentation
- Tie 1-10 score thresholds to empirical evidence from cited research
- When extrapolating from adjacent research, note the connection explicitly
- Mark heuristic-based thresholds as such (similar to C5 commit stability)

### Quality Standards
- Same rigor as C1-C6 despite field immaturity
- Distinguish foundational AI research from agent-specific findings
- Prioritize open-access sources (ArXiv, preprints) given recency
- DOI preferred, ArXiv acceptable for 2024+ papers

</decisions>

<specifics>
## Specific Ideas

- Follow URL verification protocol from Phase 18
- Maintain per-category References section structure
- Each quantified claim needs explicit source attribution
- Research novelty should be transparent (note when citing preprints or very recent work)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 25-c7-agent-evaluation-citations*
*Context gathered: 2026-02-05*
