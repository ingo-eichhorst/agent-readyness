---
phase: "02"
plan: "01"
subsystem: "parser"
tags: ["go-packages", "ast", "type-info", "pipeline", "metrics"]
dependency-graph:
  requires: ["01-03"]
  provides: ["GoPackagesParser", "ParsedPackage", "C1Metrics", "C3Metrics", "C6Metrics", "updated-pipeline-interfaces"]
  affects: ["02-02", "02-03", "02-04", "02-05"]
tech-stack:
  added: ["golang.org/x/tools/go/packages", "fzipp/gocyclo"]
  patterns: ["go/packages type-aware loading", "NeedSyntax|NeedTypes|NeedTypesInfo|NeedForTest"]
key-files:
  created:
    - "internal/parser/parser.go"
    - "internal/parser/parser_test.go"
  modified:
    - "pkg/types/types.go"
    - "internal/pipeline/interfaces.go"
    - "internal/pipeline/pipeline.go"
    - "internal/pipeline/pipeline_test.go"
    - "go.mod"
    - "go.sum"
decisions:
  - id: "02-01-01"
    decision: "NeedForTest flag required for test package identification"
    context: "go/packages ForTest field is empty without NeedForTest mode bit"
  - id: "02-01-02"
    decision: "ParsedPackage is a new type in internal/parser, not an evolution of ParsedFile"
    context: "Clean break since only stubs implemented the old interfaces"
  - id: "02-01-03"
    decision: "Pipeline Parser.Parse takes rootDir string, not []DiscoveredFile"
    context: "go/packages loads from directory, not individual files"
metrics:
  duration: "10 min"
  completed: "2026-01-31"
---

# Phase 2 Plan 1: GoPackagesParser and Typed Metric Structs Summary

**One-liner:** go/packages-backed parser loading ASTs, type info, and import graphs with typed C1/C3/C6 metric result structs and updated pipeline interfaces.

## What Was Done

### Task 1: GoPackagesParser and ParsedPackage type (0648eb8)

Created `internal/parser/parser.go` with:
- `ParsedPackage` struct carrying ID, Name, PkgPath, GoFiles, Syntax (AST), Fset, Types, TypesInfo, Imports, ForTest
- `GoPackagesParser.Parse(rootDir)` that loads all packages via `packages.Load` with `NeedName|NeedFiles|NeedImports|NeedDeps|NeedTypes|NeedSyntax|NeedTypesInfo|NeedForTest`
- Deduplication by PkgPath: source packages kept once, test packages (ForTest != "") added separately
- Packages with errors logged but included if they have partial useful data (AST + types)

Tests verify:
- Parse returns non-empty packages from this repository (15 packages loaded)
- At least one package has non-nil Syntax (AST)
- All packages have non-nil Fset
- Source packages have non-nil Types and TypesInfo
- Test packages are identified via ForTest field (4 test packages found)

### Task 2: Typed metric structs and pipeline interface updates (2b166ec)

Added to `pkg/types/types.go`:
- `MetricSummary` (Avg, Max, MaxEntity)
- `FunctionMetric` (Package, Name, File, Line, Complexity, LineCount)
- `DuplicateBlock` (FileA/B, StartA/B, EndA/B, LineCount)
- `C1Metrics` (CyclomaticComplexity, FunctionLength, FileSize, coupling maps, duplication, functions)
- `C3Metrics` (directory depth, module fanout, circular deps, import complexity, dead exports)
- `DeadExport` (Package, Name, File, Line, Kind)
- `C6Metrics` (test counts, ratio, coverage, isolation, assertion density, test functions)
- `TestFunctionMetric` (Package, Name, File, Line, AssertionCount, HasExternalDep)
- Added `Category` field to `AnalysisResult`

Updated `internal/pipeline/interfaces.go`:
- `Parser.Parse(rootDir string) ([]*parser.ParsedPackage, error)` -- takes directory, not files
- `Analyzer.Analyze(pkgs []*parser.ParsedPackage) (*types.AnalysisResult, error)` -- takes packages, not files
- StubParser returns empty slice; StubAnalyzer returns empty result

Updated `internal/pipeline/pipeline.go`:
- Stage 2 calls `p.parser.Parse(dir)` instead of `p.parser.Parse(result.Files)`
- Stage 3 passes packages to analyzers

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| 02-01-01 | Added NeedForTest to packages.Config.Mode | ForTest field is empty without this flag; needed to distinguish test packages from source |
| 02-01-02 | ParsedPackage as new type in internal/parser | Clean break from ParsedFile; go/packages operates on packages not files |
| 02-01-03 | Parser.Parse takes rootDir, not []DiscoveredFile | go/packages loads from a directory with "./..." pattern |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] NeedForTest mode flag missing**

- **Found during:** Task 1 test verification
- **Issue:** ForTest field was always empty because packages.Config.Mode did not include NeedForTest
- **Fix:** Added `packages.NeedForTest` to the Mode bitmask
- **Files modified:** internal/parser/parser.go
- **Commit:** 0648eb8

## Verification Results

| Check | Result |
|-------|--------|
| `go build ./...` | PASS -- entire project compiles |
| `go test ./... -count=1` | PASS -- all tests pass (discovery, output, parser, pipeline) |
| `go vet ./...` | PASS -- no issues |
| Parser loads real ASTs | PASS -- 15 packages loaded with syntax, types, and type info |
| Test packages identified | PASS -- 4 test packages found via ForTest field |

## Next Phase Readiness

All Phase 2 analyzers (plans 02-02, 02-03, 02-04) can now:
- Import `internal/parser` and receive `[]*ParsedPackage` with full AST, type info, and import graphs
- Return typed results using `C1Metrics`, `C3Metrics`, or `C6Metrics` from `pkg/types`
- Implement the `Analyzer` interface which accepts `[]*parser.ParsedPackage`

Dependencies installed: `golang.org/x/tools` (for go/packages) and `fzipp/gocyclo` (for cyclomatic complexity in C1).
