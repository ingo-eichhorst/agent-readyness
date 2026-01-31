# Phase 5: Hardening - Context

**Gathered:** 2026-01-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Make the ARS tool production-ready by gracefully handling real-world edge cases (symlinks, syntax errors, Unicode paths) and ensuring good performance on large codebases (50k LOC target: <30s). Progress indicators provide user feedback during long-running scans.

</domain>

<decisions>
## Implementation Decisions

### Error Handling Strategy
- **Symlinks**: Claude's discretion on whether to follow, skip with warning, or skip silently
- **Syntax errors**: Claude's discretion on whether to fail scan, skip file and continue, or include in counts but not analysis
- **Unicode paths**: Claude's discretion on level of support (full, skip with warning, or error)
- **Error reporting**: Claude's discretion on immediate logging vs end summary vs both

**Philosophy**: Tool should be resilient and continue producing results when possible, not fragile. Edge cases shouldn't prevent analyzing the rest of a valid codebase.

### Progress Indicators
- **Style**: Claude's discretion on spinner vs progress bar vs file counter
- **Update frequency**: Claude's discretion on per-file, per-package, or per-phase granularity
- **Time estimates**: No ETA display - only show progress completion, not time remaining
- **Completion behavior**: Claude's discretion on clearing, leaving final state, or replacing with summary

**Philosophy**: Progress is about user reassurance during long operations, not precise metrics. Keep it simple and non-distracting.

### Claude's Discretion
- Exact error handling approaches for all edge case categories
- Progress indicator implementation details (style, frequency, completion)
- Performance optimization strategies (parallel processing, caching, early exits, memory)
- Which additional edge cases to handle beyond the three listed (empty repos, monorepos, non-standard structures)

</decisions>

<specifics>
## Specific Ideas

- 30-second target for 50k LOC repositories is the performance benchmark
- Progress matters most for "long-running scans" - tool should detect when to show progress (threshold?)
- Unicode path handling should consider that Go itself has good Unicode support

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope

</deferred>

---

*Phase: 05-hardening*
*Context gathered: 2026-01-31*
