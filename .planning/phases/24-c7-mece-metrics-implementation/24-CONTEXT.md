# Phase 24: C7 MECE Metrics Implementation - Context

**Gathered:** 2026-02-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace C7's single `overall_score` metric with 5 MECE (Mutually Exclusive, Collectively Exhaustive) agent-assessable metrics: Task Execution Consistency, Code Behavior Comprehension, Cross-File Navigation, Identifier Interpretability, Documentation Accuracy Detection. This phase implements the metrics within the existing C7 analyzer structure, maintaining backward compatibility. Citations and research validation are handled in Phase 25.

</domain>

<decisions>
## Implementation Decisions

### Metric Set & Boundaries
- **5 metrics confirmed:** Task Execution Consistency, Code Behavior Comprehension, Cross-File Navigation, Identifier Interpretability, Documentation Accuracy Detection
- **Isolated abilities:** Each metric tests one narrow capability, not compound scenarios — easier to debug failures and understand what each measures
- **MECE framework:** Claude uses research-backed framework (SWE-bench, RepoGraph, etc.) to define boundaries and ensure mutual exclusivity
- **Evolve existing system:** Extend/evolve current 4-task system (intent_clarity, modification_confidence, cross_file_coherence, semantic_completeness) rather than replacing entirely — preserve task execution infrastructure where applicable

### Test Selection Strategy
- **Reproducible selection:** Claude decides strategy that balances reproducibility and coverage (deterministic heuristics vs stratified sampling)
- **Sample count:** Claude determines count per metric based on metric variance and cost constraints (likely 1-5 samples per metric)
- **Independent from C1-C6:** C7 selects samples independently — avoids circular dependencies, simpler implementation
- **Language-agnostic strategy:** Same selection strategy across all languages (Go/Python/TypeScript) — simpler, more consistent

### Scoring & Thresholds
- **1-10 scale:** Matches C1-C6 metrics, aligns with roadmap requirement
- **Weighted average aggregation:** C7 category score = weighted average of 5 metric scores, where weights reflect research-derived impact of each metric on agent success
- **Per-metric threshold strategy:** Each metric can use different threshold approach (binary pass/fail, benchmark percentiles, quantitative rubrics) — follow research references to determine appropriate method per metric
- **Failure handling:** Claude decides how to score timeouts/errors/refusals based on what's most informative to users

### Execution & Progress UX
- **Parallel execution:** All 5 metrics run concurrently using Claude Code agent spawning — faster, leverages roadmap suggestion
- **Real-time progress display:** Show updates as agent activity happens, including:
  - Current metric name (e.g., "Testing Task Execution Consistency...")
  - Progress per metric (e.g., "Sample 2/5")
  - Running token counter to track LLM usage
- **Variable timeouts:** Complex metrics get more time, simple metrics have shorter timeouts — adapts to metric difficulty

### Claude's Discretion
- MECE framework selection (research-based boundary definition)
- Sample selection algorithm (heuristic vs stratified)
- Sample count per metric (variance/cost tradeoff)
- Failure scoring policy (how to handle timeouts/errors)
- Update frequency within real-time display
- Specific timeout values per metric
- Weight values for weighted average (derived from research)

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches guided by research evidence (SWE-bench, RepoGraph, agent benchmark literature).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 24-c7-mece-metrics-implementation*
*Context gathered: 2026-02-05*
