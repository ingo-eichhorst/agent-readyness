# Phase 34: Testing & Quality - Context

**Gathered:** 2026-02-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Cross-cutting test suite validating evidence extraction, file size budget, JSON compatibility, prompt coverage, accessibility, and responsive layout across all features built in v0.0.6. This phase tests what's already built in phases 30-33, not adding new capabilities.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion

All implementation decisions for this phase:
- **Test organization** - Follow existing Go patterns (colocated `*_test.go`, use `testdata/` for fixtures)
- **Assertion style** - Match existing test rigor (exact value checks where deterministic, presence checks for variable output)
- **Test data strategy** - Reuse existing `testdata/` fixtures where possible, create minimal new ones only if needed
- **Coverage expectations** - Hard failures for schema breakage and missing prompt templates; warnings for file size trends; allow empty evidence for categories with no violations
- **Test types** - Mix of unit tests (evidence extraction logic), integration tests (full pipeline with assertions on JSON/HTML output), and validation tests (schema compatibility, accessibility attributes)

</decisions>

<specifics>
## Specific Ideas

No specific requirements — follow existing test patterns in the codebase:
- Colocated tests (`internal/output/html_test.go`, `internal/analyzer/c1_code_quality/go_test.go`)
- Testdata fixtures (`testdata/valid-go-project/`, `testdata/c7_responses/`)
- Table-driven tests where applicable
- Integration tests that exercise full pipeline (scan → analyze → output)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 34-testing-and-quality*
*Context gathered: 2026-02-07*
