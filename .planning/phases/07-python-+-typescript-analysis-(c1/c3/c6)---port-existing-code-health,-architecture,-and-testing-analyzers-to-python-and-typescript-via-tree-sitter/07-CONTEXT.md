# Phase 7: Python + TypeScript Analysis (C1/C3/C6) - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Extend C1 (Code Health), C3 (Architecture), and C6 (Testing) analyzers to Python and TypeScript using Tree-sitter parsing. This phase ports the three core analysis categories that already work for Go — no new analysis categories, no new languages beyond Python and TypeScript.

</domain>

<decisions>
## Implementation Decisions

### Test Framework Detection and Execution

- **Python frameworks:** Support pytest + unittest (covers ~90% of Python projects)
- **TypeScript frameworks:** Support Jest + Vitest + Mocha (comprehensive coverage including legacy)
- **Test detection strategy:** Hybrid approach — try config first, fall back to naming patterns (*_test.py, *.test.ts, __tests__/*), optionally verify with AST scanning
- **Coverage generation:** ARS actively runs tests to generate fresh coverage data (for Python, TypeScript, and Go)
- **Test failure handling:** If tests fail or timeout, skip C6 analysis and continue with other categories (C1, C3). Analysis completes despite test issues.

### Architecture Analysis Scope

- **TypeScript module systems:** Parse both ESM (import) and CommonJS (require) styles universally, regardless of package.json config. Handles mixed codebases.
- **Directory depth thresholds:** Same thresholds across all languages (4+ levels = penalty). Consistent scoring.

### Complexity Metrics Mapping

- **Cyclomatic complexity:** Identical control flow rules across languages — if/for/while/case all add +1 complexity regardless of language. Ensures consistent scoring.
- **Function length:** Same 50-line threshold for all languages (Go, Python, TypeScript). No language-specific adjustments.
- **Decorator handling:** Analyze decorator impact on complexity — decorators like @retry, error handlers add implicit complexity and should adjust metrics accordingly (not just count lines or ignore).

### Claude's Discretion

- Python module boundary definition (packages vs all directories with .py files)
- Dead code detection strategy (balancing false positives vs detection accuracy)
- Whether async/await patterns should affect complexity scoring
- Exact thresholds and heuristics for AST-based test detection
- Performance optimizations for Tree-sitter parsing

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches guided by Tree-sitter capabilities and ecosystem conventions.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 07-python-+-typescript-analysis-(c1/c3/c6)*
*Context gathered: 2026-02-01*
