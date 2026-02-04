# Phase 16: Analyzer Reorganization - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Reorganize analyzer code into category subdirectories for improved navigability. Move existing files without changing functionality. Maintain backward compatibility through re-exports.

</domain>

<decisions>
## Implementation Decisions

### Directory structure
- Category directories use mixed number-name format: `c1_code_quality/`, `c2_semantics/`, etc.
- Shared utilities stay at root level (`internal/analyzer/`)
- Language-specific files live with their category (e.g., `python.go` in `c1_code_quality/`)
- Test files move with their implementation

### File naming
- Drop category prefix when moving (e.g., `c1_codehealth.go` becomes `codehealth.go`)
- Language-specific files use just the language name (e.g., `c1_python.go` becomes `python.go`)
- Each category has `analyzer.go` as the main entry point
- Package names use short form: `package c1`, `package c2`, etc.

### Claude's Discretion
- Re-export strategy at root level
- Migration approach (all at once vs incremental)
- Exact mapping of current files to new locations
- How to handle any edge cases with imports

</decisions>

<specifics>
## Specific Ideas

- Directory names should clearly indicate both the category number and its purpose (e.g., `c1_code_quality` not just `c1`)
- Keep the codebase working throughout the migration

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 16-analyzer-reorganization*
*Context gathered: 2026-02-04*
