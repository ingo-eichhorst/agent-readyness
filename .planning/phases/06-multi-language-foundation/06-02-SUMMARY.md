---
phase: 06-multi-language-foundation
plan: 02
subsystem: pipeline
tags: [tree-sitter, multi-language, python, typescript, discovery, parsing, cgo]

dependency-graph:
  requires:
    - phase: 06-01
      provides: AnalysisTarget type, Language constants, GoAwareAnalyzer bridge
  provides:
    - Multi-language file discovery (Go, Python, TypeScript)
    - Language classifier for Python and TypeScript test patterns
    - DetectProjectLanguages helper
    - TreeSitterParser with pooled Python/TypeScript/TSX parsers
    - Test fixtures for Python, TypeScript, and polyglot projects
  affects: [06-03, 06-04, 07, 08, 09, 10]

tech-stack:
  added:
    - go-tree-sitter v0.25.0
    - tree-sitter-python v0.25.0
    - tree-sitter-typescript v0.23.2
  patterns:
    - Pooled Tree-sitter parsers with explicit Close() lifecycle
    - Extension-based language detection in walker
    - Language-specific classifier dispatch

key-files:
  created:
    - internal/parser/treesitter.go
    - internal/parser/treesitter_test.go
    - testdata/valid-python-project/app.py
    - testdata/valid-python-project/test_app.py
    - testdata/valid-python-project/pyproject.toml
    - testdata/valid-ts-project/src/index.ts
    - testdata/valid-ts-project/src/index.test.ts
    - testdata/valid-ts-project/tsconfig.json
    - testdata/valid-ts-project/package.json
    - testdata/polyglot-project/main.go
    - testdata/polyglot-project/util.py
    - testdata/polyglot-project/helper.ts
  modified:
    - internal/discovery/walker.go
    - internal/discovery/classifier.go
    - internal/discovery/walker_test.go
    - internal/discovery/classifier_test.go
    - pkg/types/types.go
    - go.mod
    - go.sum

key-decisions:
  - "Extension-based language routing in walker (langExtensions map)"
  - "Go-only generated file check (IsGeneratedFile not called for Python/TypeScript)"
  - "Separate parsers for .ts and .tsx to use correct Tree-sitter grammar"

patterns-established:
  - "TreeSitterParser.Close() and CloseAll() for memory safety"
  - "ParseFile with ext parameter to distinguish TypeScript vs TSX"
  - "PerLanguage map on ScanResult for per-language source counts"

duration: 7 min
completed: 2026-02-01
---

# Phase 6 Plan 2: Multi-Language Discovery and Tree-sitter Integration Summary

**Multi-language walker discovers .py/.ts/.tsx files, Tree-sitter parses Python and TypeScript with pooled parsers and explicit memory management**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-01T15:37:52Z
- **Completed:** 2026-02-01T15:44:47Z
- **Tasks:** 2
- **Files modified:** 19

## Accomplishments
- Walker discovers Go, Python, and TypeScript files with per-language classification and counts
- TreeSitterParser provides pooled parsers for Python, TypeScript, and TSX with explicit Close() lifecycle
- Test fixtures for Python-only, TypeScript-only, and polyglot projects enable integration testing
- DetectProjectLanguages checks for language indicators (go.mod, pyproject.toml, tsconfig.json, etc.)

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend discovery walker, classifier, and types for multi-language** - `d6e9f4f` (feat)
2. **Task 2: Create test fixtures and integrate Tree-sitter parser** - `4e1e3e4` (feat)

## Files Created/Modified
- `internal/discovery/walker.go` - Multi-language file discovery with extension-based routing
- `internal/discovery/classifier.go` - ClassifyPythonFile and ClassifyTypeScriptFile
- `internal/discovery/walker_test.go` - Tests for Python, TypeScript, polyglot discovery
- `internal/discovery/classifier_test.go` - Tests for Python and TypeScript classification
- `pkg/types/types.go` - Language field on DiscoveredFile, PerLanguage on ScanResult
- `internal/parser/treesitter.go` - TreeSitterParser with Python/TypeScript/TSX pools
- `internal/parser/treesitter_test.go` - Tests for parsing, reuse, Close, ParseTargetFiles
- `testdata/valid-python-project/` - Flask-like app with type annotations and pytest tests
- `testdata/valid-ts-project/` - Express-like app with strict TypeScript and Jest tests
- `testdata/polyglot-project/` - Go, Python, TypeScript files in one directory

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Extension-based language routing via map | Clean O(1) lookup, easy to extend for new languages |
| Go-only generated file check | Python/TypeScript have no standard "DO NOT EDIT" convention |
| Separate .ts and .tsx parsers | Tree-sitter uses different grammars for TypeScript vs TSX (JSX syntax) |
| PerLanguage as map[Language]int on ScanResult | Flexible for any number of languages, avoids separate counters |

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `tree-sitter-typescript/bindings/go` import path does not work as a separate module (go.mod declares root path). Used `github.com/tree-sitter/tree-sitter-typescript/bindings/go` import with root module `github.com/tree-sitter/tree-sitter-typescript`. Resolved by go mod tidy.

## User Setup Required

None - no external service configuration required. CGO_ENABLED=1 is required for building (already documented in STATE.md).

## Next Phase Readiness

Plan 06-03 (C2 Semantic Explicitness analyzers) can proceed. The TreeSitterParser and multi-language discovery provide the foundation for Python/TypeScript code analysis. Test fixtures are ready for analyzer testing.

---
*Phase: 06-multi-language-foundation*
*Completed: 2026-02-01*
