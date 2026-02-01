---
phase: 06-multi-language-foundation
plan: 03
subsystem: scoring-and-analysis
tags: [c2, semantic-explicitness, scoring, go-ast, map-based-config]

dependency-graph:
  requires: ["06-01"]
  provides: ["C2 Go analyzer", "map-based ScoringConfig", "C2 scoring with breakpoints"]
  affects: ["06-04"]

tech-stack:
  added: []
  patterns: ["metric-extractor-pattern", "map-based-category-config", "loc-weighted-aggregation"]

key-files:
  created:
    - internal/analyzer/c2_semantics.go
    - internal/analyzer/c2_go.go
    - internal/analyzer/c2_go_test.go
  modified:
    - internal/scoring/config.go
    - internal/scoring/config_test.go
    - internal/scoring/scorer.go
    - internal/scoring/scorer_test.go
    - internal/recommend/recommend.go
    - internal/recommend/recommend_test.go
    - pkg/types/types.go

decisions:
  - id: "06-03-01"
    decision: "Map-based ScoringConfig with Categories map[string]CategoryConfig"
    rationale: "Extensible for any number of categories (C1-C7) without code changes"
  - id: "06-03-02"
    decision: "Extractor pattern for metric extraction (metricExtractors map)"
    rationale: "Decouples scoring logic from category-specific metric extraction"
  - id: "06-03-03"
    decision: "Nil safety metric blends interface{}/any usage with nil-check ratio"
    rationale: "Both patterns reflect Go's type safety practices"

metrics:
  duration: "9 min"
  completed: "2026-02-01"
---

# Phase 06 Plan 03: C2 Go Analyzer and Map-Based Scoring Summary

**One-liner:** Map-based ScoringConfig with C2 breakpoints, Go C2 analyzer measuring interface{}/any usage, naming consistency, magic numbers, and nil safety via go/ast.

## What Was Done

### Task 1: Refactor ScoringConfig to map-based categories and add C2 types
- Changed `ScoringConfig` from hardcoded `C1`/`C3`/`C6` fields to `Categories map[string]CategoryConfig`
- Added `Category(name)` accessor method for clean lookup
- Added C2 category config with 5 metrics and breakpoints (weight 0.10):
  - type_annotation_coverage (0.30), naming_consistency (0.25), magic_number_ratio (0.20), type_strictness (0.15), null_safety (0.10)
- Added `C2Metrics` and `C2LanguageMetrics` types to pkg/types
- Refactored scorer from switch/case to extractor pattern (`metricExtractors` map)
- Added `extractC2` function for C2 metric extraction from aggregate
- Updated recommend package to use map-based category lookup
- Updated all YAML config format to use `categories:` key
- All existing scoring tests pass with identical results

### Task 2: Implement C2 Go analyzer
- Created `C2Analyzer` dispatcher with `GoAwareAnalyzer` interface (c2_semantics.go)
- Created `C2GoAnalyzer` with 4 Go-specific metrics using go/ast (c2_go.go):
  - **C2-GO-01 interface{}/any usage:** Detects empty `*ast.InterfaceType` and builtin `any` identifier
  - **C2-GO-02 Naming consistency:** Checks CamelCase exports, camelCase unexports, detects snake_case violations
  - **C2-GO-03 Magic numbers:** Finds numeric literals outside const blocks, excludes 0/1/2
  - **C2-GO-04 Nil safety:** Ratio of nil checks to pointer dereferences per function
- TypeAnnotationCoverage = 100 and TypeStrictness = 1 for Go (statically typed)
- LOC-weighted aggregation ready for multi-language support
- Unit tests verify real Go code analysis on project's own codebase
- Self-analysis results: NamingConsistency=100%, MagicNumberRatio=46.9, NullSafety=66.7%, 141 functions, 5328 LOC

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| 06-03-01 | Map-based ScoringConfig | Extensible for C4/C5/C7 without code changes |
| 06-03-02 | Extractor pattern for scoring | Decouples scoring from category-specific extraction |
| 06-03-03 | Blended nil safety metric | Both interface{}/any usage and nil-check patterns reflect Go type safety |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated recommend package for map-based config**
- **Found during:** Task 1
- **Issue:** recommend.go and recommend_test.go used `cfg.C1`, `cfg.C3`, `cfg.C6` fields which no longer exist
- **Fix:** Updated `getCategoryConfig` to use `cfg.Categories[name]` map lookup, updated test helper
- **Files modified:** internal/recommend/recommend.go, internal/recommend/recommend_test.go

## Verification

1. `go build ./...` -- passes
2. `go test ./...` -- all tests pass
3. ScoringConfig uses `Categories map[string]CategoryConfig` -- confirmed
4. C2 Go analyzer produces correct metrics in unit tests -- confirmed
5. Existing C1/C3/C6 scoring unchanged (verified by all existing tests passing)
6. C2Analyzer and C2GoAnalyzer exist and are unit-tested (pipeline registration is Plan 04)

## Next Phase Readiness

- C2Analyzer is ready to be registered in the pipeline (Plan 04 Task 2)
- Map-based ScoringConfig supports adding C4/C5/C7 categories without structural changes
- Extractor pattern allows registering new extractors via `RegisterExtractor()`
