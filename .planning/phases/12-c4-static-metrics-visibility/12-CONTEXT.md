# Phase 12: C4 Static Metrics Visibility - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix terminal output display for C4 documentation quality metrics. Users should see static documentation analysis (README presence, CHANGELOG presence, comment density, API doc coverage, examples presence, CONTRIBUTING presence) in terminal output without requiring `--enable-c4-llm` flag. LLM-based metrics show as "N/A" when LLM is disabled.

</domain>

<decisions>
## Implementation Decisions

### Metric display format
- Show all C4 metrics in unified table, with LLM metrics marked as "N/A" when --enable-c4-llm not used
- Static metrics: README, CHANGELOG, comment density, API docs, examples, CONTRIBUTING (always shown)
- LLM metrics: clarity, quality, completeness (shown as "N/A" or skipped when LLM disabled)
- Consistent with how C7 handles opt-in metrics

### Availability logic
- C4Analyzer reports Available=true when static metrics can be computed (always, for most repos)
- LLM client being nil should NOT make C4 unavailable
- If repo has zero files to analyze, Available=false (edge case)
- Matches pattern from other categories: available if ANY analysis possible

### Verbose mode details
- Standard mode: category score + metric table (same as C1/C2/C3/C5/C6)
- Verbose mode: add file-level documentation coverage breakdown (which files missing docs)
- Similar detail level to C1 verbose (shows per-file complexity) and C6 verbose (shows per-file test coverage)

### Empty state handling
- If no documentation exists: show 0% scores for applicable metrics
- Don't skip the category — show the gap so users know to improve
- Matches C6 behavior (shows 0% coverage when no tests exist)
- Error state only if analysis crashes, not if documentation is absent

### Claude's Discretion
- Exact color thresholds for C4 scores in terminal (can match existing category patterns)
- Table formatting details (spacing, alignment)
- Error message wording

</decisions>

<specifics>
## Specific Ideas

- Terminal rendering should match existing category display patterns (see renderCategoryScores in internal/output/terminal.go)
- C4Analyzer.Analyze() needs conditional logic: if llmClient == nil, skip LLM metrics but still compute static metrics
- Look at C7 terminal rendering (11-01-PLAN.md just implemented this) for reference on opt-in metrics display

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 12-c4-static-metrics-visibility*
*Context gathered: 2026-02-03*
