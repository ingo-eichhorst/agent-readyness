# Phase 30: Evidence Data Flow - Context

**Gathered:** 2026-02-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Extend the scoring pipeline to capture and flow evidence data (top-5 worst offenders per metric) from analyzers through to JSON output. This makes evidence available for downstream modal rendering without changing scoring logic, adding new metrics, or building UI components.

</domain>

<decisions>
## Implementation Decisions

### Evidence selection criteria
- Fixed limit: Always top-5 offenders per metric (not configurable)
- Sorting: Claude's discretion on per-metric severity ordering (use natural scale per metric)
- Tie-breaking: Claude's discretion (choose a reasonable deterministic approach)
- Normalization: Claude's discretion on handling different scales across metrics

### C7 evidence handling
- No evidence arrays for C7 metrics (C7 uses existing C7DebugSample data for trace modals)
- Remove overall_score metric entirely in Phase 30 (not just zero-weight)
- Final C7 state: 5 MECE metrics only (overall_score was redundant with category score)
- C7 extractors return empty evidence arrays for the 5 MECE metrics (structural consistency)

### Fallback behavior
- No offenders: Return empty array [] (field always present, not omitted)
- Partial evidence: Return what we got (even 2 items is better than none)
- Extraction errors: Log to stderr (visible during scan for debugging)
- Git unavailable (C5): Empty evidence array + log "C5 evidence unavailable: not a git repo"

### Claude's Discretion
- Exact tie-breaking logic per metric type
- Whether to normalize values across different metric scales
- Evidence extraction error message formatting

</decisions>

<specifics>
## Specific Ideas

- Evidence fields per item expected: file_path, line, value, description (from requirements)
- C7DebugSample type from v0.0.5 already captures prompts/responses for trace modals
- Success criterion: `ars scan . --json | jq '.categories[0].sub_scores[0].evidence'` returns evidence array

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 30-evidence-data-flow*
*Context gathered: 2026-02-06*
