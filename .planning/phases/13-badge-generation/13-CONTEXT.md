# Phase 13: Badge Generation - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Generate shields.io badge URLs that display ARS tier and score for embedding in READMEs. Badge color reflects tier classification. Output integrates with existing CLI output modes (terminal, JSON, HTML).

</domain>

<decisions>
## Implementation Decisions

### Output format
- Markdown image format ready to paste: `![ARS](https://img.shields.io/...)`
- `--badge` flag appends badge markdown to normal terminal output (not standalone)
- JSON output includes `badge_url` and `badge_markdown` fields when `--badge` is used
- HTML reports include a "Copy badge" section with the markdown for easy copying

### Badge content
- Label: "ARS" (short form)
- Message: Tier name + score with one decimal, e.g., "Agent-Ready 8.2/10"
- Colors: red (Agent-Hostile), orange (Agent-Limited), yellow (Agent-Assisted), green (Agent-Ready)
- Badge links to ARS repo when clicked (wrapped in markdown link)

### Claude's Discretion
- shields.io URL parameter formatting
- Exact color hex codes for each tier
- How the "Copy badge" section appears in HTML

</decisions>

<specifics>
## Specific Ideas

- Final markdown output should look like: `[![ARS](https://img.shields.io/badge/ARS-Agent--Ready%208.2%2F10-green)](https://github.com/ingo-eichhorst/agent-readyness)`
- One decimal precision matches the existing scan output format

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 13-badge-generation*
*Context gathered: 2026-02-03*
