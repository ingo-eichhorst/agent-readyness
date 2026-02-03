# Phase 14: HTML Enhancements - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Enhance HTML reports with expandable, research-backed metric descriptions. Each metric shows a brief description (always visible) and a detailed expandable section with research citations. CSS-only implementation (no JavaScript). This phase does NOT add new metrics or change scoring logic.

</domain>

<decisions>
## Implementation Decisions

### Brief Descriptions
- Focus: Both what it measures AND why it matters, combined in one sentence
- Tone: Action-oriented ("Simpler functions help agents complete tasks faster")
- Include specific thresholds in the brief description ("Keep under 10 for best agent performance")
- Position: Below the metric value (Score: 7.2 [newline] description)

### Expandable Content
- Research citations: Full academic style (author names, year, paper title, specific findings with numbers)
- Structure: Structured sections with headers ("Definition", "Impact", "Research", "Thresholds")
- Include actionable recommendations: How-to-improve guidance within each expanded section
- Length: Comprehensive (3+ paragraphs) — full educational content with all research details

### Auto-expand Behavior
- Trigger: Low individual metrics expand automatically (not category-level)
- Threshold: Claude's discretion per metric type (appropriate threshold varies by metric)
- All metrics are expandable regardless of score (good scores just start collapsed)
- Add "Expand all / Collapse all" control

### Visual Presentation
- Expand indicator: Chevron/arrow (▶ collapsed, ▼ expanded)
- Expanded content styling: Indented block with left border or background tint
- Citations: Inline parenthetical format (Borg et al., 2026)
- Expand/collapse all control: Single control at top of report affecting all sections

### Claude's Discretion
- Exact CSS styling (colors, spacing, border styles)
- Auto-expand threshold per metric type
- Wording of brief descriptions and expanded content

</decisions>

<specifics>
## Specific Ideas

- Brief descriptions should be scannable — user glances at score, sees immediately why it matters
- Expanded sections structured like mini-articles: Definition → Impact → Research → Recommendations
- Research should cite Borg et al. (2026) and other relevant papers with specific numbers (e.g., "32.8% improvement in task completion")

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 14-html-enhancements*
*Context gathered: 2026-02-03*
