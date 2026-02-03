# Phase 15: Claude Code Integration - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

All LLM features use Claude Code CLI, eliminating Anthropic SDK dependency. C4 documentation analysis switches from SDK to CLI. C7 agent evaluation continues working. LLM analysis auto-detects CLI availability. Remove ANTHROPIC_API_KEY requirement. Remove Anthropic SDK from go.mod.

</domain>

<decisions>
## Implementation Decisions

### Auto-detection behavior
- Check PATH at startup (`which claude`), cache result for entire scan
- Run `claude --version` to verify CLI responds (not just exists)
- If CLI not found or version check fails: warn and skip LLM analysis
- Add `--no-llm` flag to force-disable LLM features even when CLI is available
- Check for minimum CLI version, warn/error if too old

### Error handling & messages
- On CLI call failure: retry once, then fail gracefully (skip analysis with warning)
- Error messages should be technical: include stderr, exit codes, command that failed
- If CLI not found: include install hint with URL in warning message
- Fixed timeout for LLM responses (e.g., 60s per call), fail if exceeded

### Output consistency
- Exact score parity required: same inputs must produce same scores
- Adapt prompts if needed to work with CLI's output format (not locked to SDK prompts)
- Output format is same regardless of LLM usage — no "llm_enhanced" indicator
- Trust the migration: no comparison mode between SDK and CLI needed

### Migration experience
- Remove `--enable-c4-llm` flag entirely — unknown flag error if used
- No mention of ANTHROPIC_API_KEY if set — just ignore it
- Brief changelog mention: "Switched to Claude CLI for LLM features"

### Claude's Discretion
- Exact timeout value (suggested 60s)
- Specific prompt adjustments for CLI output parsing
- Retry delay between attempts
- Format of install hint URL

</decisions>

<specifics>
## Specific Ideas

- Clean break: remove old flag and SDK entirely, don't maintain backward compatibility
- Technical users prefer seeing error details, not sanitized messages
- The CLI detection at startup means fast failure — user knows immediately if LLM features unavailable

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 15-claude-code-integration*
*Context gathered: 2026-02-03*
