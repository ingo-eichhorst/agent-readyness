# Phase 7: Python + TypeScript Analysis (C1/C3/C6) - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Port existing code health (C1), architecture (C3), and testing (C6) analyzers from Go to Python and TypeScript. Users should get the same depth of analysis for Python/TS projects as they currently get for Go projects.

This phase extends static analysis to additional languages. Adding new metrics, new languages beyond Python/TS, or new analysis categories belongs in other phases.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion

**Full implementation flexibility** — User delegated all implementation decisions to Claude. Specifically:

- **Analyzer parity depth**: Determine how closely Python/TS analyzers should match Go's metrics. Decide whether to detect language-specific patterns (Python comprehensions, TS decorators) or stay structurally equivalent.

- **Tree-sitter query strategy**: Choose between pre-written queries vs node walking. Decide how to handle language idioms (async/await, decorators, comprehensions). Determine query reuse strategy between Python/TS.

- **Testing framework detection**: Select which test frameworks to support:
  - Python: pytest, unittest, or both
  - TypeScript: Jest, Mocha, Vitest — which ones?
  - Coverage format handling: coverage.py, Istanbul, lcov — parsing approach?

- **Dispatcher refactoring**: Design analyzer dispatcher routing to language-specific implementations. Choose registration pattern and fallback behavior for unsupported languages.

- **Metric calculation consistency**: Ensure Python/TS metrics are comparable to Go metrics where meaningful, while respecting language idioms.

- **Error handling**: Graceful degradation when Tree-sitter parsing fails or language features aren't supported.

</decisions>

<specifics>
## Specific Ideas

**From ROADMAP.md success criteria:**
- Python C1: cyclomatic complexity, function length, file size, duplication (comparable to Go)
- TypeScript C3: import graph, dead code, directory depth (comparable to Go)
- Python C6: pytest/unittest detection + coverage.py parsing
- TypeScript C6: Jest/Mocha/Vitest detection + Istanbul/lcov parsing

**From Phase 6 foundation:**
- Tree-sitter parsers already integrated for Python (.py) and TypeScript (.ts, .tsx)
- AnalysisTarget abstraction provides language-agnostic interface
- Pipeline auto-creates parsers; degrades gracefully if CGO unavailable

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 07-python-+-typescript-analysis-c1/c3/c6*
*Context gathered: 2026-02-01*
