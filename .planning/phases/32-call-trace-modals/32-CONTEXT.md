# Phase 32: Call Trace Modals - Context

**Gathered:** 2026-02-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Add "View Trace" buttons to every metric row in HTML reports. Clicking opens a modal showing exactly how that metric's score was derived. C7 (LLM-based) traces show prompt/response/score breakdown. C1-C6 (rule-based) traces show scoring breakpoints, current value, and top-5 worst offenders. This phase makes scoring transparent, not the scoring algorithm itself.

</domain>

<decisions>
## Implementation Decisions

### Trace content structure (C7 - LLM-based metrics)
- Score breakdown appears first (matched indicators with checklist format: ✓/✗)
- Full prompt and full response in collapsible sections below the score
- Checklist format for matched indicators: visual checkmarks/crosses with final score
- No truncation — show complete prompts and responses regardless of length

### Trace content structure (C1-C6 - rule-based metrics)
- Breakpoint context appears first (scoring scale table showing ranges and scores)
- Highlight the current band in the breakpoint table (where this metric's value landed)
- Top-5 worst offenders displayed below the breakpoints as supporting evidence
- Full file paths shown (no abbreviation)

### Technical content presentation
- Syntax highlighting: subtle/minimal (2-3 colors max to distinguish keys from values)
- Copy buttons: add "Copy" button to each code block (prompts, responses, shell commands)
- File paths: show complete paths (e.g., 'internal/analyzer/c1_code_quality/complexity.go:145')
- Breakpoint table: highlight current band with background color or bold text

### Size budget handling
- **No size limits** — 500KB budget removed from success criteria
- Informational warning: show file size when generating HTML reports (terminal output)
- Always embed: all C7 trace data embedded in HTML (self-contained, no external files)
- No truncation strategy needed

### Claude's Discretion
- Exact color choices for syntax highlighting (within subtle/minimal constraint)
- Collapsible section implementation (expand/collapse icons, animation)
- Copy button placement and styling
- Warning format for large file size reporting

</decisions>

<specifics>
## Specific Ideas

- C7 score breakdown uses checklist format like: "✓ Clear structure  ✗ No examples  (score: 6/10)"
- Breakpoint tables should feel like scoring rubrics — clear bands with visual emphasis on where you landed
- Copy buttons for code blocks so users can extract prompts/responses easily for debugging

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 32-call-trace-modals*
*Context gathered: 2026-02-06*
