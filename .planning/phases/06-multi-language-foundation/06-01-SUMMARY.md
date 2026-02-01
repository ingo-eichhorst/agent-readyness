---
phase: 06-multi-language-foundation
plan: 01
subsystem: pipeline
tags: [refactoring, multi-language, types, interfaces]

dependency-graph:
  requires: [01-foundation, 02-core-analysis]
  provides: [AnalysisTarget-type, GoAwareAnalyzer-interface, language-agnostic-pipeline]
  affects: [06-02, 06-03, 06-04, 07, 08, 09, 10]

tech-stack:
  added: []
  patterns: [GoAwareAnalyzer-bridge-pattern, language-agnostic-targets]

key-files:
  created: []
  modified:
    - pkg/types/types.go
    - internal/pipeline/interfaces.go
    - internal/pipeline/pipeline.go
    - internal/analyzer/c1_codehealth.go
    - internal/analyzer/c3_architecture.go
    - internal/analyzer/c6_testing.go
    - internal/analyzer/helpers.go
    - internal/pipeline/pipeline_test.go
    - internal/analyzer/c1_codehealth_test.go
    - internal/analyzer/c3_architecture_test.go
    - internal/analyzer/c6_testing_test.go

decisions:
  - id: bridge-pattern
    description: "GoAwareAnalyzer bridge pattern: analyzers store Go packages via SetGoPackages, Analyze receives language-agnostic targets"
    rationale: "Allows gradual migration -- Go analyzers keep working via stored packages while new analyzers use AnalysisTarget"
  - id: stub-parser-removed
    description: "Removed StubParser from interfaces.go"
    rationale: "Multi-parser approach replaces the single Parser interface pattern; StubParser was only used in one test"

metrics:
  duration: 4 min
  completed: 2026-02-01
---

# Phase 6 Plan 1: Language-Agnostic Foundation Types Summary

**One-liner:** Introduced AnalysisTarget/SourceFile/Language types and GoAwareAnalyzer bridge pattern to decouple pipeline from Go-specific ParsedPackage.

## What Was Done

### Task 1: Add multi-language types and refactor interfaces
- Added `Language` type with `LangGo`, `LangPython`, `LangTypeScript` constants
- Added `AnalysisTarget` struct with Language, RootDir, Files fields
- Added `SourceFile` struct with Path, RelPath, Language, Lines, Content, Class
- Changed `Analyzer.Analyze` signature from `[]*parser.ParsedPackage` to `[]*types.AnalysisTarget`
- Added `GoAwareAnalyzer` interface embedding `Analyzer` with `SetGoPackages` method
- Updated `StubAnalyzer` to match new signature
- Removed `StubParser` (replaced by multi-parser approach)

### Task 2: Refactor analyzers and pipeline
- C1Analyzer, C3Analyzer, C6Analyzer all implement `GoAwareAnalyzer`
- Each stores `[]*parser.ParsedPackage` via `SetGoPackages` and uses them in `Analyze`
- Pipeline creates `[]*types.AnalysisTarget` from parsed Go packages via `buildGoTargets`
- Pipeline calls `SetGoPackages` on any analyzer implementing `GoAwareAnalyzer` before `Analyze`
- All internal analysis logic unchanged -- purely interface refactoring
- All 7 test packages pass, `ars scan` produces identical output

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| GoAwareAnalyzer bridge pattern | Allows Go analyzers to keep using ParsedPackage while new language analyzers use AnalysisTarget directly |
| Removed StubParser | Multi-parser approach replaces single Parser interface; StubParser had minimal test coverage |
| SourceFile.Content/Lines left empty for Go | Go analyzers use ParsedPackage ASTs, not raw content; Content populated later for Tree-sitter languages |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated test files for new signatures**
- **Found during:** Task 2
- **Issue:** Test helper types (errorAnalyzer, slowAnalyzer) and test calls used old ParsedPackage signature
- **Fix:** Updated all test files to use SetGoPackages + Analyze(nil) pattern; removed StubParser test
- **Files modified:** pipeline_test.go, c1_codehealth_test.go, c3_architecture_test.go, c6_testing_test.go
- **Commit:** 1ce85b2

## Verification Results

- `go build ./...` -- passes
- `go test ./...` -- all 7 test packages pass
- `go vet ./...` -- no issues
- `ars scan .` -- produces C1/C3/C6 scores, composite 8.1, Agent-Ready rating
- AnalysisTarget type exists with Language, RootDir, Files fields
- GoAwareAnalyzer interface exists with SetGoPackages method

## Next Phase Readiness

Plan 06-02 (Tree-sitter integration) can proceed. The AnalysisTarget type is ready to receive Python/TypeScript source files, and the Analyzer interface accepts targets for new language-specific analyzers.
