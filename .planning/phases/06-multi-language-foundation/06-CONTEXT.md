# Phase 6: Multi-Language Foundation + C2 Semantic Explicitness - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Transform ARS from a Go-only structural analyzer to support multi-language analysis (Go, Python, TypeScript) with C2 semantic explicitness scoring and a configurable scoring system. This phase establishes the foundation for language-agnostic analysis while implementing the first new category (C2) across all three languages.

</domain>

<decisions>
## Implementation Decisions

### Language Detection Strategy
- File extension-based detection only (.go, .py, .ts, .tsx)
- No content inspection or shebang parsing - extensions are sufficient
- Analyze all found languages regardless of file count or LOC percentage
- No minimum threshold - even a single file in a language triggers analysis
- JavaScript and TypeScript treated as separate languages (.js separate from .ts/.tsx)

### Directory Filtering
- Standard exclusions applied: node_modules, vendor, .git, __pycache__, dist, build
- Common artifact and dependency directories automatically skipped
- No custom .gitignore parsing or user-configurable exclusions in this phase

### Claude's Discretion
- Config file format (.arsrc.yml structure and validation)
- C2 metric presentation in CLI output
- Error handling when Tree-sitter parsing fails
- Specific threshold values for C2 metrics (type coverage %, naming consistency patterns)
- Performance optimization approaches to meet 30-second budget

</decisions>

<specifics>
## Specific Ideas

No specific requirements - open to standard approaches for areas not discussed.

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope.

</deferred>

---

*Phase: 06-multi-language-foundation*
*Context gathered: 2026-02-01*
