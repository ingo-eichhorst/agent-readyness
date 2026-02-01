---
phase: 07-python-typescript-c1-c3-c6
plan: 01
subsystem: analysis
tags: [tree-sitter, python, cyclomatic-complexity, import-graph, testing, multi-language]

requires:
  - phase: 06-multi-language-foundation
    provides: Tree-sitter parser infrastructure, AnalysisTarget dispatch, C2 Python analyzer pattern
provides:
  - Python C1 analysis (complexity, function length, file size, duplication)
  - Python C3 analysis (import graph, dead code, directory depth)
  - Python C6 analysis (test detection, assertions, isolation)
  - Language dispatcher pattern in C1/C3/C6 analyzers
  - NewC1Analyzer, NewC3Analyzer, NewC6Analyzer constructors
  - Thread-safe TreeSitterParser with mutex
affects: [07-02 TypeScript analysis, future C4/C5/C7 multi-language support]

tech-stack:
  added: []
  patterns:
    - "Language dispatcher: each analyzer dispatches to language-specific functions via switch target.Language"
    - "GoAwareAnalyzer bridge: Go analysis via SetGoPackages, Python/TS via AnalysisTarget"
    - "pyFilter pattern: pyFilterSourceFiles separates test from source files"

key-files:
  created:
    - internal/analyzer/c1_python.go
    - internal/analyzer/c1_python_test.go
    - internal/analyzer/c3_python.go
    - internal/analyzer/c3_python_test.go
    - internal/analyzer/c6_python.go
    - internal/analyzer/c6_python_test.go
    - testdata/valid-python-project/utils.py
  modified:
    - internal/analyzer/c1_codehealth.go
    - internal/analyzer/c3_architecture.go
    - internal/analyzer/c6_testing.go
    - internal/pipeline/pipeline.go
    - internal/parser/treesitter.go
    - internal/discovery/walker_test.go
    - testdata/valid-python-project/test_app.py

key-decisions:
  - "Language dispatch via switch/case in each analyzer's Analyze method (matches C2 pattern)"
  - "Extract Go logic into private methods (analyzeGoC1, analyzeGoC3, analyzeGoC6) for clean separation"
  - "Thread-safe TreeSitterParser: added sync.Mutex to ParseFile since tree-sitter parsers are not goroutine-safe"
  - "Python test detection: file-path-based (test_*.py, *_test.py) matching pytest conventions"
  - "Cyclomatic complexity: base 1, skip nested function_definition nodes to avoid double-counting"

patterns-established:
  - "NewCxAnalyzer(tsParser) constructor: all analyzers now use constructor that accepts TreeSitterParser"
  - "pyFilterSourceFiles: reusable filter for separating test vs source parsed files"
  - "isTestFileByPath: centralized test file classification for Python"

duration: 10min
completed: 2026-02-01
---

# Phase 7 Plan 1: Python C1/C3/C6 Analysis Summary

**Python code health, architecture, and testing analyzers via Tree-sitter with language-dispatching refactor of C1/C3/C6**

## Performance

- **Duration:** 10 min
- **Started:** 2026-02-01T22:05:38Z
- **Completed:** 2026-02-01T22:15:21Z
- **Tasks:** 3
- **Files modified:** 13

## Accomplishments
- C1 Python analyzer: cyclomatic complexity, function length, file size, duplication detection
- C3 Python analyzer: import graph with intra-project tracking, dead code detection, directory depth
- C6 Python analyzer: test function detection, assertion counting, isolation scoring
- Refactored all three analyzers (C1/C3/C6) from Go-only to multi-language dispatch
- Pipeline wired with NewC1Analyzer, NewC3Analyzer, NewC6Analyzer constructors
- End-to-end: `ars scan` produces C1/C2/C3/C6 scores for Python projects

## Task Commits

1. **Task 1: Refactor C1 dispatcher + Python C1** - `3b1317b` (feat)
2. **Task 2: Refactor C3 dispatcher + Python C3** - `10ebc48` (feat)
3. **Task 3: Refactor C6 dispatcher + Python C6 + wire pipeline** - `9c55f6e` (feat)

## Files Created/Modified
- `internal/analyzer/c1_python.go` - Python C1 analysis (complexity, file size, duplication)
- `internal/analyzer/c1_python_test.go` - Tests for Python C1
- `internal/analyzer/c3_python.go` - Python C3 analysis (import graph, dead code, depth)
- `internal/analyzer/c3_python_test.go` - Tests for Python C3
- `internal/analyzer/c6_python.go` - Python C6 analysis (test detection, assertions, isolation)
- `internal/analyzer/c6_python_test.go` - Tests for Python C6
- `internal/analyzer/c1_codehealth.go` - Added tsParser field, NewC1Analyzer, language dispatch
- `internal/analyzer/c3_architecture.go` - Added tsParser field, NewC3Analyzer, language dispatch
- `internal/analyzer/c6_testing.go` - Added tsParser field, NewC6Analyzer, language dispatch
- `internal/pipeline/pipeline.go` - Wired NewC1/C3/C6Analyzer(tsParser)
- `internal/parser/treesitter.go` - Added sync.Mutex for thread safety
- `testdata/valid-python-project/utils.py` - Complex Python test fixture
- `testdata/valid-python-project/test_app.py` - Extended with 6 test functions

## Decisions Made
- Used switch/case language dispatch in each analyzer (matching C2 pattern from Phase 6)
- Extracted Go logic into private methods to keep dispatchers clean
- Added mutex to TreeSitterParser to prevent concurrent parsing crashes
- Python test detection uses file path conventions (test_*.py, *_test.py, conftest.py)
- Complexity counting skips nested function_definition to avoid double-counting

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TreeSitterParser concurrent access crash (SIGSEGV)**
- **Found during:** Task 3 (end-to-end smoke test)
- **Issue:** Tree-sitter parsers are not thread-safe; 4 analyzers parsing Python concurrently caused segfault
- **Fix:** Added sync.Mutex to TreeSitterParser.ParseFile() to serialize all parse operations
- **Files modified:** internal/parser/treesitter.go
- **Verification:** E2E scan completes successfully; all tests pass
- **Committed in:** 9c55f6e (Task 3 commit)

**2. [Rule 3 - Blocking] Discovery test failure from added testdata**
- **Found during:** Task 3 (full test suite verification)
- **Issue:** Adding utils.py to testdata/valid-python-project changed file counts expected by walker_test.go
- **Fix:** Updated TestDiscoverPythonProject assertions: SourceCount 1->2, TotalFiles 2->3, PerLanguage 1->2
- **Files modified:** internal/discovery/walker_test.go
- **Verification:** Full test suite passes
- **Committed in:** 9c55f6e (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both fixes necessary for correctness. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Python C1/C3/C6 analysis complete and producing scores
- Ready for Plan 02: TypeScript C1/C3/C6 analysis (same dispatcher pattern, just needs TS-specific functions)
- All dispatchers already have `case types.LangTypeScript:` placeholder comments

---
*Phase: 07-python-typescript-c1-c3-c6*
*Completed: 2026-02-01*
