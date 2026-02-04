# Phase 17: README & Testing - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Add standard status badges to README and update test documentation to include coverage profiling. Enables coverage data for C6 self-analysis.

</domain>

<decisions>
## Implementation Decisions

### Badge placement & order
- Badges immediately after the H1 title line
- Single line, inline (not stacked)
- Order: Go Reference → Go Report Card → License → Release

### Badge styling
- Flat style (shields.io default)
- Standard colors (no custom overrides)
- Link each badge to its source (pkg.go.dev, goreportcard.com, etc.)

### Test command documentation
- Update existing CLAUDE.md "Build & Test Commands" section
- Add `-coverprofile=coverage.out` to test examples
- No Makefile (keep simple, commands already documented)

### Claude's Discretion
- Exact badge markdown syntax
- Whether to add alt text to badges
- Formatting/whitespace around badge line

</decisions>

<specifics>
## Specific Ideas

No specific requirements — standard Go project conventions apply.

</specifics>

<deferred>
## Deferred Ideas

- Coverage badge (requires CI setup) — future milestone
- Codecov/Coveralls integration — future milestone

</deferred>

---

*Phase: 17-readme-testing*
*Context gathered: 2026-02-04*
