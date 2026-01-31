# Phase 4: Recommendations and Output - Context

**Gathered:** 2026-01-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver a polished terminal report that displays composite and category scores with tier ratings, plus actionable improvement recommendations ranked by impact. Includes CI gating via --threshold flag and detailed metric breakdowns via --verbose mode.

This phase presents existing data from Phase 3 scoring. New metrics or scoring changes belong in other phases.

</domain>

<decisions>
## Implementation Decisions

### Terminal presentation
- **Layout**: Top-down summary structure - composite score and tier at top, then category breakdowns, then metric details below
- **Emphasis**: Color-coded scores with symbols (green/yellow/red colors plus ✓/⚠/✗ symbols) for quick visual scanning
- **Metric organization**: Table format for category scores and metrics - aligned columns for easy reading
- **Default info**: Just scores (composite, tier, category scores) - keep minimal, full metrics only in --verbose

### Recommendation content
- **Framing**: Agent-readiness focused - frame as "This blocks agents because..." or "Agents struggle when..." to directly tie to AI assistant usability
- **Details included**:
  - Specific metric values: Show actual numbers like 'Avg complexity: 12.3 (threshold: 10)'
  - Estimated impact: Show predicted score improvement
  - Concrete action: What to actually do, not just generic advice
- **Action specificity**: Claude's discretion - balance between naming specific functions vs pointing to problem areas
- **Why explanation**: Claude's discretion - decide whether to include brief context about agent impact

### Ranking and prioritization
- **Primary ranking factor**: Estimated score impact - rank by predicted composite score improvement, biggest gains first
- **Impact estimation**: Claude's discretion - choose approach that balances accuracy and complexity
- **Recommendation count**: Top 5 by default
- **Grouping**: Claude's discretion - choose organization (by category, by metric, or flat list) that helps users prioritize

### Verbosity and detail levels
- **--verbose adds**:
  - Full metric breakdown: all collected metrics with values
  - Per-metric scores: the 1-10 score for each individual metric
  - Scoring calculation details: how category scores were calculated from metrics
- **Additional formats**: JSON output via --json flag for machine-readable output (CI integration, tooling)
- **--threshold behavior**: Claude's discretion - decide on messaging and output when threshold not met
- **JSON + verbose interaction**: Claude's discretion - decide whether --verbose affects JSON output structure

</decisions>

<specifics>
## Specific Ideas

No specific requirements - open to standard approaches within the decisions above.

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope.

</deferred>

---

*Phase: 04-recommendations-and-output*
*Context gathered: 2026-01-31*
